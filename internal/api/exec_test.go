package api

import (
	"testing"

	"github.com/awkto/ssh-to-go/internal/config"
	"github.com/awkto/ssh-to-go/internal/hub"
	"github.com/awkto/ssh-to-go/internal/keystore"
)

func newSettings(t *testing.T) *keystore.SettingsManager {
	t.Helper()
	sm, err := keystore.NewSettingsManager(t.TempDir())
	if err != nil {
		t.Fatalf("NewSettingsManager: %v", err)
	}
	return sm
}

func TestResolveExecHostExplicit(t *testing.T) {
	h := &Handlers{
		Hub:      hub.New([]config.Host{{Name: "a"}, {Name: "b"}}),
		Settings: newSettings(t),
	}
	cfg, errMsg := h.resolveExecHost("b")
	if errMsg != "" {
		t.Fatalf("unexpected error: %s", errMsg)
	}
	if cfg.Name != "b" {
		t.Fatalf("got host %q, want b", cfg.Name)
	}
}

func TestResolveExecHostSingleFallback(t *testing.T) {
	h := &Handlers{
		Hub:      hub.New([]config.Host{{Name: "only"}}),
		Settings: newSettings(t),
	}
	cfg, errMsg := h.resolveExecHost("")
	if errMsg != "" {
		t.Fatalf("unexpected error: %s", errMsg)
	}
	if cfg.Name != "only" {
		t.Fatalf("got host %q, want only", cfg.Name)
	}
}

func TestResolveExecHostDefaultSetting(t *testing.T) {
	sm := newSettings(t)
	if err := sm.Update(keystore.Settings{DefaultHost: "b"}, nil); err != nil {
		t.Fatalf("Update: %v", err)
	}
	h := &Handlers{
		Hub:      hub.New([]config.Host{{Name: "a"}, {Name: "b"}}),
		Settings: sm,
	}
	cfg, errMsg := h.resolveExecHost("")
	if errMsg != "" {
		t.Fatalf("unexpected error: %s", errMsg)
	}
	if cfg.Name != "b" {
		t.Fatalf("got host %q, want b (from default setting)", cfg.Name)
	}
}

func TestResolveExecHostAmbiguous(t *testing.T) {
	h := &Handlers{
		Hub:      hub.New([]config.Host{{Name: "a"}, {Name: "b"}}),
		Settings: newSettings(t),
	}
	if _, errMsg := h.resolveExecHost(""); errMsg == "" {
		t.Fatal("expected error for ambiguous host, got none")
	}
}

func TestResolveExecHostUnknown(t *testing.T) {
	h := &Handlers{
		Hub:      hub.New([]config.Host{{Name: "a"}}),
		Settings: newSettings(t),
	}
	if _, errMsg := h.resolveExecHost("nope"); errMsg == "" {
		t.Fatal("expected error for unknown host, got none")
	}
}
