package keystore

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type Settings struct {
	DefaultKeypair  string `json:"default_keypair"`
	DefaultUsername string `json:"default_username"`
	TmuxWindowSize  string `json:"tmux_window_size"`
	ShowPubKey      *bool  `json:"show_pub_key,omitempty"`
	TabTitleFormat  string `json:"tab_title_format,omitempty"`
	EnableMCP       bool   `json:"enable_mcp,omitempty"`
	// DefaultHost is the host used by the one-off command exec API when a
	// request omits "host". Empty means "no default" — exec then falls back
	// to the sole host when exactly one is configured.
	DefaultHost string `json:"default_host,omitempty"`
	// ScrollbackLines is the tmux history-limit set on sessions ssh-to-go
	// creates AND the browser emulator's scrollback target. 0 means "use
	// DefaultScrollbackLines". It only affects newly-created tmux panes —
	// tmux can't grow an existing pane's history buffer.
	ScrollbackLines int `json:"scrollback_lines,omitempty"`
	// SessionIconMode controls the icon/color assigned to a session when it
	// is created: "random" (the default when empty) picks a random icon+color
	// from the built-in palette; "fixed" assigns SessionIconName/Color to
	// every new session. SessionIconName/Color are only consulted in "fixed"
	// mode and fall back to terminal/default when empty.
	SessionIconMode  string `json:"session_icon_mode,omitempty"`
	SessionIconName  string `json:"session_icon_name,omitempty"`
	SessionIconColor string `json:"session_icon_color,omitempty"`
	// NewSessionDir prefills the working-directory field of the New Session
	// form. Empty means DefaultNewSessionDir. It is a UI default only — the
	// API still treats an omitted cwd as "the SSH user's home", so MCP and
	// direct API callers are unaffected.
	NewSessionDir string `json:"new_session_dir,omitempty"`
	// IdleOffloadHours puts long-idle sessions to sleep: the tmux session is
	// offloaded (killed, registry entry kept, resumable with its working
	// directory and launch command) once it has had no client and nothing
	// running for this many hours. 0 — the default — is off.
	IdleOffloadHours int `json:"idle_offload_hours"`
	// NativeMouseMode enables tmux's per-session mouse option on sessions
	// ssh-to-go creates, so the wheel scrolls (via tmux copy-mode) when
	// attaching from a native terminal instead of arriving as arrow keys.
	// Unset means on. The trade-off: with mouse on, click-drag selection in a
	// native terminal goes through tmux copy-mode rather than the terminal's
	// own selection (hold Shift to bypass). The browser terminal is unaffected
	// either way — it scrolls its own scrollback and never forwards the wheel.
	NativeMouseMode *bool `json:"native_mouse_mode,omitempty"`
}

// DefaultScrollbackLines is the scrollback depth used when the setting is
// unset. Large enough for deep history; bounded by validation in Update.
const DefaultScrollbackLines = 50000

// DefaultNewSessionDir prefills the New Session working-directory field.
// The trailing slash is deliberate: the field is meant to be clicked into
// and typed at, so the common case is appending a project name.
const DefaultNewSessionDir = "~/sessions/"

type SettingsManager struct {
	mu       sync.RWMutex
	path     string
	settings Settings
}

func NewSettingsManager(dataDir string) (*SettingsManager, error) {
	sm := &SettingsManager{
		path: filepath.Join(dataDir, "settings.json"),
	}
	if err := sm.load(); err != nil {
		return nil, err
	}
	return sm, nil
}

func (sm *SettingsManager) load() error {
	data, err := os.ReadFile(sm.path)
	if err != nil {
		if os.IsNotExist(err) {
			sm.settings = Settings{
				DefaultKeypair:  "server",
				DefaultUsername: "",
			}
			return nil
		}
		return fmt.Errorf("read settings: %w", err)
	}
	return json.Unmarshal(data, &sm.settings)
}

func (sm *SettingsManager) save() error {
	data, err := json.MarshalIndent(sm.settings, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(sm.path, data, 0600)
}

func (sm *SettingsManager) Get() Settings {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.settings
}

func (sm *SettingsManager) Update(s Settings, ks *Store) error {
	if s.DefaultKeypair != "" && !ks.Exists(s.DefaultKeypair) {
		return fmt.Errorf("keypair %q does not exist", s.DefaultKeypair)
	}

	sm.mu.Lock()
	defer sm.mu.Unlock()

	if s.DefaultKeypair != "" {
		sm.settings.DefaultKeypair = s.DefaultKeypair
	}
	sm.settings.DefaultUsername = s.DefaultUsername
	sm.settings.DefaultHost = s.DefaultHost
	if s.TmuxWindowSize != "" {
		switch s.TmuxWindowSize {
		case "largest", "smallest", "latest":
			sm.settings.TmuxWindowSize = s.TmuxWindowSize
		default:
			return fmt.Errorf("invalid tmux_window_size %q: must be largest, smallest, or latest", s.TmuxWindowSize)
		}
	}
	if s.ShowPubKey != nil {
		sm.settings.ShowPubKey = s.ShowPubKey
	}
	if s.TabTitleFormat != "" {
		switch s.TabTitleFormat {
		case "host-session", "host-only", "session-only", "session-host":
			sm.settings.TabTitleFormat = s.TabTitleFormat
		default:
			return fmt.Errorf("invalid tab_title_format %q", s.TabTitleFormat)
		}
	}
	if s.ScrollbackLines != 0 {
		if s.ScrollbackLines < 100 || s.ScrollbackLines > 1_000_000 {
			return fmt.Errorf("invalid scrollback_lines %d: must be between 100 and 1000000", s.ScrollbackLines)
		}
		sm.settings.ScrollbackLines = s.ScrollbackLines
	}
	if s.SessionIconMode != "" {
		switch s.SessionIconMode {
		case "random", "fixed":
			sm.settings.SessionIconMode = s.SessionIconMode
		default:
			return fmt.Errorf("invalid session_icon_mode %q: must be random or fixed", s.SessionIconMode)
		}
	}
	sm.settings.SessionIconName = s.SessionIconName
	sm.settings.SessionIconColor = s.SessionIconColor
	// Assigned unconditionally so clearing the field in the UI resets it to
	// DefaultNewSessionDir rather than pinning the old value forever.
	sm.settings.NewSessionDir = strings.TrimSpace(s.NewSessionDir)
	// Also unconditional: 0 is "off", which has to be reachable again once
	// auto-sleep has been turned on.
	if s.IdleOffloadHours < 0 || s.IdleOffloadHours > 24*365 {
		return fmt.Errorf("invalid idle_offload_hours %d: must be between 0 (off) and %d", s.IdleOffloadHours, 24*365)
	}
	sm.settings.IdleOffloadHours = s.IdleOffloadHours
	if s.NativeMouseMode != nil {
		sm.settings.NativeMouseMode = s.NativeMouseMode
	}

	return sm.save()
}

// NewSessionDir returns the working directory to prefill the New Session
// form with, or DefaultNewSessionDir when unset.
func (sm *SettingsManager) NewSessionDir() string {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	if sm.settings.NewSessionDir == "" {
		return DefaultNewSessionDir
	}
	return sm.settings.NewSessionDir
}

func (sm *SettingsManager) DefaultKeypairName() string {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	if sm.settings.DefaultKeypair == "" {
		return "server"
	}
	return sm.settings.DefaultKeypair
}

func (sm *SettingsManager) DefaultUsername() string {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.settings.DefaultUsername
}

func (sm *SettingsManager) TmuxWindowSize() string {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	if sm.settings.TmuxWindowSize == "" {
		return "largest"
	}
	return sm.settings.TmuxWindowSize
}

// ScrollbackLines returns the configured scrollback depth, or
// DefaultScrollbackLines when unset.
func (sm *SettingsManager) ScrollbackLines() int {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	if sm.settings.ScrollbackLines <= 0 {
		return DefaultScrollbackLines
	}
	return sm.settings.ScrollbackLines
}

// NativeMouseMode reports whether tmux mouse mode should be enabled on
// sessions ssh-to-go creates. Defaults to true when unset.
func (sm *SettingsManager) NativeMouseMode() bool {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	if sm.settings.NativeMouseMode == nil {
		return true
	}
	return *sm.settings.NativeMouseMode
}

// IdleOffloadTimeout returns how long a session may sit idle before the
// sweeper offloads it, or 0 when the feature is off.
func (sm *SettingsManager) IdleOffloadTimeout() time.Duration {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	if sm.settings.IdleOffloadHours <= 0 {
		return 0
	}
	return time.Duration(sm.settings.IdleOffloadHours) * time.Hour
}

// DefaultHost returns the configured default host for the exec API, or "".
func (sm *SettingsManager) DefaultHost() string {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.settings.DefaultHost
}

// NewSessionIcon returns the icon/color to assign to a freshly created
// session. In "fixed" mode it returns the configured icon/color (falling
// back to terminal/default when unset); otherwise — the default — it
// returns a random palette entry so each new session looks distinct.
func (sm *SettingsManager) NewSessionIcon() SessionIcon {
	sm.mu.RLock()
	mode := sm.settings.SessionIconMode
	name := sm.settings.SessionIconName
	color := sm.settings.SessionIconColor
	sm.mu.RUnlock()

	if mode == "fixed" {
		if name == "" {
			name = "terminal"
		}
		if color == "" {
			color = "default"
		}
		return SessionIcon{Icon: name, Color: color}
	}
	return RandomSessionIcon()
}

func (sm *SettingsManager) MCPEnabled() bool {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.settings.EnableMCP
}

func (sm *SettingsManager) SetMCPEnabled(enabled bool) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.settings.EnableMCP = enabled
	return sm.save()
}
