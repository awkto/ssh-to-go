package relay

import "sync"

// kickChannels maps TTY path -> channel. Each active relay registers its
// TTY so the detach handler can signal it before tmux detaches the client.
// This lets the relay send a "kicked" message to the browser while the
// WebSocket is still healthy, avoiding close-code delivery races.
var (
	mu       sync.Mutex
	channels = map[string]chan struct{}{}
)

// RegisterKickCh registers a kick channel for a TTY. Returns the channel.
func RegisterKickCh(tty string) chan struct{} {
	ch := make(chan struct{}, 1)
	mu.Lock()
	channels[tty] = ch
	mu.Unlock()
	return ch
}

// UnregisterKickCh removes a TTY from the registry.
func UnregisterKickCh(tty string) {
	mu.Lock()
	delete(channels, tty)
	mu.Unlock()
}

// SignalKick sends the kick signal for a TTY. Returns true if the TTY
// was found (i.e. a relay is actively connected on that TTY).
func SignalKick(tty string) bool {
	mu.Lock()
	ch, ok := channels[tty]
	mu.Unlock()
	if ok {
		select {
		case ch <- struct{}{}:
		default:
		}
	}
	return ok
}
