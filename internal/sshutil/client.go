package sshutil

import (
	"bytes"
	"fmt"
	"net"
	"os"
	"time"

	"golang.org/x/crypto/ssh"
)

// DefaultKeyPath is set at startup by main after EnsureKeypair runs.
var DefaultKeyPath string

func Dial(address, user, keyPath string) (*ssh.Client, error) {
	if keyPath == "" {
		keyPath = DefaultKeyPath
	}
	if keyPath == "" {
		return nil, fmt.Errorf("no SSH key configured")
	}

	keyData, err := os.ReadFile(keyPath)
	if err != nil {
		return nil, fmt.Errorf("read key %s: %w", keyPath, err)
	}

	signer, err := ssh.ParsePrivateKey(keyData)
	if err != nil {
		return nil, fmt.Errorf("parse key %s: %w", keyPath, err)
	}

	config := &ssh.ClientConfig{
		User: user,
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(signer),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         10 * time.Second,
	}

	client, err := ssh.Dial("tcp", address, config)
	if err != nil {
		return nil, fmt.Errorf("dial %s: %w", address, err)
	}

	return client, nil
}

func Exec(client *ssh.Client, cmd string) (string, error) {
	session, err := client.NewSession()
	if err != nil {
		return "", fmt.Errorf("new session: %w", err)
	}
	defer session.Close()

	out, err := session.CombinedOutput(cmd)
	if err != nil {
		return string(out), fmt.Errorf("exec %q: %w (output: %s)", cmd, err, string(out))
	}

	return string(out), nil
}

// ExecInput runs cmd with stdin fed from input and returns its stdout.
// Stderr is kept out of the result and only reported on failure, so a
// command that prints a value (a path, an id) can be parsed directly.
func ExecInput(client *ssh.Client, cmd string, input []byte) (string, error) {
	session, err := client.NewSession()
	if err != nil {
		return "", fmt.Errorf("new session: %w", err)
	}
	defer session.Close()

	var stdout, stderr bytes.Buffer
	session.Stdin = bytes.NewReader(input)
	session.Stdout = &stdout
	session.Stderr = &stderr
	if err := session.Run(cmd); err != nil {
		return "", fmt.Errorf("exec: %w (stderr: %s)", err, bytes.TrimSpace(stderr.Bytes()))
	}
	return stdout.String(), nil
}

func Shell(client *ssh.Client, cols, rows int) (*ssh.Session, error) {
	session, err := client.NewSession()
	if err != nil {
		return nil, fmt.Errorf("new session: %w", err)
	}

	modes := ssh.TerminalModes{
		ssh.ECHO:          1,
		ssh.TTY_OP_ISPEED: 14400,
		ssh.TTY_OP_OSPEED: 14400,
	}

	if err := session.RequestPty("xterm-256color", rows, cols, modes); err != nil {
		session.Close()
		return nil, fmt.Errorf("request pty: %w", err)
	}

	return session, nil
}

func KeepAlive(client *ssh.Client, interval time.Duration, done <-chan struct{}) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-done:
			return
		case <-ticker.C:
			_, _, err := client.SendRequest("keepalive@openssh.com", true, nil)
			if err != nil {
				return
			}
		}
	}
}

// WaitPort checks if a TCP port is reachable.
func WaitPort(address string, timeout time.Duration) error {
	conn, err := net.DialTimeout("tcp", address, timeout)
	if err != nil {
		return err
	}
	conn.Close()
	return nil
}
