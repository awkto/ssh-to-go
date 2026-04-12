package relay

import (
	"log"
	"sync"
)

// kicked tracks TTY paths that have been explicitly detached via the API.
// The relay checks this set when its SSH session ends to decide whether
// to send close code 4001 (kicked) instead of the default 4000 (session ended).
var kicked sync.Map

// MarkKicked registers a TTY as having been kicked.
func MarkKicked(tty string) {
	log.Printf("kick registry: marking %q", tty)
	kicked.Store(tty, struct{}{})
}

// WasKicked checks and clears the kicked flag for a TTY.
func WasKicked(tty string) bool {
	_, loaded := kicked.LoadAndDelete(tty)
	log.Printf("kick registry: check %q -> %v", tty, loaded)
	return loaded
}
