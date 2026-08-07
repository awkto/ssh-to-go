package keystore

import (
	"slices"
	"testing"
)

func TestScrollbackLinesDefault(t *testing.T) {
	sm := &SettingsManager{} // zero value: ScrollbackLines unset
	if got := sm.ScrollbackLines(); got != DefaultScrollbackLines {
		t.Errorf("ScrollbackLines() unset = %d, want default %d", got, DefaultScrollbackLines)
	}
	sm.settings.ScrollbackLines = 12345
	if got := sm.ScrollbackLines(); got != 12345 {
		t.Errorf("ScrollbackLines() = %d, want 12345", got)
	}
}

func TestUpdateValidatesScrollback(t *testing.T) {
	dir := t.TempDir()
	sm, err := NewSettingsManager(dir)
	if err != nil {
		t.Fatalf("NewSettingsManager: %v", err)
	}
	ks := &Store{} // Update only consults ks when DefaultKeypair is set, which we don't.

	// Out of range is rejected and leaves the stored value untouched.
	for _, bad := range []int{99, 1_000_001} {
		if err := sm.Update(Settings{ScrollbackLines: bad}, ks); err == nil {
			t.Errorf("Update(ScrollbackLines=%d) = nil error, want validation error", bad)
		}
	}
	if got := sm.ScrollbackLines(); got != DefaultScrollbackLines {
		t.Errorf("after rejected updates ScrollbackLines() = %d, want still-default %d", got, DefaultScrollbackLines)
	}

	// In-range is accepted and persisted.
	if err := sm.Update(Settings{ScrollbackLines: 80000}, ks); err != nil {
		t.Fatalf("Update(ScrollbackLines=80000): %v", err)
	}
	if got := sm.ScrollbackLines(); got != 80000 {
		t.Errorf("ScrollbackLines() = %d, want 80000", got)
	}

	// 0 means "leave unchanged" (mirrors the other optional fields).
	if err := sm.Update(Settings{ScrollbackLines: 0}, ks); err != nil {
		t.Fatalf("Update(ScrollbackLines=0): %v", err)
	}
	if got := sm.ScrollbackLines(); got != 80000 {
		t.Errorf("after no-op update ScrollbackLines() = %d, want unchanged 80000", got)
	}
}

func TestNativeMouseModeDefaultsOn(t *testing.T) {
	sm := &SettingsManager{} // zero value: NativeMouseMode unset
	if !sm.NativeMouseMode() {
		t.Error("NativeMouseMode() unset = false, want default true")
	}
}

func TestUpdateNativeMouseMode(t *testing.T) {
	dir := t.TempDir()
	sm, err := NewSettingsManager(dir)
	if err != nil {
		t.Fatalf("NewSettingsManager: %v", err)
	}
	ks := &Store{}

	// Omitted (nil) leaves the stored value alone.
	if err := sm.Update(Settings{}, ks); err != nil {
		t.Fatalf("Update(empty): %v", err)
	}
	if !sm.NativeMouseMode() {
		t.Error("omitting native_mouse_mode must not change it")
	}

	// Explicit false turns it off — the whole reason the field is *bool.
	off := false
	if err := sm.Update(Settings{NativeMouseMode: &off}, ks); err != nil {
		t.Fatalf("Update(off): %v", err)
	}
	if sm.NativeMouseMode() {
		t.Error("NativeMouseMode() = true after setting false")
	}

	// And back on.
	on := true
	if err := sm.Update(Settings{NativeMouseMode: &on}, ks); err != nil {
		t.Fatalf("Update(on): %v", err)
	}
	if !sm.NativeMouseMode() {
		t.Error("NativeMouseMode() = false after setting true")
	}
}

func TestNewSessionIconDefaultsToRandom(t *testing.T) {
	sm := &SettingsManager{} // zero value: SessionIconMode unset

	// Unset mode → random palette entries. Assert both fields are always a
	// valid palette member (never empty) across many draws.
	for i := 0; i < 200; i++ {
		icon := sm.NewSessionIcon()
		if !slices.Contains(iconPalette, icon.Icon) {
			t.Fatalf("random icon %q not in palette", icon.Icon)
		}
		if !slices.Contains(colorPalette, icon.Color) {
			t.Fatalf("random color %q not in palette", icon.Color)
		}
	}
}

func TestNewSessionIconFixed(t *testing.T) {
	sm := &SettingsManager{}
	sm.settings.SessionIconMode = "fixed"
	sm.settings.SessionIconName = "server"
	sm.settings.SessionIconColor = "emerald"
	if got := sm.NewSessionIcon(); got.Icon != "server" || got.Color != "emerald" {
		t.Errorf("fixed NewSessionIcon() = %+v, want server/emerald", got)
	}

	// Fixed mode with unset icon/color falls back to terminal/default.
	sm.settings.SessionIconName = ""
	sm.settings.SessionIconColor = ""
	if got := sm.NewSessionIcon(); got.Icon != "terminal" || got.Color != "default" {
		t.Errorf("fixed NewSessionIcon() fallback = %+v, want terminal/default", got)
	}
}

func TestUpdateValidatesSessionIconMode(t *testing.T) {
	dir := t.TempDir()
	sm, err := NewSettingsManager(dir)
	if err != nil {
		t.Fatalf("NewSettingsManager: %v", err)
	}
	ks := &Store{}

	if err := sm.Update(Settings{SessionIconMode: "bogus"}, ks); err == nil {
		t.Error("Update(SessionIconMode=bogus) = nil error, want validation error")
	}
	if err := sm.Update(Settings{SessionIconMode: "fixed", SessionIconName: "cpu", SessionIconColor: "sky"}, ks); err != nil {
		t.Fatalf("Update(fixed): %v", err)
	}
	if got := sm.NewSessionIcon(); got.Icon != "cpu" || got.Color != "sky" {
		t.Errorf("after Update NewSessionIcon() = %+v, want cpu/sky", got)
	}
}

func TestUpdateValidatesFileBrowser(t *testing.T) {
	sm, err := NewSettingsManager(t.TempDir())
	if err != nil {
		t.Fatalf("NewSettingsManager: %v", err)
	}
	ks := &Store{}

	// A non-http(s) URL is rejected.
	if err := sm.Update(Settings{FileBrowserBaseURL: "ftp://files.example.com"}, ks); err == nil {
		t.Error("Update(ftp URL) = nil error, want validation error")
	}
	// Enabling the button without a URL is rejected.
	if err := sm.Update(Settings{FileBrowserEnabled: true}, ks); err == nil {
		t.Error("Update(enabled, no URL) = nil error, want validation error")
	}

	// Valid: URL is stored with the trailing slash stripped.
	if err := sm.Update(Settings{FileBrowserEnabled: true, FileBrowserBaseURL: "https://files.example.com/files/"}, ks); err != nil {
		t.Fatalf("Update(valid): %v", err)
	}
	got := sm.Get()
	if !got.FileBrowserEnabled || got.FileBrowserBaseURL != "https://files.example.com/files" {
		t.Errorf("settings = enabled=%v url=%q, want enabled with trimmed URL", got.FileBrowserEnabled, got.FileBrowserBaseURL)
	}

	// Turning it off keeps the URL for easy re-enabling; clearing the URL sticks.
	if err := sm.Update(Settings{FileBrowserEnabled: false, FileBrowserBaseURL: "https://files.example.com/files"}, ks); err != nil {
		t.Fatalf("Update(disable): %v", err)
	}
	got = sm.Get()
	if got.FileBrowserEnabled || got.FileBrowserBaseURL != "https://files.example.com/files" {
		t.Errorf("after disable: enabled=%v url=%q, want disabled with URL kept", got.FileBrowserEnabled, got.FileBrowserBaseURL)
	}
	if err := sm.Update(Settings{}, ks); err != nil {
		t.Fatalf("Update(clear): %v", err)
	}
	if got := sm.Get(); got.FileBrowserBaseURL != "" {
		t.Errorf("after clear: url=%q, want empty", got.FileBrowserBaseURL)
	}
}
