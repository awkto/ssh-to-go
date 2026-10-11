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

func TestNewTagGetsPaletteColourThatSticks(t *testing.T) {
	dir := t.TempDir()
	s, err := NewSessionIconStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Set("h", "a", SessionIcon{Tags: []string{"work"}}); err != nil {
		t.Fatal(err)
	}
	hue, ok := s.TagColors()["work"]
	if !ok {
		t.Fatal("new tag got no colour")
	}
	inPalette := false
	for _, h := range TagHues {
		inPalette = inPalette || h == hue
	}
	if !inPalette {
		t.Fatalf("hue %d not from the palette", hue)
	}
	// Reusing the tag on another session keeps its colour.
	if err := s.Set("h", "b", SessionIcon{Tags: []string{"work"}}); err != nil {
		t.Fatal(err)
	}
	if got := s.TagColors()["work"]; got != hue {
		t.Fatalf("colour changed on reuse: %d -> %d", hue, got)
	}
	if err := s.SetTagColor("Work", -10); err != nil {
		t.Fatal(err)
	}
	reloaded, err := NewSessionIconStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	if got := reloaded.TagColors()["work"]; got != 350 {
		t.Fatalf("hue after set+reload = %d, want 350", got)
	}
}
