package keystore

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type SessionIcon struct {
	Icon         string `json:"icon,omitempty"`
	Color        string `json:"color,omitempty"`
	Starred      bool   `json:"starred,omitempty"`
	Theme        string `json:"theme,omitempty"`
	LastAccessed string `json:"last_accessed,omitempty"`
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
	if icon.Icon == "" && icon.Color == "" && icon.Theme == "" && icon.LastAccessed == "" && !icon.Starred {
		delete(s.icons, key)
	} else {
		s.icons[key] = icon
	}
	return s.save()
}

// Rename migrates icon data from one session name to another.
func (s *SessionIconStore) Rename(host, oldSession, newSession string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	oldKey := sessionKey(host, oldSession)
	entry, ok := s.icons[oldKey]
	if !ok {
		return nil
	}
	delete(s.icons, oldKey)
	s.icons[sessionKey(host, newSession)] = entry
	return s.save()
}

// Touch updates the last-accessed timestamp for a session.
func (s *SessionIconStore) Touch(host, session string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := sessionKey(host, session)
	entry := s.icons[key]
	entry.LastAccessed = time.Now().UTC().Format(time.RFC3339)
	s.icons[key] = entry
	return s.save()
}
