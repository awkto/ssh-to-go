package relay

import (
	"strings"
	"testing"
)

func TestEnsureSessionCmd(t *testing.T) {
	tmuxCmd := ensureSessionCmd("work", 50000, false)
	if !strings.Contains(tmuxCmd, `set-option -g history-limit 50000 \; new-session -d -s 'work'`) {
		t.Errorf("tmux: history-limit must be global and precede new-session: %q", tmuxCmd)
	}
	psmuxCmd := ensureSessionCmd("work", 50000, true)
	if strings.Contains(psmuxCmd, "-g history-limit") {
		t.Errorf("psmux: no global history-limit: %q", psmuxCmd)
	}
	if !strings.Contains(psmuxCmd, `new-session -d -s 'work' \; set-option -t 'work' history-limit 50000`) {
		t.Errorf("psmux: history-limit must follow new-session on the session: %q", psmuxCmd)
	}
	for _, c := range []string{tmuxCmd, psmuxCmd} {
		if !strings.HasPrefix(c, `tmux has-session -t 'work' 2>/dev/null || `) {
			t.Errorf("an existing session must be left alone: %q", c)
		}
	}
}

func TestQueueConnectReply(t *testing.T) {
	p := &controlParser{}
	queueConnectReply(p, false)
	if len(p.queue) != 1 || p.queue[0] != cmdConnect {
		t.Errorf("tmux: want one queued connect reply, got %v", p.queue)
	}
	if p.connectByFlag {
		t.Error("tmux: connect reply must come from the queue, not the flag")
	}
	p = &controlParser{}
	queueConnectReply(p, true)
	if len(p.queue) != 0 || !p.connectByFlag {
		t.Errorf("psmux: want nothing queued and connectByFlag, got %v %v", p.queue, p.connectByFlag)
	}
}

// TestParserPsmuxStartup replays the opening of a real psmux 3.4 control-mode
// stream: notifications, then the first reply block with no connect block
// ahead of it. psmux also echoes each command line it reads; those land in
// the parser's default case and must not disturb the reply order.
func TestParserPsmuxStartup(t *testing.T) {
	testParserPsmuxStartup(t, nil)
}

// TestParserPsmuxStartupWithConnectReply is the same stream had psmux sent
// tmux's connect reply (flag 0) first: it must be skipped, not consume the
// first queued reply.
func TestParserPsmuxStartupWithConnectReply(t *testing.T) {
	testParserPsmuxStartup(t, []string{"%begin 1792328699 265 0", "%end 1792328699 265 0"})
}

func testParserPsmuxStartup(t *testing.T, connectReply []string) {
	t.Helper()
	var emitted []byte
	p := &controlParser{emit: func(b []byte) { emitted = append(emitted, b...) }}
	queueConnectReply(p, true)

	p.pushCmd(cmdIgnore) // set-option window-size
	p.pushCmd(cmdMeta)
	var metaName string
	p.onMeta = func(n string) { metaName = n }
	p.pushCmd(cmdHistory)

	feed(p, connectReply...)
	feed(p,
		"",
		"%window-add @1",
		"%sessions-changed",
		"%session-changed $3 work",
		"set-option window-size largest",
		"%begin 1792328699 1 1",
		"%end 1792328699 1 1",
		`display-message -p -t 'work' "#{client_name},#{pane_id}"`,
		"%begin 1792328699 2 1",
		"/dev/pts/68840,%1",
		"%end 1792328699 2 1",
		"capture-pane -p -e -J -S - -E - -t 'work'",
		"%begin 1792328699 3 1",
		"$ echo hi",
		"hi",
		"%end 1792328699 3 1",
	)
	if metaName != "/dev/pts/68840" || p.paneID != "%1" {
		t.Errorf("meta reply misrouted: client=%q pane=%q", metaName, p.paneID)
	}
	if want := "$ echo hi\r\nhi\x1b[0m"; string(emitted) != want {
		t.Fatalf("history payload = %q, want %q", emitted, want)
	}

	emitted = nil
	feed(p, `%output %1 live\015\012`)
	if string(emitted) != "live\r\n" {
		t.Errorf("live output after history = %q", emitted)
	}
}

func TestPsmuxPasteCmd(t *testing.T) {
	got := psmuxPasteCmd("work", "/c/Users/me/it's $HOME \"q\";\x1b[31m\n")
	want := `tmux select-pane -t 'work' \; set-buffer -b ssh-to-go-paste -- '/c/Users/me/it'\''s $HOME "q";[31m' \; paste-buffer -p -d -b ssh-to-go-paste -t 'work'`
	if got != want {
		t.Errorf("psmuxPasteCmd:\n got  %s\n want %s", got, want)
	}
	if psmuxPasteCmd("work", "\n\x1b") != "" {
		t.Error("text that is all control characters should produce no command")
	}
}
