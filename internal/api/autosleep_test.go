package api

import (
	"testing"
	"time"

	"github.com/awkto/ssh-to-go/internal/sessionreg"
	"github.com/awkto/ssh-to-go/internal/tmux"
)

func TestShouldAutoSleep(t *testing.T) {
	now := time.Date(2026, 7, 26, 12, 0, 0, 0, time.UTC)
	day := 24 * time.Hour
	old := now.Add(-3 * day)
	recent := now.Add(-time.Hour)

	cases := []struct {
		name    string
		entry   sessionreg.Entry
		session tmux.Session
		keep    bool
		timeout time.Duration
		want    bool
	}{
		{
			name:    "idle for longer than the timeout",
			entry:   sessionreg.Entry{LastAttachedAt: old},
			session: tmux.Session{Activity: old},
			timeout: 2 * day,
			want:    true,
		},
		{
			name:    "feature off",
			entry:   sessionreg.Entry{LastAttachedAt: old},
			session: tmux.Session{Activity: old},
			timeout: 0,
			want:    false,
		},
		{
			name:    "kept awake",
			entry:   sessionreg.Entry{LastAttachedAt: old},
			session: tmux.Session{Activity: old},
			keep:    true,
			timeout: 2 * day,
			want:    false,
		},
		{
			name:    "client attached",
			entry:   sessionreg.Entry{LastAttachedAt: old},
			session: tmux.Session{Activity: old, AttachedClients: 1},
			timeout: 2 * day,
			want:    false,
		},
		{
			// The throwaway collector owns these and kills them outright;
			// offloading one would leave a permanent-looking resumable entry.
			name:    "throwaway is not ours to sleep",
			entry:   sessionreg.Entry{LastAttachedAt: old, Throwaway: true},
			session: tmux.Session{Activity: old},
			timeout: 2 * day,
			want:    false,
		},
		{
			name:    "not idle long enough",
			entry:   sessionreg.Entry{LastAttachedAt: recent},
			session: tmux.Session{Activity: recent},
			timeout: 2 * day,
			want:    false,
		},
		{
			// Nobody has attached in days, but the session itself has been
			// producing output — tmux's clock is the more recent one and wins.
			name:    "tmux activity keeps it awake",
			entry:   sessionreg.Entry{LastAttachedAt: old},
			session: tmux.Session{Activity: recent},
			timeout: 2 * day,
			want:    false,
		},
		{
			// The mirror case: tmux is silent but a client was on it recently.
			name:    "recent attach keeps it awake",
			entry:   sessionreg.Entry{LastAttachedAt: recent},
			session: tmux.Session{Activity: old},
			timeout: 2 * day,
			want:    false,
		},
		{
			name:    "no idle clock at all",
			entry:   sessionreg.Entry{},
			session: tmux.Session{},
			timeout: 2 * day,
			want:    false,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := shouldAutoSleep(c.entry, c.session, c.keep, c.timeout, now); got != c.want {
				t.Errorf("shouldAutoSleep = %v, want %v", got, c.want)
			}
		})
	}
}
