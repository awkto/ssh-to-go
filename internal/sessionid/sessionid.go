// Package sessionid assigns short numeric IDs to sessions, unique across
// every host the server manages. The assignment is made server-side and
// persisted, so the same session shows the same ID to every client — the
// web UI, stogo on any machine, the API — for as long as it exists.
//
// IDs are 3-4 digits (100-9999): long enough to be unique in any realistic
// deployment, short enough to type instead of a session name. A session
// keeps its ID across polls, offload/recreate (same host+name) and renames;
// assignments for sessions not seen for a retention period are pruned and
// their numbers eventually reused.
package sessionid

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"
)

const (
	minID = 100
	maxID = 9999
	// retention is how long an assignment outlives its last sighting. It
	// deliberately exceeds any plausible offload gap so a resumed session
	// comes back under its old number.
	retention = 30 * 24 * time.Hour
	// flushEvery bounds how often pure LastSeen refreshes hit disk. The
	// poller calls Assign for every session on every cycle; persisting each
	// touch would rewrite the file several times a minute for no benefit.
	flushEvery = 10 * time.Minute
)

type entry struct {
	Host     string    `json:"host"`
	Name     string    `json:"name"`
	ID       int       `json:"id"`
	LastSeen time.Time `json:"last_seen"`
}

// Store is a thread-safe JSON-backed ID allocator.
type Store struct {
	mu        sync.Mutex
	path      string
	byKey     map[string]*entry
	byID      map[int]*entry
	dirty     bool
	lastFlush time.Time
}

func key(host, name string) string { return host + "\x00" + name }

// NewStore opens (or creates) the allocator at <dataDir>/session-ids.json.
func NewStore(dataDir string) (*Store, error) {
	s := &Store{
		path:      filepath.Join(dataDir, "session-ids.json"),
		byKey:     make(map[string]*entry),
		byID:      make(map[int]*entry),
		lastFlush: time.Now(),
	}
	data, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return s, nil
		}
		return nil, fmt.Errorf("read session ids: %w", err)
	}
	var list []entry
	if err := json.Unmarshal(data, &list); err != nil {
		return nil, fmt.Errorf("parse session ids: %w", err)
	}
	cutoff := time.Now().Add(-retention)
	for i := range list {
		e := list[i]
		if e.ID < minID || e.ID > maxID || e.LastSeen.Before(cutoff) {
			continue
		}
		if _, taken := s.byID[e.ID]; taken {
			continue
		}
		ent := e
		s.byKey[key(e.Host, e.Name)] = &ent
		s.byID[e.ID] = &ent
	}
	return s, nil
}

// Assign returns the ID for (host, name), allocating one on first sight.
// Returns 0 only when every ID in the range is live within the retention
// window, which no realistic deployment reaches.
func (s *Store) Assign(host, name string) int {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	if e, ok := s.byKey[key(host, name)]; ok {
		e.LastSeen = now
		s.dirty = true
		s.maybeFlushLocked(now)
		return e.ID
	}

	id := s.freeIDLocked(now)
	if id == 0 {
		return 0
	}
	e := &entry{Host: host, Name: name, ID: id, LastSeen: now}
	s.byKey[key(host, name)] = e
	s.byID[id] = e
	s.flushLocked(now)
	return id
}

// Rename moves an assignment to the session's new name so the ID survives.
// No-op if the old name has no assignment.
func (s *Store) Rename(host, oldName, newName string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	e, ok := s.byKey[key(host, oldName)]
	if !ok {
		return
	}
	delete(s.byKey, key(host, oldName))
	// A stale assignment under the new name would orphan an ID; drop it.
	if prev, ok := s.byKey[key(host, newName)]; ok {
		delete(s.byID, prev.ID)
	}
	e.Name = newName
	e.LastSeen = time.Now()
	s.byKey[key(host, newName)] = e
	s.flushLocked(time.Now())
}

// freeIDLocked returns the lowest unassigned ID, pruning expired entries
// first if the range is fully allocated.
func (s *Store) freeIDLocked(now time.Time) int {
	if len(s.byID) >= maxID-minID+1 {
		cutoff := now.Add(-retention)
		for k, e := range s.byKey {
			if e.LastSeen.Before(cutoff) {
				delete(s.byKey, k)
				delete(s.byID, e.ID)
			}
		}
	}
	for id := minID; id <= maxID; id++ {
		if _, taken := s.byID[id]; !taken {
			return id
		}
	}
	return 0
}

func (s *Store) maybeFlushLocked(now time.Time) {
	if s.dirty && now.Sub(s.lastFlush) >= flushEvery {
		s.flushLocked(now)
	}
}

// flushLocked persists the current assignments; errors are swallowed — an
// unwritable data dir costs ID stability across restarts, not sessions.
func (s *Store) flushLocked(now time.Time) {
	list := make([]entry, 0, len(s.byKey))
	for _, e := range s.byKey {
		list = append(list, *e)
	}
	sort.Slice(list, func(i, j int) bool { return list[i].ID < list[j].ID })
	if data, err := json.MarshalIndent(list, "", "  "); err == nil {
		_ = os.WriteFile(s.path, data, 0600)
	}
	s.dirty = false
	s.lastFlush = now
}
