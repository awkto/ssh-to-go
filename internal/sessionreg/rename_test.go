package sessionreg

import (
	"errors"
	"testing"
	"time"
)

func newTestStore(t *testing.T) *Store {
	t.Helper()
	s, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	return s
}

// A rename is not a re-creation: everything recorded about the session has to
// survive it, or a renamed session recreates as a bare shell in the wrong
// directory and a renamed throwaway comes back looking permanent.
func TestRenamePreservesTheEntry(t *testing.T) {
	s := newTestStore(t)
	if err := s.AddSession("h1", "old", Attrs{
		WorkingDir: "/srv/app",
		Command:    "claude",
		Flags:      Flags{Throwaway: true, Incognito: true},
	}); err != nil {
		t.Fatalf("add: %v", err)
	}
	before, _ := s.Get("h1", "old")

	moved, err := s.Rename("h1", "old", "new")
	if err != nil {
		t.Fatalf("rename: %v", err)
	}
	if !moved {
		t.Fatal("rename reported no move")
	}
	if _, ok := s.Get("h1", "old"); ok {
		t.Error("old name is still tracked")
	}
	after, ok := s.Get("h1", "new")
	if !ok {
		t.Fatal("new name is not tracked")
	}
	if after.Name != "new" {
		t.Errorf("Name = %q, want %q", after.Name, "new")
	}
	if after.WorkingDir != "/srv/app" || after.Command != "claude" {
		t.Errorf("working dir/command lost: %+v", after)
	}
	if !after.Throwaway || !after.Incognito {
		t.Errorf("flavours lost: throwaway=%v incognito=%v", after.Throwaway, after.Incognito)
	}
	if !after.CreatedAt.Equal(before.CreatedAt) {
		t.Errorf("CreatedAt changed: %v -> %v", before.CreatedAt, after.CreatedAt)
	}
	// Renaming is not attaching — resetting this clock would give a
	// long-abandoned throwaway a fresh idle window every time it's renamed.
	if !after.LastAttachedAt.Equal(before.LastAttachedAt) {
		t.Errorf("LastAttachedAt changed: %v -> %v", before.LastAttachedAt, after.LastAttachedAt)
	}
}

// Renaming onto an offloaded entry's name would silently destroy that entry's
// working directory and launch command. tmux only guards the live case.
func TestRenameRefusesATakenName(t *testing.T) {
	s := newTestStore(t)
	_ = s.AddSession("h1", "old", Attrs{WorkingDir: "/a"})
	_ = s.AddSession("h1", "taken", Attrs{WorkingDir: "/b", Command: "vim"})

	if _, err := s.Rename("h1", "old", "taken"); !errors.Is(err, ErrNameTaken) {
		t.Fatalf("err = %v, want ErrNameTaken", err)
	}
	if e, ok := s.Get("h1", "taken"); !ok || e.WorkingDir != "/b" || e.Command != "vim" {
		t.Errorf("the existing entry was damaged: %+v", e)
	}
	if _, ok := s.Get("h1", "old"); !ok {
		t.Error("the source entry was dropped despite the failed rename")
	}
}

// Sessions made by hand in tmux aren't tracked. Renaming one is normal, not an
// error — the caller shouldn't have to look first.
func TestRenameUntrackedIsANoOp(t *testing.T) {
	s := newTestStore(t)
	moved, err := s.Rename("h1", "never-seen", "new")
	if err != nil {
		t.Fatalf("rename: %v", err)
	}
	if moved {
		t.Error("reported a move for an untracked session")
	}
	if _, ok := s.Get("h1", "new"); ok {
		t.Error("rename invented an entry for an untracked session")
	}
}

func TestRenameToSameNameIsANoOp(t *testing.T) {
	s := newTestStore(t)
	_ = s.AddSession("h1", "same", Attrs{WorkingDir: "/a"})
	moved, err := s.Rename("h1", "same", "same")
	if err != nil {
		t.Fatalf("rename: %v", err)
	}
	if moved {
		t.Error("reported a move for a no-op rename")
	}
	if e, ok := s.Get("h1", "same"); !ok || e.WorkingDir != "/a" {
		t.Errorf("entry disturbed: %+v, ok=%v", e, ok)
	}
}

// Hosts are separate namespaces: the same session name on another host must
// not be touched, and must not block the rename.
func TestRenameIsScopedToTheHost(t *testing.T) {
	s := newTestStore(t)
	_ = s.AddSession("h1", "old", Attrs{WorkingDir: "/one"})
	_ = s.AddSession("h2", "new", Attrs{WorkingDir: "/two"})

	if _, err := s.Rename("h1", "old", "new"); err != nil {
		t.Fatalf("rename: %v", err)
	}
	if e, ok := s.Get("h2", "new"); !ok || e.WorkingDir != "/two" {
		t.Errorf("the other host's entry was touched: %+v", e)
	}
	if e, ok := s.Get("h1", "new"); !ok || e.WorkingDir != "/one" {
		t.Errorf("rename landed wrong: %+v", e)
	}
}

// The move must reach disk — a rename lost on restart resurrects the old name
// as a phantom resumable entry.
func TestRenamePersists(t *testing.T) {
	dir := t.TempDir()
	s, err := NewStore(dir)
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	_ = s.AddSession("h1", "old", Attrs{WorkingDir: "/srv", Command: "claude", Flags: Flags{Incognito: true}})
	if _, err := s.Rename("h1", "old", "new"); err != nil {
		t.Fatalf("rename: %v", err)
	}

	reopened, err := NewStore(dir)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	if _, ok := reopened.Get("h1", "old"); ok {
		t.Error("old name came back after reload")
	}
	e, ok := reopened.Get("h1", "new")
	if !ok {
		t.Fatal("new name missing after reload")
	}
	if !e.Incognito || e.Command != "claude" {
		t.Errorf("entry did not round-trip: %+v", e)
	}
	if hidden := reopened.HiddenNames("h1"); !hidden["new"] || hidden["old"] {
		t.Errorf("HiddenNames after reload = %v, want only the new name", hidden)
	}
}

func TestRenameUpdatesLastSeen(t *testing.T) {
	s := newTestStore(t)
	_ = s.AddSession("h1", "old", Attrs{})
	before, _ := s.Get("h1", "old")
	time.Sleep(2 * time.Millisecond)

	if _, err := s.Rename("h1", "old", "new"); err != nil {
		t.Fatalf("rename: %v", err)
	}
	after, _ := s.Get("h1", "new")
	if !after.LastSeenAt.After(before.LastSeenAt) {
		t.Errorf("LastSeenAt not refreshed: %v -> %v", before.LastSeenAt, after.LastSeenAt)
	}
}
