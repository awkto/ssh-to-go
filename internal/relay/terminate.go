package relay

import "sync"

// Deliberate-termination signalling.
//
// A relay whose WebSocket simply drops is reconnected by the browser after a
// few seconds, and that reconnect runs `tmux new-session -A` — attach OR
// CREATE. For an ordinary network blip that is exactly right; for a session
// the user just killed it resurrects it three seconds later.
//
// So kill/offload announce themselves: the handler signals every relay on
// that session, each relay tells its browser WHY it is about to close, and
// the browser suppresses its reconnect and draws an end state. Same shape as
// the kick channel next door, but keyed by session rather than by client —
// several clients can be attached to one session and all of them need to
// hear it.
var (
	termMu    sync.Mutex
	termChans = map[string]map[chan string]struct{}{}
)

func termKey(host, session string) string { return host + "\x00" + session }

// RegisterTerminateCh subscribes a relay to termination signals for a
// session. The returned channel must be passed back to Unregister.
func RegisterTerminateCh(host, session string) chan string {
	ch := make(chan string, 1)
	termMu.Lock()
	k := termKey(host, session)
	if termChans[k] == nil {
		termChans[k] = map[chan string]struct{}{}
	}
	termChans[k][ch] = struct{}{}
	termMu.Unlock()
	return ch
}

// UnregisterTerminateCh drops a relay's subscription.
func UnregisterTerminateCh(host, session string, ch chan string) {
	termMu.Lock()
	k := termKey(host, session)
	if set := termChans[k]; set != nil {
		delete(set, ch)
		if len(set) == 0 {
			delete(termChans, k)
		}
	}
	termMu.Unlock()
}

// SignalTerminate tells every relay attached to a session that it is being
// terminated on purpose, and why ("killed" / "offloaded"). Returns how many
// relays were notified so the caller can decide whether to pause before
// tearing tmux down.
func SignalTerminate(host, session, reason string) int {
	termMu.Lock()
	set := termChans[termKey(host, session)]
	chans := make([]chan string, 0, len(set))
	for ch := range set {
		chans = append(chans, ch)
	}
	termMu.Unlock()
	for _, ch := range chans {
		select {
		case ch <- reason:
		default:
		}
	}
	return len(chans)
}
