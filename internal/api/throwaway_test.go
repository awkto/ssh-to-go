package api

import (
	"testing"
	"time"

	"github.com/awkto/ssh-to-go/internal/config"
	"github.com/awkto/ssh-to-go/internal/hub"
	"github.com/awkto/ssh-to-go/internal/sessionreg"
	"github.com/awkto/ssh-to-go/internal/tmux"
)

func TestShouldCollect(t *testing.T) {
	now := time.Now().UTC()
	cases := []struct {
		name     string
		entry    sessionreg.Entry
		attached bool
		want     bool
		why      string
	}{
		{
			name:  "idle past the timeout",
			entry: sessionreg.Entry{Throwaway: true, LastAttachedAt: now.Add(-IdleTimeout - time.Minute)},
			want:  true,
			why:   "nothing attached for longer than the timeout",
		},
		{
			name:  "idle but not long enough",
			entry: sessionreg.Entry{Throwaway: true, LastAttachedAt: now.Add(-IdleTimeout + time.Minute)},
			want:  false,
			why:   "still inside the grace period",
		},
		{
			name:     "attached, however old",
			entry:    sessionreg.Entry{Throwaway: true, LastAttachedAt: now.Add(-24 * time.Hour)},
			attached: true,
			want:     false,
			why:      "someone is using it; age is irrelevant",
		},
		{
			name:  "not a throwaway",
			entry: sessionreg.Entry{LastAttachedAt: now.Add(-30 * 24 * time.Hour)},
			want:  false,
			why:   "ordinary sessions are never collected, no matter how idle",
		},
		{
			name:  "exactly at the timeout",
			entry: sessionreg.Entry{Throwaway: true, LastAttachedAt: now.Add(-IdleTimeout)},
			want:  true,
			why:   "boundary counts as expired",
		},
	}
	for _, c := range cases {
		if got := shouldCollect(c.entry, c.attached, now); got != c.want {
			t.Errorf("%s: shouldCollect = %v, want %v (%s)", c.name, got, c.want, c.why)
		}
	}
}

// A throwaway created and never opened must still age out — if the clock were
// left at the zero time this would read as "idle since year 1" and collect
// instantly, and if it were never set it would never collect at all.
func TestFreshThrowawayIsNotCollectedImmediately(t *testing.T) {
	dir := t.TempDir()
	reg, err := sessionreg.NewStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := reg.AddWithFlags("h", "burner", "", sessionreg.Flags{Throwaway: true}); err != nil {
		t.Fatal(err)
	}
	e, _ := reg.Get("h", "burner")
	if shouldCollect(e, false, time.Now().UTC()) {
		t.Error("a just-created throwaway was immediately eligible for collection")
	}
	if !shouldCollect(e, false, time.Now().UTC().Add(IdleTimeout+time.Second)) {
		t.Error("a never-attached throwaway never becomes eligible")
	}
}

func TestSessionHasClientsReadsLastPoll(t *testing.T) {
	h := hub.New([]config.Host{{Name: "h1"}})
	h.Update(tmux.PollResult{
		HostName: "h1",
		Sessions: []tmux.Session{
			{Name: "busy", AttachedClients: 2},
			{Name: "lonely", AttachedClients: 0},
		},
	})
	handlers := &Handlers{Hub: h}

	if !handlers.sessionHasClients(sessionreg.Entry{Host: "h1", Name: "busy"}) {
		t.Error("busy session reported as having no clients")
	}
	if handlers.sessionHasClients(sessionreg.Entry{Host: "h1", Name: "lonely"}) {
		t.Error("session with 0 attached clients reported as busy")
	}
	// A session that isn't in the last poll is gone, not busy — reporting it
	// busy would make it immortal.
	if handlers.sessionHasClients(sessionreg.Entry{Host: "h1", Name: "vanished"}) {
		t.Error("unknown session must not be treated as attached")
	}
	if handlers.sessionHasClients(sessionreg.Entry{Host: "nohost", Name: "x"}) {
		t.Error("unknown host must not be treated as attached")
	}
}

// The disconnect path must ignore ordinary sessions entirely: a bug here
// would mean normal sessions get killed when you close the tab.
func TestOnClientDetachedIgnoresOrdinarySessions(t *testing.T) {
	reg, err := sessionreg.NewStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	_ = reg.Add("h", "keeper", "")
	handlers := &Handlers{Hub: hub.New([]config.Host{{Name: "h"}}), Registry: reg}

	handlers.onClientDetached("h", "keeper")
	handlers.onClientDetached("h", "never-seen")

	// Nothing scheduled means the entry is untouched and still present.
	if _, ok := reg.Get("h", "keeper"); !ok {
		t.Error("ordinary session entry was removed by a detach")
	}
}
