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
func Relay(ctx context.Context, ws *websocket.Conn, address, user, keyPath, sessionName, windowSize string) error {
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

	if windowSize == "" {
		windowSize = "largest"
	}

	cmd := fmt.Sprintf("tmux set-option -g mouse on 2>/dev/null; tmux set-option -t %q window-size %s 2>/dev/null; tmux attach-session -t %q || tmux new-session -s %q", sessionName, windowSize, sessionName, sessionName)
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

	cancel()
	wg.Wait()

	if err != nil {
		log.Printf("relay session ended: %v", err)
	}

	// Use a custom close code (4000) to signal "session ended normally"
	// This is checked by the client to decide whether to reconnect
	closeCtx, closeCancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer closeCancel()
	ws.Close(websocket.StatusCode(4000), "session ended")
	// Block briefly so the close frame is sent before the defer ws.CloseNow() in the caller
	<-closeCtx.Done()

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

	cmd := fmt.Sprintf("tmux attach-session -t %q", sessionName)
	return session.Run(cmd)
}
