// Command smoke-control is an end-to-end smoke test for the control-mode
// relay (mode=control). It drives the real WebSocket endpoint against a
// running ssh-to-go instance and verifies the property the pipeline exists
// for: after a disconnect/reconnect, the replayed history is accurate —
// every line present exactly once, no attach-repaint duplicates, no control
// protocol leakage.
//
// Usage:
//
//	go run ./scripts/smoke-control -base http://127.0.0.1:8099 -host pro -session smoke-ctl
package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"nhooyr.io/websocket"
)

var (
	base    = flag.String("base", "http://127.0.0.1:8099", "ssh-to-go base URL")
	host    = flag.String("host", "pro", "host name as configured in ssh-to-go")
	session = flag.String("session", "smoke-ctl", "tmux session name to create/use")
	keep    = flag.Bool("keep", false, "don't kill the session afterwards")
)

func apiURL(path string) string { return *base + path }

func wsURL() string {
	u := strings.Replace(*base, "http", "ws", 1)
	return fmt.Sprintf("%s/ws/%s/%s?mouse=off&mode=control", u, *host, *session)
}

// conn wraps a websocket with a persistent reader goroutine. nhooyr's Read
// closes the whole connection if its context expires, so polling with short
// read deadlines is not an option — we read continuously into a buffer.
type conn struct {
	ws *websocket.Conn

	mu       sync.Mutex
	buf      strings.Builder
	tty      string
	lastByte time.Time
	closed   bool
}

func dial(ctx context.Context) (*conn, error) {
	ws, _, err := websocket.Dial(ctx, wsURL(), nil)
	if err != nil {
		return nil, err
	}
	ws.SetReadLimit(1 << 20)
	c := &conn{ws: ws, lastByte: time.Now()}

	go func() {
		for {
			typ, data, err := ws.Read(ctx)
			if err != nil {
				c.mu.Lock()
				c.closed = true
				c.mu.Unlock()
				return
			}
			c.mu.Lock()
			if typ == websocket.MessageBinary {
				c.buf.Write(data)
				c.lastByte = time.Now()
			} else {
				var msg map[string]string
				if json.Unmarshal(data, &msg) == nil && msg["type"] == "tty" {
					c.tty = msg["tty"]
				}
			}
			c.mu.Unlock()
		}
	}()

	if err := ws.Write(ctx, websocket.MessageText, []byte(`{"type":"resize","cols":100,"rows":30}`)); err != nil {
		return nil, err
	}
	return c, nil
}

func (c *conn) snapshot() (string, bool, time.Time) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.buf.String(), c.closed, c.lastByte
}

// waitFor polls until the accumulated output contains want.
func (c *conn) waitFor(want string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for {
		s, closed, _ := c.snapshot()
		if strings.Contains(s, want) {
			return nil
		}
		if closed {
			return fmt.Errorf("connection closed while waiting for %q; got %d bytes", want, len(s))
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("timeout waiting for %q; got %d bytes", want, len(s))
		}
		time.Sleep(50 * time.Millisecond)
	}
}

// settle waits until no output has arrived for quiet (or timeout passes).
func (c *conn) settle(quiet, timeout time.Duration) {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		_, closed, last := c.snapshot()
		if closed || time.Since(last) >= quiet {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
}

func (c *conn) typeLine(ctx context.Context, line string) error {
	return c.ws.Write(ctx, websocket.MessageBinary, []byte(line+"\r"))
}

func nonce() string {
	b := make([]byte, 4)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func fail(format string, args ...any) {
	fmt.Printf("FAIL: "+format+"\n", args...)
	os.Exit(1)
}

func killSession() {
	req, _ := http.NewRequest(http.MethodDelete, apiURL("/api/hosts/"+*host+"/sessions/"+*session), nil)
	if resp, err := http.DefaultClient.Do(req); err == nil {
		resp.Body.Close()
		fmt.Printf("killed session %q: HTTP %d\n", *session, resp.StatusCode)
	}
}

func main() {
	flag.Parse()
	ctx := context.Background()

	body := strings.NewReader(fmt.Sprintf(`{"name":%q}`, *session))
	resp, err := http.Post(apiURL("/api/hosts/"+*host+"/sessions"), "application/json", body)
	if err != nil {
		fail("create session: %v", err)
	}
	resp.Body.Close()
	fmt.Printf("create session %q: HTTP %d\n", *session, resp.StatusCode)

	pass := false
	if !*keep {
		defer func() {
			if pass {
				killSession()
			} else {
				fmt.Printf("keeping session %q for debugging\n", *session)
			}
		}()
	}

	mark := nonce()

	// --- Connection 1: generate uniquely identifiable output. ---
	c1, err := dial(ctx)
	if err != nil {
		fail("dial #1: %v", err)
	}
	c1.settle(2*time.Second, 15*time.Second) // initial history + prompt

	// printf so the marker string appears in the OUTPUT only — the typed
	// command line contains "OUT-%s", never the contiguous marker — which
	// lets us count occurrences exactly.
	if err := c1.typeLine(ctx, fmt.Sprintf(`printf 'OUT-%%s\n' %s; seq 1 60`, mark)); err != nil {
		fail("send command: %v", err)
	}
	if err := c1.waitFor("OUT-"+mark, 10*time.Second); err != nil {
		fail("%v", err)
	}
	if err := c1.waitFor("\r\n60", 10*time.Second); err != nil {
		fail("seq output incomplete: %v", err)
	}
	_, _, _ = c1.snapshot()
	c1.mu.Lock()
	tty := c1.tty
	c1.mu.Unlock()
	if tty == "" {
		fail("no tty/client-name control message received on connection 1")
	}
	fmt.Printf("conn1 OK: marker echoed live, client name %q\n", tty)
	c1.ws.Close(websocket.StatusNormalClosure, "done")
	time.Sleep(1 * time.Second)

	// --- Connection 2: reconnect, history must contain everything exactly once. ---
	c2, err := dial(ctx)
	if err != nil {
		fail("dial #2: %v", err)
	}
	if err := c2.waitFor("OUT-"+mark, 10*time.Second); err != nil {
		fail("history prefill missing marker: %v", err)
	}
	c2.settle(2*time.Second, 15*time.Second)
	hist, _, _ := c2.snapshot()

	if n := strings.Count(hist, "OUT-"+mark); n != 1 {
		fail("marker appears %d times in replayed history, want exactly 1 (duplicate/cutoff bug)", n)
	}
	for i := 1; i <= 60; i++ {
		if !strings.Contains(hist, fmt.Sprintf("\r\n%d", i)) {
			fail("history missing seq line %d (cutoff)", i)
		}
	}
	for _, leak := range []string{"%begin", "%end ", "%output", "%exit"} {
		if strings.Contains(hist, leak) {
			fail("control protocol leaked into terminal stream: %q", leak)
		}
	}

	// Live stream must work after the prefill gate opens.
	if err := c2.typeLine(ctx, "echo LIVE-"+mark); err != nil {
		fail("send live command: %v", err)
	}
	if err := c2.waitFor("LIVE-"+mark, 10*time.Second); err != nil {
		fail("no live output after history prefill (gate stuck): %v", err)
	}

	// --- Kick flow: a second client detaches the first by client name. ---
	c3, err := dial(ctx)
	if err != nil {
		fail("dial #3: %v", err)
	}
	c3.settle(2*time.Second, 15*time.Second)
	c3.mu.Lock()
	c3name := c3.tty
	c3.mu.Unlock()
	if c3name == "" {
		fail("conn3 got no client-name control message")
	}

	kickBody := strings.NewReader(fmt.Sprintf(`{"exclude_tty":%q}`, c3name))
	resp, err = http.Post(apiURL("/api/hosts/"+*host+"/sessions/"+*session+"/detach-clients"), "application/json", kickBody)
	if err != nil {
		fail("detach-clients: %v", err)
	}
	var kickResult struct {
		Detached int `json:"detached"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&kickResult)
	resp.Body.Close()
	if kickResult.Detached < 1 {
		fail("detach-clients detached %d clients, want >= 1 (client_name kick broken)", kickResult.Detached)
	}

	// c2 must observe its connection closing; c3 must survive.
	deadline := time.Now().Add(10 * time.Second)
	for {
		_, c2closed, _ := c2.snapshot()
		if c2closed {
			break
		}
		if time.Now().After(deadline) {
			fail("kicked connection 2 never closed")
		}
		time.Sleep(100 * time.Millisecond)
	}
	if err := c3.typeLine(ctx, "echo SURVIVED-"+mark); err != nil {
		fail("conn3 dead after kicking others: %v", err)
	}
	if err := c3.waitFor("SURVIVED-"+mark, 10*time.Second); err != nil {
		fail("conn3 not live after kick: %v", err)
	}
	fmt.Printf("kick OK: detached %d client(s), excluded survivor still live\n", kickResult.Detached)
	c3.ws.Close(websocket.StatusNormalClosure, "done")

	pass = true
	fmt.Println("PASS: history accurate after reconnect (1x marker, seq 1-60 complete, no protocol leakage, live stream flows)")
}
