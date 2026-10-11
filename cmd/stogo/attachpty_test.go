package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestFindCtrlV(t *testing.T) {
	cases := []struct {
		in      string
		at, len int
	}{
		{"abc", -1, 0},
		{"ab\x16c", 2, 1},
		{"x\x1b[118;5u", 1, 8},
		{"\x1b[27;5;118~\x16", 0, 11},
		{"\x1b[118;5uq\x16", 0, 8},
	}
	for _, c := range cases {
		at, n := findCtrlV([]byte(c.in))
		if at != c.at || (at >= 0 && n != c.len) {
			t.Errorf("findCtrlV(%q) = %d,%d want %d,%d", c.in, at, n, c.at, c.len)
		}
	}
}

func TestRelayInput(t *testing.T) {
	paste := func(path string, ok bool) func() (string, bool) {
		return func() (string, bool) { return path, ok }
	}
	cases := []struct {
		name, in string
		paste    func() (string, bool)
		want     string
	}{
		{"plain keys pass through", "ls -la\r", paste("", false), "ls -la\r"},
		{"ctrl+v with image pastes the path", "a\x16b", paste("/h/.ssh-to-go/paste/x.png", true),
			"a\x1b[200~/h/.ssh-to-go/paste/x.png \x1b[201~b"},
		{"ctrl+v without image stays ctrl+v", "a\x16b", paste("", false), "a\x16b"},
		{"extended-key ctrl+v without image is kept as sent", "\x1b[118;5u", paste("", false), "\x1b[118;5u"},
		{"two ctrl+v in one read", "\x16\x16", paste("/p", true), "\x1b[200~/p \x1b[201~\x1b[200~/p \x1b[201~"},
	}
	for _, c := range cases {
		var out bytes.Buffer
		relayInput(strings.NewReader(c.in), &out, c.paste)
		if out.String() != c.want {
			t.Errorf("%s: got %q want %q", c.name, out.String(), c.want)
		}
	}
}
