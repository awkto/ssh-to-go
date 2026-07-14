package tmux

import "testing"

func TestParseWxH(t *testing.T) {
	cases := []struct {
		in   string
		w, h int
		ok   bool
	}{
		{"200x50", 200, 50, true},
		{"80x24", 80, 24, true},
		{"largest", 0, 0, false},
		{"smallest", 0, 0, false},
		{"latest", 0, 0, false},
		{"", 0, 0, false},
		{"200x", 0, 0, false},
		{"x50", 0, 0, false},
		{"0x50", 0, 0, false},
		{"200x0", 0, 0, false},
		{"axb", 0, 0, false},
	}
	for _, c := range cases {
		w, h, ok := parseWxH(c.in)
		if ok != c.ok || (ok && (w != c.w || h != c.h)) {
			t.Errorf("parseWxH(%q) = (%d,%d,%v), want (%d,%d,%v)", c.in, w, h, ok, c.w, c.h, c.ok)
		}
	}
}
