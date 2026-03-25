package relay

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"sync"
	"time"

	"github.com/awkto/ssh-to-go/internal/sshutil"
	"golang.org/x/crypto/ssh"
	"nhooyr.io/websocket"
)

type resizeMsg struct {
	Type string `json:"type"`
	Cols int    `json:"cols"`
	Rows int    `json:"rows"`
}

// Relay bridges a WebSocket connection to a tmux session over SSH.
func Relay(ctx context.Context, ws *websocket.Conn, address, user, keyPath, sessionName string) error {
	client, err := sshutil.Dial(address, user, keyPath)
	if err != nil {
		return fmt.Errorf("ssh dial: %w", err)
	}
	defer client.Close()

	// Start keepalive
	done := make(chan struct{})
	defer close(done)
	go sshutil.KeepAlive(client, 15_000_000_000, done) // 15s

	// Default size, will be resized by first client message
	cols, rows := 80, 24
	session, err := sshutil.Shell(client, cols, rows)
	if err != nil {
		return fmt.Errorf("shell: %w", err)
	}
	defer session.Close()

	stdin, err := session.StdinPipe()
	if err != nil {
		return fmt.Errorf("stdin pipe: %w", err)
	}

	stdout, err := session.StdoutPipe()
	if err != nil {
		return fmt.Errorf("stdout pipe: %w", err)
	}

	cmd := fmt.Sprintf("tmux set-option -t %s window-size largest 2>/dev/null; tmux attach-session -t %s || tmux new-session -s %s", sessionName, sessionName, sessionName)
	if err := session.Start(cmd); err != nil {
		return fmt.Errorf("start tmux: %w", err)
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	var wg sync.WaitGroup

	// SSH stdout -> WebSocket
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer cancel()
		buf := make([]byte, 32*1024)
		for {
			n, err := stdout.Read(buf)
			if n > 0 {
				if writeErr := ws.Write(ctx, websocket.MessageBinary, buf[:n]); writeErr != nil {
					return
				}
			}
			if err != nil {
				return
			}
		}
	}()

	// WebSocket -> SSH stdin (text messages are control, binary is data)
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer cancel()
		for {
			typ, data, err := ws.Read(ctx)
			if err != nil {
				return
			}

			if typ == websocket.MessageText {
				// Try to parse as control message
				var msg resizeMsg
				if json.Unmarshal(data, &msg) == nil && msg.Type == "resize" {
					if msg.Cols > 0 && msg.Rows > 0 {
						_ = session.WindowChange(msg.Rows, msg.Cols)
					}
					continue
				}
			}

			// Raw terminal input
			if _, err := stdin.Write(data); err != nil {
				return
			}
		}
	}()

	// Wait for SSH command to finish
	err = session.Wait()

	// Signal the client BEFORE cancelling goroutines, while the WS is still alive
	closeCtx, closeCancel := context.WithTimeout(context.Background(), 2*time.Second)
	_ = ws.Write(closeCtx, websocket.MessageText, []byte(`{"type":"session_ended"}`))
	closeCancel()

	cancel()
	wg.Wait()

	if err != nil {
		log.Printf("relay session ended: %v", err)
	}

	return nil
}

// PipeIO is a simpler relay for non-WebSocket use (future CLI attach).
func PipeIO(ctx context.Context, client *ssh.Client, sessionName string, r io.Reader, w io.Writer) error {
	session, err := sshutil.Shell(client, 80, 24)
	if err != nil {
		return err
	}
	defer session.Close()

	session.Stdin = r
	session.Stdout = w
	session.Stderr = w

	cmd := fmt.Sprintf("tmux attach-session -t %s", sessionName)
	return session.Run(cmd)
}
