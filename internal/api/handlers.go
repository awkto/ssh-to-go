package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/awkto/ssh-to-go/internal/config"
	"github.com/awkto/ssh-to-go/internal/hub"
	"github.com/awkto/ssh-to-go/internal/sshutil"
	"github.com/awkto/ssh-to-go/internal/tmux"
)

type Handlers struct {
	Hub          *hub.Hub
	Tmux         *tmux.Manager
	ConfigPath   string
	PollInterval time.Duration
	PollResults  chan<- tmux.PollResult
	Done         <-chan struct{}
}

func (h *Handlers) ListSessions(w http.ResponseWriter, r *http.Request) {
	sessions := h.Hub.AllSessions()
	if sessions == nil {
		sessions = []hub.HostSession{}
	}
	writeJSON(w, sessions)
}

func (h *Handlers) ListHosts(w http.ResponseWriter, r *http.Request) {
	hosts := h.Hub.AllHosts()
	if hosts == nil {
		hosts = []hub.HostState{}
	}
	writeJSON(w, hosts)
}

type createSessionReq struct {
	Name string `json:"name"`
}

func (h *Handlers) CreateSession(w http.ResponseWriter, r *http.Request) {
	hostName := r.PathValue("host")
	hostCfg, ok := h.Hub.GetHostConfig(hostName)
	if !ok {
		http.Error(w, "host not found", http.StatusNotFound)
		return
	}

	var req createSessionReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if req.Name == "" {
		http.Error(w, "session name required", http.StatusBadRequest)
		return
	}

	client, err := sshutil.Dial(hostCfg.Address, hostCfg.User, hostCfg.KeyPath)
	if err != nil {
		http.Error(w, fmt.Sprintf("ssh connect failed: %v", err), http.StatusBadGateway)
		return
	}
	defer client.Close()

	if err := h.Tmux.CreateSession(client, req.Name); err != nil {
		http.Error(w, fmt.Sprintf("create session failed: %v", err), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	writeJSON(w, map[string]string{"status": "created", "name": req.Name})
}

func (h *Handlers) KillSession(w http.ResponseWriter, r *http.Request) {
	hostName := r.PathValue("host")
	sessionName := r.PathValue("session")

	hostCfg, ok := h.Hub.GetHostConfig(hostName)
	if !ok {
		http.Error(w, "host not found", http.StatusNotFound)
		return
	}

	client, err := sshutil.Dial(hostCfg.Address, hostCfg.User, hostCfg.KeyPath)
	if err != nil {
		http.Error(w, fmt.Sprintf("ssh connect failed: %v", err), http.StatusBadGateway)
		return
	}
	defer client.Close()

	if err := h.Tmux.KillSession(client, sessionName); err != nil {
		http.Error(w, fmt.Sprintf("kill session failed: %v", err), http.StatusInternalServerError)
		return
	}

	writeJSON(w, map[string]string{"status": "killed"})
}

func (h *Handlers) Handoff(w http.ResponseWriter, r *http.Request) {
	hostName := r.PathValue("host")
	sessionName := r.PathValue("session")

	hostCfg, ok := h.Hub.GetHostConfig(hostName)
	if !ok {
		http.Error(w, "host not found", http.StatusNotFound)
		return
	}

	cmd := h.Tmux.HandoffCommand(hostCfg.User, hostCfg.Address, sessionName)
	writeJSON(w, map[string]string{"command": cmd})
}

// ScanHost triggers an immediate poll of a specific host.
func (h *Handlers) ScanHost(w http.ResponseWriter, r *http.Request) {
	hostName := r.PathValue("host")

	state, err := h.Hub.ScanHost(hostName, h.Tmux)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		http.Error(w, fmt.Sprintf("scan failed: %v", err), http.StatusBadGateway)
		return
	}

	writeJSON(w, state)
}

// ScanAll triggers an immediate poll of all hosts.
func (h *Handlers) ScanAll(w http.ResponseWriter, r *http.Request) {
	hosts := h.Hub.AllHosts()
	var results []hub.HostState
	for _, host := range hosts {
		state, err := h.Hub.ScanHost(host.Config.Name, h.Tmux)
		if err != nil {
			log.Printf("scan %s: %v", host.Config.Name, err)
			// Still include the host state (it was updated with the error)
			updated, ok := h.Hub.GetHost(host.Config.Name)
			if ok {
				results = append(results, *updated)
			}
			continue
		}
		results = append(results, *state)
	}
	if results == nil {
		results = []hub.HostState{}
	}
	writeJSON(w, results)
}

type addHostReq struct {
	Name    string `json:"name"`
	Address string `json:"address"`
	User    string `json:"user"`
	KeyPath string `json:"key_path"`
}

// AddHost adds a new host at runtime and saves it to the config file.
func (h *Handlers) AddHost(w http.ResponseWriter, r *http.Request) {
	var req addHostReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if req.Name == "" || req.Address == "" || req.User == "" {
		http.Error(w, "name, address, and user are required", http.StatusBadRequest)
		return
	}

	// Normalize address to include port if missing
	address := req.Address
	if !strings.Contains(address, ":") {
		address = address + ":22"
	}

	host := config.Host{
		Name:    req.Name,
		Address: address,
		User:    req.User,
		KeyPath: req.KeyPath,
	}

	if !h.Hub.AddHost(host) {
		http.Error(w, fmt.Sprintf("host %q already exists", req.Name), http.StatusConflict)
		return
	}

	// Save to config file
	if err := config.AppendHost(h.ConfigPath, config.Host{
		Name:    req.Name,
		Address: req.Address, // save original (without normalized port)
		User:    req.User,
		KeyPath: req.KeyPath,
	}); err != nil {
		log.Printf("warning: host added at runtime but config save failed: %v", err)
	}

	// Start a poller for the new host
	tmux.StartPoller(host, h.PollInterval, h.PollResults, h.Done)
	log.Printf("started poller for new host %s (%s@%s)", host.Name, host.User, host.Address)

	w.WriteHeader(http.StatusCreated)
	writeJSON(w, map[string]string{"status": "added", "name": req.Name})
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}
