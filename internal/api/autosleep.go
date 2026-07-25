package api

import (
	"log"
	"time"

	"github.com/awkto/ssh-to-go/internal/sessionreg"
	"github.com/awkto/ssh-to-go/internal/sshutil"
	"github.com/awkto/ssh-to-go/internal/tmux"
)

// Auto-sleep: offload sessions that have been idle for a long time.
//
// "Sleep" is deliberately offload, not kill — the tmux session goes away to
// free the host's RAM, the registry entry stays, and it comes back from the
// Resumable panel with its working directory and launch command. Being
// recoverable is what makes it safe to run unattended.
//
// The whole design fails safe. Never sleeping a session costs a few hundred
// megabytes; sleeping one mid-work costs the user their work. So every check
// is a veto and any doubt (unreachable host, unreadable panes, an unfamiliar
// foreground process) leaves the session alone.

// shouldAutoSleep is the sweeper's decision, kept free of SSH so it can be
// tested directly. The SSH-side quiet check is applied separately by the
// caller — this covers everything answerable from state we already hold.
func shouldAutoSleep(e sessionreg.Entry, s tmux.Session, keepAwake bool, timeout time.Duration, now time.Time) bool {
	if timeout <= 0 || keepAwake {
		return false
	}
	// Throwaways belong to the throwaway collector, which kills them
	// outright on a much shorter clock. Offloading one would resurrect it
	// as a permanent-looking resumable entry — the opposite of the point.
	if e.Throwaway {
		return false
	}
	if s.AttachedClients > 0 {
		return false
	}
	// Two independent idle clocks, whichever is more recent wins:
	// LastAttachedAt is ours (a client was on it), session_activity is
	// tmux's (output or input happened in it). Only ever attaching would
	// leave the first stale; a session that talks to itself keeps the
	// second fresh.
	last := e.LastAttachedAt
	if s.Activity.After(last) {
		last = s.Activity
	}
	if last.IsZero() {
		return false // no idea how long it's been idle — leave it
	}
	return now.Sub(last) >= timeout
}

// sweepIdleSessions is one pass of the auto-sleep sweeper.
func (h *Handlers) sweepIdleSessions(now time.Time) {
	if h.Registry == nil {
		return
	}
	timeout := h.Settings.IdleOffloadTimeout()
	if timeout <= 0 {
		return
	}
	for _, e := range h.Registry.List() {
		// Incognito sessions are swept like any other: GetHost (unlike the
		// UI listing) still sees them, and hidden is not the same as
		// protected — an incognito session left idle for days is exactly
		// the RAM this feature is meant to reclaim.
		s, live := h.liveSession(e)
		if !live {
			continue // already offloaded, or the host is unreachable
		}
		keep := false
		if h.SessionIcons != nil {
			keep = h.SessionIcons.Get(e.Host, s.Name).KeepAwake
		}
		if !shouldAutoSleep(e, s, keep, timeout, now) {
			continue
		}
		h.autoSleep(e, s.Name)
	}
}

// autoSleep offloads one idle session, after confirming over SSH that it is
// really unattended and really running nothing. Every failure path here is a
// no-op: the next sweep tries again.
func (h *Handlers) autoSleep(e sessionreg.Entry, liveName string) {
	hostCfg, ok := h.Hub.GetHostConfig(e.Host)
	if !ok {
		return
	}
	client, err := sshutil.Dial(hostCfg.DialAddress(), hostCfg.User, h.resolveKey(hostCfg))
	if err != nil {
		log.Printf("auto-sleep %s/%s: ssh failed, will retry: %v", e.Host, liveName, err)
		return
	}
	defer client.Close()

	// Re-check attachment against tmux rather than the last poll: someone
	// may have attached in the seconds since, and the poll interval is the
	// one window where this decision can go stale.
	if clients, err := h.Tmux.ListClients(client, liveName); err != nil || len(clients) > 0 {
		return
	}

	// The hard part of the feature: an overnight build or a long `claude`
	// run is detached and silent, and must not be slept. Every pane in
	// every window has to be a shell sitting at a prompt.
	quiet, err := h.Tmux.SessionQuiet(client, liveName)
	if err != nil {
		log.Printf("auto-sleep %s/%s: pane check failed, skipping: %v", e.Host, liveName, err)
		return
	}
	if !quiet {
		return
	}

	// Offload, not kill: same announcement as the manual path so an open web
	// terminal doesn't reconnect and recreate what we just put to sleep.
	h.announceTermination(e.Host, liveName, "offloaded")
	if err := h.Tmux.KillSession(client, liveName); err != nil {
		log.Printf("auto-sleep %s/%s: kill: %v", e.Host, liveName, err)
		return
	}
	// Registry entry stays — that's what makes it resumable. The flag lets
	// the UI say the session slept rather than looking like it vanished.
	h.Registry.MarkAutoOffloaded(e.Host, e.Name)
	log.Printf("auto-slept (offloaded) idle session: host=%s session=%s", e.Host, liveName)
}

// liveSession finds a tracked entry's session in the last poll. Names are
// compared sanitized, since a legacy entry may carry a spaced name while the
// live session is hyphenated.
func (h *Handlers) liveSession(e sessionreg.Entry) (tmux.Session, bool) {
	state, ok := h.Hub.GetHost(e.Host)
	if !ok || !state.Online {
		return tmux.Session{}, false
	}
	want := sanitizeSessionName(e.Name)
	for _, s := range state.Sessions {
		if sanitizeSessionName(s.Name) == want {
			return s, true
		}
	}
	return tmux.Session{}, false
}
