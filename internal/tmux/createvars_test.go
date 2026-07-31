package tmux

import (
	"strings"
	"testing"
	"time"

	"github.com/awkto/ssh-to-go/internal/sessionvars"
)

// The handlers expand $name/$date before buildCreateCmd ever sees the
// strings, so what arrives here is ordinary text — but text that now comes
// from a session name rather than from something the user typed as a path.
// These guard the seam between the two.

// $name is the sanitized name, so it cannot contain a space — but the quoting
// must not be relying on that. A path with a space in it stays one argument.
func TestExpandedPathWithASpaceIsStillQuoted(t *testing.T) {
	cwd := sessionvars.Expand("~/sessions/$name", sessionvars.Vars{Name: "my session"})
	got := buildCreateCmd("work", CreateOptions{Cwd: cwd, CreateDir: true})
	if !strings.Contains(got, `mkdir -p "$HOME"/'sessions/my session' && tmux`) {
		t.Errorf("expanded path not quoted as one argument: %q", got)
	}
	if !strings.Contains(got, `-c "$HOME"/'sessions/my session'`) {
		t.Errorf("missing quoted -c working dir: %q", got)
	}
}

// The whole "leave the shell alone" rule, end to end: a command with no
// variables of ours must reach the pane byte-for-byte identically to how it
// did before expansion existed — including the $-forms the shell owns.
func TestExpansionLeavesOrdinaryCommandsByteIdentical(t *testing.T) {
	for _, cmd := range []string{
		"claude",
		"claude --resume",
		"cd $HOME && exec $SHELL -l",
		"echo $(whoami) `id`",
		"PATH=$PATH:/opt/bin claude",
	} {
		want := buildCreateCmd("work", CreateOptions{Cwd: "~/p", Command: cmd})
		expanded := sessionvars.Expand(cmd, sessionvars.Vars{Name: "work", Now: time.Now()})
		got := buildCreateCmd("work", CreateOptions{Cwd: "~/p", Command: expanded})
		if got != want {
			t.Errorf("expansion changed the create command for %q:\n got %q\nwant %q", cmd, got, want)
		}
	}
}

// And the case the feature is for: the expanded command is single-quoted into
// send-keys exactly like any other, so the pane types the real name.
func TestExpandedCommandReachesThePane(t *testing.T) {
	cmd := sessionvars.Expand("claude --name $name", sessionvars.Vars{Name: "api-refactor"})
	got := buildCreateCmd("api-refactor", CreateOptions{Command: cmd})
	if !strings.Contains(got, `send-keys -t "api-refactor" 'claude --name api-refactor' Enter`) {
		t.Errorf("expanded command not typed into the pane: %q", got)
	}
}
