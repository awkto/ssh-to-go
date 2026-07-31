package tmux

import "testing"

func TestHandoffCommandPlain(t *testing.T) {
	m := NewManager()
	got := m.HandoffCommand("alice", "host.example", 0, "work", false)
	want := `ssh -t alice@host.example tmux attach-session -t "work"`
	if got != want {
		t.Errorf("HandoffCommand = %q, want %q", got, want)
	}
	got = m.HandoffCommand("alice", "host.example", 2222, "work", false)
	want = `ssh -t -p 2222 alice@host.example tmux attach-session -t "work"`
	if got != want {
		t.Errorf("HandoffCommand(port) = %q, want %q", got, want)
	}
}

// With mouse, the command backfills the per-session option before attaching so
// sessions created before native_mouse_mode existed still get a working wheel.
// The whole remote command is single-quoted: the string is pasted into the
// user's LOCAL shell, so tmux's own "\;" chaining would need double escaping.
func TestHandoffCommandBackfillsMouse(t *testing.T) {
	m := NewManager()
	got := m.HandoffCommand("alice", "host.example", 0, "work", true)
	want := `ssh -t alice@host.example 'tmux set-option -t "work" mouse on 2>/dev/null; exec tmux attach-session -t "work"'`
	if got != want {
		t.Errorf("HandoffCommand(mouse) = %q, want %q", got, want)
	}
	got = m.HandoffCommand("alice", "host.example", 2222, "work", true)
	want = `ssh -t -p 2222 alice@host.example 'tmux set-option -t "work" mouse on 2>/dev/null; exec tmux attach-session -t "work"'`
	if got != want {
		t.Errorf("HandoffCommand(mouse, port) = %q, want %q", got, want)
	}
}
