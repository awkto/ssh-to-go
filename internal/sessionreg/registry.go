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
	Host       string `json:"host"`
	Name       string `json:"name"`
	WorkingDir string `json:"working_dir,omitempty"`
	// Command is what the session was launched with (e.g. "claude"), so
	// Recreate can replay it and Duplicate can copy it. Entries written
	// before this field existed simply have none and recreate as a bare
	// shell, exactly as they did then.
	Command    string    `json:"command,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
	LastSeenAt time.Time `json:"last_seen_at,omitempty"`
	// Throwaway sessions are collected once nothing is attached — see
	// Flags. LastAttachedAt drives that idle clock: it is the last moment
	// a client was observed on the session, or the creation time if one
	// never has been.
	Throwaway      bool      `json:"throwaway,omitempty"`
	LastAttachedAt time.Time `json:"last_attached_at,omitempty"`
	// Incognito hides the session from every UI surface. The entry still
	// exists — hiding something requires knowing it is there, and the tmux
	// session outlives this process, so the flag has to be on disk.
	Incognito bool `json:"incognito,omitempty"`
	// AutoOffloaded records that the idle sweeper put this session to sleep
	// rather than a person offloading it by hand. Without the distinction a
	// session moving to Resumable overnight looks like a bug. Cleared the
	// moment the entry is re-added (i.e. recreated).
	AutoOffloaded bool `json:"auto_offloaded,omitempty"`
}

// Flags are the session flavours chosen at creation. Zero value is an
// ordinary session, so existing callers are unaffected.
type Flags struct {
	Throwaway bool
	Incognito bool
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

// Attrs is everything recorded about a session at creation time.
type Attrs struct {
	WorkingDir string
	// Command the session was launched with. Empty on an existing entry
	// leaves the recorded one alone — Add()'s callers (rename, poller cwd
	// refresh) don't know it and must not erase it.
	Command string
	Flags   Flags
}

// Add records a newly-created session. If one already exists with the same
// (host, name), its WorkingDir is updated and CreatedAt is left untouched.
func (s *Store) Add(host, name, workingDir string) error {
	return s.AddSession(host, name, Attrs{WorkingDir: workingDir})
}

// AddWithFlags is Add with the session flavours set. Flags only apply to a
// NEW entry: re-adding an existing session (rename, recreate) keeps whatever
// it was created as, so a throwaway can't quietly become permanent.
func (s *Store) AddWithFlags(host, name, workingDir string, flags Flags) error {
	return s.AddSession(host, name, Attrs{WorkingDir: workingDir, Flags: flags})
}

// AddSession is Add with every recorded attribute. See Attrs.
func (s *Store) AddSession(host, name string, a Attrs) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	k := key(host, name)
	now := time.Now().UTC()
	if existing, ok := s.entries[k]; ok {
		if a.WorkingDir != "" {
			existing.WorkingDir = a.WorkingDir
		}
		if a.Command != "" {
			existing.Command = a.Command
		}
		// The session is alive again, so it is no longer asleep. Leaving
		// this set would badge a recreated session as auto-offloaded for
		// the rest of its life.
		existing.AutoOffloaded = false
		existing.LastSeenAt = now
		s.entries[k] = existing
	} else {
		s.entries[k] = Entry{
			Host:       host,
			Name:       name,
			WorkingDir: a.WorkingDir,
			Command:    a.Command,
			CreatedAt:  now,
			LastSeenAt: now,
			Throwaway:  a.Flags.Throwaway,
			Incognito:  a.Flags.Incognito,
			// Never attached yet — the idle clock starts now, so a session
			// created and forgotten is still collected.
			LastAttachedAt: now,
		}
	}
	return s.saveLocked()
}

// ErrNameTaken is returned by Rename when the destination name is already
// tracked on the host. Overwriting would silently destroy the other entry's
// working directory and launch command — the very things it exists to hold.
var ErrNameTaken = fmt.Errorf("a session with that name is already tracked")

// Rename moves an entry to a new name on the same host, keeping everything
// recorded about it: creation time, launch command, flavours and idle clock.
// Renaming is not re-creating — a renamed throwaway is still the same
// throwaway, as old and as idle as it was a moment ago.
//
// Reports whether it moved anything; an untracked oldName is not an error, so
// callers can rename a hand-made tmux session without special-casing it.
func (s *Store) Rename(host, oldName, newName string) (bool, error) {
	if oldName == newName {
		return false, nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	oldKey, newKey := key(host, oldName), key(host, newName)
	e, ok := s.entries[oldKey]
	if !ok {
		return false, nil
	}
	if _, taken := s.entries[newKey]; taken {
		return false, ErrNameTaken
	}
	delete(s.entries, oldKey)
	e.Name = newName
	e.LastSeenAt = time.Now().UTC()
	s.entries[newKey] = e
	return true, s.saveLocked()
}

// MarkAutoOffloaded flags an entry as slept by the idle sweeper rather than
// offloaded by hand, so the UI can say so. No-op for untracked sessions.
func (s *Store) MarkAutoOffloaded(host, name string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	k := key(host, name)
	e, ok := s.entries[k]
	if !ok {
		return
	}
	e.AutoOffloaded = true
	s.entries[k] = e
	_ = s.saveLocked()
}

// List returns a copy of every entry, for the idle-offload sweeper.
func (s *Store) List() []Entry {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]Entry, 0, len(s.entries))
	for _, e := range s.entries {
		out = append(out, e)
	}
	return out
}

// MarkAttached records that a client is (or just was) on the session,
// resetting the throwaway idle clock. Cheap no-op for untracked sessions.
func (s *Store) MarkAttached(host, name string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	k := key(host, name)
	e, ok := s.entries[k]
	if !ok {
		return
	}
	now := time.Now().UTC()
	// Persist at most once a minute: this fires on every poll cycle for
	// every attached session, and the idle threshold is minutes wide.
	shouldSave := now.Sub(e.LastAttachedAt) > time.Minute
	e.LastAttachedAt = now
	e.LastSeenAt = now
	s.entries[k] = e
	if shouldSave {
		_ = s.saveLocked()
	}
}

// HiddenNames returns the incognito session names on a host, for the UI
// filter. Returned as a set so callers can test membership directly.
func (s *Store) HiddenNames(host string) map[string]bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var hidden map[string]bool
	for _, e := range s.entries {
		if e.Host == host && e.Incognito {
			if hidden == nil {
				hidden = make(map[string]bool)
			}
			hidden[e.Name] = true
		}
	}
	return hidden
}

// Throwaways returns a copy of every throwaway entry, for the idle sweeper.
func (s *Store) Throwaways() []Entry {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []Entry
	for _, e := range s.entries {
		if e.Throwaway {
			out = append(out, e)
		}
	}
	return out
}

// Flavours returns the flags recorded for a session. Missing entries report
// the zero value (an ordinary session).
func (s *Store) Flavours(host, name string) Flags {
	s.mu.RLock()
	defer s.mu.RUnlock()
	e, ok := s.entries[key(host, name)]
	if !ok {
		return Flags{}
	}
	return Flags{Throwaway: e.Throwaway, Incognito: e.Incognito}
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
