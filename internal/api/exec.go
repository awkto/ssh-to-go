package api

import (
	"encoding/json"
	"fmt"
	"net/http"
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
	// TimeoutSeconds, when > 0, wraps the command in `timeout` so it can't
	// run forever. 0 means no timeout.
	TimeoutSeconds int `json:"timeout_seconds,omitempty"`
}

// execJobResponse is the JSON shape returned for a job's status.
type execJobResponse struct {
	ID        string    `json:"id"`
	Host      string    `json:"host"`
	Command   string    `json:"command"`
	Status    string    `json:"status"`
	ExitCode  *int      `json:"exit_code,omitempty"`
	Output    string    `json:"output,omitempty"`
	CreatedAt time.Time `json:"created_at"`
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

// RunCommand launches a one-off shell command on a host inside a throwaway
// tmux session and returns a job id immediately. The command keeps running
// on the remote host after this handler returns; callers poll GetExec with
// the id for status, exit code, and output.
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
	if req.TimeoutSeconds < 0 {
		http.Error(w, "timeout_seconds must be >= 0", http.StatusBadRequest)
		return
	}

	hostCfg, errMsg := h.resolveExecHost(req.Host)
	if errMsg != "" {
		http.Error(w, errMsg, http.StatusBadRequest)
		return
	}

	client, err := sshutil.Dial(hostCfg.DialAddress(), hostCfg.User, h.resolveKey(hostCfg))
	if err != nil {
		http.Error(w, fmt.Sprintf("ssh connect failed: %v", err), http.StatusBadGateway)
		return
	}
	defer client.Close()

	id := execjob.NewID()
	if _, err := sshutil.Exec(client, execjob.StartScript(id, req.Command, req.TimeoutSeconds)); err != nil {
		http.Error(w, fmt.Sprintf("launch failed: %v", err), http.StatusInternalServerError)
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

	w.WriteHeader(http.StatusAccepted)
	writeJSON(w, execJobResponse{
		ID:        job.ID,
		Host:      job.Host,
		Command:   job.Command,
		Status:    string(execjob.StatusRunning),
		CreatedAt: job.CreatedAt,
	})
}

// fetchExecStatus reconnects to the job's host and reads its current state,
// exit code, and (unless skipOutput) captured output.
func (h *Handlers) fetchExecStatus(job *execjob.Job, skipOutput bool) (execjob.StatusResult, string, error) {
	hostCfg, ok := h.Hub.GetHostConfig(job.Host)
	if !ok {
		return execjob.StatusResult{}, "", fmt.Errorf("host %q not found", job.Host)
	}
	client, err := sshutil.Dial(hostCfg.DialAddress(), hostCfg.User, h.resolveKey(hostCfg))
	if err != nil {
		return execjob.StatusResult{}, "", fmt.Errorf("ssh connect failed: %w", err)
	}
	defer client.Close()

	statusOut, err := sshutil.Exec(client, execjob.StatusScript(job.ID))
	if err != nil {
		return execjob.StatusResult{}, "", fmt.Errorf("status query failed: %w", err)
	}
	res := execjob.ParseStatus(statusOut)

	var output string
	if !skipOutput && res.Status != execjob.StatusGone {
		// cat is guarded with `|| true`, so a missing file is empty, not an error.
		output, _ = sshutil.Exec(client, execjob.OutputScript(job.ID))
	}
	return res, output, nil
}

// GetExec returns a job's status, exit code, and captured output. Pass
// ?output=false to skip fetching (potentially large) output.
func (h *Handlers) GetExec(w http.ResponseWriter, r *http.Request) {
	if h.ExecJobs == nil {
		http.Error(w, "exec is not available", http.StatusNotImplemented)
		return
	}
	id := r.PathValue("id")
	job, ok := h.ExecJobs.Get(id)
	if !ok {
		http.Error(w, "job not found", http.StatusNotFound)
		return
	}

	skipOutput := r.URL.Query().Get("output") == "false"
	res, output, err := h.fetchExecStatus(job, skipOutput)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}

	writeJSON(w, execJobResponse{
		ID:        job.ID,
		Host:      job.Host,
		Command:   job.Command,
		Status:    string(res.Status),
		ExitCode:  res.ExitCode,
		Output:    output,
		CreatedAt: job.CreatedAt,
	})
}

// GetExecOutput returns the job's captured combined output as plain text —
// convenient for `curl` and for LLM agents that just want the raw result.
func (h *Handlers) GetExecOutput(w http.ResponseWriter, r *http.Request) {
	if h.ExecJobs == nil {
		http.Error(w, "exec is not available", http.StatusNotImplemented)
		return
	}
	id := r.PathValue("id")
	job, ok := h.ExecJobs.Get(id)
	if !ok {
		http.Error(w, "job not found", http.StatusNotFound)
		return
	}
	_, output, err := h.fetchExecStatus(job, false)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Write([]byte(output))
}

// ListExec returns the in-memory index of launched jobs (newest first),
// without querying remote status. Metadata only.
func (h *Handlers) ListExec(w http.ResponseWriter, r *http.Request) {
	if h.ExecJobs == nil {
		http.Error(w, "exec is not available", http.StatusNotImplemented)
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
