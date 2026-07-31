package api

import (
	"log"
	"time"

	"github.com/awkto/ssh-to-go/internal/sessionreg"
	"github.com/awkto/ssh-to-go/internal/sshutil"
)

// Throwaway sessions self-destruct. Two things can collect one:
//
//   - the last client detaching (checked when a websocket relay returns), or
//   - IdleTimeout passing with nothing attached (the sweeper below).
//
// Collecting means killing the tmux session AND dropping the registry entry,
// so it leaves neither a live session nor a "resumable" ghost in the UI.
const (
	// IdleTimeout is how long a throwaway may sit with nothing attached.
	IdleTimeout = 15 * time.Minute

	// DetachGrace is the pause between a client disconnecting and the
	// zero-client check. A browser refresh drops the websocket for a moment
	// and reconnects; without this, F5 would destroy the session. Walking
	// away still collects it — this only has to outlast a reload.
	DetachGrace = 20 * time.Second

	// sweepInterval is how often idle throwaways are looked for. The
	// timeout is minutes wide, so this need not be tight.
	sweepInterval = time.Minute
)

// collectIfIdle kills and purges a throwaway session when nothing is
// attached to it. Reports whether it collected.
//
// The kill and the registry removal are deliberately ordered: the entry is
// only dropped once tmux confirms the session is gone. Purging first would
// orphan a live session with no record of it — invisible and immortal.
func (h *Handlers) collectIfIdle(hostName, sessionName string) bool {
	hostCfg, ok := h.Hub.GetHostConfig(hostName)
	if !ok {
		return false
	}
	client, err := sshutil.Dial(hostCfg.DialAddress(), hostCfg.User, h.resolveKey(hostCfg))
	if err != nil {
		// Host unreachable: leave the entry alone and try again next sweep.
		log.Printf("throwaway %s/%s: ssh failed, will retry: %v", hostName, sessionName, err)
		return false
	}
	defer client.Close()

	clients, err := h.Tmux.ListClients(client, sessionName)
	if err == nil && len(clients) > 0 {
		return false // someone is still attached (or came back)
	}

	// Nothing is attached, so there is usually no relay to tell — but a
	// client that reconnected between the check above and the kill would
	// otherwise recreate the session with `new-session -A`. Announcing arms
	// the reconnect guard as well as signalling. See terminate.go.
	h.announceTermination(hostName, sessionName, "killed")

	if err := h.Tmux.KillSession(client, sessionName); err != nil {
		// The session may simply be gone already (user typed exit); that
		// still means it should stop being tracked, so fall through.
		log.Printf("throwaway %s/%s: kill: %v", hostName, sessionName, err)
	}
	if h.Registry != nil {
		if err := h.Registry.Remove(hostName, sessionName); err != nil {
			log.Printf("throwaway %s/%s: registry remove: %v", hostName, sessionName, err)
		}
		h.refreshHidden(hostName)
	}
	if h.SessionIcons != nil {
		_ = h.SessionIcons.Delete(hostName, sessionName)
	}
	log.Printf("throwaway collected: host=%s session=%s", hostName, sessionName)
	return true
}

// onClientDetached is called when a websocket relay ends. For a throwaway it
// schedules the zero-client check after the grace window.
func (h *Handlers) onClientDetached(hostName, sessionName string) {
	if h.Registry == nil || !h.Registry.Flavours(hostName, sessionName).Throwaway {
		return
	}
	go func() {
		time.Sleep(DetachGrace)
		// Re-check the flag: the session could have been killed, renamed or
		// collected by the sweeper during the grace window.
		if h.Registry.Flavours(hostName, sessionName).Throwaway {
			h.collectIfIdle(hostName, sessionName)
		}
	}()
}

// StartSweepers runs the background collectors until ctx-less shutdown.
//
// The throwaway pass also keeps the idle clock fed: any throwaway with
// clients attached (per the last poll) has its LastAttachedAt refreshed, so
// "idle" means "no client for IdleTimeout", not "created IdleTimeout ago".
//
// The auto-sleep pass shares the tick — both timeouts are far wider than a
// minute, and a second ticker would buy nothing. It is a no-op unless the
// idle-offload setting is on.
func (h *Handlers) StartSweepers() {
	go func() {
		for range time.Tick(sweepInterval) {
			now := time.Now().UTC()
			h.sweepThrowaways(now)
			h.sweepIdleSessions(now)
		}
	}()
}

// sweepThrowaways is one pass of the sweeper, split out for testing.
func (h *Handlers) sweepThrowaways(now time.Time) {
	if h.Registry == nil {
		return
	}
	for _, e := range h.Registry.Throwaways() {
		attached := h.sessionHasClients(e)
		if attached {
			h.Registry.MarkAttached(e.Host, e.Name)
		}
		if shouldCollect(e, attached, now) {
			h.collectIfIdle(e.Host, e.Name)
		}
	}
}

// shouldCollect is the sweeper's decision, kept free of SSH so it can be
// tested directly: collect a throwaway with nothing attached that has been
// that way for at least IdleTimeout.
func shouldCollect(e sessionreg.Entry, attached bool, now time.Time) bool {
	if !e.Throwaway || attached {
		return false
	}
	return now.Sub(e.LastAttachedAt) >= IdleTimeout
}

// sessionHasClients answers from the last poll rather than dialing: the
// sweeper runs every minute across every throwaway, and the poller already
// collects attach counts on a connection it is holding anyway.
func (h *Handlers) sessionHasClients(e sessionreg.Entry) bool {
	state, ok := h.Hub.GetHost(e.Host)
	if !ok {
		return false
	}
	for _, s := range state.Sessions {
		if s.Name == e.Name {
			return s.AttachedClients > 0
		}
	}
	return false
}
