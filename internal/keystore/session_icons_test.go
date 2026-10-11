package keystore

import (
	"reflect"
	"testing"
)

func TestNormalizeTags(t *testing.T) {
	got := NormalizeTags([]string{" Work ", "work", "", "  ", "Side Project", "personal"})
	want := []string{"personal", "side-project", "work"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("NormalizeTags = %v, want %v", got, want)
	}
	if NormalizeTags(nil) != nil || NormalizeTags([]string{" "}) != nil {
		t.Fatal("no tags should normalise to nil so the field is omitted")
	}
	long := NormalizeTags([]string{"abcdefghijklmnopqrstuvwxyz0123456789"})
	if len(long) != 1 || len(long[0]) != maxTagLen {
		t.Fatalf("long tag not cut to %d: %v", maxTagLen, long)
	}
}

func TestSessionTagsPersistAndRename(t *testing.T) {
	dir := t.TempDir()
	s, err := NewSessionIconStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Set("h", "a", SessionIcon{Tags: []string{"Work"}}); err != nil {
		t.Fatal(err)
	}
	// Tags alone keep the entry alive.
	if got := s.Get("h", "a").Tags; !reflect.DeepEqual(got, []string{"work"}) {
		t.Fatalf("tags = %v", got)
	}
	if err := s.Rename("h", "a", "b"); err != nil {
		t.Fatal(err)
	}
	reloaded, err := NewSessionIconStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	if got := reloaded.Get("h", "b").Tags; !reflect.DeepEqual(got, []string{"work"}) {
		t.Fatalf("tags after rename+reload = %v", got)
	}
	// Clearing the last tag drops the entry entirely.
	if err := reloaded.Set("h", "b", SessionIcon{}); err != nil {
		t.Fatal(err)
	}
	if _, ok := reloaded.GetAll()["h:b"]; ok {
		t.Fatal("empty entry should be deleted")
	}
}
