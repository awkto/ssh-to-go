package api

import (
	"testing"

	"github.com/awkto/ssh-to-go/internal/config"
	"github.com/awkto/ssh-to-go/internal/hub"
	"github.com/awkto/ssh-to-go/internal/sessionreg"
	"github.com/awkto/ssh-to-go/internal/tmux"
)

// hiddenFixture builds a Handlers over a real registry and a hub holding one
// live session, which is the state the leak needs: the hub filters listings by
// name, the registry owns the flag.
func hiddenFixture(t *testing.T, sessionName string, flags sessionreg.Flags) (*Handlers, *hub.Hub, *sessionreg.Store) {
	t.Helper()
	reg, err := sessionreg.NewStore(t.TempDir())
	if err != nil {
		t.Fatalf("registry: %v", err)
	}
	if err := reg.AddSession("h1", sessionName, sessionreg.Attrs{WorkingDir: "/srv", Flags: flags}); err != nil {
		t.Fatalf("add session: %v", err)
	}
	hb := hub.New([]config.Host{{Name: "h1"}})
	hb.Update(tmux.PollResult{HostName: "h1", Sessions: []tmux.Session{{Name: sessionName}}})
	// Seed the hide-list the way main.go does at startup, not via
	// refreshHidden — otherwise these tests would pass on a no-op fix.
	hb.SetHidden("h1", reg.HiddenNames("h1"))
	return &Handlers{Hub: hb, Registry: reg}, hb, reg
}

func visibleNames(hb *hub.Hub) []string {
	var out []string
	for _, s := range hb.AllSessions() {
		out = append(out, s.Session.Name)
	}
	return out
}

func contains(names []string, want string) bool {
	for _, n := range names {
		if n == want {
			return true
		}
	}
	return false
}

// The bug: the hub hides by name, so renaming an incognito session left it
// hiding a name that no longer existed while the live session — same session,
// new name — rendered in every listing until the process restarted.
func TestRenamedIncognitoSessionStaysHidden(t *testing.T) {
	h, hb, reg := hiddenFixture(t, "secret", sessionreg.Flags{Incognito: true})
	if contains(visibleNames(hb), "secret") {
		t.Fatal("incognito session was visible before the rename; fixture is wrong")
	}

	// What RenameSession does after tmux renames the live session.
	if _, err := reg.Rename("h1", "secret", "notes"); err != nil {
		t.Fatalf("registry rename: %v", err)
	}
	hb.Update(tmux.PollResult{HostName: "h1", Sessions: []tmux.Session{{Name: "notes"}}})
	h.refreshHidden("h1")

	if got := visibleNames(hb); contains(got, "notes") {
		t.Errorf("renamed incognito session leaked into the listing: %v", got)
	}
	if !hb.IsHidden("h1", "notes") {
		t.Error("hub should hide the new name")
	}
	if hb.IsHidden("h1", "secret") {
		t.Error("hub is still hiding the old name, which nothing answers to now")
	}
}

// Renaming an ordinary session must not start hiding it.
func TestRenamedOrdinarySessionStaysVisible(t *testing.T) {
	h, hb, reg := hiddenFixture(t, "work", sessionreg.Flags{})

	if _, err := reg.Rename("h1", "work", "work-2"); err != nil {
		t.Fatalf("registry rename: %v", err)
	}
	hb.Update(tmux.PollResult{HostName: "h1", Sessions: []tmux.Session{{Name: "work-2"}}})
	h.refreshHidden("h1")

	if got := visibleNames(hb); !contains(got, "work-2") {
		t.Errorf("renamed ordinary session vanished from the listing: %v", got)
	}
}

// Killing or forgetting an incognito session should drop its name from the
// hide-list; a stale name there would hide an unrelated session created under
// it later.
func TestForgettingIncognitoClearsTheHideList(t *testing.T) {
	h, hb, reg := hiddenFixture(t, "secret", sessionreg.Flags{Incognito: true})

	if err := reg.Remove("h1", "secret"); err != nil {
		t.Fatalf("registry remove: %v", err)
	}
	h.refreshHidden("h1")

	if hb.IsHidden("h1", "secret") {
		t.Error("hub still hides a name the registry no longer knows")
	}
}

// A nil registry is a valid configuration (see the nil guards throughout the
// handlers); refreshing must not panic on it.
func TestRefreshHiddenWithoutRegistry(t *testing.T) {
	h := &Handlers{Hub: hub.New([]config.Host{{Name: "h1"}})}
	h.refreshHidden("h1") // must not panic
}
