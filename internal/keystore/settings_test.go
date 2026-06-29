package keystore

import "testing"

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
