package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/awkto/ssh-to-go/internal/auth"
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
	SessionIcons *keystore.SessionIconStore
	Auth         *auth.Manager
	ConfigPath   string
	PollInterval time.Duration
	PollResults  chan<- tmux.PollResult
	Done         <-chan struct{}
	Version      string
}

func (h *Handlers) GetVersion(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, map[string]string{"version": h.Version})
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
	Cwd  string `json:"cwd,omitempty"`
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

	client, err := sshutil.Dial(hostCfg.DialAddress(), hostCfg.User, h.resolveKey(hostCfg))
	if err != nil {
		http.Error(w, fmt.Sprintf("ssh connect failed: %v", err), http.StatusBadGateway)
		return
	}
	defer client.Close()

	if err := h.Tmux.CreateSession(client, req.Name, h.Settings.TmuxWindowSize(), req.Cwd); err != nil {
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

	client, err := sshutil.Dial(hostCfg.DialAddress(), hostCfg.User, h.resolveKey(hostCfg))
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

	client, err := sshutil.Dial(hostCfg.DialAddress(), hostCfg.User, h.resolveKey(hostCfg))
	if err != nil {
		http.Error(w, fmt.Sprintf("ssh connect failed: %v", err), http.StatusBadGateway)
		return
	}
	defer client.Close()

	if err := h.Tmux.RenameSession(client, sessionName, req.NewName); err != nil {
		http.Error(w, fmt.Sprintf("rename failed: %v", err), http.StatusInternalServerError)
		return
	}

	// Migrate session icon/color/star data to the new name
	_ = h.SessionIcons.Rename(hostName, sessionName, req.NewName)

	writeJSON(w, map[string]string{"status": "renamed", "old_name": sessionName, "new_name": req.NewName})
}

func (h *Handlers) SessionCwd(w http.ResponseWriter, r *http.Request) {
	hostName := r.PathValue("host")
	sessionName := r.PathValue("session")

	hostCfg, ok := h.Hub.GetHostConfig(hostName)
	if !ok {
		http.Error(w, "host not found", http.StatusNotFound)
		return
	}

	client, err := sshutil.Dial(hostCfg.DialAddress(), hostCfg.User, h.resolveKey(hostCfg))
	if err != nil {
		http.Error(w, fmt.Sprintf("ssh connect failed: %v", err), http.StatusBadGateway)
		return
	}
	defer client.Close()

	cwd, err := h.Tmux.SessionCwd(client, sessionName)
	if err != nil {
		http.Error(w, fmt.Sprintf("get cwd failed: %v", err), http.StatusInternalServerError)
		return
	}

	writeJSON(w, map[string]string{"cwd": cwd})
}

func (h *Handlers) Handoff(w http.ResponseWriter, r *http.Request) {
	hostName := r.PathValue("host")
	sessionName := r.PathValue("session")

	hostCfg, ok := h.Hub.GetHostConfig(hostName)
	if !ok {
		http.Error(w, "host not found", http.StatusNotFound)
		return
	}

	cmd := h.Tmux.HandoffCommand(hostCfg.User, hostCfg.Address, hostCfg.Port, sessionName)
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
	Name      string `json:"name"`
	Address   string `json:"address"`
	Port      int    `json:"port"`
	User      string `json:"user"`
	KeyName   string `json:"key_name,omitempty"`
	Icon      string `json:"icon,omitempty"`
	IconColor string `json:"icon_color,omitempty"`
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

	port := req.Port
	if port == 0 {
		port = 22
	}

	host := config.Host{
		Name:      req.Name,
		Address:   req.Address,
		Port:      port,
		User:      user,
		KeyName:   req.KeyName,
		Icon:      req.Icon,
		IconColor: req.IconColor,
	}

	if !h.Hub.AddHost(host) {
		http.Error(w, fmt.Sprintf("host %q already exists", req.Name), http.StatusConflict)
		return
	}

	if err := config.AppendHost(h.ConfigPath, host); err != nil {
		log.Printf("warning: host added at runtime but config save failed: %v", err)
	}

	resolveKey := func(hc config.Host) string {
		return keystore.ResolveKeyPath(hc, h.KeyStore, h.Settings)
	}
	tmux.StartPoller(host, h.PollInterval, resolveKey, h.PollResults, h.Done)
	log.Printf("started poller for new host %s (%s@%s)", host.Name, host.User, host.DialAddress())

	w.WriteHeader(http.StatusCreated)
	writeJSON(w, map[string]string{"status": "added", "name": req.Name})
}

type updateHostReq struct {
	Address   string `json:"address"`
	Port      int    `json:"port"`
	User      string `json:"user"`
	KeyName   string `json:"key_name"`
	OS        string `json:"os"`
	Icon      string `json:"icon"`
	IconColor string `json:"icon_color"`
}

// UpdateHost updates a host's config at runtime and in the config file.
func (h *Handlers) UpdateHost(w http.ResponseWriter, r *http.Request) {
	hostName := r.PathValue("host")

	hostCfg, ok := h.Hub.GetHostConfig(hostName)
	if !ok {
		http.Error(w, "host not found", http.StatusNotFound)
		return
	}

	var req updateHostReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.Address != "" {
		hostCfg.Address = req.Address
	}
	if req.Port > 0 {
		hostCfg.Port = req.Port
	}
	if req.User != "" {
		hostCfg.User = req.User
	}
	hostCfg.KeyName = req.KeyName
	hostCfg.OS = req.OS
	hostCfg.Icon = req.Icon
	hostCfg.IconColor = req.IconColor

	h.Hub.UpdateHost(hostCfg)

	if err := config.UpdateHost(h.ConfigPath, hostName, hostCfg); err != nil {
		log.Printf("warning: host updated at runtime but config save failed: %v", err)
	}

	writeJSON(w, map[string]string{"status": "updated"})
}

// DeleteHost removes a host from the hub and config file.
func (h *Handlers) DeleteHost(w http.ResponseWriter, r *http.Request) {
	hostName := r.PathValue("host")

	if !h.Hub.RemoveHost(hostName) {
		http.Error(w, "host not found", http.StatusNotFound)
		return
	}

	if err := config.RemoveHost(h.ConfigPath, hostName); err != nil {
		log.Printf("warning: host removed at runtime but config save failed: %v", err)
	}

	writeJSON(w, map[string]string{"status": "deleted"})
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

	// Check if any hosts explicitly reference this keypair
	var usingHosts []string
	for _, hs := range h.Hub.AllHosts() {
		if hs.Config.KeyName == name {
			usingHosts = append(usingHosts, hs.Config.Name)
		}
	}

	// If force=true query param, proceed anyway; otherwise warn
	if len(usingHosts) > 0 && r.URL.Query().Get("force") != "true" {
		writeJSON(w, map[string]any{
			"warning":     "keypair is in use by hosts",
			"hosts_using": usingHosts,
		})
		return
	}

	if err := h.KeyStore.Delete(name); err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	writeJSON(w, map[string]string{"status": "deleted"})
}

// RenameKeypair renames a keypair.
func (h *Handlers) RenameKeypair(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")

	var req struct {
		NewName string `json:"new_name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if req.NewName == "" {
		http.Error(w, "new_name is required", http.StatusBadRequest)
		return
	}

	if err := h.KeyStore.Rename(name, req.NewName); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Update any hosts referencing the old name
	for _, hs := range h.Hub.AllHosts() {
		if hs.Config.KeyName == name {
			updated := hs.Config
			updated.KeyName = req.NewName
			h.Hub.UpdateHost(updated)
			if err := config.UpdateHost(h.ConfigPath, hs.Config.Name, updated); err != nil {
				log.Printf("warning: host %s keypair ref update failed: %v", hs.Config.Name, err)
			}
		}
	}

	// Update default keypair if it was the renamed one
	if h.Settings.DefaultKeypairName() == name {
		s := h.Settings.Get()
		s.DefaultKeypair = req.NewName
		_ = h.Settings.Update(s, h.KeyStore)
	}

	writeJSON(w, map[string]string{"status": "renamed", "new_name": req.NewName})
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

// ── Auth ──

type setupPasswordReq struct {
	Password string `json:"password"`
}

func (h *Handlers) AuthSetup(w http.ResponseWriter, r *http.Request) {
	if h.Auth.HasPassword() {
		http.Error(w, "password already set", http.StatusConflict)
		return
	}

	var req setupPasswordReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if len(req.Password) < 4 {
		http.Error(w, "password must be at least 4 characters", http.StatusBadRequest)
		return
	}

	if err := h.Auth.SetPassword("", req.Password); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Auto-login after setup
	token, err := h.Auth.CreateSession()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	http.SetCookie(w, sessionCookie(token))
	writeJSON(w, map[string]string{"status": "ok"})
}

type loginReq struct {
	Password string `json:"password"`
}

func (h *Handlers) AuthLogin(w http.ResponseWriter, r *http.Request) {
	var req loginReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if !h.Auth.CheckPassword(req.Password) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid password"})
		return
	}

	token, err := h.Auth.CreateSession()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	http.SetCookie(w, sessionCookie(token))
	writeJSON(w, map[string]string{"status": "ok"})
}

func (h *Handlers) AuthLogout(w http.ResponseWriter, r *http.Request) {
	if cookie, err := r.Cookie("session"); err == nil {
		h.Auth.DeleteSession(cookie.Value)
	}
	http.SetCookie(w, &http.Cookie{
		Name:     "session",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
	writeJSON(w, map[string]string{"status": "ok"})
}

type changePasswordReq struct {
	CurrentPassword string `json:"current_password"`
	NewPassword     string `json:"new_password"`
}

func (h *Handlers) AuthChangePassword(w http.ResponseWriter, r *http.Request) {
	var req changePasswordReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if len(req.NewPassword) < 4 {
		http.Error(w, "password must be at least 4 characters", http.StatusBadRequest)
		return
	}

	if err := h.Auth.SetPassword(req.CurrentPassword, req.NewPassword); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	writeJSON(w, map[string]string{"status": "ok"})
}

type createTokenReq struct {
	Name string `json:"name"`
}

func (h *Handlers) AuthListTokens(w http.ResponseWriter, r *http.Request) {
	tokens := h.Auth.ListAPITokens()
	if tokens == nil {
		tokens = []auth.APIToken{}
	}
	writeJSON(w, tokens)
}

func (h *Handlers) AuthCreateToken(w http.ResponseWriter, r *http.Request) {
	var req createTokenReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	plain, err := h.Auth.CreateAPIToken(req.Name)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusCreated)
	writeJSON(w, map[string]string{"name": req.Name, "token": plain})
}

func (h *Handlers) AuthDeleteToken(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if err := h.Auth.DeleteAPIToken(name); err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	writeJSON(w, map[string]string{"status": "deleted"})
}

func sessionCookie(token string) *http.Cookie {
	return &http.Cookie{
		Name:     "session",
		Value:    token,
		Path:     "/",
		MaxAge:   7 * 24 * 60 * 60, // 7 days
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	}
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}

// ── Session Icons ──

func (h *Handlers) GetSessionIcons(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, h.SessionIcons.GetAll())
}

func (h *Handlers) SetSessionIcon(w http.ResponseWriter, r *http.Request) {
	host := r.PathValue("host")
	session := r.PathValue("session")

	var req keystore.SessionIcon
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if err := h.SessionIcons.Set(host, session, req); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, map[string]string{"status": "ok"})
}
