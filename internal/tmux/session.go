package tmux

import (
	"math/bits"
	"strconv"
	"strings"
	"time"
)

type Session struct {
	// ID is the server-assigned short numeric ID, unique across hosts.
	// Zero when no assigner is configured (or from parsers/tests) — the
	// wire format then omits it, which older clients read as before.
	ID              int       `json:"id,omitempty"`
	Name            string    `json:"name"`
	Windows         int       `json:"windows"`
	Created         time.Time `json:"created"`
	Activity        time.Time `json:"activity"`
	Attached        bool      `json:"attached"`
	AttachedClients int       `json:"attached_clients"`

	// LastOutput is when a pane in the session last printed anything
	// (tmux window_activity, newest window). Activity above is
	// session_activity, which tmux bumps on client *input*, so it can't
	// tell a busy program from an idle one. Zero when the host didn't
	// report it.
	LastOutput time.Time `json:"last_output,omitempty"`
	// State is working, waiting or idle — see Classify. Empty when the
	// host didn't report pane data.
	State string `json:"state,omitempty"`

	// outputAge is seconds since LastOutput by the host's own clock, so
	// clock skew between us and the host can't make a session look busy.
	// -1 means unknown.
	outputAge int
	// shellOnly is true when every pane is a shell at its prompt.
	shellOnly bool
	// recent holds one bit per poll, newest in bit 0: was there output
	// since the previous poll. Carried across polls by Classify.
	recent uint8
}

const (
	StateWorking = "working"
	StateWaiting = "waiting"
	StateIdle    = "idle"
)

// freshAge is how recent output must be (seconds, host clock) to count
// as "output since the last poll" at the default 5s poll interval.
const freshAge = 5

// Classify sets State from this poll's pane data and the session's
// previous poll (nil for a new session).
//
// Working needs output in at least 2 of the last 3 polls. A single
// recent burst isn't enough: idle TUIs such as Claude Code still repaint
// every 20s or so, and a keystroke's echo is output too. Otherwise the
// session is idle when every pane is a shell at its prompt, and waiting
// when some program sits in the foreground.
func (s *Session) Classify(prev *Session) {
	if s.outputAge < 0 {
		return
	}
	if prev != nil {
		s.recent = prev.recent << 1
	}
	if s.outputAge < freshAge {
		s.recent |= 1
	}
	s.recent &= 0b111
	switch {
	case bits.OnesCount8(s.recent) >= 2:
		s.State = StateWorking
	case s.shellOnly:
		s.State = StateIdle
	default:
		s.State = StateWaiting
	}
}

// Format string for tmux list-sessions -F
const ListFormat = "#{session_name}\t#{session_windows}\t#{session_created}\t#{session_attached}\t#{session_activity}"

// ParseSessions parses output from `tmux list-sessions -F <ListFormat>`.
func ParseSessions(output string) []Session {
	var sessions []Session
	for _, line := range strings.Split(strings.TrimSpace(output), "\n") {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "\t", 5)
		if len(parts) < 4 {
			continue
		}

		windows, _ := strconv.Atoi(parts[1])
		createdUnix, _ := strconv.ParseInt(parts[2], 10, 64)
		attachedCount, _ := strconv.Atoi(parts[3])

		// session_activity is the timestamp (seconds since epoch) of the
		// last user/program activity in the session — perfect signal for
		// "recently used" sorting in the UI. Older tmux builds may not
		// emit this field; fall back to the created time so we still
		// have a usable value.
		activity := time.Unix(createdUnix, 0)
		if len(parts) >= 5 {
			if activityUnix, err := strconv.ParseInt(strings.TrimSpace(parts[4]), 10, 64); err == nil && activityUnix > 0 {
				activity = time.Unix(activityUnix, 0)
			}
		}

		sessions = append(sessions, Session{
			outputAge:       -1,
			Name:            parts[0],
			Windows:         windows,
			Created:         time.Unix(createdUnix, 0),
			Activity:        activity,
			Attached:        attachedCount > 0,
			AttachedClients: attachedCount,
		})
	}
	return sessions
}

// PaneFormat is the per-pane format for `tmux list-panes -a -F`.
const PaneFormat = "#{session_name}\t#{window_activity}\t#{pane_current_command}\t#{pane_pid}"

// paneMarker separates list-sessions output from the host clock and
// list-panes output in ListSessions' single remote command; procMarker
// then separates the `ps -A -o pid=,pgid=,tpgid=` listing.
const (
	paneMarker = "__stogo_panes__"
	procMarker = "__stogo_procs__"
)

// fgGroups reads `ps -o pid=,pgid=,tpgid=` output into each process's
// terminal foreground group (tpgid) and the size of every process group.
func fgGroups(out string) (tpgid map[int]int, size map[int]int) {
	tpgid, size = map[int]int{}, map[int]int{}
	for _, line := range strings.Split(out, "\n") {
		f := strings.Fields(line)
		if len(f) != 3 {
			continue
		}
		pid, err1 := strconv.Atoi(f[0])
		pgid, err2 := strconv.Atoi(f[1])
		tp, err3 := strconv.Atoi(f[2])
		if err1 != nil || err2 != nil || err3 != nil {
			continue
		}
		tpgid[pid] = tp
		size[pgid]++
	}
	return tpgid, size
}

// atPrompt reports whether a pane is a shell sitting at its prompt.
// pane_current_command alone can't tell: a running shell script (or
// `bash -c` as the pane command) reports "bash" too. So when ps data is
// available, the pane's own shell must also lead the terminal's
// foreground process group and be alone in it.
func atPrompt(cmd string, panePID int, tpgid, size map[int]int) bool {
	if !shellCommands[cmd] {
		return false
	}
	fg, ok := tpgid[panePID]
	if !ok {
		return true // no ps data for this pane: trust the command name
	}
	return fg == panePID && size[fg] == 1
}

// applyPanes fills LastOutput, outputAge and shellOnly from the part of
// ListSessions' output after paneMarker: the host's `date +%s` on the
// first line, then PaneFormat lines, then procMarker and the ps listing. now is our own clock, used to turn
// the host-relative age into a timestamp. Sessions without pane lines
// keep outputAge -1 and stay unclassified.
func applyPanes(sessions []Session, out string, now time.Time) {
	out, procOut, _ := strings.Cut(out, procMarker+"\n")
	tpgid, size := fgGroups(procOut)
	lines := strings.Split(strings.TrimSpace(out), "\n")
	hostNow, err := strconv.ParseInt(strings.TrimSpace(lines[0]), 10, 64)
	if err != nil || hostNow <= 0 {
		return
	}
	type agg struct {
		latest    int64
		shellOnly bool
	}
	by := map[string]*agg{}
	for _, line := range lines[1:] {
		parts := strings.SplitN(line, "\t", 4)
		if len(parts) < 3 {
			continue
		}
		panePID := -1
		if len(parts) == 4 {
			if n, err := strconv.Atoi(strings.TrimSpace(parts[3])); err == nil {
				panePID = n
			}
		}
		act, err := strconv.ParseInt(parts[1], 10, 64)
		if err != nil {
			continue
		}
		a := by[parts[0]]
		if a == nil {
			a = &agg{shellOnly: true}
			by[parts[0]] = a
		}
		if act > a.latest {
			a.latest = act
		}
		if !atPrompt(strings.TrimSpace(parts[2]), panePID, tpgid, size) {
			a.shellOnly = false
		}
	}
	for i := range sessions {
		a := by[sessions[i].Name]
		if a == nil || a.latest <= 0 {
			continue
		}
		age := hostNow - a.latest
		if age < 0 {
			age = 0
		}
		sessions[i].outputAge = int(age)
		sessions[i].shellOnly = a.shellOnly
		sessions[i].LastOutput = now.Add(-time.Duration(age) * time.Second).Truncate(time.Second)
	}
}
