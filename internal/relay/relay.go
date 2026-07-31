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

// Options controls per-attach relay behavior.
type Options struct {
	// Mouse controls whether tmux mouse mode is enabled for this attach.
	// Default ("on" or "") matches historical behavior for the web client.
	// "off" skips the set-option so the client can handle scroll gestures
	// locally without tmux entering copy-mode (used by native clients that
	// render their own smooth-scroll scrollback).
	Mouse string

	// Mode selects the relay pipeline. "" = classic PTY attach (with the
	// Mouse-dependent capture-pane prefill). "control" = tmux control mode:
	// history prefilled over the control channel, live output streamed as
	// %output events. No screen repaints, so the browser emulator owns the
	// scrollback with no duplicates or width clipping. See control.go.
	Mode string

	// HistoryLimit, when >0, is set globally on the tmux server before any
	// session this relay creates, so new panes get deeper scrollback. tmux
	// can't grow an existing pane's buffer, so this only helps fresh sessions.
	HistoryLimit int

	// Host is the ssh-to-go host name this session belongs to. Only used to
	// key the deliberate-termination registry (see terminate.go) — the relay
	// itself dials by address. Empty disables termination signalling.
	Host string
}

// Relay bridges a WebSocket connection to a tmux session over SSH.
func Relay(ctx context.Context, ws *websocket.Conn, address, user, keyPath, sessionName, windowSize string) error {
	return RelayWithOptions(ctx, ws, address, user, keyPath, sessionName, windowSize, Options{})
}

// RelayWithOptions is the same as Relay but accepts per-attach Options.
func RelayWithOptions(ctx context.Context, ws *websocket.Conn, address, user, keyPath, sessionName, windowSize string, opts Options) error {
	client, err := sshutil.Dial(address, user, keyPath)
	if err != nil {
		return fmt.Errorf("ssh dial: %w", err)
	}
	defer client.Close()

	// Start keepalive
	done := make(chan struct{})
	defer close(done)
	go sshutil.KeepAlive(client, 15_000_000_000, done) // 15s

	if opts.Mode == "control" {
		return relayControlMode(ctx, ws, client, sessionName, windowSize, opts.HistoryLimit, opts.Host)
	}

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
	// Chained AFTER new-session so the session exists, and deliberately
	// without -g: the old server-global write leaked mouse mode to every
	// session on the host and nothing ever restored it. Without -g or -t,
	// set-option applies to the session this client just attached.
	mouseOpt := ` \; set-option mouse on`
	if opts.Mouse == "off" {
		mouseOpt = ""
	}

	// For native clients running with mouse=off we don't let tmux handle
	// scrollback (its row-quantised wheel binding ruins smooth scroll on
	// touch devices). Instead we dump up-to-5000 lines of pane history
	// over the wire BEFORE attaching, so the client's local emulator
	// fills its scrollback with the real session history. tmux's attach
	// then paints the live frame on top; the scrollback above it remains.
	// Fold history-limit INTO the new-session invocations. A pane's scrollback
	// depth is fixed at creation, and a standalone `tmux set -g history-limit`
	// is useless here: with no session it starts a server that exits before
	// the next command runs, so the pane is born in a fresh default server.
	// Empty when HistoryLimit<=0.
	histInline := ""
	if opts.HistoryLimit > 0 {
		histInline = fmt.Sprintf("set-option -g history-limit %d \\; ", opts.HistoryLimit)
	}
	var cmd string
	if opts.Mouse == "off" {
		cmd = fmt.Sprintf(
			`printf '%%s\x00' "$(tty)"; `+
				`tmux has-session -t %q 2>/dev/null || tmux %snew-session -d -s %q; `+
				`tmux capture-pane -p -e -S -5000 -E -1 -t %q 2>/dev/null; `+
				`printf '\x1b[0m'; `+
				`exec tmux %snew-session -A -s %q \; set-option window-size %s`,
			sessionName, histInline, sessionName, sessionName, histInline, sessionName, windowSize,
		)
	} else {
		cmd = fmt.Sprintf(
			`printf '%%s\x00' "$(tty)"; exec tmux %snew-session -A -s %q \; set-option window-size %s%s`,
			histInline, sessionName, windowSize, mouseOpt,
		)
	}
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

	// Register a kick channel so the detach handler can signal us before
	// tmux detaches the client. This lets us send a control message to
	// the browser while the WebSocket is still healthy.
	wasKicked := false
	var kickCh chan struct{}
	if ttyPath != "" {
		kickCh = RegisterKickCh(ttyPath)
		defer UnregisterKickCh(ttyPath)
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	var wg sync.WaitGroup

	// Watch for a deliberate kill/offload of this session — tell the browser
	// why while the socket is still healthy, so it suppresses its reconnect
	// instead of recreating the session with `new-session -A`.
	termReason := ""
	if opts.Host != "" {
		termCh := RegisterTerminateCh(opts.Host, sessionName)
		defer UnregisterTerminateCh(opts.Host, sessionName, termCh)
		wg.Add(1)
		go func() {
			defer wg.Done()
			select {
			case reason := <-termCh:
				termReason = reason
				msg, _ := json.Marshal(map[string]string{"type": "terminated", "reason": reason})
				_ = ws.Write(ctx, websocket.MessageText, msg)
			case <-ctx.Done():
			}
		}()
	}

	// Watch for kick signal — send control message to browser before tmux detaches us
	if kickCh != nil {
		wg.Add(1)
		go func() {
			defer wg.Done()
			select {
			case <-kickCh:
				wasKicked = true
				kickMsg, _ := json.Marshal(map[string]string{"type": "kicked"})
				_ = ws.Write(ctx, websocket.MessageText, kickMsg)
			case <-ctx.Done():
			}
		}()
	}

	// SSH stdout -> WebSocket. For mouse=off attaches we also strip
	// alt-screen-buffer escape sequences so TUI apps' output flows into
	// the client emulator's main buffer (and scrollback), letting native
	// clients smooth-scroll over it.
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer cancel()
		var stripper *altBufferStripper
		if opts.Mouse == "off" {
			stripper = &altBufferStripper{}
		}
		buf := make([]byte, 32*1024)
		for {
			n, err := stdout.Read(buf)
			if n > 0 {
				out := buf[:n]
				if stripper != nil {
					out = stripper.Process(out)
				}
				if len(out) > 0 {
					if writeErr := ws.Write(ctx, websocket.MessageBinary, out); writeErr != nil {
						return
					}
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

	closeCode := websocket.StatusCode(4000) // session ended
	closeMsg := "session ended"
	if wasKicked {
		closeCode = 4001
		closeMsg = "detached by another client"
	}
	if termReason != "" {
		closeCode = 4002
		closeMsg = termReason
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
