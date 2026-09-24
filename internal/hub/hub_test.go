package hub

import (
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
