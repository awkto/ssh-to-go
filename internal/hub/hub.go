package hub

import (
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/awkto/ssh-to-go/internal/config"
	"github.com/awkto/ssh-to-go/internal/metrics"
	"github.com/awkto/ssh-to-go/internal/sshutil"
	"github.com/awkto/ssh-to-go/internal/tmux"
)

// LoadHistorySize is the number of 1-minute load samples retained per host.
// At a typical 5-10s poll interval this covers the last 5-10 minutes.
const LoadHistorySize = 60

type HostState struct {
	Config       config.Host      `json:"config"`
	TmuxDetected bool             `json:"tmux_detected"`
	TmuxVersion  string           `json:"tmux_version"`
	Sessions     []tmux.Session   `json:"sessions"`
	LastPoll     time.Time        `json:"last_poll"`
	Error        string           `json:"error,omitempty"`
	Online       bool             `json:"online"`
	DetectedOS   string           `json:"detected_os,omitempty"`
	Metrics      *metrics.Metrics `json:"metrics,omitempty"`
	LoadHistory  []float64        `json:"load_history,omitempty"`
}

// HostSession is a flattened view of a session with its host info.
type HostSession struct {
	HostName string       `json:"host_name"`
	Host     config.Host  `json:"-"`
	Session  tmux.Session `json:"session"`
}

type Hub struct {
	mu    sync.RWMutex
	hosts map[string]*HostState
}

func New(hosts []config.Host) *Hub {
	h := &Hub{
		hosts: make(map[string]*HostState, len(hosts)),
	}
	for _, host := range hosts {
		h.hosts[host.Name] = &HostState{Config: host}
	}
	return h
}

// Update applies a poll result to the hub state.
func (h *Hub) Update(result tmux.PollResult) {
	h.mu.Lock()
	defer h.mu.Unlock()

	state, ok := h.hosts[result.HostName]
	if !ok {
		return
	}

	state.LastPoll = time.Now()
	state.TmuxDetected = result.TmuxDetected
	state.TmuxVersion = result.TmuxVersion

	if result.Error != nil {
		state.Error = result.Error.Error()
		state.Online = false
	} else {
		state.Error = ""
		state.Online = true
		state.Sessions = result.Sessions
	}
	if result.DetectedOS != "" {
		state.DetectedOS = result.DetectedOS
	}
	if result.Metrics != nil {
		state.Metrics = result.Metrics
		// Append the 1-min load to the ring buffer, trimming from the front
		// once we exceed LoadHistorySize. Frontend treats it as oldest-first.
		state.LoadHistory = append(state.LoadHistory, result.Metrics.Load1)
		if len(state.LoadHistory) > LoadHistorySize {
			state.LoadHistory = state.LoadHistory[len(state.LoadHistory)-LoadHistorySize:]
		}
	}
}

// AllSessions returns a flat list of all sessions across all hosts.
func (h *Hub) AllSessions() []HostSession {
	h.mu.RLock()
	defer h.mu.RUnlock()

	var all []HostSession
	for _, state := range h.hosts {
		for _, s := range state.Sessions {
			all = append(all, HostSession{
				HostName: state.Config.Name,
				Host:     state.Config,
				Session:  s,
			})
		}
	}
	return all
}

// AllHosts returns the state of all hosts, sorted by name for stable ordering.
func (h *Hub) AllHosts() []HostState {
	h.mu.RLock()
	defer h.mu.RUnlock()

	all := make([]HostState, 0, len(h.hosts))
	for _, state := range h.hosts {
		all = append(all, *state)
	}
	sort.Slice(all, func(i, j int) bool {
		return all[i].Config.Name < all[j].Config.Name
	})
	return all
}

// GetHost returns the state of a specific host.
func (h *Hub) GetHost(name string) (*HostState, bool) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	state, ok := h.hosts[name]
	if !ok {
		return nil, false
	}
	copy := *state
	return &copy, true
}

// GetHostConfig returns the config for a specific host.
func (h *Hub) GetHostConfig(name string) (config.Host, bool) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	state, ok := h.hosts[name]
	if !ok {
		return config.Host{}, false
	}
	return state.Config, true
}

// AddHost registers a new host in the hub. Returns false if the host already exists.
func (h *Hub) AddHost(host config.Host) bool {
	h.mu.Lock()
	defer h.mu.Unlock()

	if _, exists := h.hosts[host.Name]; exists {
		return false
	}
	h.hosts[host.Name] = &HostState{Config: host}
	return true
}

// UpdateHost updates the config for an existing host.
func (h *Hub) UpdateHost(host config.Host) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if state, ok := h.hosts[host.Name]; ok {
		state.Config = host
	}
}

// RemoveHost removes a host from the hub. Returns false if not found.
func (h *Hub) RemoveHost(name string) bool {
	h.mu.Lock()
	defer h.mu.Unlock()

	if _, ok := h.hosts[name]; !ok {
		return false
	}
	delete(h.hosts, name)
	return true
}

// ScanHost performs an immediate poll of a single host and updates the hub state.
// It returns the updated HostState. keyPath is the resolved private key path.
func (h *Hub) ScanHost(name string, tm *tmux.Manager, keyPath string) (*HostState, error) {
	hostCfg, ok := h.GetHostConfig(name)
	if !ok {
		return nil, fmt.Errorf("host %q not found", name)
	}

	result := tmux.PollResult{HostName: name}

	client, err := sshutil.Dial(hostCfg.DialAddress(), hostCfg.User, keyPath)
	if err != nil {
		result.Error = err
		h.Update(result)
		return nil, fmt.Errorf("ssh connect failed: %w", err)
	}
	defer client.Close()

	version, err := tm.DetectTmux(client)
	if err != nil {
		result.Error = err
		h.Update(result)
		return nil, fmt.Errorf("tmux detection failed: %w", err)
	}
	result.TmuxDetected = true
	result.TmuxVersion = version

	sessions, err := tm.ListSessions(client)
	if err != nil {
		result.Error = err
		h.Update(result)
		return nil, fmt.Errorf("list sessions failed: %w", err)
	}
	result.Sessions = sessions
	result.DetectedOS = tmux.DetectOSViaClient(client)

	if m, err := metrics.Collect(client); err == nil {
		result.Metrics = &m
	}

	h.Update(result)

	state, _ := h.GetHost(name)
	return state, nil
}
