// Package sessionreg persists a record of tmux sessions created through
// ssh-to-go. When a remote host reboots, its tmux server forgets every
// session — but the registry remembers the names (and last-known working
// directories) we'd intended to keep around, so the UI can offer a
// "recreate" action.
package sessionreg

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"
)

// Entry is one tracked session.
type Entry struct {
	Host       string    `json:"host"`
	Name       string    `json:"name"`
	WorkingDir string    `json:"working_dir,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
	LastSeenAt time.Time `json:"last_seen_at,omitempty"`
}

// Store is a thread-safe JSON-backed map of registered sessions.
type Store struct {
	mu      sync.RWMutex
	path    string
	entries map[string]Entry // key: host + "\x00" + name
}

func key(host, name string) string { return host + "\x00" + name }

// NewStore opens (or creates) a registry at <dataDir>/session-registry.json.
func NewStore(dataDir string) (*Store, error) {
	s := &Store{
		path:    filepath.Join(dataDir, "session-registry.json"),
		entries: make(map[string]Entry),
	}
	if err := s.load(); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *Store) load() error {
	data, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("read session registry: %w", err)
	}
	var list []Entry
	if err := json.Unmarshal(data, &list); err != nil {
		return fmt.Errorf("parse session registry: %w", err)
	}
	for _, e := range list {
		s.entries[key(e.Host, e.Name)] = e
	}
	return nil
}

// caller must hold s.mu (write)
func (s *Store) saveLocked() error {
	list := make([]Entry, 0, len(s.entries))
	for _, e := range s.entries {
		list = append(list, e)
	}
	sort.Slice(list, func(i, j int) bool {
		if list[i].Host != list[j].Host {
			return list[i].Host < list[j].Host
		}
		return list[i].Name < list[j].Name
	})
	data, err := json.MarshalIndent(list, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, data, 0600)
}

// Add records a newly-created session. If one already exists with the same
// (host, name), its WorkingDir is updated and CreatedAt is left untouched.
func (s *Store) Add(host, name, workingDir string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	k := key(host, name)
	now := time.Now().UTC()
	if existing, ok := s.entries[k]; ok {
		if workingDir != "" {
			existing.WorkingDir = workingDir
		}
		existing.LastSeenAt = now
		s.entries[k] = existing
	} else {
		s.entries[k] = Entry{
			Host:       host,
			Name:       name,
			WorkingDir: workingDir,
			CreatedAt:  now,
			LastSeenAt: now,
		}
	}
	return s.saveLocked()
}

// Remove drops an entry. Missing entries are not an error.
func (s *Store) Remove(host, name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	k := key(host, name)
	if _, ok := s.entries[k]; !ok {
		return nil
	}
	delete(s.entries, k)
	return s.saveLocked()
}

// RemoveHost drops every entry for a host. Used when a host itself is
// removed from the configuration.
func (s *Store) RemoveHost(host string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	changed := false
	for k, e := range s.entries {
		if e.Host == host {
			delete(s.entries, k)
			changed = true
		}
	}
	if !changed {
		return nil
	}
	return s.saveLocked()
}

// Get returns the entry and true if present.
func (s *Store) Get(host, name string) (Entry, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	e, ok := s.entries[key(host, name)]
	return e, ok
}

// ListByHost returns every entry for a host, sorted by name.
func (s *Store) ListByHost(host string) []Entry {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []Entry
	for _, e := range s.entries {
		if e.Host == host {
			out = append(out, e)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

// TrackedNames returns just the session names for a host. Implements the
// poller's Tracker interface.
func (s *Store) TrackedNames(host string) []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []string
	for _, e := range s.entries {
		if e.Host == host {
			out = append(out, e.Name)
		}
	}
	return out
}

// SetCwd records the most recently observed working directory of a
// tracked session. No-op if the entry isn't tracked.
func (s *Store) SetCwd(host, name, cwd string) {
	if cwd == "" {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	k := key(host, name)
	e, ok := s.entries[k]
	if !ok {
		return
	}
	if e.WorkingDir == cwd {
		// Avoid disk write storms when cwd is stable.
		e.LastSeenAt = time.Now().UTC()
		s.entries[k] = e
		return
	}
	e.WorkingDir = cwd
	e.LastSeenAt = time.Now().UTC()
	s.entries[k] = e
	if err := s.saveLocked(); err != nil {
		// Save failures here are non-fatal — we'll retry on next poll.
		return
	}
}

// MarkSeen updates LastSeenAt without changing cwd. Used to record that
// a tracked session is currently alive in tmux.
func (s *Store) MarkSeen(host, name string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	k := key(host, name)
	e, ok := s.entries[k]
	if !ok {
		return
	}
	e.LastSeenAt = time.Now().UTC()
	s.entries[k] = e
}
