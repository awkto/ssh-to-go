package sessionreg

import (
	"testing"
	"time"
)

func newStore(t *testing.T) *Store {
	t.Helper()
	s, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	return s
}

func TestFlagsPersistAndReadBack(t *testing.T) {
	s := newStore(t)
	if err := s.AddWithFlags("h", "burner", "~/tmp", Flags{Throwaway: true, Incognito: true}); err != nil {
		t.Fatal(err)
	}
	got := s.Flavours("h", "burner")
	if !got.Throwaway || !got.Incognito {
		t.Errorf("Flavours = %+v, want both set", got)
	}
	if f := s.Flavours("h", "missing"); f.Throwaway || f.Incognito {
		t.Errorf("unknown session should report no flags, got %+v", f)
	}
}

func TestFlagsSurviveReload(t *testing.T) {
	dir := t.TempDir()
	s, err := NewStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.AddWithFlags("h", "ghost", "", Flags{Incognito: true}); err != nil {
		t.Fatal(err)
	}
	// A hidden session outlives the process, so the flag must come back
	// from disk — otherwise it reappears in the UI after a restart.
	reloaded, err := NewStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	if !reloaded.Flavours("h", "ghost").Incognito {
		t.Error("incognito flag lost across reload")
	}
	if !reloaded.HiddenNames("h")["ghost"] {
		t.Error("HiddenNames lost the session across reload")
	}
}

func TestPlainAddSetsNoFlags(t *testing.T) {
	s := newStore(t)
	if err := s.Add("h", "normal", ""); err != nil {
		t.Fatal(err)
	}
	if f := s.Flavours("h", "normal"); f.Throwaway || f.Incognito {
		t.Errorf("Add must create an ordinary session, got %+v", f)
	}
}

func TestReAddDoesNotChangeFlags(t *testing.T) {
	s := newStore(t)
	_ = s.AddWithFlags("h", "burner", "", Flags{Throwaway: true})
	// A recreate/offload path re-adds without flags; that must not promote
	// a throwaway into a permanent session.
	_ = s.Add("h", "burner", "/new/dir")
	if !s.Flavours("h", "burner").Throwaway {
		t.Error("re-adding cleared the throwaway flag")
	}
	if e, _ := s.Get("h", "burner"); e.WorkingDir != "/new/dir" {
		t.Errorf("working dir not updated: %q", e.WorkingDir)
	}
}

func TestHiddenNamesOnlyIncognitoAndOnlyThatHost(t *testing.T) {
	s := newStore(t)
	_ = s.AddWithFlags("h1", "hidden", "", Flags{Incognito: true})
	_ = s.AddWithFlags("h1", "shown", "", Flags{})
	_ = s.AddWithFlags("h2", "otherhost", "", Flags{Incognito: true})

	h1 := s.HiddenNames("h1")
	if !h1["hidden"] || h1["shown"] || h1["otherhost"] {
		t.Errorf("HiddenNames(h1) = %v", h1)
	}
	if len(s.HiddenNames("h3")) != 0 {
		t.Error("host with no entries should report nothing hidden")
	}
}

func TestThrowawaysListsOnlyThrowaways(t *testing.T) {
	s := newStore(t)
	_ = s.AddWithFlags("h", "burner", "", Flags{Throwaway: true})
	_ = s.AddWithFlags("h", "keeper", "", Flags{})
	got := s.Throwaways()
	if len(got) != 1 || got[0].Name != "burner" {
		t.Fatalf("Throwaways = %+v", got)
	}
}

func TestNewThrowawayStartsIdleClockAtCreation(t *testing.T) {
	s := newStore(t)
	before := time.Now().UTC()
	_ = s.AddWithFlags("h", "burner", "", Flags{Throwaway: true})
	e, ok := s.Get("h", "burner")
	if !ok {
		t.Fatal("entry missing")
	}
	// A session created and never opened must still age out, so the clock
	// starts now rather than sitting at the zero time (which would read as
	// idle forever) or being left unset.
	if e.LastAttachedAt.Before(before.Add(-time.Second)) || e.LastAttachedAt.IsZero() {
		t.Errorf("LastAttachedAt = %v, want ~now", e.LastAttachedAt)
	}
}

func TestMarkAttachedResetsIdleClock(t *testing.T) {
	s := newStore(t)
	_ = s.AddWithFlags("h", "burner", "", Flags{Throwaway: true})
	// Backdate so MarkAttached has something to move.
	s.mu.Lock()
	e := s.entries[key("h", "burner")]
	e.LastAttachedAt = time.Now().UTC().Add(-30 * time.Minute)
	s.entries[key("h", "burner")] = e
	s.mu.Unlock()

	s.MarkAttached("h", "burner")
	got, _ := s.Get("h", "burner")
	if time.Since(got.LastAttachedAt) > time.Minute {
		t.Errorf("LastAttachedAt not reset: %v", got.LastAttachedAt)
	}
	// Unknown sessions must not panic or create entries.
	s.MarkAttached("h", "nope")
	if _, ok := s.Get("h", "nope"); ok {
		t.Error("MarkAttached created an entry for an unknown session")
	}
}
