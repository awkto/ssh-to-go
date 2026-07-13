package execjob

import (
	"encoding/base64"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

// SessionPrefix names the throwaway tmux sessions this package creates.
// Callers can filter it out of normal session listings if desired.
const SessionPrefix = "stg-exec-"

// Tunables for the exec job lifecycle. These are deliberately constants, not
// per-request knobs: they bound worst-case behavior on the remote host.
const (
	// DefaultTimeoutSecs bounds a job when the caller doesn't say otherwise.
	// Generous enough for long `claude -p` runs and builds; an explicit
	// timeout_seconds of 0 opts out entirely (documented as dangerous).
	DefaultTimeoutSecs = 3600
	// KillGraceSecs is how long after SIGTERM the runner waits before
	// escalating to SIGKILL (`timeout -k`).
	KillGraceSecs = 10
	// DefaultMaxOutputBytes caps each returned stream. The primary consumer
	// is an LLM agent with a finite context window — a multi-MB payload is
	// fatal there, so truncation is the default and callers page explicitly.
	DefaultMaxOutputBytes = 256 * 1024
	// MaxOutputBytesLimit is the hard ceiling a caller can request per stream.
	MaxOutputBytesLimit = 4 * 1024 * 1024
	// GCTTLMinutes: job dirs older than this are pruned opportunistically at
	// the next launch on the same host (live sessions are never pruned).
	GCTTLMinutes = 24 * 60
	// MaxWaitSeconds caps server-side long-polling so a request can't hold a
	// connection open indefinitely.
	MaxWaitSeconds = 60
)

// RunSpec describes a one-off command launch.
type RunSpec struct {
	// Command is the shell script to run (may be multi-line).
	Command string
	// TimeoutSecs kills the job after this many seconds (exit 124).
	// 0 means unlimited — callers should default to DefaultTimeoutSecs.
	TimeoutSecs int
	// Cwd, when set, is the working directory. Validated at launch time.
	Cwd string
	// Env are extra environment variables, exported before the command runs.
	// Written to a 0600 file on the host, never inlined into cmd.sh.
	Env map[string]string
	// Stdin, when non-nil, is fed to the command's stdin. Otherwise stdin is
	// /dev/null so prompting commands fail fast instead of hanging.
	Stdin *string
}

// remoteDir is the per-job directory under $HOME on the remote host that
// holds the command script, its output streams, and the exit code.
func remoteDir(id string) string {
	return "$HOME/.ssh-to-go/exec/" + id
}

// SessionName returns the throwaway tmux session name for a job id.
func SessionName(id string) string {
	return SessionPrefix + id
}

var envKeyRe = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

// shellSingleQuote wraps s in single quotes, escaping embedded quotes, so it
// is safe to paste into a shell script.
func shellSingleQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

// envFile renders Env as a sourceable file of export statements with
// single-quoted values, sorted for determinism.
func envFile(env map[string]string) string {
	keys := make([]string, 0, len(env))
	for k := range env {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var b strings.Builder
	for _, k := range keys {
		b.WriteString("export " + k + "=" + shellSingleQuote(env[k]) + "\n")
	}
	return b.String()
}

// writeB64 emits a launcher line that materializes content into $D/name.
// Base64 transfer means arbitrary bytes (quotes, newlines, $) can't break
// out of the launcher script.
func writeB64(name, content string) string {
	b64 := base64.StdEncoding.EncodeToString([]byte(content))
	return fmt.Sprintf(`printf %%s '%s' | base64 -d > "$D/%s"`, b64, name)
}

// runnerScript builds the per-job run.sh. Invariants it maintains:
//
//   - umask 077: all artifacts are 0600 (job dirs hold every command and its
//     output — treat them like credentials).
//   - stdin is /dev/null (or the provided stdin file), never the pane TTY, so
//     prompting commands get EOF instead of hanging until timeout.
//   - the command always runs under `timeout -k`, in its own process group
//     (timeout calls setpgid), with SIGKILL escalation; `timeout 0` disables
//     the clock but keeps the group semantics, so on exit the whole group is
//     reaped and only a deliberate setsid can outlive the job.
//   - stdout and stderr are captured separately (out / err).
//   - a pid file records the process-group id for kill_command.
//   - rc is written via tmp+rename so a poll can never observe a partial
//     exit code, and only ever after the group is reaped.
func runnerScript(timeoutSecs int) string {
	if timeoutSecs < 0 {
		timeoutSecs = 0
	}
	return strings.Join([]string{
		"#!/bin/bash",
		"umask 077",
		`D="$(cd "$(dirname "$0")" && pwd)"`,
		// Documented contract: non-interactive bash, no profile sourcing,
		// inherited base PATH plus the common user bin dirs.
		`export PATH="$HOME/.local/bin:$HOME/bin:$PATH"`,
		`[ -f "$D/env" ] && . "$D/env"`,
		`if [ -f "$D/cwd" ]; then`,
		`  cd "$(cat "$D/cwd")" || { echo "working directory no longer exists" > "$D/err"; printf %s 127 > "$D/rc.tmp" && mv "$D/rc.tmp" "$D/rc"; exit 1; }`,
		`else`,
		`  cd "$HOME" 2>/dev/null || true`,
		`fi`,
		`IN=/dev/null`,
		`[ -f "$D/stdin" ] && IN="$D/stdin"`,
		fmt.Sprintf(`timeout -k %d %d bash "$D/cmd.sh" < "$IN" > "$D/out" 2> "$D/err" &`, KillGraceSecs, timeoutSecs),
		`pid=$!`,
		`printf %s "$pid" > "$D/pid"`,
		`wait "$pid"`,
		`rc=$?`,
		`kill -KILL -"$pid" 2>/dev/null`,
		`printf %s "$rc" > "$D/rc.tmp" && mv "$D/rc.tmp" "$D/rc"`,
		"",
	}, "\n")
}

// StartScript builds the shell script (run on the remote host's login shell)
// that heals permissions, prunes expired job dirs, materializes the job
// directory, and launches the command in a detached tmux session.
//
// The command and every user-supplied value are transferred base64-encoded so
// they can contain arbitrary characters without any shell-quoting hazard. The
// runner exits when the command does — which ends the single-window tmux
// session, so "session gone" reliably means "runner finished".
//
// A non-existent Cwd fails the launch loudly (exit 3 with the path on
// stderr) rather than silently running somewhere unexpected.
func StartScript(id string, spec RunSpec) (string, error) {
	for k := range spec.Env {
		if !envKeyRe.MatchString(k) {
			return "", fmt.Errorf("invalid env variable name %q", k)
		}
	}

	lines := []string{
		"umask 077",
		`B="$HOME/.ssh-to-go"`,
		`E="$B/exec"`,
		`mkdir -p "$E"`,
		// Self-heal artifacts created by older releases with the default
		// umask; cheap once GC keeps the tree small.
		`chmod 700 "$B" "$E" 2>/dev/null`,
		`chmod -R go-rwx "$E" 2>/dev/null`,
		// Opportunistic GC: prune expired job dirs, never live sessions.
		fmt.Sprintf(`find "$E" -mindepth 1 -maxdepth 1 -type d -mmin +%d 2>/dev/null | while read -r d; do`, GCTTLMinutes),
		`  i="${d##*/}"`,
		fmt.Sprintf(`  tmux has-session -t "%s$i" 2>/dev/null || rm -rf "$d"`, SessionPrefix),
		`done`,
		fmt.Sprintf(`D="%s"`, remoteDir(id)),
		`mkdir -p "$D"`,
		writeB64("cmd.sh", spec.Command),
		writeB64("run.sh", runnerScript(spec.TimeoutSecs)),
	}
	if spec.Cwd != "" {
		lines = append(lines,
			writeB64("cwd", spec.Cwd),
			`C="$(cat "$D/cwd")"`,
			`if [ ! -d "$C" ]; then echo "cwd not found: $C" >&2; rm -rf "$D"; exit 3; fi`,
		)
	}
	if len(spec.Env) > 0 {
		lines = append(lines, writeB64("env", envFile(spec.Env)))
	}
	if spec.Stdin != nil {
		lines = append(lines, writeB64("stdin", *spec.Stdin))
	}
	lines = append(lines,
		fmt.Sprintf(`tmux new-session -d -s '%s' "bash \"$D/run.sh\""`, SessionName(id)),
	)
	return strings.Join(lines, "\n"), nil
}

// StatusScript builds a small, output-safe script that reports the job's
// state and recorded exit code as key=value lines. It never echoes command
// output, so parsing can't be confused by user data.
//
// State decision: rc present → finished; else session alive → running; else
// the runner died before recording an exit code → crashed. A job is never
// reported finished without an exit code.
func StatusScript(id string) string {
	return strings.Join([]string{
		fmt.Sprintf(`D="%s"`, remoteDir(id)),
		`if [ ! -d "$D" ]; then echo STATE=gone; exit 0; fi`,
		`RC="$(cat "$D/rc" 2>/dev/null | tr -d '[:space:]')"`,
		`if [ -n "$RC" ]; then`,
		`  echo STATE=finished`,
		`  echo "RC=$RC"`,
		fmt.Sprintf(`elif tmux has-session -t '%s' 2>/dev/null; then`, SessionName(id)),
		`  echo STATE=running`,
		`else`,
		`  echo STATE=crashed`,
		`fi`,
	}, "\n")
}

// OutputOpts bounds what OutputScript returns per stream.
type OutputOpts struct {
	// TailLines, when > 0, returns only the last N lines of each stream
	// (still subject to the byte cap).
	TailLines int
	// MaxBytes caps each returned stream; <= 0 selects
	// DefaultMaxOutputBytes. Clamped to MaxOutputBytesLimit.
	MaxBytes int
}

func (o OutputOpts) maxBytes() int {
	m := o.MaxBytes
	if m <= 0 {
		m = DefaultMaxOutputBytes
	}
	if m > MaxOutputBytesLimit {
		m = MaxOutputBytesLimit
	}
	return m
}

// OutputScript builds a script that reports each stream's total size and a
// bounded, base64-encoded slice of its content (the tail — for logs the end
// is almost always what matters). Markers contain '_' so they can never
// collide with base64 payload lines.
func OutputScript(id string, opts OutputOpts) string {
	max := opts.maxBytes()
	slice := func(file string) string {
		if opts.TailLines > 0 {
			return fmt.Sprintf(`tail -n %d "$D/%s" 2>/dev/null | tail -c %d | base64`, opts.TailLines, file, max)
		}
		return fmt.Sprintf(`tail -c %d "$D/%s" 2>/dev/null | base64`, max, file)
	}
	stream := func(label, file string) []string {
		return []string{
			fmt.Sprintf(`T=0; [ -f "$D/%s" ] && T="$(wc -c < "$D/%s" | tr -d '[:space:]')"`, file, file),
			fmt.Sprintf(`echo "%s_TOTAL=$T"`, label),
			fmt.Sprintf(`echo %s_BEGIN`, label),
			slice(file),
			fmt.Sprintf(`echo %s_END`, label),
		}
	}
	lines := []string{fmt.Sprintf(`D="%s"`, remoteDir(id))}
	lines = append(lines, stream("OUT", "out")...)
	lines = append(lines, stream("ERR", "err")...)
	return strings.Join(lines, "\n")
}

// KillScript builds a script that stops a running job by signalling its
// process group. The runner is left alive to reap the group and record the
// exit code, so a killed job resolves to a real terminal state (finished,
// rc 143/137) rather than hanging in running or crashing without a code.
// Jobs launched by older releases have no pid file; for those the tmux
// session is killed instead (they resolve to crashed).
func KillScript(id string, force bool) string {
	lines := []string{
		fmt.Sprintf(`D="%s"`, remoteDir(id)),
		`if [ ! -d "$D" ]; then echo RESULT=gone; exit 0; fi`,
		`RC="$(cat "$D/rc" 2>/dev/null | tr -d '[:space:]')"`,
		`if [ -n "$RC" ]; then echo RESULT=already_finished; exit 0; fi`,
		`P="$(cat "$D/pid" 2>/dev/null | tr -d '[:space:]')"`,
		`if [ -n "$P" ]; then`,
		`  kill -TERM -"$P" 2>/dev/null || kill -TERM "$P" 2>/dev/null`,
	}
	if force {
		lines = append(lines,
			`  sleep 2`,
			`  kill -KILL -"$P" 2>/dev/null || kill -KILL "$P" 2>/dev/null`,
		)
	}
	lines = append(lines,
		`  echo RESULT=killed`,
		`else`,
		fmt.Sprintf(`  tmux kill-session -t '%s' 2>/dev/null`, SessionName(id)),
		`  echo RESULT=killed_session`,
		`fi`,
	)
	return strings.Join(lines, "\n")
}

// ListScript builds a script that inventories every job dir on the host —
// including jobs launched by other server instances or evicted from the
// in-memory index — as pipe-delimited JOB lines. The command preview is
// base64-encoded so user data can't break the framing.
func ListScript() string {
	return strings.Join([]string{
		`E="$HOME/.ssh-to-go/exec"`,
		`[ -d "$E" ] || exit 0`,
		`for d in "$E"/*/; do`,
		`  [ -d "$d" ] || continue`,
		`  i="${d%/}"; i="${i##*/}"`,
		`  RC="$(cat "${d}rc" 2>/dev/null | tr -d '[:space:]')"`,
		`  if [ -n "$RC" ]; then S=finished`,
		fmt.Sprintf(`  elif tmux has-session -t "%s$i" 2>/dev/null; then S=running`, SessionPrefix),
		`  else S=crashed; fi`,
		`  OB=0; [ -f "${d}out" ] && OB="$(wc -c < "${d}out" | tr -d '[:space:]')"`,
		`  EB=0; [ -f "${d}err" ] && EB="$(wc -c < "${d}err" | tr -d '[:space:]')"`,
		`  MT="$(stat -c %Y "${d}cmd.sh" 2>/dev/null || echo 0)"`,
		`  CB="$(head -c 120 "${d}cmd.sh" 2>/dev/null | base64 | tr -d '\n')"`,
		`  echo "JOB|$i|$S|$RC|$OB|$EB|$MT|$CB"`,
		`done`,
	}, "\n")
}

// StatusResult is the parsed form of StatusScript's output.
type StatusResult struct {
	Status   Status
	ExitCode *int
}

// Terminal reports whether the job has reached a final state.
func (r StatusResult) Terminal() bool {
	return r.Status != StatusRunning
}

// ParseStatus parses the key=value output of StatusScript. A crashed job —
// or a "finished" one whose rc is missing or unparseable — reports exit code
// -1: a failure must never be mistakable for success.
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
			case "crashed":
				res.Status = StatusCrashed
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
	if res.Status == StatusFinished && res.ExitCode == nil {
		res.Status = StatusCrashed
	}
	if res.Status == StatusCrashed {
		crashed := -1
		res.ExitCode = &crashed
	}
	return res
}

// OutputResult is the parsed form of OutputScript's output.
type OutputResult struct {
	Stdout      string
	Stderr      string
	StdoutBytes int64 // total size on disk, not the returned slice
	StderrBytes int64
}

// Truncated reports whether either returned stream is a subset of what's on
// disk (byte cap hit, or tail_lines requested).
func (r OutputResult) Truncated() bool {
	return int64(len(r.Stdout)) < r.StdoutBytes || int64(len(r.Stderr)) < r.StderrBytes
}

// ParseOutput parses OutputScript's marker-framed output.
func ParseOutput(out string) (OutputResult, error) {
	var res OutputResult
	var b64 strings.Builder
	collect := "" // which stream we're inside, if any
	finish := func() error {
		raw := b64.String()
		b64.Reset()
		dec, err := base64.StdEncoding.DecodeString(raw)
		if err != nil {
			return fmt.Errorf("decode %s stream: %w", collect, err)
		}
		if collect == "OUT" {
			res.Stdout = string(dec)
		} else {
			res.Stderr = string(dec)
		}
		return nil
	}
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(line, "OUT_TOTAL="):
			res.StdoutBytes, _ = strconv.ParseInt(strings.TrimPrefix(line, "OUT_TOTAL="), 10, 64)
		case strings.HasPrefix(line, "ERR_TOTAL="):
			res.StderrBytes, _ = strconv.ParseInt(strings.TrimPrefix(line, "ERR_TOTAL="), 10, 64)
		case line == "OUT_BEGIN":
			collect = "OUT"
		case line == "ERR_BEGIN":
			collect = "ERR"
		case line == "OUT_END" || line == "ERR_END":
			if err := finish(); err != nil {
				return res, err
			}
			collect = ""
		default:
			if collect != "" {
				b64.WriteString(line)
			}
		}
	}
	return res, nil
}

// RemoteJob is one entry from ListScript: a job dir as it exists on the
// host, independent of the server's in-memory index.
type RemoteJob struct {
	ID             string    `json:"id"`
	Status         Status    `json:"status"`
	ExitCode       *int      `json:"exit_code,omitempty"`
	StdoutBytes    int64     `json:"stdout_bytes"`
	StderrBytes    int64     `json:"stderr_bytes"`
	StartedAt      time.Time `json:"started_at"`
	CommandPreview string    `json:"command_preview"`
}

// ParseJobList parses ListScript's JOB lines, newest first.
func ParseJobList(out string) []RemoteJob {
	var jobs []RemoteJob
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "JOB|") {
			continue
		}
		f := strings.Split(line, "|")
		if len(f) != 8 {
			continue
		}
		j := RemoteJob{ID: f[1]}
		switch f[2] {
		case "finished":
			j.Status = StatusFinished
		case "running":
			j.Status = StatusRunning
		default:
			j.Status = StatusCrashed
		}
		if n, err := strconv.Atoi(f[3]); err == nil {
			j.ExitCode = &n
		} else if j.Status == StatusCrashed {
			crashed := -1
			j.ExitCode = &crashed
		}
		j.StdoutBytes, _ = strconv.ParseInt(f[4], 10, 64)
		j.StderrBytes, _ = strconv.ParseInt(f[5], 10, 64)
		if sec, err := strconv.ParseInt(f[6], 10, 64); err == nil && sec > 0 {
			j.StartedAt = time.Unix(sec, 0).UTC()
		}
		if dec, err := base64.StdEncoding.DecodeString(f[7]); err == nil {
			j.CommandPreview = string(dec)
		}
		jobs = append(jobs, j)
	}
	sort.Slice(jobs, func(a, b int) bool {
		return jobs[a].StartedAt.After(jobs[b].StartedAt)
	})
	return jobs
}
