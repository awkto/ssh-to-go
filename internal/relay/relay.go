package relay

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"strings"
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

	// Discover our PTY path before attaching to tmux, so the client can
	// pass it to the detach-clients endpoint to exclude itself.
	// We print the tty on a single line followed by a NUL byte delimiter,
	// then exec into tmux. The NUL byte ensures we can split cleanly even
	// if tmux output arrives in the same read buffer.
	cmd := fmt.Sprintf(
		`printf '%%s\x00' "$(tty)"; exec tmux set-option -g mouse on 2>/dev/null \; new-session -A -s %q \; set-option window-size %s`,
		sessionName, windowSize,
	)
	if err := session.Start(cmd); err != nil {
		return fmt.Errorf("start tmux: %w", err)
	}

	// Read the TTY path from stdout (everything before the NUL delimiter).
	ttyBuf := make([]byte, 256)
	ttyPath := ""
	n, err := stdout.Read(ttyBuf)
	if err == nil && n > 0 {
		chunk := string(ttyBuf[:n])
		if idx := strings.IndexByte(chunk, 0); idx >= 0 {
			ttyPath = chunk[:idx]
			// Forward any remaining bytes after the NUL to the WebSocket
			remainder := chunk[idx+1:]
			if len(remainder) > 0 {
				_ = ws.Write(ctx, websocket.MessageBinary, []byte(remainder))
			}
		}
	}

	// Send the TTY path to the client as a control message
	if ttyPath != "" {
		ttyMsg, _ := json.Marshal(map[string]string{"type": "tty", "tty": ttyPath})
		_ = ws.Write(ctx, websocket.MessageText, ttyMsg)
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

	// When the context is canceled (WebSocket dropped), close the SSH
	// session so tmux gets SIGHUP and the PTY is released. Without this,
	// abandoned relays leak SSH sessions and PTY devices on the remote
	// host until it runs out ("open terminal failed: not a terminal").
	waitCh := make(chan error, 1)
	go func() {
		waitCh <- session.Wait()
	}()

	select {
	case err = <-waitCh:
		// SSH command exited normally (user detached/session killed)
	case <-ctx.Done():
		// WebSocket dropped — force-close the SSH session to free the PTY
		session.Close()
		err = <-waitCh
	}

	cancel()
	wg.Wait()

	if err != nil {
		log.Printf("relay session ended: %v", err)
	}

	// Determine close code: check if the session still exists to distinguish
	// "kicked/detached" (session alive, we were removed) from "session ended"
	// (session destroyed). Both prevent reconnect but show different messages.
	closeCode := websocket.StatusCode(4000) // session ended
	closeMsg := "session ended"
	if err == nil {
		// Clean exit — tmux detached us. Check if the session still exists.
		checkClient, checkErr := sshutil.Dial(address, user, keyPath)
		if checkErr == nil {
			out, _ := sshutil.Exec(checkClient, fmt.Sprintf("tmux has-session -t %q 2>/dev/null && echo yes", sessionName))
			checkClient.Close()
			if strings.TrimSpace(out) == "yes" {
				closeCode = 4001 // kicked/detached
				closeMsg = "detached by another client"
			}
		}
	}

	closeCtx, closeCancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer closeCancel()
	ws.Close(closeCode, closeMsg)
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
