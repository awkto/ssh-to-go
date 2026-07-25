package sessionreg

import "testing"

// The launch command has to survive the round trip to disk, or Recreate
// brings a session back as a bare shell — the bug this field exists to fix.
func TestCommandPersists(t *testing.T) {
	dir := t.TempDir()
	s, err := NewStore(dir)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	if err := s.AddSession("h", "work", Attrs{WorkingDir: "/srv/app", Command: "claude"}); err != nil {
		t.Fatalf("AddSession: %v", err)
	}

	reopened, err := NewStore(dir)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	e, ok := reopened.Get("h", "work")
	if !ok {
		t.Fatal("entry missing after reopen")
	}
	if e.Command != "claude" || e.WorkingDir != "/srv/app" {
		t.Errorf("got command=%q dir=%q, want claude /srv/app", e.Command, e.WorkingDir)
	}
}

// Add() and the poller's cwd refresh don't know the command; re-adding an
// entry without one must not erase what was recorded.
func TestEmptyCommandDoesNotEraseRecordedOne(t *testing.T) {
	s, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	_ = s.AddSession("h", "work", Attrs{WorkingDir: "/a", Command: "claude"})
	_ = s.Add("h", "work", "/b") // e.g. a rename or a cwd refresh
	e, _ := s.Get("h", "work")
	if e.Command != "claude" {
		t.Errorf("command = %q, want it preserved as claude", e.Command)
	}
	if e.WorkingDir != "/b" {
		t.Errorf("working dir = %q, want the updated /b", e.WorkingDir)
	}
}

// A session that slept and came back is not still asleep.
func TestReAddClearsAutoOffloaded(t *testing.T) {
	s, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	_ = s.AddSession("h", "work", Attrs{WorkingDir: "/a"})
	s.MarkAutoOffloaded("h", "work")
	if e, _ := s.Get("h", "work"); !e.AutoOffloaded {
		t.Fatal("MarkAutoOffloaded did not stick")
	}
	_ = s.AddSession("h", "work", Attrs{WorkingDir: "/a"}) // recreate
	if e, _ := s.Get("h", "work"); e.AutoOffloaded {
		t.Error("recreated session is still flagged as auto-offloaded")
	}
}

// Flavour flags must not be re-applied (or dropped) by a later re-add.
func TestFlagsSurviveReAdd(t *testing.T) {
	s, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	_ = s.AddSession("h", "tmp", Attrs{Flags: Flags{Throwaway: true, Incognito: true}})
	_ = s.AddSession("h", "tmp", Attrs{WorkingDir: "/x"})
	e, _ := s.Get("h", "tmp")
	if !e.Throwaway || !e.Incognito {
		t.Errorf("flags lost on re-add: throwaway=%v incognito=%v", e.Throwaway, e.Incognito)
	}
}
