package hub

import (
	"testing"

	"github.com/awkto/ssh-to-go/internal/config"
	"github.com/awkto/ssh-to-go/internal/tmux"
)

func hubWithSessions(names ...string) *Hub {
	h := New([]config.Host{{Name: "h1"}})
	sessions := make([]tmux.Session, 0, len(names))
	for _, n := range names {
		sessions = append(sessions, tmux.Session{Name: n})
	}
	h.hosts["h1"].Sessions = sessions
	return h
}

func sessionNames(h *Hub) []string {
	var out []string
	for _, s := range h.AllSessions() {
		out = append(out, s.Session.Name)
	}
	return out
}

func TestAllSessionsHidesIncognito(t *testing.T) {
	h := hubWithSessions("work", "secret", "other")
	h.SetHidden("h1", map[string]bool{"secret": true})

	got := sessionNames(h)
	for _, n := range got {
		if n == "secret" {
			t.Fatalf("incognito session leaked into AllSessions: %v", got)
		}
	}
	if len(got) != 2 {
		t.Errorf("AllSessions = %v, want the 2 visible ones", got)
	}
}

func TestAllHostsHidesIncognito(t *testing.T) {
	h := hubWithSessions("work", "secret")
	h.SetHidden("h1", map[string]bool{"secret": true})

	hosts := h.AllHosts()
	if len(hosts) != 1 {
		t.Fatalf("want 1 host, got %d", len(hosts))
	}
	if len(hosts[0].Sessions) != 1 || hosts[0].Sessions[0].Name != "work" {
		t.Errorf("AllHosts sessions = %+v, want only \"work\"", hosts[0].Sessions)
	}
}

// Filtering must not mutate the hub's own slice — AllHosts returns HostState
// by value, but Sessions is a slice header over shared backing memory.
func TestFilteringDoesNotMutateHubState(t *testing.T) {
	h := hubWithSessions("work", "secret")
	h.SetHidden("h1", map[string]bool{"secret": true})

	_ = h.AllHosts()
	_ = h.AllSessions()

	h.SetHidden("h1", nil) // un-hide
	got := sessionNames(h)
	if len(got) != 2 {
		t.Errorf("after un-hiding, AllSessions = %v; the hidden session was destroyed, not filtered", got)
	}
}

func TestGetHostStillSeesHiddenSessions(t *testing.T) {
	// Name-collision checks and kill/rename paths go through GetHost. If it
	// filtered too, you could create a second session with a hidden one's
	// name and tmux would reject it with a confusing error.
	h := hubWithSessions("work", "secret")
	h.SetHidden("h1", map[string]bool{"secret": true})

	state, ok := h.GetHost("h1")
	if !ok {
		t.Fatal("host missing")
	}
	found := false
	for _, s := range state.Sessions {
		if s.Name == "secret" {
			found = true
		}
	}
	if !found {
		t.Error("GetHost must still see hidden sessions for collision checks")
	}
}

func TestSetHiddenNilClears(t *testing.T) {
	h := hubWithSessions("work", "secret")
	h.SetHidden("h1", map[string]bool{"secret": true})
	if !h.IsHidden("h1", "secret") {
		t.Fatal("expected hidden")
	}
	h.SetHidden("h1", nil)
	if h.IsHidden("h1", "secret") {
		t.Error("SetHidden(nil) should clear the host's hide list")
	}
}

func TestNoHiddenSetIsPassThrough(t *testing.T) {
	h := hubWithSessions("a", "b")
	if got := sessionNames(h); len(got) != 2 {
		t.Errorf("unfiltered hub changed the listing: %v", got)
	}
}
