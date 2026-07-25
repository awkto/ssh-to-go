package tmux

import "testing"

func TestPanesQuiet(t *testing.T) {
	cases := []struct {
		name string
		out  string
		want bool
	}{
		{"single shell at a prompt", "bash\n", true},
		{"several shells across windows", "bash\nzsh\nfish\n", true},
		{"one pane running a build", "bash\nmake\n", false},
		{"an agent running", "claude\n", false},
		// A shell inside ssh/docker reports the wrapper, not the shell. That
		// reads as busy, which is the safe direction to be wrong in.
		{"wrapper reads as busy", "ssh\n", false},
		{"no panes at all", "", false},
		{"whitespace only", "  \n\n", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := panesQuiet(c.out); got != c.want {
				t.Errorf("panesQuiet(%q) = %v, want %v", c.out, got, c.want)
			}
		})
	}
}
