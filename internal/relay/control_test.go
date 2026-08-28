package relay

import (
	"bytes"
	"strings"
	"testing"
)

func TestUnescapeControl(t *testing.T) {
	cases := []struct {
		in   string
		want []byte
	}{
		{"plain text", []byte("plain text")},
		{`a\015\012b`, []byte("a\r\nb")},         // octal CR LF
		{`\033[31mred`, []byte("\x1b[31mred")},   // octal ESC
		{`back\\slash`, []byte(`back\slash`)},    // escaped backslash
		{`caf\303\251`, []byte("café")},          // UTF-8 via octal bytes
		{`tab\011nl\012`, []byte("tab\tnl\n")},   // octal control chars
		{`c\tstyle\r\n`, []byte("c\tstyle\r\n")}, // C-style fallbacks
		{`dangling\`, []byte(`dangling\`)},       // trailing backslash
		{`\7x`, []byte(`\7x`)},                   // too-short octal passes through
		{`\377`, []byte{0xff}},                   // max octal byte
	}
	for _, c := range cases {
		if got := unescapeControl(c.in); !bytes.Equal(got, c.want) {
			t.Errorf("unescapeControl(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestHexWords(t *testing.T) {
	if got := hexWords([]byte("hi\r")); got != "68 69 0d" {
		t.Errorf("hexWords = %q", got)
	}
}

// feed pushes raw protocol lines through the parser.
func feed(p *controlParser, lines ...string) {
	for _, l := range lines {
		p.feedLine(l)
	}
}

func TestParserHistoryThenLive(t *testing.T) {
	var emitted []byte
	p := &controlParser{emit: func(b []byte) { emitted = append(emitted, b...) }}

	// Connect reply block (queued before any stdin command).
	p.pushCmd(cmdConnect)
	feed(p, "%begin 100 1 0", "%end 100 1 0")

	// %output arriving before the capture reply must be dropped — its bytes
	// are already inside the capture.
	feed(p, `%output %0 already-in-history`)
	if len(emitted) != 0 {
		t.Fatalf("pre-history %%output leaked: %q", emitted)
	}

	// set-option reply.
	p.pushCmd(cmdIgnore)
	feed(p, "%begin 100 2 0", "%end 100 2 0")

	// Meta reply: client name + active pane.
	p.pushCmd(cmdMeta)
	var metaName string
	p.onMeta = func(n string) { metaName = n }
	feed(p, "%begin 100 3 0", "client-4242,%0", "%end 100 3 0")
	if metaName != "client-4242" {
		t.Errorf("client name = %q", metaName)
	}
	if p.paneID != "%0" {
		t.Errorf("paneID = %q", p.paneID)
	}

	// Capture reply: history lines, trailing blank pane rows trimmed, a
	// content line that looks like "%end" with the WRONG ids stays content.
	p.pushCmd(cmdHistory)
	feed(p,
		"%begin 100 4 0",
		"$ echo hello",
		"hello",
		"%end 999 999 0", // spoof — pane content, ids don't match
		"$ ",
		"",
		"   ",
		"%end 100 4 0",
	)
	want := "$ echo hello\r\nhello\r\n%end 999 999 0\r\n$ \x1b[0m"
	if string(emitted) != want {
		t.Errorf("history payload = %q, want %q", emitted, want)
	}

	// Live output for the active pane flows; other panes are filtered.
	emitted = nil
	feed(p, `%output %0 ls\015\012`)
	feed(p, `%output %7 hidden pane noise`)
	if string(emitted) != "ls\r\n" {
		t.Errorf("live output = %q", emitted)
	}

	// Active pane change redirects the filter.
	feed(p, "%window-pane-changed @1 %7")
	emitted = nil
	feed(p, `%output %7 now-visible`)
	feed(p, `%output %0 now-hidden`)
	if string(emitted) != "now-visible" {
		t.Errorf("after pane change = %q", emitted)
	}
}

func TestParserErrorBlock(t *testing.T) {
	var emitted []byte
	p := &controlParser{emit: func(b []byte) { emitted = append(emitted, b...) }}

	p.pushCmd(cmdHistory)
	feed(p, "%begin 5 9 0", "bad command", "%error 5 9 0")
	if len(emitted) != 0 {
		t.Errorf("error block emitted output: %q", emitted)
	}
	// History gate must still open so live output isn't swallowed forever.
	if !p.historyDone {
		t.Error("historyDone not set after error reply")
	}
	feed(p, `%output %0 after-error`)
	if string(emitted) != "after-error" {
		t.Errorf("live output after error = %q", emitted)
	}
}

func TestParserEmptyQueueIsIgnore(t *testing.T) {
	p := &controlParser{emit: func(b []byte) { t.Errorf("unexpected emit: %q", b) }}
	// Unsolicited block (shouldn't happen, but must not panic or emit).
	feed(p, "%begin 1 1 0", "stray", "%end 1 1 0")
}

func TestTmuxQuote(t *testing.T) {
	if got := tmuxQuote("my-session"); got != "'my-session'" {
		t.Errorf("tmuxQuote = %q", got)
	}
	if got := tmuxQuote("a'b"); got != "'ab'" {
		t.Errorf("tmuxQuote strips quotes: %q", got)
	}
}

func TestLayoutDims(t *testing.T) {
	cases := []struct {
		in   string
		w, h int
		ok   bool
	}{
		{"b25d,120x30,0,0,1", 120, 30, true},
		{"cafe,200x50,0,0{100x50,0,0,1,99x50,101,0,2}", 200, 50, true},
		{"dead,93x35,0,0[93x17,0,0,1,93x17,0,18,2]", 93, 35, true},
		{"bogus", 0, 0, false},
		{"abcd,notdims,0,0", 0, 0, false},
	}
	for _, c := range cases {
		w, h, ok := layoutDims(c.in)
		if w != c.w || h != c.h || ok != c.ok {
			t.Errorf("layoutDims(%q) = %d,%d,%v want %d,%d,%v", c.in, w, h, ok, c.w, c.h, c.ok)
		}
	}
}

func TestParserLayoutChangeCallback(t *testing.T) {
	var gotW, gotH int
	p := &controlParser{onLayout: func(w, h int) { gotW, gotH = w, h }}
	feed(p, "%layout-change @1 b25d,120x30,0,0,1 b25d,120x30,0,0,1 *")
	if gotW != 120 || gotH != 30 {
		t.Errorf("onLayout got %dx%d, want 120x30", gotW, gotH)
	}
}

// A repaint after a foreign resize must clear the screen, paint the captured
// frame, gate %output while pending, and restore the cursor from the
// cmdCursor reply.
func TestParserRepaintSequence(t *testing.T) {
	var emitted []byte
	p := &controlParser{emit: func(b []byte) { emitted = append(emitted, b...) }}
	p.openOutput() // attach finished long ago

	// Foreign resize detected: relay gates output and queues the repaint.
	p.gateOutput()
	p.pushCmd(cmdRepaint)
	p.pushCmd(cmdCursor)

	// %output racing the repaint is stale — the capture will contain it.
	feed(p, `%output %0 stale-width-garbage`)
	if len(emitted) != 0 {
		t.Fatalf("gated %%output leaked: %q", emitted)
	}

	feed(p,
		"%begin 7 1 0",
		"frame line 1",
		"frame line 2",
		"",
		"%end 7 1 0",
	)
	want := "\x1b[2J\x1b[Hframe line 1\r\nframe line 2\x1b[0m"
	if string(emitted) != want {
		t.Errorf("repaint payload = %q, want %q", emitted, want)
	}

	// Cursor restore: 0-based x,y -> 1-based CUP row;col.
	emitted = nil
	feed(p, "%begin 7 2 0", "5,2", "%end 7 2 0")
	if string(emitted) != "\x1b[3;6H" {
		t.Errorf("cursor restore = %q, want ESC[3;6H", emitted)
	}

	// Live output flows again after the repaint reply.
	emitted = nil
	feed(p, `%output %0 fresh`)
	if string(emitted) != "fresh" {
		t.Errorf("post-repaint output = %q", emitted)
	}
}

func TestParserHistoryEmptySession(t *testing.T) {
	p := &controlParser{emit: func(b []byte) { t.Errorf("empty capture emitted: %q", b) }}
	p.pushCmd(cmdHistory)
	// A brand-new session captures only blank rows.
	feed(p, "%begin 1 1 0", "", "", "", "%end 1 1 0")
	if !p.historyDone {
		t.Error("historyDone not set")
	}
	_ = strings.TrimSpace("")
}

// Attaching while a fullscreen TUI already owns the pane: the cmdAlt reply
// must enter the client's alt buffer AND re-assert the pane's mouse modes,
// so the TUI (opencode) keeps receiving real wheel reports instead of
// xterm.js's alt-scroll arrow keys.
func TestParserAltSyncReassertsMouseModes(t *testing.T) {
	var emitted []byte
	p := &controlParser{emit: func(b []byte) { emitted = append(emitted, b...) }}

	// alternate_on=1, button(1002)+sgr(1006) set, std/all/utf8 unset.
	p.pushCmd(cmdAlt)
	feed(p, "%begin 3 1 0", "1,0,1,0,1,0", "%end 3 1 0")
	want := "\x1b[?1049h\x1b[?1002h\x1b[?1006h"
	if string(emitted) != want {
		t.Errorf("alt sync = %q, want %q", emitted, want)
	}

	// Normal buffer: nothing emitted, regardless of the mouse flags.
	emitted = nil
	p2 := &controlParser{emit: func(b []byte) { emitted = append(emitted, b...) }}
	p2.pushCmd(cmdAlt)
	feed(p2, "%begin 3 2 0", "0,0,1,0,1,0", "%end 3 2 0")
	if len(emitted) != 0 {
		t.Errorf("normal-buffer alt sync emitted: %q", emitted)
	}

	// Older tmux without the mouse format names answers "1" alone.
	emitted = nil
	p3 := &controlParser{emit: func(b []byte) { emitted = append(emitted, b...) }}
	p3.pushCmd(cmdAlt)
	feed(p3, "%begin 3 3 0", "1", "%end 3 3 0")
	if string(emitted) != "\x1b[?1049h" {
		t.Errorf("legacy alt sync = %q, want just the alt switch", emitted)
	}
}
