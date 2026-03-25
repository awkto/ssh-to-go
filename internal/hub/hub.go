package hub

import (
	"fmt"
	"sync"
	"time"

	"github.com/awkto/ssh-to-go/internal/config"
	"github.com/awkto/ssh-to-go/internal/sshutil"
	"github.com/awkto/ssh-to-go/internal/tmux"
)

type HostState struct {
	Config       config.Host    `json:"config"`
	TmuxDetected bool           `json:"tmux_detected"`
	TmuxVersion  string         `json:"tmux_version"`
	Sessions     []tmux.Session `json:"sessions"`
	LastPoll     time.Time      `json:"last_poll"`
	Error        string         `json:"error,omitempty"`
	Online       bool           `json:"online"`
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

// AllHosts returns the state of all hosts.
func (h *Hub) AllHosts() []HostState {
	h.mu.RLock()
	defer h.mu.RUnlock()

	var all []HostState
	for _, state := range h.hosts {
		all = append(all, *state)
	}
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

	h.Update(result)

	state, _ := h.GetHost(name)
	return state, nil
}
