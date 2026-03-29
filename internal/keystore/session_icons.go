package keystore

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

type SessionIcon struct {
	Icon    string `json:"icon,omitempty"`
	Color   string `json:"color,omitempty"`
	Starred bool   `json:"starred,omitempty"`
	Theme   string `json:"theme,omitempty"`
}

// SessionIconStore persists session icon/color overrides to a JSON file.
type SessionIconStore struct {
	mu    sync.RWMutex
	path  string
	icons map[string]SessionIcon // key: "host:session"
}

func NewSessionIconStore(dataDir string) (*SessionIconStore, error) {
	s := &SessionIconStore{
		path:  filepath.Join(dataDir, "session-icons.json"),
		icons: make(map[string]SessionIcon),
	}
	if err := s.load(); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *SessionIconStore) load() error {
	data, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("read session icons: %w", err)
	}
	return json.Unmarshal(data, &s.icons)
}

func (s *SessionIconStore) save() error {
	data, err := json.MarshalIndent(s.icons, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, data, 0600)
}

func sessionKey(host, session string) string {
	return host + ":" + session
}

func (s *SessionIconStore) Get(host, session string) SessionIcon {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.icons[sessionKey(host, session)]
}

func (s *SessionIconStore) GetAll() map[string]SessionIcon {
	s.mu.RLock()
	defer s.mu.RUnlock()
	cp := make(map[string]SessionIcon, len(s.icons))
	for k, v := range s.icons {
		cp[k] = v
	}
	return cp
}

func (s *SessionIconStore) Set(host, session string, icon SessionIcon) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := sessionKey(host, session)
	if icon.Icon == "" && icon.Color == "" && icon.Theme == "" && !icon.Starred {
		delete(s.icons, key)
	} else {
		s.icons[key] = icon
	}
	return s.save()
}
