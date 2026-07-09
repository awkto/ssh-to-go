package execjob

import (
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"
)

// SessionPrefix names the throwaway tmux sessions this package creates.
// Callers can filter it out of normal session listings if desired.
const SessionPrefix = "stg-exec-"

// remoteDir is the per-job directory under $HOME on the remote host that
// holds the command script, its combined output, and the exit code.
func remoteDir(id string) string {
	return "$HOME/.ssh-to-go/exec/" + id
}

// SessionName returns the throwaway tmux session name for a job id.
func SessionName(id string) string {
	return SessionPrefix + id
}

// StartScript builds the shell script (run on the remote host's login
// shell) that materializes the job directory and launches the command in a
// detached tmux session.
//
// The user command and a small runner are transferred base64-encoded so
// the command can contain arbitrary characters (quotes, newlines, $, etc.)
// without any shell-quoting hazard. The runner captures combined output to
// out and the exit code to rc, then exits — which ends the single-window
// tmux session, so "session gone" reliably means "command finished".
//
// timeoutSecs, when > 0, wraps the command in coreutils `timeout` so a
// runaway command can't occupy the host forever (exit code 124 on timeout).
func StartScript(id, command string, timeoutSecs int) string {
	dir := remoteDir(id)
	sess := SessionName(id)

	timeoutPrefix := ""
	if timeoutSecs > 0 {
		timeoutPrefix = "timeout " + strconv.Itoa(timeoutSecs) + " "
	}
	// Self-contained runner: resolves its own directory so it needs no
	// arguments and no interpolation of the (possibly space-containing)
	// remote path at launch time.
	runner := strings.Join([]string{
		"#!/bin/bash",
		`D="$(cd "$(dirname "$0")" && pwd)"`,
		timeoutPrefix + `bash "$D/cmd.sh" > "$D/out" 2>&1`,
		`echo $? > "$D/rc"`,
		"",
	}, "\n")

	cmdB64 := base64.StdEncoding.EncodeToString([]byte(command))
	runB64 := base64.StdEncoding.EncodeToString([]byte(runner))

	return strings.Join([]string{
		fmt.Sprintf(`D="%s"`, dir),
		`mkdir -p "$D"`,
		fmt.Sprintf(`printf %%s '%s' | base64 -d > "$D/cmd.sh"`, cmdB64),
		fmt.Sprintf(`printf %%s '%s' | base64 -d > "$D/run.sh"`, runB64),
		fmt.Sprintf(`tmux new-session -d -s '%s' "bash \"$D/run.sh\""`, sess),
	}, "\n")
}

// StatusScript builds a small, output-safe script that reports the job's
// state (gone/running/finished) and recorded exit code as key=value lines.
// It never echoes the command output, so parsing can't be confused by user
// data.
func StatusScript(id string) string {
	dir := remoteDir(id)
	sess := SessionName(id)
	return strings.Join([]string{
		fmt.Sprintf(`D="%s"`, dir),
		`if [ ! -d "$D" ]; then echo STATE=gone; exit 0; fi`,
		fmt.Sprintf(`if tmux has-session -t '%s' 2>/dev/null; then echo STATE=running; else echo STATE=finished; fi`, sess),
		`printf 'RC='; cat "$D/rc" 2>/dev/null | tr -d '[:space:]'; echo`,
	}, "\n")
}

// OutputScript builds a script that emits the job's captured combined
// output verbatim (empty if the file is missing).
func OutputScript(id string) string {
	dir := remoteDir(id)
	return fmt.Sprintf(`cat "%s/out" 2>/dev/null || true`, dir)
}

// StatusResult is the parsed form of StatusScript's output.
type StatusResult struct {
	Status   Status
	ExitCode *int
}

// ParseStatus parses the key=value output of StatusScript. An absent or
// unparseable exit code leaves ExitCode nil (e.g. a job still running, or a
// finished job whose rc file wasn't flushed yet).
func ParseStatus(out string) StatusResult {
	res := StatusResult{Status: StatusGone}
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(line, "STATE="):
			switch strings.TrimPrefix(line, "STATE=") {
			case "running":
				res.Status = StatusRunning
			case "finished":
				res.Status = StatusFinished
			case "gone":
				res.Status = StatusGone
			}
		case strings.HasPrefix(line, "RC="):
			v := strings.TrimSpace(strings.TrimPrefix(line, "RC="))
			if v != "" {
				if n, err := strconv.Atoi(v); err == nil {
					res.ExitCode = &n
				}
			}
		}
	}
	return res
}
