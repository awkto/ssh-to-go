package tmux

import (
	"log"
	"time"

	"github.com/awkto/ssh-to-go/internal/config"
	"github.com/awkto/ssh-to-go/internal/sshutil"
)

// PollResult is sent from a poller to the hub on each poll cycle.
type PollResult struct {
	HostName     string
	Sessions     []Session
	TmuxVersion  string
	TmuxDetected bool
	Error        error
}

// StartPoller launches a goroutine that periodically discovers tmux sessions
// on the given host and sends results to the provided channel.
func StartPoller(host config.Host, interval time.Duration, results chan<- PollResult, done <-chan struct{}) {
	go func() {
		mgr := NewManager()
		poll := func() {
			result := PollResult{HostName: host.Name}

			client, err := sshutil.Dial(host.Address, host.User, host.KeyPath)
			if err != nil {
				result.Error = err
				results <- result
				return
			}
			defer client.Close()

			version, err := mgr.DetectTmux(client)
			if err != nil {
				result.Error = err
				results <- result
				return
			}
			result.TmuxDetected = true
			result.TmuxVersion = version

			sessions, err := mgr.ListSessions(client)
			if err != nil {
				result.Error = err
				results <- result
				return
			}
			result.Sessions = sessions
			results <- result
		}

		// Initial poll immediately
		poll()

		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-done:
				log.Printf("poller for %s stopped", host.Name)
				return
			case <-ticker.C:
				poll()
			}
		}
	}()
}
