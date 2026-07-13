// Package execjob runs one-off shell commands on a remote host inside a
// throwaway, detached tmux session and lets callers poll for status, exit
// code, and captured output by job id.
//
// The design is intentionally fire-and-forget: the server dials the host,
// launches the command in a detached tmux session that redirects combined
// output to a file and records the exit code, then closes the SSH
// connection. The command keeps running on the remote host independent of
// any HTTP request, so long tasks (e.g. `claude -p ...`) survive well past
// the request that started them. Status lookups reconnect and inspect the
// tmux session + the on-disk result files.
//
// Job metadata (id -> host/command/session) lives in an in-memory store on
// the ssh-to-go server. The command itself and its output live on the
// remote host under ~/.ssh-to-go/exec/<id>/, so they persist across a
// server restart even though the in-memory index does not.
package execjob

import (
	"crypto/rand"
	"encoding/hex"
	"sort"
	"sync"
	"time"
)

// Status is the lifecycle state of a job as observed on the remote host.
type Status string

const (
	// StatusRunning: the throwaway tmux session is still alive.
	StatusRunning Status = "running"
	// StatusFinished: the session ended and an exit code was recorded.
	StatusFinished Status = "finished"
	// StatusCrashed: the runner died without recording an exit code (tmux
	// server restart, OOM kill, host reboot mid-job). Reported with exit
	// code -1 so a crash can never be mistaken for success.
	StatusCrashed Status = "crashed"
	// StatusGone: the job directory no longer exists on the host (host
	// rebooted, /tmp-style cleanup, or never created). Output is lost.
	StatusGone Status = "gone"
)

// Job is the server-side index entry for a launched command. The heavy
// data (command output) is not held here — it stays on the remote host and
// is fetched on demand.
type Job struct {
	ID        string    `json:"id"`
	Host      string    `json:"host"`
	Command   string    `json:"command"`
	Session   string    `json:"session"`
	CreatedAt time.Time `json:"created_at"`
}

// maxJobs bounds the in-memory index so a long-lived server that launches
// many jobs doesn't grow without limit. On overflow the oldest entries are
// dropped from the index (the remote result files are untouched).
const maxJobs = 1000

// Store is a concurrency-safe in-memory index of launched jobs.
type Store struct {
	mu   sync.RWMutex
	jobs map[string]*Job
}

// NewStore returns an empty job index.
func NewStore() *Store {
	return &Store{jobs: make(map[string]*Job)}
}

// Add records a job, pruning the oldest entries if the index is full.
func (s *Store) Add(j *Job) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.jobs) >= maxJobs {
		s.pruneOldestLocked(len(s.jobs) - maxJobs + 1)
	}
	s.jobs[j.ID] = j
}

// Get returns the job with the given id.
func (s *Store) Get(id string) (*Job, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	j, ok := s.jobs[id]
	return j, ok
}

// List returns all indexed jobs, newest first.
func (s *Store) List() []*Job {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*Job, 0, len(s.jobs))
	for _, j := range s.jobs {
		out = append(out, j)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].CreatedAt.After(out[j].CreatedAt)
	})
	return out
}

// pruneOldestLocked drops the n oldest jobs. Caller must hold the lock.
func (s *Store) pruneOldestLocked(n int) {
	if n <= 0 {
		return
	}
	all := make([]*Job, 0, len(s.jobs))
	for _, j := range s.jobs {
		all = append(all, j)
	}
	sort.Slice(all, func(i, j int) bool {
		return all[i].CreatedAt.Before(all[j].CreatedAt)
	})
	for i := 0; i < n && i < len(all); i++ {
		delete(s.jobs, all[i].ID)
	}
}

// NewID returns a random 12-hex-char job id. Short enough to paste into a
// URL, wide enough (48 bits) to avoid collisions in the bounded index.
func NewID() string {
	var b [6]byte
	// crypto/rand.Read never returns a short read on the platforms we run.
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}
