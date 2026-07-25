package api

import (
	"sync"
	"time"

	"github.com/awkto/ssh-to-go/internal/relay"
)

// Killing or offloading a session used to bring it straight back.
//
// The kill itself always worked; the attached web terminal then noticed its
// socket drop and reconnected, and a reconnect runs `tmux new-session -A`,
// which CREATES the session when it isn't there. Three seconds after the
// kill, the poller saw a live tmux session with no registry entry and it
// reappeared in the UI.
//
// Two things stop that. relay.SignalTerminate tells every attached relay the
// session is going away on purpose, so the browser suppresses its reconnect
// (see terminal.js). And this guard covers the race the signal can't: a
// reconnect already in flight when the kill lands, or a client that never got
// the message. For a short window after a deliberate termination, attaching
// to that session is refused outright rather than served by `-A`.
const terminateGuard = 15 * time.Second

// terminateSignalGrace is how long we let the "you were killed" message
// travel before tmux is torn down under the relay. Mirrors the same pause in
// DetachClients for the kick message.
const terminateSignalGrace = 150 * time.Millisecond

type terminationGuard struct {
	mu   sync.Mutex
	when map[string]time.Time // host \x00 session -> termination time
}

func guardKey(host, session string) string { return host + "\x00" + session }

// announceTermination tells attached relays why they are about to be cut off
// and arms the reconnect guard. Call it BEFORE killing tmux.
func (h *Handlers) announceTermination(host, session, reason string) {
	h.markTerminated(host, session)
	if n := relay.SignalTerminate(host, session, reason); n > 0 {
		time.Sleep(terminateSignalGrace)
	}
}

func (h *Handlers) markTerminated(host, session string) {
	h.terminated.mu.Lock()
	defer h.terminated.mu.Unlock()
	if h.terminated.when == nil {
		h.terminated.when = make(map[string]time.Time)
	}
	now := time.Now()
	h.terminated.when[guardKey(host, session)] = now
	// Opportunistic sweep: the map is tiny and only grows on kill/offload,
	// but there's no other moment to drop stale keys.
	for k, t := range h.terminated.when {
		if now.Sub(t) > terminateGuard {
			delete(h.terminated.when, k)
		}
	}
}

// clearTerminated lifts the guard, so a session deliberately brought back
// (create, recreate, duplicate) can be attached immediately.
func (h *Handlers) clearTerminated(host, session string) {
	h.terminated.mu.Lock()
	defer h.terminated.mu.Unlock()
	delete(h.terminated.when, guardKey(host, session))
}

// recentlyTerminated reports whether attaching should be refused because the
// session was deliberately killed moments ago.
func (h *Handlers) recentlyTerminated(host, session string) bool {
	h.terminated.mu.Lock()
	defer h.terminated.mu.Unlock()
	t, ok := h.terminated.when[guardKey(host, session)]
	return ok && time.Since(t) < terminateGuard
}
