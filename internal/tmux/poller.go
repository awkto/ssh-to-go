package tmux

import (
	"log"
	"strings"
	"time"

	"github.com/awkto/ssh-to-go/internal/config"
	"github.com/awkto/ssh-to-go/internal/metrics"
	"github.com/awkto/ssh-to-go/internal/sshutil"
	"golang.org/x/crypto/ssh"
)

// DetectOSViaClient tries to identify the remote OS by reading /etc/os-release,
// falling back to uname.
func DetectOSViaClient(client *ssh.Client) string {
	out, err := sshutil.Exec(client, "cat /etc/os-release 2>/dev/null")
	if err == nil && out != "" {
		// Parse NAME= or PRETTY_NAME= from os-release
		for _, line := range strings.Split(out, "\n") {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "PRETTY_NAME=") {
				val := strings.TrimPrefix(line, "PRETTY_NAME=")
				val = strings.Trim(val, "\"")
				return val
			}
		}
		for _, line := range strings.Split(out, "\n") {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "NAME=") {
				val := strings.TrimPrefix(line, "NAME=")
				val = strings.Trim(val, "\"")
				return val
			}
		}
	}
	// Fallback to uname
	out, err = sshutil.Exec(client, "uname -s 2>/dev/null")
	if err == nil && out != "" {
		return strings.TrimSpace(out)
	}
	return "Linux"
}

// KeyResolver returns the private key path for a given host.
type KeyResolver func(host config.Host) string

// PollResult is sent from a poller to the hub on each poll cycle.
type PollResult struct {
	HostName     string
	Sessions     []Session
	TmuxVersion  string
	TmuxDetected bool
	DetectedOS   string
	Metrics      *metrics.Metrics
	Error        error
}

// StartPoller launches a goroutine that periodically discovers tmux sessions
// on the given host and sends results to the provided channel.
func StartPoller(host config.Host, interval time.Duration, resolveKey KeyResolver, results chan<- PollResult, done <-chan struct{}) {
	go func() {
		mgr := NewManager()
		var cachedOS string
		poll := func() {
			result := PollResult{HostName: host.Name}

			keyPath := resolveKey(host)
			client, err := sshutil.Dial(host.DialAddress(), host.User, keyPath)
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

			// Detect OS once per poller lifetime
			if cachedOS == "" {
				cachedOS = DetectOSViaClient(client)
			}
			result.DetectedOS = cachedOS

			// Best-effort metrics collection. Don't fail the poll if it errors —
			// hosts without /proc or busybox-stripped systems can still report
			// tmux state without metrics.
			if m, err := metrics.Collect(client); err == nil {
				result.Metrics = &m
			} else {
				log.Printf("metrics for %s: %v", host.Name, err)
			}

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
