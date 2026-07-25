package keystore

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// recentCommandLimit caps how many distinct commands are remembered. The New
// Session modal only shows the first few as chips; the rest exist so the
// Settings list is worth opening, and so a command you use monthly is still
// there when you come back to it.
const recentCommandLimit = 20

// RecentCommand is one command that was actually launched into a session,
// with enough context for the UI to rank and label it.
type RecentCommand struct {
	Command  string `json:"command"`
	LastUsed string `json:"last_used"`
	Count    int    `json:"count"`
}

// RecentCommandStore remembers the commands sessions were started with.
//
// This lives on the server rather than in localStorage on purpose: a command
// is a property of the deployment, not of the browser that happened to type
// it. Sessions created from another device, from the HTTP API, or by an agent
// over MCP all feed the same list, and the list survives clearing site data.
type RecentCommandStore struct {
	mu   sync.RWMutex
	path string
	cmds []RecentCommand // most-recently-used first
}

func NewRecentCommandStore(dataDir string) (*RecentCommandStore, error) {
	s := &RecentCommandStore{path: filepath.Join(dataDir, "recent-commands.json")}
	if err := s.load(); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *RecentCommandStore) load() error {
	data, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("read recent commands: %w", err)
	}
	if err := json.Unmarshal(data, &s.cmds); err != nil {
		return fmt.Errorf("parse recent commands: %w", err)
	}
	// Trust the file for content but not for order: sorting on load means a
	// hand-edited or older-format file still comes back MRU-first.
	s.sortLocked()
	return nil
}

func (s *RecentCommandStore) save() error {
	list := s.cmds
	if list == nil {
		list = []RecentCommand{} // an emptied list is [], not null
	}
	data, err := json.MarshalIndent(list, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, data, 0600)
}

func (s *RecentCommandStore) sortLocked() {
	sort.SliceStable(s.cmds, func(i, j int) bool {
		return s.cmds[i].LastUsed > s.cmds[j].LastUsed
	})
}

// Record moves a command to the front of the list, bumping its use count if
// it was already known. Blank commands are ignored, so callers can hand it
// whatever the "start in shell" path produced without checking first.
func (s *RecentCommandStore) Record(command string) error {
	command = strings.TrimSpace(command)
	if command == "" {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UTC().Format(time.RFC3339)
	for i, c := range s.cmds {
		if c.Command == command {
			c.Count++
			c.LastUsed = now
			s.cmds = append(s.cmds[:i], s.cmds[i+1:]...)
			s.cmds = append([]RecentCommand{c}, s.cmds...)
			return s.save()
		}
	}
	s.cmds = append([]RecentCommand{{Command: command, LastUsed: now, Count: 1}}, s.cmds...)
	if len(s.cmds) > recentCommandLimit {
		s.cmds = s.cmds[:recentCommandLimit]
	}
	return s.save()
}

// List returns the remembered commands, most recently used first.
func (s *RecentCommandStore) List() []RecentCommand {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]RecentCommand, len(s.cmds))
	copy(out, s.cmds)
	return out
}

// Delete forgets a single command. Reports whether it was there, so the API
// can 404 a stale chip instead of silently succeeding.
func (s *RecentCommandStore) Delete(command string) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, c := range s.cmds {
		if c.Command == command {
			s.cmds = append(s.cmds[:i], s.cmds[i+1:]...)
			return true, s.save()
		}
	}
	return false, nil
}

// Clear forgets every command.
func (s *RecentCommandStore) Clear() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cmds = nil
	return s.save()
}
