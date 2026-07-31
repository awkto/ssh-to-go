package tmux

import "testing"

// Every handoff raises escape-time: tmux 3.5's 10ms default flushes a
// fragmented terminal-probe reply into the pane as literal keystrokes
// (issue #79). The remote command is single-quoted for the user's LOCAL
// shell; the "\;" between the set-options survives one unquoting hop and
// reaches tmux as its command separator.
func TestHandoffCommandPlain(t *testing.T) {
	m := NewManager()
	got := m.HandoffCommand("alice", "host.example", 0, "work", false)
	want := `ssh -t alice@host.example 'tmux set-option -s escape-time 200 2>/dev/null; exec tmux attach-session -t "work"'`
	if got != want {
		t.Errorf("HandoffCommand = %q, want %q", got, want)
	}
	got = m.HandoffCommand("alice", "host.example", 2222, "work", false)
	want = `ssh -t -p 2222 alice@host.example 'tmux set-option -s escape-time 200 2>/dev/null; exec tmux attach-session -t "work"'`
	if got != want {
		t.Errorf("HandoffCommand(port) = %q, want %q", got, want)
	}
}

// With mouse, the command also backfills the per-session option before
// attaching so sessions created before native_mouse_mode existed still get a
// working wheel.
func TestHandoffCommandBackfillsMouse(t *testing.T) {
	m := NewManager()
	got := m.HandoffCommand("alice", "host.example", 0, "work", true)
	want := `ssh -t alice@host.example 'tmux set-option -s escape-time 200 \; set-option -t "work" mouse on 2>/dev/null; exec tmux attach-session -t "work"'`
	if got != want {
		t.Errorf("HandoffCommand(mouse) = %q, want %q", got, want)
	}
	got = m.HandoffCommand("alice", "host.example", 2222, "work", true)
	want = `ssh -t -p 2222 alice@host.example 'tmux set-option -s escape-time 200 \; set-option -t "work" mouse on 2>/dev/null; exec tmux attach-session -t "work"'`
	if got != want {
		t.Errorf("HandoffCommand(mouse, port) = %q, want %q", got, want)
	}
}
