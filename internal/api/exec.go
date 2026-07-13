package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/awkto/ssh-to-go/internal/config"
	"github.com/awkto/ssh-to-go/internal/execjob"
	"github.com/awkto/ssh-to-go/internal/sshutil"
)

// execRunReq is the POST /api/exec body.
type execRunReq struct {
	// Command is the shell command to run (may be multi-line). Required.
	Command string `json:"command"`
	// Host is the target host name. Optional — falls back to the configured
	// default host, or the only host when exactly one is registered.
	Host string `json:"host,omitempty"`
	// TimeoutSeconds kills the command after this many seconds (exit code
	// 124). Omitted → execjob.DefaultTimeoutSecs; an explicit 0 disables the
	// timeout entirely (dangerous: the job can run forever).
	TimeoutSeconds *int `json:"timeout_seconds,omitempty"`
	// Cwd is the working directory for the command. Optional — defaults to
	// $HOME. A non-existent directory fails the launch.
	Cwd string `json:"cwd,omitempty"`
	// Env are extra environment variables exported before the command runs.
	// Stored in a 0600 file on the host, never inlined into the command.
	Env map[string]string `json:"env,omitempty"`
	// Stdin, when present, is fed to the command's standard input. Otherwise
	// stdin is /dev/null.
	Stdin *string `json:"stdin,omitempty"`
	// WaitSeconds, when > 0, long-polls server-side: if the job finishes
	// within the window the full result (exit code + output) is returned in
	// this same response. Capped at execjob.MaxWaitSeconds.
	WaitSeconds int `json:"wait_seconds,omitempty"`
}

// execJobResponse is the JSON shape returned for a job's status.
type execJobResponse struct {
	ID          string    `json:"id"`
	Host        string    `json:"host"`
	Command     string    `json:"command"`
	Status      string    `json:"status"`
	ExitCode    *int      `json:"exit_code,omitempty"`
	Stdout      string    `json:"stdout,omitempty"`
	Stderr      string    `json:"stderr,omitempty"`
	StdoutBytes int64     `json:"stdout_bytes,omitempty"`
	StderrBytes int64     `json:"stderr_bytes,omitempty"`
	Truncated   bool      `json:"truncated,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

// resolveExecHost picks the target host for an exec request: the explicit
// name if given, else the configured default host, else the sole host when
// exactly one is registered. Returns the resolved config and a client-facing
// error message when it can't decide.
func (h *Handlers) resolveExecHost(name string) (config.Host, string) {
	if name == "" {
		name = h.Settings.DefaultHost()
	}
	if name == "" {
		hosts := h.Hub.AllHosts()
		if len(hosts) == 1 {
			name = hosts[0].Config.Name
		} else if len(hosts) == 0 {
			return config.Host{}, "no hosts are configured"
		} else {
			return config.Host{}, "no host specified and no default host configured; set a default in Settings or pass \"host\""
		}
	}
	cfg, ok := h.Hub.GetHostConfig(name)
	if !ok {
		return config.Host{}, fmt.Sprintf("host %q not found", name)
	}
	return cfg, ""
}

// dialExecer connects to the host and wraps the client as an execjob.Execer.
// The caller must call the returned closer when done.
func (h *Handlers) dialExecer(hostCfg config.Host) (execjob.Execer, func(), error) {
	client, err := sshutil.Dial(hostCfg.DialAddress(), hostCfg.User, h.resolveKey(hostCfg))
	if err != nil {
		return nil, nil, err
	}
	run := func(script string) (string, error) { return sshutil.Exec(client, script) }
	return run, func() { client.Close() }, nil
}

func (h *Handlers) buildExecResponse(run execjob.Execer, job *execjob.Job,
	res execjob.StatusResult, includeOutput bool, opts execjob.OutputOpts) (execJobResponse, error) {

	resp := execJobResponse{
		ID:        job.ID,
		Host:      job.Host,
		Command:   job.Command,
		Status:    string(res.Status),
		ExitCode:  res.ExitCode,
		CreatedAt: job.CreatedAt,
	}
	if includeOutput && res.Status != execjob.StatusGone {
		out, err := execjob.FetchOutput(run, job.ID, opts)
		if err != nil {
			return resp, err
		}
		resp.Stdout = out.Stdout
		resp.Stderr = out.Stderr
		resp.StdoutBytes = out.StdoutBytes
		resp.StderrBytes = out.StderrBytes
		resp.Truncated = out.Truncated()
	}
	return resp, nil
}

// RunCommand launches a one-off shell command on a host inside a throwaway
// tmux session. Without wait_seconds it returns a job id immediately and the
// command keeps running on the remote host; with wait_seconds it long-polls
// and, when the job finishes in time, returns the completed result directly.
func (h *Handlers) RunCommand(w http.ResponseWriter, r *http.Request) {
	if h.ExecJobs == nil {
		http.Error(w, "exec is not available", http.StatusNotImplemented)
		return
	}

	var req execRunReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if req.Command == "" {
		http.Error(w, "command required", http.StatusBadRequest)
		return
	}
	timeoutSecs := execjob.DefaultTimeoutSecs
	if req.TimeoutSeconds != nil {
		if *req.TimeoutSeconds < 0 {
			http.Error(w, "timeout_seconds must be >= 0", http.StatusBadRequest)
			return
		}
		timeoutSecs = *req.TimeoutSeconds
	}

	hostCfg, errMsg := h.resolveExecHost(req.Host)
	if errMsg != "" {
		http.Error(w, errMsg, http.StatusBadRequest)
		return
	}

	run, closer, err := h.dialExecer(hostCfg)
	if err != nil {
		http.Error(w, fmt.Sprintf("ssh connect failed: %v", err), http.StatusBadGateway)
		return
	}
	defer closer()

	id := execjob.NewID()
	spec := execjob.RunSpec{
		Command:     req.Command,
		TimeoutSecs: timeoutSecs,
		Cwd:         req.Cwd,
		Env:         req.Env,
		Stdin:       req.Stdin,
	}
	if err := execjob.Launch(run, id, spec); err != nil {
		http.Error(w, fmt.Sprintf("launch failed: %v", err), http.StatusBadRequest)
		return
	}

	job := &execjob.Job{
		ID:        id,
		Host:      hostCfg.Name,
		Command:   req.Command,
		Session:   execjob.SessionName(id),
		CreatedAt: time.Now(),
	}
	h.ExecJobs.Add(job)

	if req.WaitSeconds > 0 {
		res, err := execjob.PollStatus(run, id, time.Duration(req.WaitSeconds)*time.Second)
		if err == nil && res.Terminal() {
			resp, err := h.buildExecResponse(run, job, res, true, execjob.OutputOpts{})
			if err == nil {
				writeJSON(w, resp)
				return
			}
		}
		// Fall through: still running (or a transient poll error) — return
		// the async shape and let the caller poll.
	}

	w.WriteHeader(http.StatusAccepted)
	writeJSON(w, execJobResponse{
		ID:        job.ID,
		Host:      job.Host,
		Command:   job.Command,
		Status:    string(execjob.StatusRunning),
		CreatedAt: job.CreatedAt,
	})
}

// execerForJob resolves a job's host and connects to it.
func (h *Handlers) execerForJob(job *execjob.Job) (execjob.Execer, func(), error) {
	hostCfg, ok := h.Hub.GetHostConfig(job.Host)
	if !ok {
		return nil, nil, fmt.Errorf("host %q not found", job.Host)
	}
	run, closer, err := h.dialExecer(hostCfg)
	if err != nil {
		return nil, nil, fmt.Errorf("ssh connect failed: %w", err)
	}
	return run, closer, nil
}

// GetExec returns a job's status, exit code, and a bounded slice of its
// output. Query params: output=false (status only), tail_lines=N,
// max_output_bytes=N, wait_seconds=N (long-poll until terminal).
func (h *Handlers) GetExec(w http.ResponseWriter, r *http.Request) {
	if h.ExecJobs == nil {
		http.Error(w, "exec is not available", http.StatusNotImplemented)
		return
	}
	job, ok := h.ExecJobs.Get(r.PathValue("id"))
	if !ok {
		http.Error(w, "job not found", http.StatusNotFound)
		return
	}

	q := r.URL.Query()
	includeOutput := q.Get("output") != "false"
	opts := execjob.OutputOpts{}
	opts.TailLines, _ = strconv.Atoi(q.Get("tail_lines"))
	opts.MaxBytes, _ = strconv.Atoi(q.Get("max_output_bytes"))
	waitSecs, _ := strconv.Atoi(q.Get("wait_seconds"))

	run, closer, err := h.execerForJob(job)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	defer closer()

	res, err := execjob.PollStatus(run, job.ID, time.Duration(waitSecs)*time.Second)
	if err != nil {
		http.Error(w, fmt.Sprintf("status query failed: %v", err), http.StatusBadGateway)
		return
	}
	resp, err := h.buildExecResponse(run, job, res, includeOutput, opts)
	if err != nil {
		http.Error(w, fmt.Sprintf("output fetch failed: %v", err), http.StatusBadGateway)
		return
	}
	writeJSON(w, resp)
}

// GetExecOutput returns one of the job's output streams as plain text —
// convenient for `curl`. Query params: stream=out|err (default out),
// tail_lines=N, max_output_bytes=N.
func (h *Handlers) GetExecOutput(w http.ResponseWriter, r *http.Request) {
	if h.ExecJobs == nil {
		http.Error(w, "exec is not available", http.StatusNotImplemented)
		return
	}
	job, ok := h.ExecJobs.Get(r.PathValue("id"))
	if !ok {
		http.Error(w, "job not found", http.StatusNotFound)
		return
	}
	q := r.URL.Query()
	opts := execjob.OutputOpts{}
	opts.TailLines, _ = strconv.Atoi(q.Get("tail_lines"))
	opts.MaxBytes, _ = strconv.Atoi(q.Get("max_output_bytes"))

	run, closer, err := h.execerForJob(job)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	defer closer()

	out, err := execjob.FetchOutput(run, job.ID, opts)
	if err != nil {
		http.Error(w, fmt.Sprintf("output fetch failed: %v", err), http.StatusBadGateway)
		return
	}
	body := out.Stdout
	if q.Get("stream") == "err" {
		body = out.Stderr
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Write([]byte(body))
}

// KillExec stops a running job by signalling its process group. The job
// resolves to a real terminal state with an exit code. ?force=true escalates
// to SIGKILL after a short grace period.
func (h *Handlers) KillExec(w http.ResponseWriter, r *http.Request) {
	if h.ExecJobs == nil {
		http.Error(w, "exec is not available", http.StatusNotImplemented)
		return
	}
	job, ok := h.ExecJobs.Get(r.PathValue("id"))
	if !ok {
		http.Error(w, "job not found", http.StatusNotFound)
		return
	}
	run, closer, err := h.execerForJob(job)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	defer closer()

	result, err := execjob.Kill(run, job.ID, r.URL.Query().Get("force") == "true")
	if err != nil {
		http.Error(w, fmt.Sprintf("kill failed: %v", err), http.StatusBadGateway)
		return
	}
	writeJSON(w, map[string]string{"id": job.ID, "result": result})
}

// ListExec returns jobs newest first. By default it reports the in-memory
// index (metadata only, no remote calls). With ?remote=true it scans the
// host's job directory instead, which also covers jobs launched by other
// server instances or evicted from the index. ?host= selects the host for
// the remote scan.
func (h *Handlers) ListExec(w http.ResponseWriter, r *http.Request) {
	if h.ExecJobs == nil {
		http.Error(w, "exec is not available", http.StatusNotImplemented)
		return
	}
	if r.URL.Query().Get("remote") == "true" {
		hostCfg, errMsg := h.resolveExecHost(r.URL.Query().Get("host"))
		if errMsg != "" {
			http.Error(w, errMsg, http.StatusBadRequest)
			return
		}
		run, closer, err := h.dialExecer(hostCfg)
		if err != nil {
			http.Error(w, fmt.Sprintf("ssh connect failed: %v", err), http.StatusBadGateway)
			return
		}
		defer closer()
		jobs, err := execjob.ListRemote(run)
		if err != nil {
			http.Error(w, fmt.Sprintf("list failed: %v", err), http.StatusBadGateway)
			return
		}
		if jobs == nil {
			jobs = []execjob.RemoteJob{}
		}
		writeJSON(w, jobs)
		return
	}

	jobs := h.ExecJobs.List()
	out := make([]execJobResponse, 0, len(jobs))
	for _, j := range jobs {
		out = append(out, execJobResponse{
			ID:        j.ID,
			Host:      j.Host,
			Command:   j.Command,
			CreatedAt: j.CreatedAt,
		})
	}
	writeJSON(w, out)
}
