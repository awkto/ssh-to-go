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
	"github.com/awkto/ssh-to-go/internal/execjob"
	"github.com/awkto/ssh-to-go/internal/hub"
	"github.com/awkto/ssh-to-go/internal/keystore"
	"github.com/awkto/ssh-to-go/internal/sessionreg"
	"github.com/awkto/ssh-to-go/internal/sshutil"
	"github.com/awkto/ssh-to-go/internal/tmux"
)

type Handlers struct {
	Hub          *hub.Hub
	Tmux         *tmux.Manager
	KeyStore     *keystore.Store
	Settings     *keystore.SettingsManager
	SessionIcons *keystore.SessionIconStore
	Registry     *sessionreg.Store
	Auth         *auth.Manager
	ExecJobs     *execjob.Store
	ConfigPath   string
	PollInterval time.Duration
	PollResults  chan<- tmux.PollResult
	Done         <-chan struct{}
	Version      string
}

func (h *Handlers) GetVersion(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, map[string]string{"version": h.Version})
}

// Me returns whether the caller is authenticated and the server version.
// Used by native clients (Android) to verify a stored bearer token at
// server-add time. Middleware has already gated unauthenticated requests
// unless NoAuth is on; either way reaching this handler means "OK".
func (h *Handlers) Me(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, map[string]any{
		"authenticated": true,
		"no_auth":       h.Auth != nil && h.Auth.NoAuth(),
		"version":       h.Version,
	})
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

// hostResponse mirrors hub.HostState and adds missing-session data
// computed from the registry. The embedded HostState preserves the
// existing wire format so older clients keep working.
type hostResponse struct {
	hub.HostState
	MissingSessions []sessionreg.Entry `json:"missing_sessions,omitempty"`
}

// missingFor returns app-tracked sessions that the host's last poll
// didn't see in tmux. Only returns entries when the host is online —
// otherwise we don't know what's actually missing vs unreachable.
func (h *Handlers) missingFor(host hub.HostState) []sessionreg.Entry {
	if h.Registry == nil || !host.Online {
		return nil
	}
	tracked := h.Registry.ListByHost(host.Config.Name)
	if len(tracked) == 0 {
		return nil
	}
	// Compare by sanitized name. Sessions created/recreated through ssh-to-go
	// are stored hyphenated (sanitizeSessionName), but a legacy or offloaded
	// entry may still carry the original spaced name. Without normalizing, a
	// live "Foo-Bar" and a tracked "Foo Bar" look distinct, so the same
	// session shows up twice — once live, once as a phantom "missing" entry.
	alive := make(map[string]struct{}, len(host.Sessions))
	for _, s := range host.Sessions {
		alive[sanitizeSessionName(s.Name)] = struct{}{}
	}
	var missing []sessionreg.Entry
	seen := make(map[string]struct{}, len(tracked))
	for _, e := range tracked {
		// Incognito sessions are filtered out of host.Sessions, so without
		// this they'd resurface here as phantom "resumable" entries — the
		// one place a hidden session would leak back into the UI.
		if e.Incognito {
			continue
		}
		norm := sanitizeSessionName(e.Name)
		if _, ok := alive[norm]; ok {
			continue
		}
		// Collapse multiple tracked entries that normalize to the same name
		// (e.g. a spaced legacy entry plus its sanitized recreation).
		if _, dup := seen[norm]; dup {
			continue
		}
		seen[norm] = struct{}{}
		missing = append(missing, e)
	}
	return missing
}

func (h *Handlers) ListHosts(w http.ResponseWriter, r *http.Request) {
	hosts := h.Hub.AllHosts()
	resp := make([]hostResponse, 0, len(hosts))
	for _, host := range hosts {
		resp = append(resp, hostResponse{HostState: host, MissingSessions: h.missingFor(host)})
	}
	writeJSON(w, resp)
}

type createSessionReq struct {
	Name string `json:"name"`
	Cwd  string `json:"cwd,omitempty"`
	// CreateDir makes Cwd first when it doesn't exist. Without it a missing
	// directory fails the create, which is the safer default for API callers.
	CreateDir bool `json:"create_dir,omitempty"`
	// Command is typed into the new session's shell once it starts (e.g.
	// "claude"). The shell outlives it, so exiting the command leaves a
	// prompt in Cwd rather than killing the session.
	Command string `json:"command,omitempty"`
	// Throwaway sessions are killed and purged once nothing is attached.
	Throwaway bool `json:"throwaway,omitempty"`
	// Incognito sessions run normally but never appear in the UI.
	Incognito bool `json:"incognito,omitempty"`
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
	rawName := req.Name
	req.Name = sanitizeSessionName(req.Name)
	if req.Name == "" {
		http.Error(w, "session name required", http.StatusBadRequest)
		return
	}

	// Pre-flight: refuse the request if a session with this name is already
	// alive on the host, OR if there's an offloaded entry under that name.
	// Without this, tmux returns "duplicate session" with a 500 (live case)
	// or — much worse — the offloaded entry gets silently overwritten with
	// a new working directory (resume context lost). The error message
	// surfaces both the typed name and the sanitized one when they differ
	// so the user understands where the collision came from.
	if exists, kind := h.sessionExists(hostName, req.Name); exists {
		typedHint := ""
		if rawName != req.Name {
			typedHint = fmt.Sprintf(" (you submitted %q which sanitizes to %q)", rawName, req.Name)
		}
		var msg string
		switch kind {
		case "live":
			msg = fmt.Sprintf("session %q is already running on %q%s", req.Name, hostName, typedHint)
		case "offloaded":
			msg = fmt.Sprintf("an offloaded session %q is already tracked on %q%s — use Recreate to bring it back, or Forget it first", req.Name, hostName, typedHint)
		}
		http.Error(w, msg, http.StatusConflict)
		return
	}

	client, err := sshutil.Dial(hostCfg.DialAddress(), hostCfg.User, h.resolveKey(hostCfg))
	if err != nil {
		http.Error(w, fmt.Sprintf("ssh connect failed: %v", err), http.StatusBadGateway)
		return
	}
	defer client.Close()

	if err := h.Tmux.CreateSessionWith(client, req.Name, tmux.CreateOptions{
		WindowSize:   h.Settings.TmuxWindowSize(),
		Cwd:          req.Cwd,
		HistoryLimit: h.Settings.ScrollbackLines(),
		CreateDir:    req.CreateDir,
		Command:      req.Command,
	}); err != nil {
		http.Error(w, fmt.Sprintf("create session failed: %v", err), http.StatusInternalServerError)
		return
	}

	if h.Registry != nil {
		// Incognito sessions are registered too. Hiding one requires knowing
		// it exists, and the tmux session outlives this process — the entry
		// is simply never surfaced (see Hub's hidden filter).
		flags := sessionreg.Flags{Throwaway: req.Throwaway, Incognito: req.Incognito}
		if err := h.Registry.AddWithFlags(hostName, req.Name, req.Cwd, flags); err != nil {
			log.Printf("session registry add %s/%s: %v", hostName, req.Name, err)
		}
		if req.Incognito {
			h.Hub.SetHidden(hostName, h.Registry.HiddenNames(hostName))
		}
	}

	// Assign the session's icon/color per the configured mode (random by
	// default, or a fixed icon/color). Non-fatal: a session without an
	// override just renders with the default terminal icon.
	if h.SessionIcons != nil {
		if err := h.SessionIcons.Set(hostName, req.Name, h.Settings.NewSessionIcon()); err != nil {
			log.Printf("session icon assign %s/%s: %v", hostName, req.Name, err)
		}
	}

	w.WriteHeader(http.StatusCreated)
	writeJSON(w, map[string]string{"status": "created", "name": req.Name})
}

// sessionExists reports whether a session with the given name is already
// known on the host — either as a live tmux session (kind="live") or as
// an offloaded entry in the registry (kind="offloaded"). The live check
// uses the hub's last polled snapshot rather than a fresh tmux ls so it's
// O(N) in-memory; stale state would just let the tmux create call catch
// the duplicate the slow way.
func (h *Handlers) sessionExists(hostName, name string) (bool, string) {
	if state, ok := h.Hub.GetHost(hostName); ok {
		for _, s := range state.Sessions {
			if s.Name == name {
				return true, "live"
			}
		}
	}
	if h.Registry != nil {
		if _, ok := h.Registry.Get(hostName, name); ok {
			return true, "offloaded"
		}
	}
	return false, ""
}

// sanitizeSessionName collapses any run of whitespace (spaces, tabs) in a
// requested session name into single dashes. tmux happily accepts spaces in
// session names, but the SSH handoff command joins arguments at two shell
// layers and any name with a space gets split into multiple tmux args
// ("too many arguments"). Replacing spaces with dashes sidesteps the
// quoting mess entirely. Other special characters are left alone.
func sanitizeSessionName(name string) string {
	name = strings.TrimSpace(name)
	// Collapse internal whitespace runs into a single dash.
	var b strings.Builder
	b.Grow(len(name))
	prevSpace := false
	for _, r := range name {
		if r == ' ' || r == '\t' {
			if !prevSpace {
				b.WriteByte('-')
				prevSpace = true
			}
			continue
		}
		b.WriteRune(r)
		prevSpace = false
	}
	return b.String()
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

	if h.Registry != nil {
		if err := h.Registry.Remove(hostName, sessionName); err != nil {
			log.Printf("session registry remove %s/%s: %v", hostName, sessionName, err)
		}
	}
}

// OffloadSession kills the tmux session like KillSession does, but keeps the
// registry entry intact. The session will then appear in the dashboard's
// "Resumable sessions" panel with its last-known working directory, ready to
// be recreated. Used to free up RAM on long-running idle sessions without
// losing the ability to come back to them later.
func (h *Handlers) OffloadSession(w http.ResponseWriter, r *http.Request) {
	hostName := r.PathValue("host")
	sessionName := r.PathValue("session")

	if h.Registry == nil {
		http.Error(w, "session registry unavailable", http.StatusServiceUnavailable)
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

	// Registry is keyed by the sanitized name (the form Recreate/Create use),
	// so an offloaded session round-trips to the same name instead of
	// spawning a hyphenated duplicate. Live tmux ops below still use the real
	// session name from the path.
	regName := sanitizeSessionName(sessionName)

	// If we're not yet tracking this session (it was created manually via
	// tmux), claim it: discover its current working directory and write a
	// registry entry. That way Offload works on any tmux session the user
	// can see in the dashboard, not just ones spawned via this app.
	if _, ok := h.Registry.Get(hostName, regName); !ok {
		cwd, cwdErr := h.Tmux.SessionCwd(client, sessionName)
		if cwdErr != nil {
			// Couldn't read cwd (session might not exist or have any panes).
			// Still proceed but with empty cwd — Recreate will land in $HOME.
			log.Printf("offload: get cwd %s/%s: %v", hostName, sessionName, cwdErr)
			cwd = ""
		}
		if err := h.Registry.Add(hostName, regName, cwd); err != nil {
			log.Printf("offload: registry add %s/%s: %v", hostName, regName, err)
		}
	}

	if err := h.Tmux.KillSession(client, sessionName); err != nil {
		http.Error(w, fmt.Sprintf("kill session failed: %v", err), http.StatusInternalServerError)
		return
	}

	// Intentionally leave the registry entry in place — that's the whole point.
	writeJSON(w, map[string]string{"status": "offloaded", "name": sessionName})
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
	rawNew := req.NewName
	req.NewName = sanitizeSessionName(req.NewName)
	if req.NewName == "" {
		http.Error(w, "new_name is required", http.StatusBadRequest)
		return
	}
	if req.NewName == sessionName {
		writeJSON(w, map[string]string{"status": "renamed", "old_name": sessionName, "new_name": req.NewName})
		return
	}
	if exists, kind := h.sessionExists(hostName, req.NewName); exists {
		typedHint := ""
		if rawNew != req.NewName {
			typedHint = fmt.Sprintf(" (you submitted %q which sanitizes to %q)", rawNew, req.NewName)
		}
		var msg string
		switch kind {
		case "live":
			msg = fmt.Sprintf("session %q is already running on %q%s", req.NewName, hostName, typedHint)
		case "offloaded":
			msg = fmt.Sprintf("an offloaded session %q is already tracked on %q%s — use Recreate to bring it back, or Forget it first", req.NewName, hostName, typedHint)
		}
		http.Error(w, msg, http.StatusConflict)
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

	// Move the registry entry along with the rename so a future reboot
	// recreates the session under its new name.
	if h.Registry != nil {
		if entry, ok := h.Registry.Get(hostName, sessionName); ok {
			_ = h.Registry.Remove(hostName, sessionName)
			// Carry the flavours across: renaming a throwaway must not
			// quietly promote it to a permanent session (nor un-hide an
			// incognito one).
			_ = h.Registry.AddWithFlags(hostName, req.NewName, entry.WorkingDir,
				sessionreg.Flags{Throwaway: entry.Throwaway, Incognito: entry.Incognito})
		}
	}

	writeJSON(w, map[string]string{"status": "renamed", "old_name": sessionName, "new_name": req.NewName})
}

func (h *Handlers) ListClients(w http.ResponseWriter, r *http.Request) {
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

	clients, err := h.Tmux.ListClients(client, sessionName)
	if err != nil {
		http.Error(w, fmt.Sprintf("list clients failed: %v", err), http.StatusInternalServerError)
		return
	}
	if clients == nil {
		clients = []tmux.Client{}
	}
	writeJSON(w, clients)
}

type detachClientsReq struct {
	ExcludeTTY string `json:"exclude_tty,omitempty"`
}

func (h *Handlers) DetachClients(w http.ResponseWriter, r *http.Request) {
	hostName := r.PathValue("host")
	sessionName := r.PathValue("session")

	var req detachClientsReq
	_ = json.NewDecoder(r.Body).Decode(&req) // body is optional

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

	detached, err := h.Tmux.DetachClients(client, sessionName, req.ExcludeTTY)
	if err != nil {
		http.Error(w, fmt.Sprintf("detach clients failed: %v", err), http.StatusInternalServerError)
		return
	}

	writeJSON(w, map[string]any{"status": "ok", "detached": detached})
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
	tmux.StartPoller(host, h.PollInterval, resolveKey, h.Registry, h.PollResults, h.Done)
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

	if h.Registry != nil {
		if err := h.Registry.RemoveHost(hostName); err != nil {
			log.Printf("session registry purge %s: %v", hostName, err)
		}
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

// ── Tracked sessions ──

// RecreateSession brings back a tracked session that no longer exists in
// tmux (typically lost to a host reboot). The session is created with
// the working directory we last observed for it. Refuses if the session
// is already alive or isn't tracked.
func (h *Handlers) RecreateSession(w http.ResponseWriter, r *http.Request) {
	hostName := r.PathValue("host")
	sessionName := r.PathValue("session")

	if h.Registry == nil {
		http.Error(w, "session registry unavailable", http.StatusServiceUnavailable)
		return
	}
	entry, ok := h.Registry.Get(hostName, sessionName)
	if !ok {
		http.Error(w, "session not tracked", http.StatusNotFound)
		return
	}

	hostCfg, ok := h.Hub.GetHostConfig(hostName)
	if !ok {
		http.Error(w, "host not found", http.StatusNotFound)
		return
	}

	// The session is recreated under its sanitized name (the form Create
	// uses), so a legacy spaced entry comes back as the canonical hyphenated
	// session rather than spawning a second one.
	createName := sanitizeSessionName(sessionName)

	// Refuse if it's already alive on the host — nothing to recreate. Compare
	// sanitized, since the live session may be hyphenated even when this
	// tracked entry still carries the original spaced name.
	if state, ok := h.Hub.GetHost(hostName); ok {
		for _, s := range state.Sessions {
			if sanitizeSessionName(s.Name) == createName {
				http.Error(w, "session already exists", http.StatusConflict)
				return
			}
		}
	}

	client, err := sshutil.Dial(hostCfg.DialAddress(), hostCfg.User, h.resolveKey(hostCfg))
	if err != nil {
		http.Error(w, fmt.Sprintf("ssh connect failed: %v", err), http.StatusBadGateway)
		return
	}
	defer client.Close()

	if err := h.Tmux.CreateSession(client, createName, h.Settings.TmuxWindowSize(), entry.WorkingDir, h.Settings.ScrollbackLines()); err != nil {
		http.Error(w, fmt.Sprintf("create session failed: %v", err), http.StatusInternalServerError)
		return
	}

	// Migrate a legacy spaced entry to its sanitized key so it stops
	// appearing separately, then refresh LastSeenAt. Add() preserves
	// CreatedAt and WorkingDir when the entry already exists.
	if createName != sessionName {
		_ = h.Registry.Remove(hostName, sessionName)
	}
	if err := h.Registry.Add(hostName, createName, entry.WorkingDir); err != nil {
		log.Printf("session registry touch %s/%s: %v", hostName, createName, err)
	}

	writeJSON(w, map[string]string{
		"status":      "recreated",
		"name":        createName,
		"working_dir": entry.WorkingDir,
	})
}

// ForgetSession drops a tracked session from the registry without
// touching tmux. Intended for "this session is gone and I don't want
// it back" — clears the entry so it stops appearing as missing.
func (h *Handlers) ForgetSession(w http.ResponseWriter, r *http.Request) {
	hostName := r.PathValue("host")
	sessionName := r.PathValue("session")

	if h.Registry == nil {
		http.Error(w, "session registry unavailable", http.StatusServiceUnavailable)
		return
	}
	if err := h.Registry.Remove(hostName, sessionName); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, map[string]string{"status": "forgotten"})
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
