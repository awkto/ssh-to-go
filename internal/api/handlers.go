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
	"github.com/awkto/ssh-to-go/internal/keystore"
	"github.com/awkto/ssh-to-go/internal/sshutil"
	"github.com/awkto/ssh-to-go/internal/tmux"
)

type Handlers struct {
	Hub          *hub.Hub
	Tmux         *tmux.Manager
	KeyStore     *keystore.Store
	Settings     *keystore.SettingsManager
	ConfigPath   string
	PollInterval time.Duration
	PollResults  chan<- tmux.PollResult
	Done         <-chan struct{}
}

func (h *Handlers) resolveKey(host config.Host) string {
	return keystore.ResolveKeyPath(host, h.KeyStore, h.Settings)
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

	client, err := sshutil.Dial(hostCfg.Address, hostCfg.User, h.resolveKey(hostCfg))
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

	client, err := sshutil.Dial(hostCfg.Address, hostCfg.User, h.resolveKey(hostCfg))
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

type renameSessionReq struct {
	NewName string `json:"new_name"`
}

func (h *Handlers) RenameSession(w http.ResponseWriter, r *http.Request) {
	hostName := r.PathValue("host")
	sessionName := r.PathValue("session")

	var req renameSessionReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if req.NewName == "" {
		http.Error(w, "new_name is required", http.StatusBadRequest)
		return
	}

	hostCfg, ok := h.Hub.GetHostConfig(hostName)
	if !ok {
		http.Error(w, "host not found", http.StatusNotFound)
		return
	}

	client, err := sshutil.Dial(hostCfg.Address, hostCfg.User, h.resolveKey(hostCfg))
	if err != nil {
		http.Error(w, fmt.Sprintf("ssh connect failed: %v", err), http.StatusBadGateway)
		return
	}
	defer client.Close()

	if err := h.Tmux.RenameSession(client, sessionName, req.NewName); err != nil {
		http.Error(w, fmt.Sprintf("rename failed: %v", err), http.StatusInternalServerError)
		return
	}

	writeJSON(w, map[string]string{"status": "renamed", "old_name": sessionName, "new_name": req.NewName})
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
	hostCfg, ok := h.Hub.GetHostConfig(hostName)
	if !ok {
		http.Error(w, "host not found", http.StatusNotFound)
		return
	}

	state, err := h.Hub.ScanHost(hostName, h.Tmux, h.resolveKey(hostCfg))
	if err != nil {
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
		state, err := h.Hub.ScanHost(host.Config.Name, h.Tmux, h.resolveKey(host.Config))
		if err != nil {
			log.Printf("scan %s: %v", host.Config.Name, err)
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
	KeyName string `json:"key_name,omitempty"`
}

// AddHost adds a new host at runtime and saves it to the config file.
func (h *Handlers) AddHost(w http.ResponseWriter, r *http.Request) {
	var req addHostReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if req.Address == "" {
		http.Error(w, "address is required", http.StatusBadRequest)
		return
	}

	// Default name to hostname portion of address
	if req.Name == "" {
		name := req.Address
		if idx := strings.Index(name, ":"); idx != -1 {
			name = name[:idx]
		}
		req.Name = name
	}

	// Use default username if not provided
	user := req.User
	if user == "" {
		user = h.Settings.DefaultUsername()
		if user == "" {
			http.Error(w, "user is required (no default username set)", http.StatusBadRequest)
			return
		}
	}

	address := req.Address
	if !strings.Contains(address, ":") {
		address = address + ":22"
	}

	host := config.Host{
		Name:    req.Name,
		Address: address,
		User:    user,
		KeyName: req.KeyName,
	}

	if !h.Hub.AddHost(host) {
		http.Error(w, fmt.Sprintf("host %q already exists", req.Name), http.StatusConflict)
		return
	}

	if err := config.AppendHost(h.ConfigPath, config.Host{
		Name:    req.Name,
		Address: req.Address,
		User:    user,
		KeyName: req.KeyName,
	}); err != nil {
		log.Printf("warning: host added at runtime but config save failed: %v", err)
	}

	resolveKey := func(hc config.Host) string {
		return keystore.ResolveKeyPath(hc, h.KeyStore, h.Settings)
	}
	tmux.StartPoller(host, h.PollInterval, resolveKey, h.PollResults, h.Done)
	log.Printf("started poller for new host %s (%s@%s)", host.Name, host.User, host.Address)

	w.WriteHeader(http.StatusCreated)
	writeJSON(w, map[string]string{"status": "added", "name": req.Name})
}

// PubKey returns the default keypair's public key.
func (h *Handlers) PubKey(w http.ResponseWriter, r *http.Request) {
	name := h.Settings.DefaultKeypairName()
	pubKey, err := h.KeyStore.PublicKey(name)
	if err != nil {
		http.Error(w, "public key not available", http.StatusInternalServerError)
		return
	}
	writeJSON(w, map[string]string{"public_key": strings.TrimSpace(pubKey), "keypair_name": name})
}

// ── Keypair management ──

func (h *Handlers) ListKeypairs(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, h.KeyStore.List())
}

func (h *Handlers) GetKeypair(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	meta, err := h.KeyStore.Get(name)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	pubKey, _ := h.KeyStore.PublicKey(name)

	writeJSON(w, map[string]any{
		"meta":       meta,
		"public_key": strings.TrimSpace(pubKey),
	})
}

type createKeypairReq struct {
	Name string `json:"name"`
}

func (h *Handlers) CreateKeypair(w http.ResponseWriter, r *http.Request) {
	var req createKeypairReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if req.Name == "" {
		http.Error(w, "name is required", http.StatusBadRequest)
		return
	}

	meta, err := h.KeyStore.Generate(req.Name)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	pubKey, _ := h.KeyStore.PublicKey(req.Name)
	w.WriteHeader(http.StatusCreated)
	writeJSON(w, map[string]any{
		"meta":       meta,
		"public_key": strings.TrimSpace(pubKey),
	})
}

type importKeypairReq struct {
	Name       string `json:"name"`
	PrivateKey string `json:"private_key,omitempty"`
	ServerPath string `json:"server_path,omitempty"`
}

func (h *Handlers) ImportKeypair(w http.ResponseWriter, r *http.Request) {
	var req importKeypairReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if req.Name == "" {
		http.Error(w, "name is required", http.StatusBadRequest)
		return
	}

	var meta *keystore.KeypairMeta
	var err error

	if req.ServerPath != "" {
		meta, err = h.KeyStore.ImportFromPath(req.Name, req.ServerPath)
	} else if req.PrivateKey != "" {
		meta, err = h.KeyStore.Import(req.Name, []byte(req.PrivateKey))
	} else {
		http.Error(w, "either private_key or server_path is required", http.StatusBadRequest)
		return
	}

	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	pubKey, _ := h.KeyStore.PublicKey(req.Name)
	w.WriteHeader(http.StatusCreated)
	writeJSON(w, map[string]any{
		"meta":       meta,
		"public_key": strings.TrimSpace(pubKey),
	})
}

func (h *Handlers) DeleteKeypair(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")

	if name == h.Settings.DefaultKeypairName() {
		http.Error(w, "cannot delete the default keypair", http.StatusForbidden)
		return
	}

	if err := h.KeyStore.Delete(name); err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	writeJSON(w, map[string]string{"status": "deleted"})
}

// ── Settings ──

func (h *Handlers) GetSettings(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, h.Settings.Get())
}

func (h *Handlers) UpdateSettings(w http.ResponseWriter, r *http.Request) {
	var s keystore.Settings
	if err := json.NewDecoder(r.Body).Decode(&s); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if err := h.Settings.Update(s, h.KeyStore); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Update the global default key path
	sshutil.DefaultKeyPath = h.KeyStore.PrivateKeyPath(h.Settings.DefaultKeypairName())

	writeJSON(w, h.Settings.Get())
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}
