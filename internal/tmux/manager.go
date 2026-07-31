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
	return m.CreateSessionWith(client, name, CreateOptions{
		WindowSize:   windowSize,
		Cwd:          cwd,
		HistoryLimit: historyLimit,
	})
}

// CreateOptions carries the optional knobs of session creation. Zero value
// behaves exactly like the old positional CreateSession.
type CreateOptions struct {
	WindowSize   string
	Cwd          string
	HistoryLimit int
	// CreateDir makes the working directory before starting tmux, instead
	// of failing when it doesn't exist yet. Only meaningful with Cwd.
	CreateDir bool
	// Command, when set, is typed into the new session's shell after it
	// starts. Sent with send-keys rather than passed to new-session so the
	// shell OUTLIVES the command — when claude/codex/vim exits the user is
	// left at a prompt in the right directory, not with a dead session.
	Command string
	// Mouse enables tmux's mouse option on the new session (per-session -t,
	// never -g, so other sessions on the host are untouched). Without it a
	// native-terminal attach gets the wheel as arrow keys: tmux holds the
	// outer terminal in the alt screen and, with no mouse tracking asked for,
	// the terminal's alternateScroll kicks in. The browser terminal ignores
	// this option entirely — control-mode clients get no mouse DECSETs and
	// the relay strips them on the legacy path.
	Mouse bool
}

// CreateSessionWith is CreateSession with the optional knobs. See CreateOptions.
func (m *Manager) CreateSessionWith(client *ssh.Client, name string, opts CreateOptions) error {
	out, err := sshutil.Exec(client, buildCreateCmd(name, opts))
	if err != nil {
		// sshutil.Exec embeds the whole command in its error, which for a
		// plain "that directory isn't there" is a wall of shell noise in the
		// UI. Surface the guard's own message when it fired.
		if msg := guardFailure(out + err.Error()); msg != "" {
			return fmt.Errorf("%s", msg)
		}
		return fmt.Errorf("create session %q: %w", name, err)
	}
	return nil
}

// missingDirMarker is echoed by the pre-flight check in buildCreateCmd.
const missingDirMarker = "working directory does not exist: "

// guardFailure pulls the pre-flight message out of a failed exec, or "".
// Uses the LAST occurrence: sshutil.Exec echoes the command it ran (which
// contains the marker as literal text) before appending the real output.
func guardFailure(s string) string {
	i := strings.LastIndex(s, missingDirMarker)
	if i < 0 {
		return ""
	}
	rest := s[i:]
	if j := strings.IndexAny(rest, "\r\n"); j >= 0 {
		rest = rest[:j]
	}
	return strings.TrimSpace(rest)
}

// buildCreateCmd renders the remote shell command that creates the session.
// Split out from CreateSessionWith so the quoting and ordering are testable
// without an SSH connection.
func buildCreateCmd(name string, opts CreateOptions) string {
	windowSize := opts.WindowSize
	if windowSize == "" {
		windowSize = "largest"
	}
	// One tmux invocation; commands separated by tmux's own "\;". The
	// history-limit set-option must come first so the new pane inherits it.
	hist := ""
	if opts.HistoryLimit > 0 {
		hist = fmt.Sprintf("set-option -g history-limit %d \\; ", opts.HistoryLimit)
	}
	geom, sizeOpt := "", windowSize
	if w, h, ok := parseWxH(windowSize); ok {
		geom = fmt.Sprintf(" -x %d -y %d", w, h)
		sizeOpt = "manual"
	}
	base := fmt.Sprintf("tmux %snew-session -d -s %q%s", hist, name, geom)
	dir := remotePath(opts.Cwd)
	if dir != "" {
		base += " -c " + dir
	}
	cmd := fmt.Sprintf("%s \\; set-option -t %q window-size %s", base, name, sizeOpt)
	if opts.Mouse {
		cmd += fmt.Sprintf(" \\; set-option -t %q mouse on", name)
	}
	if dir != "" {
		if opts.CreateDir {
			// && so a failed mkdir (permissions, a file in the way) surfaces
			// as a create error instead of silently starting elsewhere.
			cmd = "mkdir -p " + dir + " && " + cmd
		} else {
			// tmux does NOT fail on a missing -c directory — it quietly falls
			// back to $HOME, so the session comes up in the wrong place with
			// no indication. Check first and fail with a usable message.
			cmd = "{ [ -d " + dir + " ] || { echo " + shellSingleQuote(missingDirMarker) + dir + " >&2; exit 3; }; } && " + cmd
		}
	}
	if c := strings.TrimSpace(opts.Command); c != "" {
		// NOT chained into the tmux "\;" sequence: send-keys would fire
		// microseconds after the pane's shell spawns — before it has sourced
		// rc files or printed a prompt — so the raw tty echoes the command at
		// column 0 and a TUI then draws over the prompt line (issue #75).
		// Wait until the shell has drawn its prompt (cursor_x > 0) before
		// typing, with a ~5s cap; on timeout send anyway (";" not "&&") so an
		// empty-prompt shell still gets the command, matching old behavior.
		cmd += fmt.Sprintf("; for _ in $(seq 1 50); do [ \"$(tmux display-message -p -t %q '#{cursor_x}')\" != 0 ] && break; sleep 0.1; done; tmux send-keys -t %q %s Enter",
			name, name, shellSingleQuote(c))
	}
	return cmd
}

// remotePath renders a user-supplied directory for the remote shell. A
// leading ~ is expanded to "$HOME" OUTSIDE the quotes — inside them the
// shell would take it literally and tmux would create a directory actually
// named "~". The rest is single-quoted so spaces and $ in the path stay
// literal. Returns "" for an empty path (meaning "let tmux pick").
func remotePath(p string) string {
	p = strings.TrimSpace(p)
	if p == "" {
		return ""
	}
	if p == "~" {
		return `"$HOME"`
	}
	if strings.HasPrefix(p, "~/") {
		rest := strings.TrimPrefix(p, "~/")
		if rest == "" {
			return `"$HOME"`
		}
		return `"$HOME"/` + shellSingleQuote(rest)
	}
	return shellSingleQuote(p)
}

// shellSingleQuote wraps s in single quotes, escaping embedded quotes, so it
// survives the remote shell verbatim.
func shellSingleQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
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

// shellCommands are the pane_current_command values that mean "a shell
// sitting at a prompt" — i.e. nothing is running. Anything else (make, vim,
// claude, ssh, docker) counts as busy. The list is deliberately short: a
// wrapper we don't recognise makes a session look busy forever, which only
// costs us an auto-offload. Mistaking a running job for a shell would cost
// the user their work.
var shellCommands = map[string]bool{
	"bash": true, "zsh": true, "fish": true, "sh": true,
	"dash": true, "ksh": true, "ash": true, "csh": true, "tcsh": true,
}

// SessionQuiet reports whether every pane of every window in the session is
// a shell at a prompt. A session with no panes at all (it vanished between
// the poll and this call) reports not-quiet, so the caller leaves it alone.
func (m *Manager) SessionQuiet(client *ssh.Client, name string) (bool, error) {
	// -s covers all windows in the session, not just the current one.
	out, err := sshutil.Exec(client, fmt.Sprintf("tmux list-panes -s -t %q -F '#{pane_current_command}'", name))
	if err != nil {
		return false, fmt.Errorf("list panes for %q: %w", name, err)
	}
	return panesQuiet(out), nil
}

// panesQuiet decides quietness from `list-panes -F #{pane_current_command}`
// output. Split out so the rule is testable without an SSH connection.
func panesQuiet(out string) bool {
	seen := false
	for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
		cmd := strings.TrimSpace(line)
		if cmd == "" {
			continue
		}
		seen = true
		if !shellCommands[cmd] {
			return false
		}
	}
	return seen
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
// With mouse, it also backfills the per-session mouse option before attaching
// so sessions created before that option existed still get a working wheel.
// Two separate tmux invocations inside single quotes rather than tmux's own
// "\;" chaining: the string is pasted into the user's local shell, and "\;"
// would need a second layer of escaping to survive the trip to the remote one.
//
// escape-time is raised on every handoff because tmux 3.5 dropped its default
// from 500ms to 10ms, and at attach tmux probes the connecting terminal
// (DA1 "who are you", etc). When the terminal's reply arrives fragmented
// across ssh packets with a >escape-time gap, tmux flushes the partial escape
// sequence into the pane as literal keystrokes — the user sees garbage like
// "61;4;6;7;...52c" typed at the prompt (issue #79, reproduced). 200ms rides
// out network jitter while keeping a bare ESC keypress feeling instant.
func (m *Manager) HandoffCommand(user, address string, port int, sessionName string, mouse bool) string {
	if port == 0 {
		port = 22
	}
	portOpt := ""
	if port != 22 {
		portOpt = fmt.Sprintf(" -p %d", port)
	}
	if mouse {
		return fmt.Sprintf("ssh -t%s %s@%s 'tmux set-option -s escape-time 200 \\; set-option -t %q mouse on 2>/dev/null; exec tmux attach-session -t %q'",
			portOpt, user, address, sessionName, sessionName)
	}
	return fmt.Sprintf("ssh -t%s %s@%s 'tmux set-option -s escape-time 200 2>/dev/null; exec tmux attach-session -t %q'",
		portOpt, user, address, sessionName)
}
