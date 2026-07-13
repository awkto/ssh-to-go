// Package mcp implements Model Context Protocol (JSON-RPC 2.0 over SSE)
// for ssh-to-go, providing AI tool integrations for session management.
package mcp

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/awkto/ssh-to-go/internal/auth"
	"github.com/awkto/ssh-to-go/internal/config"
	"github.com/awkto/ssh-to-go/internal/execjob"
	"github.com/awkto/ssh-to-go/internal/hub"
	"github.com/awkto/ssh-to-go/internal/keystore"
	"github.com/awkto/ssh-to-go/internal/sshutil"
	"github.com/awkto/ssh-to-go/internal/tmux"

	"crypto/rand"
	"encoding/hex"
)

// Tool defines an MCP tool exposed to clients.
type Tool struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	InputSchema InputSchema `json:"inputSchema"`
}

type InputSchema struct {
	Type       string                    `json:"type"`
	Properties map[string]PropertySchema `json:"properties"`
	Required   []string                  `json:"required,omitempty"`
}

type PropertySchema struct {
	Type        string `json:"type"`
	Description string `json:"description"`
}

var tools = []Tool{
	{
		Name:        "list_sessions",
		Description: "List all tmux sessions across all hosts. Returns host name, session name, window count, creation time, and attached status.",
		InputSchema: InputSchema{Type: "object", Properties: map[string]PropertySchema{}},
	},
	{
		Name:        "list_hosts",
		Description: "List all configured SSH hosts with their online status, tmux version, session count, and OS.",
		InputSchema: InputSchema{Type: "object", Properties: map[string]PropertySchema{}},
	},
	{
		Name:        "create_session",
		Description: "Create a new tmux session on a host.",
		InputSchema: InputSchema{
			Type: "object",
			Properties: map[string]PropertySchema{
				"host": {Type: "string", Description: "Host name to create the session on"},
				"name": {Type: "string", Description: "Name for the new tmux session"},
				"cwd":  {Type: "string", Description: "Optional working directory for the session"},
			},
			Required: []string{"host", "name"},
		},
	},
	{
		Name:        "kill_session",
		Description: "Kill (destroy) a tmux session on a host.",
		InputSchema: InputSchema{
			Type: "object",
			Properties: map[string]PropertySchema{
				"host":    {Type: "string", Description: "Host name"},
				"session": {Type: "string", Description: "Session name to kill"},
			},
			Required: []string{"host", "session"},
		},
	},
	{
		Name:        "rename_session",
		Description: "Rename a tmux session on a host.",
		InputSchema: InputSchema{
			Type: "object",
			Properties: map[string]PropertySchema{
				"host":     {Type: "string", Description: "Host name"},
				"session":  {Type: "string", Description: "Current session name"},
				"new_name": {Type: "string", Description: "New session name"},
			},
			Required: []string{"host", "session", "new_name"},
		},
	},
	{
		Name:        "detach_clients",
		Description: "Detach all tmux clients from a session. The caller's WebSocket relay will auto-reconnect.",
		InputSchema: InputSchema{
			Type: "object",
			Properties: map[string]PropertySchema{
				"host":    {Type: "string", Description: "Host name"},
				"session": {Type: "string", Description: "Session name"},
			},
			Required: []string{"host", "session"},
		},
	},
	{
		Name:        "scan_host",
		Description: "Force an immediate poll of a specific host to refresh its session list.",
		InputSchema: InputSchema{
			Type: "object",
			Properties: map[string]PropertySchema{
				"host": {Type: "string", Description: "Host name to scan"},
			},
			Required: []string{"host"},
		},
	},
	{
		Name:        "scan_all",
		Description: "Force an immediate poll of all hosts to refresh session lists.",
		InputSchema: InputSchema{Type: "object", Properties: map[string]PropertySchema{}},
	},
	{
		Name:        "health_check",
		Description: "Check ssh-to-go service health and return host count and version.",
		InputSchema: InputSchema{Type: "object", Properties: map[string]PropertySchema{}},
	},
	{
		Name:        "run_command",
		Description: "Run a one-off shell command on a host in a throwaway detached tmux session. Runs non-interactively (no shell profile, stdin from /dev/null unless 'stdin' is given), so long tasks (e.g. 'claude -p ...') keep running after this call returns. Pass wait_seconds to get short commands' full result (exit code + output) in this single call; otherwise poll with get_command using the returned job id.",
		InputSchema: InputSchema{
			Type: "object",
			Properties: map[string]PropertySchema{
				"command":         {Type: "string", Description: "Shell command to run (may be multi-line)"},
				"host":            {Type: "string", Description: "Target host name. Optional — defaults to the configured default host, or the only host if just one is registered."},
				"timeout_seconds": {Type: "number", Description: "Kill the command after this many seconds (exit code 124, SIGKILL escalation after 10s). Default 3600. An explicit 0 disables the timeout — dangerous, the job can run forever."},
				"cwd":             {Type: "string", Description: "Working directory for the command. Default $HOME. Fails the launch if it doesn't exist."},
				"env":             {Type: "object", Description: "Extra environment variables (string values) exported before the command runs. Stored in a 0600 file on the host, never inlined into the command string — use this for tokens instead of interpolating them."},
				"stdin":           {Type: "string", Description: "Text fed to the command's standard input. Omitted → /dev/null (prompting commands get EOF instead of hanging)."},
				"wait_seconds":    {Type: "number", Description: "Long-poll server-side up to this many seconds (max 60). If the job finishes in time, the full result is returned in this call; otherwise you get the async {id, status: running} shape."},
			},
			Required: []string{"command"},
		},
	},
	{
		Name:        "get_command",
		Description: "Get the status, exit code, and captured output of a command launched with run_command. Status is 'running', 'finished' (always with exit_code), 'crashed' (runner died without recording an exit code; exit_code -1), or 'gone' (job directory no longer on the host). Output is returned as separate stdout/stderr, capped at 256KB per stream by default with total sizes and a truncated flag.",
		InputSchema: InputSchema{
			Type: "object",
			Properties: map[string]PropertySchema{
				"id":               {Type: "string", Description: "Job id returned by run_command"},
				"output":           {Type: "boolean", Description: "Include captured output (default true). Set false for status/exit code only."},
				"tail_lines":       {Type: "number", Description: "Return only the last N lines of each stream."},
				"max_output_bytes": {Type: "number", Description: "Cap each returned stream at this many bytes (default 262144, max 4194304). The tail of the stream is kept."},
				"wait_seconds":     {Type: "number", Description: "Long-poll up to this many seconds (max 60) for the job to reach a terminal state before responding."},
			},
			Required: []string{"id"},
		},
	},
	{
		Name:        "list_commands",
		Description: "List exec jobs on a host by scanning its job directory — includes jobs launched by other clients or forgotten after a server restart. Returns id, status, exit code, output sizes, start time, and a command preview.",
		InputSchema: InputSchema{
			Type: "object",
			Properties: map[string]PropertySchema{
				"host":   {Type: "string", Description: "Host to scan. Optional — defaults like run_command."},
				"status": {Type: "string", Description: "Filter: 'running', 'finished', or 'crashed'. Omit for all."},
				"limit":  {Type: "number", Description: "Max jobs to return (default 20), newest first."},
			},
		},
	},
	{
		Name:        "kill_command",
		Description: "Stop a running exec job by signalling its process group (SIGTERM). The job resolves to a terminal state with a real exit code. Set force=true to escalate to SIGKILL after a short grace period.",
		InputSchema: InputSchema{
			Type: "object",
			Properties: map[string]PropertySchema{
				"id":    {Type: "string", Description: "Job id returned by run_command"},
				"force": {Type: "boolean", Description: "Escalate to SIGKILL if the process ignores SIGTERM."},
			},
			Required: []string{"id"},
		},
	},
}

// Session represents an active SSE connection.
type Session struct {
	ID      string
	Queue   chan []byte
	Created time.Time
}

// Server holds MCP state and dependencies.
type Server struct {
	Hub      *hub.Hub
	Tmux     *tmux.Manager
	KeyStore *keystore.Store
	Settings *keystore.SettingsManager
	Auth     *auth.Manager
	ExecJobs *execjob.Store
	Version  string

	mu       sync.Mutex
	sessions map[string]*Session
}

func NewServer(h *hub.Hub, tm *tmux.Manager, ks *keystore.Store,
	sm *keystore.SettingsManager, am *auth.Manager, ej *execjob.Store, version string) *Server {
	return &Server{
		Hub:      h,
		Tmux:     tm,
		KeyStore: ks,
		Settings: sm,
		Auth:     am,
		ExecJobs: ej,
		Version:  version,
		sessions: make(map[string]*Session),
	}
}

func (s *Server) validateBearer(r *http.Request) bool {
	if s.Auth.NoAuth() {
		return true
	}
	h := r.Header.Get("Authorization")
	if !strings.HasPrefix(h, "Bearer ") {
		return false
	}
	return s.Auth.ValidAPIToken(strings.TrimPrefix(h, "Bearer "))
}

// HandleSSE handles GET /mcp/sse — creates an SSE session.
func (s *Server) HandleSSE(w http.ResponseWriter, r *http.Request) {
	if !s.Settings.MCPEnabled() {
		http.Error(w, `{"error":"MCP is not enabled. Enable it in Settings."}`, http.StatusNotFound)
		return
	}
	if !s.validateBearer(r) {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}

	id := randomID()
	sess := &Session{ID: id, Queue: make(chan []byte, 64), Created: time.Now()}
	s.mu.Lock()
	s.sessions[id] = sess
	s.mu.Unlock()

	log.Printf("MCP SSE session created: %s", id)

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming not supported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	// Send endpoint event
	fmt.Fprintf(w, "event: endpoint\ndata: /mcp/messages?session_id=%s\n\n", id)
	flusher.Flush()

	ctx := r.Context()
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	defer func() {
		s.mu.Lock()
		delete(s.sessions, id)
		s.mu.Unlock()
		log.Printf("MCP SSE session closed: %s", id)
	}()

	for {
		select {
		case <-ctx.Done():
			return
		case msg := <-sess.Queue:
			fmt.Fprintf(w, "event: message\ndata: %s\n\n", msg)
			flusher.Flush()
		case <-ticker.C:
			fmt.Fprint(w, ": keepalive\n\n")
			flusher.Flush()
		}
	}
}

// HandleMessages handles POST /mcp/messages?session_id=...
func (s *Server) HandleMessages(w http.ResponseWriter, r *http.Request) {
	if !s.Settings.MCPEnabled() {
		http.Error(w, `{"error":"MCP is not enabled"}`, http.StatusNotFound)
		return
	}
	if !s.validateBearer(r) {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}

	sessionID := r.URL.Query().Get("session_id")
	s.mu.Lock()
	sess, ok := s.sessions[sessionID]
	s.mu.Unlock()
	if !ok {
		http.Error(w, `{"error":"invalid or missing session_id"}`, http.StatusBadRequest)
		return
	}

	var msg map[string]any
	if err := json.NewDecoder(r.Body).Decode(&msg); err != nil {
		http.Error(w, `{"error":"invalid JSON"}`, http.StatusBadRequest)
		return
	}

	log.Printf("MCP message [%s]: method=%v", sessionID, msg["method"])

	resp := s.handleMessage(msg)
	if resp != nil {
		data, _ := json.Marshal(resp)
		select {
		case sess.Queue <- data:
		default:
			log.Printf("MCP session %s queue full, dropping message", sessionID)
		}
	}

	w.WriteHeader(http.StatusAccepted)
}

func (s *Server) handleMessage(msg map[string]any) map[string]any {
	method, _ := msg["method"].(string)
	id := msg["id"] // may be nil for notifications
	params, _ := msg["params"].(map[string]any)

	switch method {
	case "initialize":
		return jsonRPCResult(id, map[string]any{
			"protocolVersion": "2024-11-05",
			"capabilities":    map[string]any{"tools": map[string]any{"listChanged": false}},
			"serverInfo":      map[string]any{"name": "ssh-to-go-mcp", "version": s.Version},
		})

	case "notifications/initialized":
		return nil

	case "tools/list":
		return jsonRPCResult(id, map[string]any{"tools": tools})

	case "tools/call":
		name, _ := params["name"].(string)
		args, _ := params["arguments"].(map[string]any)
		result := s.callToolSafe(name, args)
		return jsonRPCResult(id, result)

	case "ping":
		return jsonRPCResult(id, map[string]any{})

	default:
		if id == nil {
			return nil
		}
		return map[string]any{
			"jsonrpc": "2.0",
			"id":      id,
			"error":   map[string]any{"code": -32601, "message": "Method not found: " + method},
		}
	}
}

// callToolSafe wraps callTool with panic recovery so a bug in one tool
// surfaces as a structured, logged error instead of an opaque transport
// failure at the client.
func (s *Server) callToolSafe(name string, args map[string]any) (result map[string]any) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("MCP tool %s panicked: %v", name, r)
			result = toolFail("INTERNAL", fmt.Sprintf("internal error in %s: %v", name, r), true, nil)
		}
	}()
	return s.callTool(name, args)
}

func (s *Server) callTool(name string, args map[string]any) map[string]any {
	switch name {
	case "list_sessions":
		sessions := s.Hub.AllSessions()
		if sessions == nil {
			sessions = []hub.HostSession{}
		}
		return toolResult(sessions, false)

	case "list_hosts":
		hosts := s.Hub.AllHosts()
		if hosts == nil {
			hosts = []hub.HostState{}
		}
		type hostSummary struct {
			Name         string `json:"name"`
			Address      string `json:"address"`
			User         string `json:"user"`
			Online       bool   `json:"online"`
			TmuxVersion  string `json:"tmux_version,omitempty"`
			SessionCount int    `json:"session_count"`
			DetectedOS   string `json:"detected_os,omitempty"`
			Error        string `json:"error,omitempty"`
		}
		summaries := make([]hostSummary, len(hosts))
		for i, h := range hosts {
			summaries[i] = hostSummary{
				Name: h.Config.Name, Address: h.Config.Address, User: h.Config.User,
				Online: h.Online, TmuxVersion: h.TmuxVersion,
				SessionCount: len(h.Sessions), DetectedOS: h.DetectedOS, Error: h.Error,
			}
		}
		return toolResult(summaries, false)

	case "create_session":
		host, _ := args["host"].(string)
		sessionName, _ := args["name"].(string)
		cwd, _ := args["cwd"].(string)
		if host == "" || sessionName == "" {
			return toolError("host and name are required")
		}
		hostCfg, ok := s.Hub.GetHostConfig(host)
		if !ok {
			return toolError("host not found: " + host)
		}
		keyPath := keystore.ResolveKeyPath(hostCfg, s.KeyStore, s.Settings)
		client, err := sshutil.Dial(hostCfg.DialAddress(), hostCfg.User, keyPath)
		if err != nil {
			return toolFail("HOST_UNREACHABLE", "SSH connect failed: "+err.Error(), true,
				map[string]any{"host": host})
		}
		defer client.Close()
		if err := s.Tmux.CreateSession(client, sessionName, s.Settings.TmuxWindowSize(), cwd, s.Settings.ScrollbackLines()); err != nil {
			return toolError("create session failed: " + err.Error())
		}
		return toolText(fmt.Sprintf("Session '%s' created on %s.", sessionName, host))

	case "kill_session":
		host, _ := args["host"].(string)
		session, _ := args["session"].(string)
		if host == "" || session == "" {
			return toolError("host and session are required")
		}
		hostCfg, ok := s.Hub.GetHostConfig(host)
		if !ok {
			return toolError("host not found: " + host)
		}
		keyPath := keystore.ResolveKeyPath(hostCfg, s.KeyStore, s.Settings)
		client, err := sshutil.Dial(hostCfg.DialAddress(), hostCfg.User, keyPath)
		if err != nil {
			return toolFail("HOST_UNREACHABLE", "SSH connect failed: "+err.Error(), true,
				map[string]any{"host": host})
		}
		defer client.Close()
		if err := s.Tmux.KillSession(client, session); err != nil {
			return toolError("kill session failed: " + err.Error())
		}
		return toolText(fmt.Sprintf("Session '%s' killed on %s.", session, host))

	case "rename_session":
		host, _ := args["host"].(string)
		session, _ := args["session"].(string)
		newName, _ := args["new_name"].(string)
		if host == "" || session == "" || newName == "" {
			return toolError("host, session, and new_name are required")
		}
		hostCfg, ok := s.Hub.GetHostConfig(host)
		if !ok {
			return toolError("host not found: " + host)
		}
		keyPath := keystore.ResolveKeyPath(hostCfg, s.KeyStore, s.Settings)
		client, err := sshutil.Dial(hostCfg.DialAddress(), hostCfg.User, keyPath)
		if err != nil {
			return toolFail("HOST_UNREACHABLE", "SSH connect failed: "+err.Error(), true,
				map[string]any{"host": host})
		}
		defer client.Close()
		if err := s.Tmux.RenameSession(client, session, newName); err != nil {
			return toolError("rename failed: " + err.Error())
		}
		return toolText(fmt.Sprintf("Session renamed from '%s' to '%s' on %s.", session, newName, host))

	case "detach_clients":
		host, _ := args["host"].(string)
		session, _ := args["session"].(string)
		if host == "" || session == "" {
			return toolError("host and session are required")
		}
		hostCfg, ok := s.Hub.GetHostConfig(host)
		if !ok {
			return toolError("host not found: " + host)
		}
		keyPath := keystore.ResolveKeyPath(hostCfg, s.KeyStore, s.Settings)
		client, err := sshutil.Dial(hostCfg.DialAddress(), hostCfg.User, keyPath)
		if err != nil {
			return toolFail("HOST_UNREACHABLE", "SSH connect failed: "+err.Error(), true,
				map[string]any{"host": host})
		}
		defer client.Close()
		detached, err := s.Tmux.DetachClients(client, session, "")
		if err != nil {
			return toolError("detach clients failed: " + err.Error())
		}
		return toolText(fmt.Sprintf("Detached %d client(s) from session '%s' on %s.", detached, session, host))

	case "scan_host":
		host, _ := args["host"].(string)
		if host == "" {
			return toolError("host is required")
		}
		hostCfg, ok := s.Hub.GetHostConfig(host)
		if !ok {
			return toolError("host not found: " + host)
		}
		keyPath := keystore.ResolveKeyPath(hostCfg, s.KeyStore, s.Settings)
		state, err := s.Hub.ScanHost(host, s.Tmux, keyPath)
		if err != nil {
			return toolError("scan failed: " + err.Error())
		}
		return toolResult(map[string]any{
			"host": host, "online": state.Online, "sessions": len(state.Sessions),
		}, false)

	case "scan_all":
		hosts := s.Hub.AllHosts()
		var results []map[string]any
		for _, h := range hosts {
			keyPath := keystore.ResolveKeyPath(h.Config, s.KeyStore, s.Settings)
			state, err := s.Hub.ScanHost(h.Config.Name, s.Tmux, keyPath)
			entry := map[string]any{"host": h.Config.Name}
			if err != nil {
				entry["error"] = err.Error()
			} else {
				entry["online"] = state.Online
				entry["sessions"] = len(state.Sessions)
			}
			results = append(results, entry)
		}
		return toolResult(results, false)

	case "health_check":
		hosts := s.Hub.AllHosts()
		online := 0
		for _, h := range hosts {
			if h.Online {
				online++
			}
		}
		return toolResult(map[string]any{
			"status": "healthy", "service": "ssh-to-go",
			"version": s.Version, "hosts": len(hosts), "hosts_online": online,
		}, false)

	case "run_command":
		command, _ := args["command"].(string)
		if command == "" {
			return toolFail("BAD_REQUEST", "command is required", false, nil)
		}
		if s.ExecJobs == nil {
			return toolFail("EXEC_UNAVAILABLE", "exec is not available", false, nil)
		}
		hostArg, _ := args["host"].(string)
		hostCfg, errMsg := s.resolveExecHost(hostArg)
		if errMsg != "" {
			return toolFail("HOST_NOT_FOUND", errMsg, false, nil)
		}
		spec := execjob.RunSpec{Command: command, TimeoutSecs: execjob.DefaultTimeoutSecs}
		if v, ok := args["timeout_seconds"].(float64); ok && v >= 0 {
			spec.TimeoutSecs = int(v)
		}
		spec.Cwd, _ = args["cwd"].(string)
		if envArg, ok := args["env"].(map[string]any); ok && len(envArg) > 0 {
			spec.Env = make(map[string]string, len(envArg))
			for k, v := range envArg {
				sv, ok := v.(string)
				if !ok {
					return toolFail("BAD_REQUEST", fmt.Sprintf("env value for %q must be a string", k), false, nil)
				}
				spec.Env[k] = sv
			}
		}
		if v, ok := args["stdin"].(string); ok {
			spec.Stdin = &v
		}
		run, closer, fail := s.execerForHost(hostCfg)
		if fail != nil {
			return fail
		}
		defer closer()
		id := execjob.NewID()
		if err := execjob.Launch(run, id, spec); err != nil {
			return toolFail("EXEC_LAUNCH_FAILED", "launch failed: "+err.Error(), false,
				map[string]any{"host": hostCfg.Name})
		}
		s.ExecJobs.Add(&execjob.Job{
			ID: id, Host: hostCfg.Name, Command: command,
			Session: execjob.SessionName(id), CreatedAt: time.Now(),
		})
		if w, ok := args["wait_seconds"].(float64); ok && w > 0 {
			res, err := execjob.PollStatus(run, id, time.Duration(w)*time.Second)
			if err == nil && res.Terminal() {
				return s.execResult(run, id, hostCfg.Name, res, true, execjob.OutputOpts{})
			}
			// Still running (or transient poll error): fall through to the
			// async shape and let the caller poll with get_command.
		}
		return toolResult(map[string]any{
			"id": id, "host": hostCfg.Name, "status": string(execjob.StatusRunning),
		}, false)

	case "get_command":
		id, _ := args["id"].(string)
		if id == "" {
			return toolFail("BAD_REQUEST", "id is required", false, nil)
		}
		if s.ExecJobs == nil {
			return toolFail("EXEC_UNAVAILABLE", "exec is not available", false, nil)
		}
		job, ok := s.ExecJobs.Get(id)
		if !ok {
			return toolFail("JOB_NOT_FOUND", "job not found: "+id, false,
				map[string]any{"hint": "job ids don't survive a server restart; use list_commands to find jobs still on the host"})
		}
		includeOutput := true
		if v, ok := args["output"].(bool); ok {
			includeOutput = v
		}
		opts := execjob.OutputOpts{}
		if v, ok := args["tail_lines"].(float64); ok && v > 0 {
			opts.TailLines = int(v)
		}
		if v, ok := args["max_output_bytes"].(float64); ok && v > 0 {
			opts.MaxBytes = int(v)
		}
		hostCfg, ok := s.Hub.GetHostConfig(job.Host)
		if !ok {
			return toolFail("HOST_NOT_FOUND", "host not found: "+job.Host, false, nil)
		}
		run, closer, fail := s.execerForHost(hostCfg)
		if fail != nil {
			return fail
		}
		defer closer()
		wait := time.Duration(0)
		if v, ok := args["wait_seconds"].(float64); ok && v > 0 {
			wait = time.Duration(v) * time.Second
		}
		res, err := execjob.PollStatus(run, id, wait)
		if err != nil {
			return toolFail("STATUS_QUERY_FAILED", "status query failed: "+err.Error(), true,
				map[string]any{"host": job.Host})
		}
		return s.execResult(run, id, job.Host, res, includeOutput, opts)

	case "list_commands":
		if s.ExecJobs == nil {
			return toolFail("EXEC_UNAVAILABLE", "exec is not available", false, nil)
		}
		hostArg, _ := args["host"].(string)
		hostCfg, errMsg := s.resolveExecHost(hostArg)
		if errMsg != "" {
			return toolFail("HOST_NOT_FOUND", errMsg, false, nil)
		}
		run, closer, fail := s.execerForHost(hostCfg)
		if fail != nil {
			return fail
		}
		defer closer()
		jobs, err := execjob.ListRemote(run)
		if err != nil {
			return toolFail("LIST_FAILED", "list failed: "+err.Error(), true,
				map[string]any{"host": hostCfg.Name})
		}
		if statusFilter, _ := args["status"].(string); statusFilter != "" {
			filtered := jobs[:0]
			for _, j := range jobs {
				if string(j.Status) == statusFilter {
					filtered = append(filtered, j)
				}
			}
			jobs = filtered
		}
		limit := 20
		if v, ok := args["limit"].(float64); ok && v > 0 {
			limit = int(v)
		}
		total := len(jobs)
		if len(jobs) > limit {
			jobs = jobs[:limit]
		}
		if jobs == nil {
			jobs = []execjob.RemoteJob{}
		}
		return toolResult(map[string]any{
			"host": hostCfg.Name, "total": total, "jobs": jobs,
		}, false)

	case "kill_command":
		id, _ := args["id"].(string)
		if id == "" {
			return toolFail("BAD_REQUEST", "id is required", false, nil)
		}
		if s.ExecJobs == nil {
			return toolFail("EXEC_UNAVAILABLE", "exec is not available", false, nil)
		}
		job, ok := s.ExecJobs.Get(id)
		if !ok {
			return toolFail("JOB_NOT_FOUND", "job not found: "+id, false, nil)
		}
		hostCfg, ok := s.Hub.GetHostConfig(job.Host)
		if !ok {
			return toolFail("HOST_NOT_FOUND", "host not found: "+job.Host, false, nil)
		}
		run, closer, fail := s.execerForHost(hostCfg)
		if fail != nil {
			return fail
		}
		defer closer()
		force, _ := args["force"].(bool)
		result, err := execjob.Kill(run, id, force)
		if err != nil {
			return toolFail("KILL_FAILED", "kill failed: "+err.Error(), true,
				map[string]any{"host": job.Host})
		}
		return toolResult(map[string]any{"id": id, "host": job.Host, "result": result}, false)

	default:
		return toolFail("UNKNOWN_TOOL", "Unknown tool: "+name, false, nil)
	}
}

// execerForHost dials the host and wraps the SSH client as an
// execjob.Execer. On failure it returns a ready-made structured tool error.
func (s *Server) execerForHost(hostCfg config.Host) (execjob.Execer, func(), map[string]any) {
	keyPath := keystore.ResolveKeyPath(hostCfg, s.KeyStore, s.Settings)
	client, err := sshutil.Dial(hostCfg.DialAddress(), hostCfg.User, keyPath)
	if err != nil {
		return nil, nil, toolFail("HOST_UNREACHABLE", "SSH connect failed: "+err.Error(), true,
			map[string]any{"host": hostCfg.Name})
	}
	run := func(script string) (string, error) { return sshutil.Exec(client, script) }
	return run, func() { client.Close() }, nil
}

// execResult renders a job's status (and optionally a bounded slice of its
// output) as a tool result.
func (s *Server) execResult(run execjob.Execer, id, host string,
	res execjob.StatusResult, includeOutput bool, opts execjob.OutputOpts) map[string]any {

	result := map[string]any{"id": id, "host": host, "status": string(res.Status)}
	if res.ExitCode != nil {
		result["exit_code"] = *res.ExitCode
	}
	if includeOutput && res.Status != execjob.StatusGone {
		out, err := execjob.FetchOutput(run, id, opts)
		if err != nil {
			return toolFail("OUTPUT_FETCH_FAILED", "output fetch failed: "+err.Error(), true,
				map[string]any{"host": host, "id": id, "status": string(res.Status)})
		}
		result["stdout"] = out.Stdout
		result["stderr"] = out.Stderr
		result["stdout_bytes"] = out.StdoutBytes
		result["stderr_bytes"] = out.StderrBytes
		if out.Truncated() {
			result["truncated"] = true
		}
	}
	return toolResult(result, false)
}

// resolveExecHost mirrors the REST API's host resolution for the exec tools:
// explicit name, else the configured default host, else the sole host.
func (s *Server) resolveExecHost(name string) (config.Host, string) {
	if name == "" {
		name = s.Settings.DefaultHost()
	}
	if name == "" {
		hosts := s.Hub.AllHosts()
		if len(hosts) == 1 {
			name = hosts[0].Config.Name
		} else if len(hosts) == 0 {
			return config.Host{}, "no hosts are configured"
		} else {
			return config.Host{}, "no host specified and no default host configured; pass \"host\""
		}
	}
	cfg, ok := s.Hub.GetHostConfig(name)
	if !ok {
		return config.Host{}, "host not found: " + name
	}
	return cfg, ""
}

// HandleDocs serves GET /mcpdocs — tool documentation page.
func (s *Server) HandleDocs(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(renderMCPDocs(s.Settings.MCPEnabled())))
}

// HandleGetConfig serves GET /api/settings/mcp
func (s *Server) HandleGetConfig(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"enabled": s.Settings.MCPEnabled()})
}

// HandleSetConfig serves PUT /api/settings/mcp
func (s *Server) HandleSetConfig(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Enabled bool `json:"enabled"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	s.Settings.SetMCPEnabled(body.Enabled)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"success": true, "enabled": body.Enabled})
}

// --- helpers ---

func jsonRPCResult(id any, result any) map[string]any {
	return map[string]any{"jsonrpc": "2.0", "id": id, "result": result}
}

func toolResult(data any, isError bool) map[string]any {
	text, _ := json.MarshalIndent(data, "", "  ")
	r := map[string]any{
		"content": []map[string]any{{"type": "text", "text": string(text)}},
	}
	if isError {
		r["isError"] = true
	}
	return r
}

func toolText(msg string) map[string]any {
	return map[string]any{
		"content": []map[string]any{{"type": "text", "text": msg}},
	}
}

func toolError(msg string) map[string]any {
	return map[string]any{
		"content": []map[string]any{{"type": "text", "text": msg}},
		"isError": true,
	}
}

// toolFail returns a structured error payload. The retryable flag is the
// single most useful field for an agent consumer: it distinguishes
// "transient, try again" (SSH dial timeout, host briefly offline) from
// "broken, stop" (unknown host, bad arguments).
func toolFail(code, msg string, retryable bool, extra map[string]any) map[string]any {
	payload := map[string]any{"error": msg, "code": code, "retryable": retryable}
	for k, v := range extra {
		payload[k] = v
	}
	text, _ := json.MarshalIndent(payload, "", "  ")
	return map[string]any{
		"content": []map[string]any{{"type": "text", "text": string(text)}},
		"isError": true,
	}
}

func randomID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// --- mcpdocs HTML ---

func renderMCPDocs(enabled bool) string {
	var toolHTML strings.Builder
	for _, t := range tools {
		toolHTML.WriteString(`<div class="tool"><div class="tool-name">`)
		toolHTML.WriteString(t.Name)
		toolHTML.WriteString(`</div><div class="tool-desc">`)
		toolHTML.WriteString(t.Description)
		toolHTML.WriteString(`</div>`)

		if len(t.InputSchema.Properties) > 0 {
			toolHTML.WriteString(`<div class="params"><div class="params-title">Parameters</div>`)
			reqSet := make(map[string]bool)
			for _, r := range t.InputSchema.Required {
				reqSet[r] = true
			}
			for pname, pinfo := range t.InputSchema.Properties {
				badge := ""
				if reqSet[pname] {
					badge = `<span class="required">required</span>`
				}
				toolHTML.WriteString(fmt.Sprintf(
					`<div class="param"><span class="param-name">%s%s</span><span class="param-type">%s</span><span class="param-desc">%s</span></div>`,
					pname, badge, pinfo.Type, pinfo.Description,
				))
			}
			toolHTML.WriteString(`</div>`)
		} else {
			toolHTML.WriteString(`<div class="params"><span class="no-params">No parameters</span></div>`)
		}
		toolHTML.WriteString(`</div>`)
	}

	banner := ""
	if !enabled {
		banner = `<div class="disabled-banner">MCP is currently disabled. Enable it in Settings to use MCP endpoints.</div>`
	}

	return fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>SSH-to-go — MCP Tools</title>
<style>
  :root { --bg: #0f1117; --surface: #1a1d27; --border: #2d3148; --text: #e1e4ed; --muted: #8b8fa3; --accent: #6c8cff; --accent2: #4fc1a6; --danger: #f87171; }
  * { margin: 0; padding: 0; box-sizing: border-box; }
  body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; background: var(--bg); color: var(--text); line-height: 1.6; padding: 2rem; max-width: 960px; margin: 0 auto; }
  h1 { font-size: 1.8rem; margin-bottom: 0.25rem; }
  .subtitle { color: var(--muted); margin-bottom: 2rem; font-size: 0.95rem; }
  .endpoint-info { background: var(--surface); border: 1px solid var(--border); border-radius: 8px; padding: 1rem 1.25rem; margin-bottom: 2rem; }
  .endpoint-info code { background: var(--bg); padding: 2px 6px; border-radius: 4px; font-size: 0.9rem; color: var(--accent); }
  .tool { background: var(--surface); border: 1px solid var(--border); border-radius: 8px; padding: 1.25rem; margin-bottom: 1rem; }
  .tool-name { font-size: 1.1rem; font-weight: 600; color: var(--accent2); font-family: monospace; }
  .tool-desc { color: var(--muted); margin: 0.5rem 0; font-size: 0.9rem; }
  .params { margin-top: 0.75rem; }
  .params-title { font-size: 0.8rem; text-transform: uppercase; color: var(--muted); letter-spacing: 0.05em; margin-bottom: 0.4rem; }
  .param { display: flex; gap: 0.75rem; padding: 0.3rem 0; font-size: 0.88rem; }
  .param-name { font-family: monospace; color: var(--accent); min-width: 120px; }
  .param-type { color: var(--muted); min-width: 70px; }
  .param-desc { color: var(--text); }
  .required { color: var(--danger); font-size: 0.75rem; margin-left: 4px; }
  .no-params { color: var(--muted); font-size: 0.85rem; font-style: italic; }
  a { color: var(--accent); }
  .disabled-banner { background: #b91c1c; color: #fff; padding: 0.75rem 1.25rem; border-radius: 8px; margin-bottom: 1.5rem; font-weight: 600; }
</style>
</head>
<body>
<h1>SSH-to-go MCP Server</h1>
<p class="subtitle">Model Context Protocol tools for managing tmux sessions across SSH hosts</p>
%s
<div class="endpoint-info">
  <strong>SSE Endpoint:</strong> <code>GET /mcp/sse</code><br>
  <strong>Messages Endpoint:</strong> <code>POST /mcp/messages?session_id=...</code><br>
  <strong>Auth:</strong> Bearer token required (same API tokens as REST API). Create tokens in <a href="/settings">Settings</a>.
</div>
%s
</body>
</html>`, banner, toolHTML.String())
}
