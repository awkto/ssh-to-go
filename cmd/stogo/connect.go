package main

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"
)

// cmdConnect attaches to a session using the server's handoff command —
// the same `ssh -t user@host 'exec tmux attach-session ...'` one-liner the
// dashboard's "handoff to native terminal" feature prints. The heavy
// lifting is local ssh and the target host's tmux; stogo only resolves the
// session and execs the command.
func cmdConnect(args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("usage: stogo connect <session>")
	}
	cfg, err := loadConfig()
	if err != nil {
		return err
	}
	c := newClient(cfg)
	host, session, err := c.resolveSession(args[0])
	if err != nil {
		return err
	}

	var out struct {
		Command string `json:"command"`
	}
	if err := c.do("GET", sessionPath(host, session)+"/handoff", nil, &out); err != nil {
		return err
	}
	if out.Command == "" {
		return fmt.Errorf("server returned an empty handoff command")
	}

	if _, err := exec.LookPath("ssh"); err != nil {
		return fmt.Errorf("ssh not found — install an SSH client (e.g. `sudo apt install openssh-client`)")
	}

	fmt.Fprintf(os.Stderr, "connecting to %s/%s (detach: Ctrl-B d)\n", host, session)

	// Replace the stogo process entirely: ssh gets the real terminal, and
	// detaching or session exit behaves exactly like a hand-typed command.
	shell := "/bin/sh"
	if err := syscall.Exec(shell, []string{"sh", "-c", out.Command}, os.Environ()); err != nil {
		return fmt.Errorf("exec handoff command: %w", err)
	}
	return nil // unreachable
}
