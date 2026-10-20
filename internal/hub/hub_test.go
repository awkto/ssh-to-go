package hub

import (
	"errors"
	"strings"
	"testing"

	"github.com/awkto/ssh-to-go/internal/config"
	"github.com/awkto/ssh-to-go/internal/tmux"
)

func TestDropSession(t *testing.T) {
	h := New([]config.Host{{Name: "pro"}})
	h.Update(tmux.PollResult{HostName: "pro", TmuxDetected: true, Sessions: []tmux.Session{{Name: "a"}, {Name: "b"}, {Name: "c"}}})

	h.DropSession("pro", "b")
	h.DropSession("pro", "nope")
	h.DropSession("ghost", "a")

	state, _ := h.GetHost("pro")
	var names []string
	for _, s := range state.Sessions {
		names = append(names, s.Name)
	}
	if got := strings.Join(names, ","); got != "a,c" {
		t.Fatalf("sessions after drop = %q, want a,c", got)
	}
}

func TestHostIsPsmuxSurvivesFailedPolls(t *testing.T) {
	h := New([]config.Host{{Name: "win"}})
	if h.HostIsPsmux("win") {
		t.Fatal("before any poll: want tmux")
	}
	h.Update(tmux.PollResult{HostName: "win", TmuxDetected: true, TmuxVersion: "tmux 3.4.0\npsmux 3.4.0 (20aa1d7 2026-10-13)"})
	if !h.HostIsPsmux("win") {
		t.Fatal("after detecting psmux: want psmux")
	}
	// An SSH drop fails the poll before tmux -V, so the result has no version.
	h.Update(tmux.PollResult{HostName: "win", Error: errors.New("ssh: connect: connection refused")})
	if !h.HostIsPsmux("win") {
		t.Fatal("after a failed poll: the psmux flag must be kept")
	}
	h.Update(tmux.PollResult{HostName: "win", TmuxDetected: true, TmuxVersion: "tmux 3.4"})
	if h.HostIsPsmux("win") {
		t.Fatal("after detecting plain tmux: want tmux")
	}
	if h.HostIsPsmux("nope") {
		t.Fatal("unknown host: want tmux")
	}
}
