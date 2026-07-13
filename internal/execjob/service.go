package execjob

import (
	"strings"
	"time"
)

// Execer runs a shell script on the remote host and returns its combined
// output. Both the REST and MCP layers wrap an established SSH client in
// one of these, so all job orchestration lives here and stays testable
// without a network.
type Execer func(script string) (string, error)

// Launch materializes and starts a job on the host.
func Launch(run Execer, id string, spec RunSpec) error {
	script, err := StartScript(id, spec)
	if err != nil {
		return err
	}
	_, err = run(script)
	return err
}

// FetchStatus reads a job's current state and exit code.
func FetchStatus(run Execer, id string) (StatusResult, error) {
	out, err := run(StatusScript(id))
	if err != nil {
		return StatusResult{}, err
	}
	return ParseStatus(out), nil
}

// FetchOutput reads a bounded slice of a job's output streams.
func FetchOutput(run Execer, id string, opts OutputOpts) (OutputResult, error) {
	out, err := run(OutputScript(id, opts))
	if err != nil {
		return OutputResult{}, err
	}
	return ParseOutput(out)
}

// PollStatus polls a job until it reaches a terminal state or the wait
// budget runs out, whichever comes first, and returns the last observed
// status. A zero or negative wait is a single immediate check. Polling
// reuses the caller's connection with gentle backoff, so a short command
// resolves in one round trip from the client's point of view.
func PollStatus(run Execer, id string, wait time.Duration) (StatusResult, error) {
	if wait > time.Duration(MaxWaitSeconds)*time.Second {
		wait = time.Duration(MaxWaitSeconds) * time.Second
	}
	deadline := time.Now().Add(wait)
	delay := 300 * time.Millisecond
	for {
		res, err := FetchStatus(run, id)
		if err != nil {
			return StatusResult{}, err
		}
		remaining := time.Until(deadline)
		if res.Terminal() || remaining <= 0 {
			return res, nil
		}
		if delay > remaining {
			delay = remaining
		}
		time.Sleep(delay)
		if delay < 2*time.Second {
			delay *= 2
		}
	}
}

// Kill signals a running job's process group and returns the remote result
// keyword: killed, killed_session, already_finished, or gone.
func Kill(run Execer, id string, force bool) (string, error) {
	out, err := run(KillScript(id, force))
	if err != nil {
		return "", err
	}
	for _, line := range strings.Split(out, "\n") {
		if v, ok := strings.CutPrefix(strings.TrimSpace(line), "RESULT="); ok {
			return v, nil
		}
	}
	return "unknown", nil
}

// ListRemote inventories all job dirs on the host, newest first.
func ListRemote(run Execer) ([]RemoteJob, error) {
	out, err := run(ListScript())
	if err != nil {
		return nil, err
	}
	return ParseJobList(out), nil
}
