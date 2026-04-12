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
				"host":    {Type: "string", Description: "Host name to create the session on"},
				"name":    {Type: "string", Description: "Name for the new tmux session"},
				"cwd":     {Type: "string", Description: "Optional working directory for the session"},
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
	Version  string

	mu       sync.Mutex
	sessions map[string]*Session
}

func NewServer(h *hub.Hub, tm *tmux.Manager, ks *keystore.Store,
	sm *keystore.SettingsManager, am *auth.Manager, version string) *Server {
	return &Server{
		Hub:      h,
		Tmux:     tm,
		KeyStore: ks,
		Settings: sm,
		Auth:     am,
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
			"capabilities":   map[string]any{"tools": map[string]any{"listChanged": false}},
			"serverInfo":     map[string]any{"name": "ssh-to-go-mcp", "version": "1.0.0"},
		})

	case "notifications/initialized":
		return nil

	case "tools/list":
		return jsonRPCResult(id, map[string]any{"tools": tools})

	case "tools/call":
		name, _ := params["name"].(string)
		args, _ := params["arguments"].(map[string]any)
		result := s.callTool(name, args)
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
			return toolError("SSH connect failed: " + err.Error())
		}
		defer client.Close()
		if err := s.Tmux.CreateSession(client, sessionName, s.Settings.TmuxWindowSize(), cwd); err != nil {
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
			return toolError("SSH connect failed: " + err.Error())
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
			return toolError("SSH connect failed: " + err.Error())
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
			return toolError("SSH connect failed: " + err.Error())
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

	default:
		return toolError("Unknown tool: " + name)
	}
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
