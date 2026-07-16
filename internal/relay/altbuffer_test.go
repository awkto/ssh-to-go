package relay

import "testing"

// Alt-screen buffer switches must pass THROUGH untouched so a fullscreen TUI
// (claude code, vim, …) can drive the client's alternate buffer and render
// each frame cleanly instead of stacking frames on the scrolling buffer
// (issue #59).
func TestStripperPassesAltScreenThrough(t *testing.T) {
	altSeqs := []string{
		"\x1b[?1049h", "\x1b[?1049l",
		"\x1b[?1047h", "\x1b[?1047l",
		"\x1b[?47h", "\x1b[?47l",
	}
	for _, seq := range altSeqs {
		s := &altBufferStripper{}
		in := "before" + seq + "after"
		got := string(s.Process([]byte(in)))
		if got != in {
			t.Errorf("alt-screen seq %q was altered: got %q, want %q", seq, got, in)
		}
	}
}

// Mouse-tracking DECSET enables must still be stripped so line-oriented shell
// output keeps the client's native (local scrollback) wheel scrolling.
func TestStripperStripsMouseTracking(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"a\x1b[?1000hb", "ab"},
		{"a\x1b[?1002hb", "ab"},
		{"a\x1b[?1003lb", "ab"},
		{"a\x1b[?1006hb", "ab"},
		{"a\x1b[?9hb", "ab"},
	}
	for _, c := range cases {
		s := &altBufferStripper{}
		if got := string(s.Process([]byte(c.in))); got != c.want {
			t.Errorf("Process(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

// A strip sequence split across two Process calls must still be removed, and a
// pass-through sequence split across two calls must still be reassembled.
func TestStripperHandlesSplitReads(t *testing.T) {
	// mouse enable split mid-sequence -> stripped
	s := &altBufferStripper{}
	out := string(s.Process([]byte("x\x1b[?10")))
	out += string(s.Process([]byte("00hy")))
	if out != "xy" {
		t.Errorf("split mouse seq: got %q, want %q", out, "xy")
	}

	// alt-screen split mid-sequence -> passes through intact
	s2 := &altBufferStripper{}
	out2 := string(s2.Process([]byte("x\x1b[?104")))
	out2 += string(s2.Process([]byte("9hy")))
	if out2 != "x\x1b[?1049hy" {
		t.Errorf("split alt seq: got %q, want %q", out2, "x\x1b[?1049hy")
	}
}
