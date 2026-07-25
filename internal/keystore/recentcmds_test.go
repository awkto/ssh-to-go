package keystore

import (
	"os"
	"path/filepath"
	"strconv"
	"testing"
)

func newRecentStore(t *testing.T) (*RecentCommandStore, string) {
	t.Helper()
	dir := t.TempDir()
	s, err := NewRecentCommandStore(dir)
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	return s, dir
}

func commandsOf(s *RecentCommandStore) []string {
	list := s.List()
	out := make([]string, len(list))
	for i, c := range list {
		out[i] = c.Command
	}
	return out
}

func TestRecordPutsNewestFirst(t *testing.T) {
	s, _ := newRecentStore(t)
	for _, c := range []string{"claude", "htop", "lazygit"} {
		if err := s.Record(c); err != nil {
			t.Fatalf("record %q: %v", c, err)
		}
	}
	got := commandsOf(s)
	want := []string{"lazygit", "htop", "claude"}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("order = %v, want %v", got, want)
		}
	}
}

func TestRecordDedupesAndCounts(t *testing.T) {
	s, _ := newRecentStore(t)
	s.Record("claude")
	s.Record("htop")
	s.Record("claude")

	list := s.List()
	if len(list) != 2 {
		t.Fatalf("len = %d, want 2 (claude must not be duplicated): %v", len(list), commandsOf(s))
	}
	if list[0].Command != "claude" {
		t.Fatalf("re-running a command must move it to the front, got %v", commandsOf(s))
	}
	if list[0].Count != 2 {
		t.Fatalf("count = %d, want 2", list[0].Count)
	}
}

// Whitespace-only differences are the same command as far as a user is
// concerned, and a blank command (the "start in shell" path) is not a command
// at all — neither should reach the list.
func TestRecordTrimsAndIgnoresBlank(t *testing.T) {
	s, _ := newRecentStore(t)
	s.Record("  claude  ")
	s.Record("")
	s.Record("   ")

	got := commandsOf(s)
	if len(got) != 1 || got[0] != "claude" {
		t.Fatalf("got %v, want [claude]", got)
	}
}

func TestRecordEvictsOldestPastLimit(t *testing.T) {
	s, _ := newRecentStore(t)
	for i := 0; i < recentCommandLimit+5; i++ {
		s.Record("cmd-" + strconv.Itoa(i))
	}
	got := commandsOf(s)
	if len(got) != recentCommandLimit {
		t.Fatalf("len = %d, want %d", len(got), recentCommandLimit)
	}
	if got[0] != "cmd-"+strconv.Itoa(recentCommandLimit+4) {
		t.Fatalf("newest command missing, front is %q", got[0])
	}
	for _, c := range got {
		if c == "cmd-0" {
			t.Fatal("cmd-0 should have been evicted")
		}
	}
}

func TestDeleteReportsWhetherItExisted(t *testing.T) {
	s, _ := newRecentStore(t)
	s.Record("claude")

	ok, err := s.Delete("claude")
	if err != nil || !ok {
		t.Fatalf("delete existing = %v, %v; want true, nil", ok, err)
	}
	if got := commandsOf(s); len(got) != 0 {
		t.Fatalf("still present: %v", got)
	}
	ok, err = s.Delete("claude")
	if err != nil || ok {
		t.Fatalf("delete missing = %v, %v; want false, nil", ok, err)
	}
}

func TestClearEmptiesTheList(t *testing.T) {
	s, _ := newRecentStore(t)
	s.Record("claude")
	s.Record("htop")
	if err := s.Clear(); err != nil {
		t.Fatalf("clear: %v", err)
	}
	if got := commandsOf(s); len(got) != 0 {
		t.Fatalf("got %v, want empty", got)
	}
}

func TestSurvivesReload(t *testing.T) {
	s, dir := newRecentStore(t)
	s.Record("claude")
	s.Record("npm run dev")

	reopened, err := NewRecentCommandStore(dir)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	got := commandsOf(reopened)
	if len(got) != 2 || got[0] != "npm run dev" {
		t.Fatalf("after reload got %v, want MRU order preserved", got)
	}
}

// A data dir with no file yet is the first-run case, not an error.
func TestMissingFileIsEmptyNotAnError(t *testing.T) {
	s, dir := newRecentStore(t)
	if got := commandsOf(s); len(got) != 0 {
		t.Fatalf("got %v, want empty", got)
	}
	if _, err := os.Stat(filepath.Join(dir, "recent-commands.json")); !os.IsNotExist(err) {
		t.Fatal("store should not write a file until something is recorded")
	}
}
