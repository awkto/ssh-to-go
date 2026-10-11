package api

import (
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"
)

func TestPasteImageName(t *testing.T) {
	now := time.Date(2026, 10, 12, 9, 30, 5, 0, time.UTC)
	a, b := pasteImageName("png", now), pasteImageName("png", now)
	if !regexp.MustCompile(`^20261012-093005-[0-9a-f]{6}\.png$`).MatchString(a) {
		t.Fatalf("unexpected name %q", a)
	}
	if a == b {
		t.Fatalf("two pastes in the same second collided: %q", a)
	}
}

// The script runs on the host's shell; run it under sh with a scratch
// HOME to check it writes stdin, prints the path, keeps the file private
// and prunes expired pastes but not fresh ones.
func TestPasteImageScript(t *testing.T) {
	home := t.TempDir()
	dir := filepath.Join(home, ".ssh-to-go", "paste")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	old, fresh := filepath.Join(dir, "old.png"), filepath.Join(dir, "fresh.png")
	for _, p := range []string{old, fresh} {
		os.WriteFile(p, []byte("x"), 0o600)
	}
	past := time.Now().Add(-(pasteTTLMinutes + 5) * time.Minute)
	os.Chtimes(old, past, past)

	cmd := exec.Command("sh", "-c", pasteImageScript("new.png"))
	cmd.Env = []string{"HOME=" + home, "PATH=" + os.Getenv("PATH")}
	cmd.Stdin = strings.NewReader("\x89PNG data")
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("script failed: %v", err)
	}

	want := filepath.Join(dir, "new.png")
	if string(out) != want {
		t.Fatalf("printed %q, want %q", out, want)
	}
	got, _ := os.ReadFile(want)
	if string(got) != "\x89PNG data" {
		t.Fatalf("file holds %q", got)
	}
	if fi, _ := os.Stat(want); fi.Mode().Perm() != 0o600 {
		t.Fatalf("file mode %v, want 0600", fi.Mode().Perm())
	}
	if _, err := os.Stat(old); !os.IsNotExist(err) {
		t.Fatal("expired paste was not pruned")
	}
	if _, err := os.Stat(fresh); err != nil {
		t.Fatal("fresh paste was pruned")
	}
}
