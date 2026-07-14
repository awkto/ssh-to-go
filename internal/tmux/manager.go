package tmux

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/awkto/ssh-to-go/internal/execjob"
	"github.com/awkto/ssh-to-go/internal/relay"
	"github.com/awkto/ssh-to-go/internal/sshutil"
	"golang.org/x/crypto/ssh"
)

type Manager struct{}

func NewManager() *Manager {
	return &Manager{}
}

// DetectTmux checks if tmux is installed on the remote host.
// Returns the version string if found, or an error.
func (m *Manager) DetectTmux(client *ssh.Client) (string, error) {
	out, err := sshutil.Exec(client, "tmux -V")
	if err != nil {
		return "", fmt.Errorf("tmux not found: %w", err)
	}
	return strings.TrimSpace(out), nil
}

// ListSessions returns all tmux sessions on the remote host.
func (m *Manager) ListSessions(client *ssh.Client) ([]Session, error) {
	out, err := sshutil.Exec(client, fmt.Sprintf("tmux list-sessions -F '%s'", ListFormat))
	if err != nil {
		// No tmux server running or no sessions is not an error — just empty
		errStr := err.Error()
		if strings.Contains(errStr, "no server running") ||
			strings.Contains(errStr, "no sessions") ||
			strings.Contains(errStr, "error connecting to") ||
			strings.Contains(errStr, "no current") {
			return nil, nil
		}
		return nil, fmt.Errorf("list sessions: %w", err)
	}
	// Hide throwaway sessions created by the exec API: their pane output is
	// redirected to a file (nothing to attach to) and they'd only clutter
	// the dashboard's session/MRU list.
	all := ParseSessions(out)
	filtered := all[:0]
	for _, s := range all {
		if strings.HasPrefix(s.Name, execjob.SessionPrefix) {
			continue
		}
		filtered = append(filtered, s)
	}
	return filtered, nil
}

// CreateSession creates a new detached tmux session on the remote host.
//
// windowSize is either a window-size *option* value (largest/smallest/latest —
// how the pane tracks attaching clients, used by the web/Android terminals) or
// a concrete "WIDTHxHEIGHT" geometry. A geometry is applied via new-session
// -x/-y with window-size set to "manual" (the window-size option itself only
// accepts the enum values), giving a stable pane size for MCP-created agent
// sessions that no interactive client attaches to.
//
// historyLimit (>0) is applied globally BEFORE new-session so the session's
// first pane inherits the deeper scrollback — a pane's depth is fixed at
// creation, so setting it afterwards wouldn't grow that pane.
func (m *Manager) CreateSession(client *ssh.Client, name, windowSize, cwd string, historyLimit int) error {
	if windowSize == "" {
		windowSize = "largest"
	}
	// One tmux invocation; commands separated by tmux's own "\;". The
	// history-limit set-option must come first so the new pane inherits it.
	hist := ""
	if historyLimit > 0 {
		hist = fmt.Sprintf("set-option -g history-limit %d \\; ", historyLimit)
	}
	geom, sizeOpt := "", windowSize
	if w, h, ok := parseWxH(windowSize); ok {
		geom = fmt.Sprintf(" -x %d -y %d", w, h)
		sizeOpt = "manual"
	}
	base := fmt.Sprintf("tmux %snew-session -d -s %q%s", hist, name, geom)
	if cwd != "" {
		base += fmt.Sprintf(" -c %q", cwd)
	}
	cmd := fmt.Sprintf("%s \\; set-option -t %q window-size %s", base, name, sizeOpt)
	_, err := sshutil.Exec(client, cmd)
	if err != nil {
		return fmt.Errorf("create session %q: %w", name, err)
	}
	return nil
}

// parseWxH parses a concrete "WIDTHxHEIGHT" geometry (e.g. "200x50"). It
// returns ok=false for the window-size enum values (largest/smallest/latest)
// so those keep the original set-option path.
func parseWxH(s string) (int, int, bool) {
	i := strings.IndexByte(s, 'x')
	if i <= 0 || i >= len(s)-1 {
		return 0, 0, false
	}
	w, err1 := strconv.Atoi(s[:i])
	h, err2 := strconv.Atoi(s[i+1:])
	if err1 != nil || err2 != nil || w <= 0 || h <= 0 {
		return 0, 0, false
	}
	return w, h, true
}

// SessionCwd returns the current working directory of the active pane in a session.
func (m *Manager) SessionCwd(client *ssh.Client, name string) (string, error) {
	out, err := sshutil.Exec(client, fmt.Sprintf("tmux display-message -t %q -p '#{pane_current_path}'", name))
	if err != nil {
		return "", fmt.Errorf("get cwd for %q: %w", name, err)
	}
	return strings.TrimSpace(out), nil
}

// KillSession kills a tmux session on the remote host.
func (m *Manager) KillSession(client *ssh.Client, name string) error {
	_, err := sshutil.Exec(client, fmt.Sprintf("tmux kill-session -t %q", name))
	if err != nil {
		return fmt.Errorf("kill session %q: %w", name, err)
	}
	return nil
}

// RenameSession renames a tmux session on the remote host.
func (m *Manager) RenameSession(client *ssh.Client, oldName, newName string) error {
	_, err := sshutil.Exec(client, fmt.Sprintf("tmux rename-session -t %q %q", oldName, newName))
	if err != nil {
		return fmt.Errorf("rename session %q -> %q: %w", oldName, newName, err)
	}
	return nil
}

// Client represents a tmux client attached to a session.
type Client struct {
	TTY     string `json:"tty"`
	Session string `json:"session"`
	Width   int    `json:"width"`
	Height  int    `json:"height"`
}

// ListClients returns all tmux clients attached to a session on the remote host.
func (m *Manager) ListClients(client *ssh.Client, sessionName string) ([]Client, error) {
	// client_name is the tty path for terminal clients and a synthetic
	// "client-<pid>" for tty-less control-mode clients; detach-client -t
	// accepts either, so it works as the universal client identifier.
	out, err := sshutil.Exec(client, fmt.Sprintf("tmux list-clients -t %q -F '#{client_name}\t#{client_session}\t#{client_width}\t#{client_height}'", sessionName))
	if err != nil {
		errStr := err.Error()
		if strings.Contains(errStr, "no clients") || strings.Contains(errStr, "no server running") {
			return nil, nil
		}
		return nil, fmt.Errorf("list clients for %q: %w", sessionName, err)
	}
	var clients []Client
	for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "\t", 4)
		if len(parts) < 4 {
			continue
		}
		w, _ := strconv.Atoi(parts[2])
		h, _ := strconv.Atoi(parts[3])
		clients = append(clients, Client{
			TTY:     parts[0],
			Session: parts[1],
			Width:   w,
			Height:  h,
		})
	}
	return clients, nil
}

// DetachClients detaches all clients from a session, optionally excluding a specific TTY.
func (m *Manager) DetachClients(client *ssh.Client, sessionName, excludeTTY string) (int, error) {
	clients, err := m.ListClients(client, sessionName)
	if err != nil {
		return 0, err
	}
	// Signal each relay before detaching so it can send a "kicked" message
	// to the browser while the WebSocket is still healthy.
	for _, c := range clients {
		if c.TTY != excludeTTY {
			relay.SignalKick(c.TTY)
		}
	}
	// Brief pause to let the kicked message reach browsers before we detach
	time.Sleep(100 * time.Millisecond)

	detached := 0
	for _, c := range clients {
		if c.TTY == excludeTTY {
			continue
		}
		_, err := sshutil.Exec(client, fmt.Sprintf("tmux detach-client -t %q", c.TTY))
		if err == nil {
			detached++
		}
	}
	return detached, nil
}

// HandoffCommand returns the SSH command to directly attach to a session.
func (m *Manager) HandoffCommand(user, address string, port int, sessionName string) string {
	if port == 0 {
		port = 22
	}
	if port == 22 {
		return fmt.Sprintf("ssh -t %s@%s tmux attach-session -t %q", user, address, sessionName)
	}
	return fmt.Sprintf("ssh -t -p %d %s@%s tmux attach-session -t %q", port, user, address, sessionName)
}
