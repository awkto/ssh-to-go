package keystore

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

type Settings struct {
	DefaultKeypair  string `json:"default_keypair"`
	DefaultUsername string `json:"default_username"`
	TmuxWindowSize  string `json:"tmux_window_size"`
	ShowPubKey      *bool  `json:"show_pub_key,omitempty"`
}

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

	return sm.save()
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
