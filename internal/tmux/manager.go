package tmux

import (
	"fmt"
	"strings"

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
	return ParseSessions(out), nil
}

// CreateSession creates a new detached tmux session on the remote host.
// Sets window-size so multiple clients behave as configured (largest/smallest/latest).
func (m *Manager) CreateSession(client *ssh.Client, name, windowSize string) error {
	if windowSize == "" {
		windowSize = "largest"
	}
	cmd := fmt.Sprintf("tmux new-session -d -s %q \\; set-option -t %q window-size %s", name, name, windowSize)
	_, err := sshutil.Exec(client, cmd)
	if err != nil {
		return fmt.Errorf("create session %q: %w", name, err)
	}
	return nil
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

// HandoffCommand returns the SSH command to directly attach to a session.
func (m *Manager) HandoffCommand(user, address, sessionName string) string {
	// Strip port if it's the default
	host := address
	if strings.HasSuffix(host, ":22") {
		host = strings.TrimSuffix(host, ":22")
	}
	return fmt.Sprintf("ssh -t %s@%s tmux attach-session -t %q", user, host, sessionName)
}
