package relay

import "testing"

// Alt-screen buffer switches must pass THROUGH untouched so a fullscreen TUI
// (opencode, vim, …) can drive the client's alternate buffer and render
// each frame cleanly instead of stacking frames on the scrolling buffer
// (issue #59).
func TestFilterPassesAltScreenThrough(t *testing.T) {
	altSeqs := []string{
		"\x1b[?1049h", "\x1b[?1049l",
		"\x1b[?1047h", "\x1b[?1047l",
		"\x1b[?47h", "\x1b[?47l",
	}
	for _, seq := range altSeqs {
		s := &mouseModeFilter{}
		in := "before" + seq + "after"
		got := string(s.Process([]byte(in)))
		if got != in {
			t.Errorf("alt-screen seq %q was altered: got %q, want %q", seq, got, in)
		}
	}
}

// Mouse-tracking ENABLES are stripped on the normal buffer so line-oriented
// shell output keeps the client's native (local scrollback) wheel scrolling.
// DISABLES always pass: harmless no-op, and they keep the client's state
// machine in step with the app.
func TestFilterStripsMouseEnablesOnNormalBuffer(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"a\x1b[?1000hb", "ab"},
		{"a\x1b[?1002hb", "ab"},
		{"a\x1b[?1006hb", "ab"},
		{"a\x1b[?1016hb", "ab"},
		{"a\x1b[?9hb", "ab"},
		// Disables pass through even on the normal buffer.
		{"a\x1b[?1003lb", "a\x1b[?1003lb"},
		{"a\x1b[?1015lb", "a\x1b[?1015lb"},
		{"a\x1b[?1016lb", "a\x1b[?1016lb"},
		{"a\x1b[?9lb", "a\x1b[?9lb"},
	}
	for _, c := range cases {
		s := &mouseModeFilter{}
		if got := string(s.Process([]byte(c.in))); got != c.want {
			t.Errorf("Process(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

// On the alternate buffer, mouse-tracking sequences pass through so a
// fullscreen TUI (opencode) receives real wheel reports and scrolls its own
// feed — instead of xterm.js's alt-scroll fallback turning the wheel into
// arrow keys (prompt-history navigation).
func TestFilterPassesMouseSeqsOnAltBuffer(t *testing.T) {
	s := &mouseModeFilter{}
	out := string(s.Process([]byte("\x1b[?1049h")))
	out += string(s.Process([]byte("a\x1b[?1002hb\x1b[?1006hc")))
	want := "\x1b[?1049h" + "a\x1b[?1002hb\x1b[?1006hc"
	if out != want {
		t.Errorf("alt-buffer mouse seqs: got %q, want %q", out, want)
	}
}

// An app that enables mouse tracking BEFORE switching to the alt buffer had
// the enable stripped; entering the alt buffer replays it.
func TestFilterReplaysStrippedEnableOnAltEnter(t *testing.T) {
	s := &mouseModeFilter{}
	out := string(s.Process([]byte("\x1b[?1002h")))   // stripped (normal buffer)
	out += string(s.Process([]byte("\x1b[?1006h")))   // stripped (normal buffer)
	out += string(s.Process([]byte("\x1b[?1049h")))   // alt enter + replay
	want := "\x1b[?1049h\x1b[?1002h\x1b[?1006h"
	if out != want {
		t.Errorf("replay on alt enter: got %q, want %q", out, want)
	}
}

// Leaving the alt buffer force-disables any mouse modes still live on the
// client, so a TUI killed before sending its own disables can't leave the
// client stuck in mouse mode on the normal buffer (which would silently
// kill local scrolling for the shell the user lands back in).
func TestFilterForceDisablesOnAltExit(t *testing.T) {
	s := &mouseModeFilter{}
	out := string(s.Process([]byte("\x1b[?1049h\x1b[?1002h")))
	out += string(s.Process([]byte("\x1b[?1049l")))
	want := "\x1b[?1049h\x1b[?1002h" + "\x1b[?1049l\x1b[?1002l"
	if out != want {
		t.Errorf("force disable on exit: got %q, want %q", out, want)
	}

	// After the exit, the normal-buffer rule applies again: enables stripped.
	out += string(s.Process([]byte("\x1b[?1002h")))
	if out != want {
		t.Errorf("post-exit enable not stripped: got %q, want %q", out, want)
	}
}

// An app that cleanly disables tracking before leaving the alt buffer must
// not get a second, redundant disable synthesized.
func TestFilterNoDoubleDisableWhenAppCleansUp(t *testing.T) {
	s := &mouseModeFilter{}
	out := string(s.Process([]byte("\x1b[?1049h\x1b[?1002h")))
	out += string(s.Process([]byte("\x1b[?1002l\x1b[?1049l")))
	want := "\x1b[?1049h\x1b[?1002h" + "\x1b[?1002l\x1b[?1049l"
	if out != want {
		t.Errorf("clean shutdown: got %q, want %q", out, want)
	}
}

// A disable arriving on the normal buffer cancels a remembered stripped
// enable, so it isn't replayed on a later alt-buffer entry.
func TestFilterDisableCancelsStrippedEnable(t *testing.T) {
	s := &mouseModeFilter{}
	out := string(s.Process([]byte("\x1b[?1002h")))   // stripped
	out += string(s.Process([]byte("\x1b[?1002l")))   // passes; cancels memory
	out += string(s.Process([]byte("\x1b[?1049h")))   // nothing to replay
	want := "\x1b[?1002l" + "\x1b[?1049h"
	if out != want {
		t.Errorf("disable cancels memory: got %q, want %q", out, want)
	}
}

// A tracked sequence split across two Process calls must still be handled,
// whichever family it belongs to.
func TestFilterHandlesSplitReads(t *testing.T) {
	// mouse enable split mid-sequence -> stripped (normal buffer)
	s := &mouseModeFilter{}
	out := string(s.Process([]byte("x\x1b[?10")))
	out += string(s.Process([]byte("00hy")))
	if out != "xy" {
		t.Errorf("split mouse seq: got %q, want %q", out, "xy")
	}

	// alt-screen split mid-sequence -> passes through intact, state tracked
	s2 := &mouseModeFilter{}
	out2 := string(s2.Process([]byte("x\x1b[?104")))
	out2 += string(s2.Process([]byte("9hy")))
	if out2 != "x\x1b[?1049hy" {
		t.Errorf("split alt seq: got %q, want %q", out2, "x\x1b[?1049hy")
	}
	if !s2.alt {
		t.Error("alt state not set after split ?1049h")
	}

	// mouse enable split across the boundary, in alt mode -> passes
	out3 := string(s2.Process([]byte("a\x1b[?10")))
	out3 += string(s2.Process([]byte("02hb")))
	if out3 != "a\x1b[?1002hb" {
		t.Errorf("split mouse seq in alt: got %q, want %q", out3, "a\x1b[?1002hb")
	}
}
