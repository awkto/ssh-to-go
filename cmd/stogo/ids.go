package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
)

// Short numeric session IDs. `stogo list` shows an ID column and commands
// accept the ID anywhere a session name is expected, so nobody has to type
// "customer-prod-debug-2" to kill it. Assignments live in a small cache file
// and are stable across runs: a session keeps its number for as long as it
// exists; numbers of vanished sessions are pruned and eventually reused.

func idCachePath() (string, error) {
	dir, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "stogo", "ids.json"), nil
}

func sessionKey(hs hostSession) string {
	return hs.HostName + "/" + hs.Session.Name
}

// assignIDs returns a host/name → ID map covering exactly the given
// sessions, reusing cached assignments where possible and persisting the
// result. Cache I/O errors are ignored — IDs then just aren't stable, which
// is not worth failing a list over.
func assignIDs(sessions []hostSession) map[string]int {
	saved := map[string]int{}
	path, err := idCachePath()
	if err == nil {
		if data, rerr := os.ReadFile(path); rerr == nil {
			_ = json.Unmarshal(data, &saved)
		}
	}

	ids := make(map[string]int, len(sessions))
	used := make(map[int]bool, len(sessions))
	var fresh []hostSession
	for _, hs := range sessions {
		if id, ok := saved[sessionKey(hs)]; ok && id > 0 && !used[id] {
			ids[sessionKey(hs)] = id
			used[id] = true
		} else {
			fresh = append(fresh, hs)
		}
	}

	// New sessions get the lowest free numbers, oldest first, so the
	// assignment is deterministic regardless of listing order.
	sort.SliceStable(fresh, func(i, j int) bool {
		return fresh[i].Session.Created.Before(fresh[j].Session.Created)
	})
	next := 1
	for _, hs := range fresh {
		for used[next] {
			next++
		}
		ids[sessionKey(hs)] = next
		used[next] = true
	}

	if err == nil {
		if data, merr := json.MarshalIndent(ids, "", "  "); merr == nil {
			_ = os.MkdirAll(filepath.Dir(path), 0700)
			_ = os.WriteFile(path, append(data, '\n'), 0600)
		}
	}
	return ids
}
