package mcp

import (
	"strings"
	"testing"
)

// The read_pane / wait_for_pane cursor is a content hash: identical content
// must yield an identical cursor (that equality drives the {changed:false}
// no-op path), and any change must move it.
func TestPaneCursorEquality(t *testing.T) {
	a := paneCursor("hello\nworld")
	b := paneCursor("hello\nworld")
	if a != b {
		t.Fatalf("same content produced different cursors: %q vs %q", a, b)
	}
	if !strings.HasPrefix(a, "sha256:") {
		t.Fatalf("cursor missing sha256: prefix: %q", a)
	}
	if c := paneCursor("hello\nworld!"); c == a {
		t.Fatalf("changed content produced the same cursor: %q", c)
	}
}

func TestLastLines(t *testing.T) {
	in := "l1\nl2\nl3\nl4\nl5"
	if got := lastLines(in, 0); got != in {
		t.Errorf("lastLines n=0 should keep all, got %q", got)
	}
	if got := lastLines(in, 2); got != "l4\nl5" {
		t.Errorf("lastLines n=2 = %q", got)
	}
	if got := lastLines(in, 99); got != in {
		t.Errorf("lastLines n>len should keep all, got %q", got)
	}
}

func TestCapPane(t *testing.T) {
	if s, tr := capPane("short"); tr || s != "short" {
		t.Errorf("small content should not truncate: %q %v", s, tr)
	}
	big := strings.Repeat("x", paneMaxBytes+100)
	s, tr := capPane(big)
	if !tr {
		t.Fatal("oversized content should truncate")
	}
	if len(s) != paneMaxBytes {
		t.Errorf("capped length = %d, want %d", len(s), paneMaxBytes)
	}
	// The tail (most recent bytes) is kept.
	if s != big[len(big)-paneMaxBytes:] {
		t.Error("capPane should keep the tail")
	}
}
