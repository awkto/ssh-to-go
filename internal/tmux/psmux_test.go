package tmux

import (
	"strings"
	"testing"
)

func TestIsPsmux(t *testing.T) {
	cases := map[string]bool{
		"tmux 3.4.0\npsmux 3.4.0 (20aa1d7 2026-10-13)": true,
		"psmux 3.4.0":   true,
		"tmux 3.4":      false,
		"tmux next-3.6": false,
		"":              false,
	}
	for version, want := range cases {
		if got := IsPsmux(version); got != want {
			t.Errorf("IsPsmux(%q) = %v, want %v", version, got, want)
		}
	}
}

// psmux runs one server per session: a global set-option ahead of new-session
// has no server to land on and fails the whole chain, so the limit goes on the
// session right after it is created.
func TestBuildCreateCmdPsmuxSetsHistoryLimitPerSession(t *testing.T) {
	got := buildCreateCmd("work", CreateOptions{HistoryLimit: 50000, Cwd: "~/p", CreateDir: true, Psmux: true})
	if strings.Contains(got, "-g history-limit") {
		t.Errorf("psmux must not get a global history-limit: %q", got)
	}
	want := `set-option -t "work" history-limit 50000`
	hist := strings.Index(got, want)
	newSess := strings.Index(got, "new-session")
	if hist < 0 || hist < newSess {
		t.Fatalf("psmux history-limit must be per-session and follow new-session: %q", got)
	}
}

func TestBuildCreateCmdPsmuxRestUnchanged(t *testing.T) {
	opts := CreateOptions{Cwd: "~/p", CreateDir: true, Command: "claude", Mouse: true}
	tmuxCmd := buildCreateCmd("work", opts)
	opts.Psmux = true
	if psmuxCmd := buildCreateCmd("work", opts); psmuxCmd != tmuxCmd {
		t.Errorf("without a history limit psmux and tmux should get the same command:\n tmux:  %q\n psmux: %q", tmuxCmd, psmuxCmd)
	}
}
