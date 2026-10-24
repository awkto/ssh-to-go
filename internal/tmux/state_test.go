package tmux

import (
	"testing"
	"time"
)

func TestApplyPanes(t *testing.T) {
	now := time.Unix(2_000_000_000, 0)
	sessions := []Session{
		{Name: "agent", outputAge: -1},
		{Name: "shell", outputAge: -1},
		{Name: "gone", outputAge: -1},
	}
	// Host clock is 100s ahead of ours; ages must come from the host clock.
	out := "1000000100\n" +
		"agent\t1000000090\tbash\n" +
		"agent\t1000000098\tclaude\n" +
		"shell\t1000000000\tzsh\n"
	applyPanes(sessions, out, now)

	if s := sessions[0]; s.outputAge != 2 || s.shellOnly {
		t.Errorf("agent: age %d shellOnly %v, want 2 false", s.outputAge, s.shellOnly)
	}
	if want := now.Add(-2 * time.Second); !sessions[0].LastOutput.Equal(want) {
		t.Errorf("agent LastOutput = %v, want %v", sessions[0].LastOutput, want)
	}
	if s := sessions[1]; s.outputAge != 100 || !s.shellOnly {
		t.Errorf("shell: age %d shellOnly %v, want 100 true", s.outputAge, s.shellOnly)
	}
	if s := sessions[2]; s.outputAge != -1 || !s.LastOutput.IsZero() {
		t.Errorf("gone: age %d LastOutput %v, want untouched", s.outputAge, s.LastOutput)
	}
}

func TestApplyPanesNoClock(t *testing.T) {
	sessions := []Session{{Name: "a", outputAge: -1}}
	applyPanes(sessions, "", time.Now())
	applyPanes(sessions, "not-a-number\na\t1\tbash\n", time.Now())
	if sessions[0].outputAge != -1 {
		t.Errorf("outputAge = %d, want -1 without a host clock", sessions[0].outputAge)
	}
}

func TestClassify(t *testing.T) {
	// ages is the output age seen on each successive poll.
	cases := []struct {
		name      string
		ages      []int
		shellOnly bool
		want      string
	}{
		{"steady output", []int{0, 1, 0}, false, StateWorking},
		{"two polls in a row is enough", []int{0, 0}, false, StateWorking},
		{"one repaint is not working", []int{30, 2, 7}, false, StateWaiting},
		{"periodic repaint every 4th poll", []int{1, 6, 11, 16, 1, 6}, false, StateWaiting},
		{"stops working after two quiet polls", []int{0, 0, 0, 5, 10}, false, StateWaiting},
		{"still working after one quiet poll", []int{0, 0, 0, 5}, false, StateWorking},
		{"quiet shell is idle", []int{300, 305}, true, StateIdle},
		{"busy shell is working", []int{0, 0}, true, StateWorking},
		{"new session with one burst", []int{0}, false, StateWaiting},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var prev *Session
			for _, age := range c.ages {
				s := Session{outputAge: age, shellOnly: c.shellOnly}
				s.Classify(prev)
				prev = &s
			}
			if prev.State != c.want {
				t.Errorf("State = %q, want %q", prev.State, c.want)
			}
		})
	}
}

func TestClassifyUnknown(t *testing.T) {
	s := Session{outputAge: -1}
	s.Classify(nil)
	if s.State != "" {
		t.Errorf("State = %q, want empty when the host gave no pane data", s.State)
	}
}

func TestAtPrompt(t *testing.T) {
	// ps: 100 is an interactive shell at its prompt with a background
	// helper (own group); 200 is a shell running a script (pid 210, with
	// its sleep 211); 300 is `bash -c` as the pane command, mid-sleep.
	tpgid, size := fgGroups(" 100 100 100\n 101 101 100\n" +
		" 200 200 210\n 210 210 210\n 211 210 210\n" +
		" 300 300 300\n 301 300 300\n")
	cases := []struct {
		name string
		cmd  string
		pid  int
		want bool
	}{
		{"shell at prompt", "zsh", 100, true},
		{"script running from a prompt", "bash", 200, false},
		{"bash -c loop", "bash", 300, false},
		{"not a shell", "claude", 100, false},
		{"no ps data trusts the name", "bash", 999, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := atPrompt(c.cmd, c.pid, tpgid, size); got != c.want {
				t.Errorf("atPrompt(%q, %d) = %v, want %v", c.cmd, c.pid, got, c.want)
			}
		})
	}
}
