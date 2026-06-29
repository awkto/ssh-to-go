package api

import (
	"testing"

	"github.com/awkto/ssh-to-go/internal/config"
	"github.com/awkto/ssh-to-go/internal/hub"
	"github.com/awkto/ssh-to-go/internal/sessionreg"
	"github.com/awkto/ssh-to-go/internal/tmux"
)

func TestSanitizeSessionName(t *testing.T) {
	cases := map[string]string{
		"SysAdmin GSTN":  "SysAdmin-GSTN",
		"  trim me  ":    "trim-me",
		"a  b   c":       "a-b-c", // runs of whitespace collapse to one hyphen
		"already-fine":   "already-fine",
		"tab\tseparated": "tab-separated",
	}
	for in, want := range cases {
		if got := sanitizeSessionName(in); got != want {
			t.Errorf("sanitizeSessionName(%q) = %q, want %q", in, got, want)
		}
	}
}

// missingFor must treat a tracked spaced name and a live hyphenated name as
// the same session, so a recreated/offloaded session never shows up twice.
func TestMissingForDedupesBySanitizedName(t *testing.T) {
	store, err := sessionreg.NewStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	const host = "h1"
	// A legacy/offloaded entry with a space, plus its sanitized recreation,
	// plus a genuinely-gone session that has no live counterpart.
	_ = store.Add(host, "SysAdmin GSTN", "")
	_ = store.Add(host, "SysAdmin-GSTN", "")
	_ = store.Add(host, "Gone-Session", "")

	h := &Handlers{Registry: store}
	state := hub.HostState{
		Config:   config.Host{Name: host},
		Online:   true,
		Sessions: []tmux.Session{{Name: "SysAdmin-GSTN"}},
	}

	missing := h.missingFor(state)

	// The live "SysAdmin-GSTN" covers both tracked SysAdmin entries; only the
	// genuinely-gone session should remain, exactly once.
	if len(missing) != 1 {
		t.Fatalf("missing = %d entries (%v), want 1 (Gone-Session only)", len(missing), names(missing))
	}
	if missing[0].Name != "Gone-Session" {
		t.Errorf("missing[0] = %q, want %q", missing[0].Name, "Gone-Session")
	}
}

// Offline hosts return no missing sessions (we can't distinguish gone from
// unreachable).
func TestMissingForOfflineHost(t *testing.T) {
	store, err := sessionreg.NewStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	_ = store.Add("h1", "whatever", "")
	h := &Handlers{Registry: store}
	if got := h.missingFor(hub.HostState{Config: config.Host{Name: "h1"}, Online: false}); got != nil {
		t.Errorf("offline host missing = %v, want nil", got)
	}
}

func names(es []sessionreg.Entry) []string {
	out := make([]string, len(es))
	for i, e := range es {
		out[i] = e.Name
	}
	return out
}
