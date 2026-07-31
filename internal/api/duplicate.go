package api

import (
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"time"

	"github.com/awkto/ssh-to-go/internal/sessionreg"
	"github.com/awkto/ssh-to-go/internal/sessionvars"
	"github.com/awkto/ssh-to-go/internal/sshutil"
	"github.com/awkto/ssh-to-go/internal/tmux"
)

// copySuffix matches the "-COPY", "-COPY2", "-COPY3" tail this file appends.
var copySuffix = regexp.MustCompile(`-COPY([0-9]*)$`)

// copyBase strips a trailing copy suffix, so duplicating fooCOPY2 continues
// the chain (foo-COPY3) instead of nesting (foo-COPY2-COPY).
func copyBase(name string) string {
	return copySuffix.ReplaceAllString(name, "")
}

// copyName returns the next free copy name for base, given every session name
// already known on the host. Numbering scans for the HIGHEST existing suffix
// rather than the first free one: deleting the middle of a chain must not make
// the next duplicate collide with the end of it.
func copyName(base string, taken []string) string {
	highest := 0
	for _, name := range taken {
		if copyBase(name) != base {
			continue
		}
		m := copySuffix.FindStringSubmatch(name)
		if m == nil {
			continue // the original, not a copy
		}
		n := 1 // bare "-COPY" is the first
		if m[1] != "" {
			var err error
			if n, err = strconv.Atoi(m[1]); err != nil {
				continue
			}
		}
		if n > highest {
			highest = n
		}
	}
	next := highest + 1
	if next == 1 {
		return base + "-COPY"
	}
	return fmt.Sprintf("%s-COPY%d", base, next)
}

// knownNames lists every session name on a host — live in tmux plus tracked
// in the registry — so a copy name can't collide with an offloaded entry.
func (h *Handlers) knownNames(hostName string) []string {
	var names []string
	if state, ok := h.Hub.GetHost(hostName); ok {
		for _, s := range state.Sessions {
			names = append(names, s.Name)
		}
	}
	if h.Registry != nil {
		for _, e := range h.Registry.ListByHost(hostName) {
			names = append(names, e.Name)
		}
	}
	return names
}

// duplicateLaunch works out where a copy should start and what it should run.
//
// A session created from a template re-expands it against the COPY's own
// name: that is the whole point of `claude --name $name`, so the copy
// announces itself as the copy rather than as the original. A session created
// without variables has no template and keeps the original behaviour exactly
// — the live pane's directory (falling back to the recorded one) and the
// command as recorded.
//
// Re-expanding the path means the copy lands in a directory of its own, which
// by definition does not exist yet, so createDir comes back true. The old
// reasoning — "the source session demonstrably runs in this directory" —
// holds only for as long as the path is not recomputed.
func duplicateLaunch(entry sessionreg.Entry, liveCwd, newName string, now time.Time) (cwd, command string, createDir bool) {
	cwd, command = liveCwd, entry.Command
	if cwd == "" {
		cwd = entry.WorkingDir
	}
	vars := sessionvars.Vars{Name: newName, Now: now}
	if entry.CommandTemplate != "" {
		command = sessionvars.Expand(entry.CommandTemplate, vars)
	}
	if entry.WorkingDirTemplate != "" {
		cwd = sessionvars.Expand(entry.WorkingDirTemplate, vars)
		createDir = true
	}
	return cwd, command, createDir
}

// DuplicateSession creates a second session alongside an existing one,
// carrying over what makes it that session: the directory it is sitting in
// right now, the command it was launched with, and its icon/colour/theme.
//
// Deliberately NOT carried over: throwaway and incognito. Those are choices
// about a session's lifetime made when it was created, and inheriting them
// silently would be a surprise — a duplicate of a throwaway would evaporate.
func (h *Handlers) DuplicateSession(w http.ResponseWriter, r *http.Request) {
	hostName := r.PathValue("host")
	sessionName := r.PathValue("session")

	hostCfg, ok := h.Hub.GetHostConfig(hostName)
	if !ok {
		http.Error(w, "host not found", http.StatusNotFound)
		return
	}

	newName := copyName(copyBase(sanitizeSessionName(sessionName)), h.knownNames(hostName))

	client, err := sshutil.Dial(hostCfg.DialAddress(), hostCfg.User, h.resolveKey(hostCfg))
	if err != nil {
		http.Error(w, fmt.Sprintf("ssh connect failed: %v", err), http.StatusBadGateway)
		return
	}
	defer client.Close()

	// The live pane's cwd beats the registry's copy: the registry records
	// where the session started (refreshed by the poller), but the pane is
	// the truth right now. Fall back to the tracked entry when the source
	// session isn't running.
	cwd, err := h.Tmux.SessionCwd(client, sessionName)
	if err != nil {
		cwd = ""
	}
	var entry sessionreg.Entry
	if h.Registry != nil {
		entry, _ = h.Registry.Get(hostName, sanitizeSessionName(sessionName))
	}
	cwd, command, createDir := duplicateLaunch(entry, cwd, newName, time.Now())

	if err := h.Tmux.CreateSessionWith(client, newName, tmux.CreateOptions{
		WindowSize:   h.Settings.TmuxWindowSize(),
		Cwd:          cwd,
		HistoryLimit: h.Settings.ScrollbackLines(),
		CreateDir:    createDir,
		Command:      command,
		Mouse:        h.Settings.NativeMouseMode(),
	}); err != nil {
		http.Error(w, fmt.Sprintf("create session failed: %v", err), http.StatusInternalServerError)
		return
	}

	h.clearTerminated(hostName, newName)

	if h.Registry != nil {
		if err := h.Registry.AddSession(hostName, newName, sessionreg.Attrs{
			WorkingDir: cwd,
			Command:    command,
			// The templates travel too, so duplicating a copy re-expands
			// again rather than freezing the first copy's name into the
			// chain forever.
			WorkingDirTemplate: entry.WorkingDirTemplate,
			CommandTemplate:    entry.CommandTemplate,
		}); err != nil {
			log.Printf("session registry add %s/%s: %v", hostName, newName, err)
		}
	}

	// A copy should look like what it copied. Icon, colour and terminal
	// theme travel; "starred" and the last-accessed stamp do not — those
	// are about the original, not the new session.
	if h.SessionIcons != nil {
		src := h.SessionIcons.Get(hostName, sessionName)
		icon := h.Settings.NewSessionIcon()
		if src.Icon != "" || src.Color != "" {
			icon.Icon, icon.Color = src.Icon, src.Color
		}
		icon.Theme = src.Theme
		if err := h.SessionIcons.Set(hostName, newName, icon); err != nil {
			log.Printf("session icon assign %s/%s: %v", hostName, newName, err)
		}
	}

	w.WriteHeader(http.StatusCreated)
	writeJSON(w, map[string]string{
		"status":      "duplicated",
		"name":        newName,
		"source":      sessionName,
		"working_dir": cwd,
		"command":     command,
	})
}
