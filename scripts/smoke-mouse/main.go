// Command smoke-mouse is an end-to-end test for the alt-aware mouse-mode
// filter in the control-mode relay. It drives the real WebSocket endpoint
// against a running ssh-to-go instance and a fake fullscreen TUI (printf +
// cat) in a local tmux session, verifying:
//
//  1. Mouse-tracking enables are stripped while the pane is on the normal
//     buffer (shell keeps local wheel scrolling).
//  2. Once the pane enters the alternate buffer, mouse enables pass through
//     (and an enable stripped earlier is replayed on the alt switch).
//  3. A client mouse report (SGR wheel) typed at the "user" reaches the
//     TUI's stdin — the opencode scroll path.
//  4. Attaching while the TUI is already running re-asserts the alt buffer
//     and the pane's mouse modes (cmdAlt handshake).
//  5. Leaving the alt buffer force-disables the modes that were passed
//     through, so a dead TUI can't leave the client stuck in mouse mode.
//
// The host must be the machine this script runs on (the TUI's stdin sink is
// a local file, and the fake TUI is cleaned up via local tmux/pkill).
//
// Usage:
//
//	go run ./scripts/smoke-mouse -base http://127.0.0.1:8199 -host local
package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"nhooyr.io/websocket"
)

var (
	base = flag.String("base", "http://127.0.0.1:8199", "ssh-to-go base URL")
	host = flag.String("host", "local", "host name as configured in ssh-to-go")
)

var (
	session  string
	sinkFile string
)

func cleanup() {
	_ = exec.Command("tmux", "kill-session", "-t", session).Run()
	os.Remove(sinkFile)
}

func fatalf(format string, args ...any) {
	fmt.Printf("FAIL: "+format+"\n", args...)
	cleanup()
	os.Exit(1)
}

// conn accumulates everything the relay sends into one buffer; assertions
// scan slices of it between checkpoints.
type conn struct {
	ws  *websocket.Conn
	mu  sync.Mutex
	buf []byte
}

func (c *conn) readLoop(ctx context.Context) {
	for {
		typ, data, err := c.ws.Read(ctx)
		if err != nil {
			return
		}
		if typ == websocket.MessageBinary {
			c.mu.Lock()
			c.buf = append(c.buf, data...)
			c.mu.Unlock()
		}
	}
}

func (c *conn) checkpoint() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.buf)
}

func (c *conn) since(mark int) string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return string(c.buf[mark:])
}

func (c *conn) send(data []byte) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := c.ws.Write(ctx, websocket.MessageBinary, data); err != nil {
		fatalf("ws write: %v", err)
	}
}

func waitFor(c *conn, mark int, want string, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if strings.Contains(c.since(mark), want) {
			return true
		}
		time.Sleep(50 * time.Millisecond)
	}
	return false
}

func dial(hostArg, session string) *conn {
	u := strings.Replace(*base, "http", "ws", 1) +
		fmt.Sprintf("/ws/%s/%s?mouse=off&mode=control", hostArg, session)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	ws, _, err := websocket.Dial(ctx, u, nil)
	if err != nil {
		fatalf("dial %s: %v", u, err)
	}
	ws.SetReadLimit(1 << 20)
	c := &conn{ws: ws}
	go c.readLoop(context.Background())
	// Declare a grid size like the browser would.
	ws.Write(context.Background(), websocket.MessageText,
		[]byte(`{"type":"resize","cols":120,"rows":30}`))
	return c
}

// killTUI terminates the cat at the bottom of the fake TUI by pid (found via
// the pane's process tree, since a redirect doesn't show in argv).
func killTUI() {
	out, err := exec.Command("tmux", "display-message", "-p", "-t", session, "#{pane_pid}").Output()
	if err != nil {
		return
	}
	shellPID := strings.TrimSpace(string(out))
	_ = exec.Command("pkill", "-P", shellPID).Run()
}

func main() {
	flag.Parse()
	rand8 := make([]byte, 4)
	_, _ = rand.Read(rand8)
	session = "smoke-mouse-" + hex.EncodeToString(rand8)
	sinkFile = "/tmp/smoke-mouse-stdin-" + hex.EncodeToString(rand8) + ".bin"
	defer cleanup()

	// ── Phase 1: shell on the normal buffer — mouse enables are stripped.
	c1 := dial(*host, session)
	time.Sleep(1500 * time.Millisecond) // let attach + capture settle
	mark := c1.checkpoint()
	c1.send([]byte("printf '\\033[?1002h'\n"))
	if !waitFor(c1, mark, "printf", 5*time.Second) {
		fatalf("phase 1: typed command never echoed")
	}
	time.Sleep(1500 * time.Millisecond) // give the (stripped) seq time to arrive
	if got := c1.since(mark); strings.Contains(got, "\x1b[?1002h") {
		fatalf("phase 1: mouse enable leaked on normal buffer: %q", got)
	}
	fmt.Println("ok 1: mouse enable stripped on normal buffer")

	// ── Phase 2: fake TUI enters alt buffer and enables mouse. The enable
	// stripped in phase 1 must be replayed on the alt switch, and the TUI's
	// own enables must pass through.
	mark = c1.checkpoint()
	c1.send([]byte("printf '\\033[?1049h\\033[?1002h\\033[?1006h'; stty raw -echo; cat > " + sinkFile + "\n"))
	if !waitFor(c1, mark, "\x1b[?1049h", 5*time.Second) {
		fatalf("phase 2: alt-buffer switch missing from stream: %q", c1.since(mark))
	}
	if !waitFor(c1, mark, "\x1b[?1006h", 5*time.Second) {
		fatalf("phase 2: SGR mouse enable not passed through in alt buffer: %q", c1.since(mark))
	}
	got := c1.since(mark)
	altIdx := strings.Index(got, "\x1b[?1049h")
	if !strings.Contains(got[altIdx:], "\x1b[?1002h") {
		fatalf("phase 2: button-mouse enable missing after alt switch (replay+passthrough): %q", got)
	}
	fmt.Println("ok 2: alt switch passed, stripped enable replayed, live enables passed")

	// ── Phase 3: a wheel-up report sent as user input reaches the TUI's
	// stdin (cat's output file on the target host — same machine here).
	time.Sleep(500 * time.Millisecond) // let stty/cat start
	c1.send([]byte("\x1b[<64;10;5M"))
	deadline := time.Now().Add(8 * time.Second)
	for {
		if data, err := os.ReadFile(sinkFile); err == nil && strings.Contains(string(data), "\x1b[<64;10;5M") {
			break
		}
		if time.Now().After(deadline) {
			fatalf("phase 3: wheel report never reached the pane's stdin")
		}
		time.Sleep(100 * time.Millisecond)
	}
	fmt.Println("ok 3: SGR wheel report delivered to the pane's stdin")

	// ── Phase 4: a second client attaching NOW (TUI already running) gets
	// the alt switch and the pane's live mouse modes from the cmdAlt
	// handshake, ahead of the history capture.
	c2 := dial(*host, session)
	if !waitFor(c2, 0, "\x1b[?1049h", 8*time.Second) {
		fatalf("phase 4: attach sync did not enter alt buffer: %q", c2.since(0))
	}
	got2 := c2.since(0)
	if !strings.Contains(got2, "\x1b[?1002h") || !strings.Contains(got2, "\x1b[?1006h") {
		fatalf("phase 4: attach sync did not re-assert mouse modes: %q", got2)
	}
	fmt.Println("ok 4: attach to running TUI re-asserts alt buffer + mouse modes")

	// ── Phase 5: leaving the alt buffer force-disables the live modes.
	mark = c1.checkpoint()
	// Kill cat (it's in raw mode, so C-c is just a byte to it) and have the
	// shell leave the alt screen WITHOUT disabling mouse — like a TUI that
	// died before cleaning up.
	killTUI()
	time.Sleep(500 * time.Millisecond)
	c1.send([]byte("printf '\\033[?1049l'\n"))
	if !waitFor(c1, mark, "\x1b[?1049l", 5*time.Second) {
		fatalf("phase 5: alt exit missing: %q", c1.since(mark))
	}
	time.Sleep(500 * time.Millisecond)
	got = c1.since(mark)
	exitIdx := strings.Index(got, "\x1b[?1049l")
	tail := got[exitIdx:]
	if !strings.Contains(tail, "\x1b[?1002l") || !strings.Contains(tail, "\x1b[?1006l") {
		fatalf("phase 5: modes not force-disabled on alt exit: %q", tail)
	}
	fmt.Println("ok 5: alt exit force-disables live mouse modes")

	c1.ws.Close(websocket.StatusNormalClosure, "done")
	c2.ws.Close(websocket.StatusNormalClosure, "done")
	fmt.Println("PASS: all mouse-filter phases")
}
