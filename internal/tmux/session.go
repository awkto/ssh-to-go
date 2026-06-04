package tmux

import (
	"strconv"
	"strings"
	"time"
)

type Session struct {
	Name            string    `json:"name"`
	Windows         int       `json:"windows"`
	Created         time.Time `json:"created"`
	Activity        time.Time `json:"activity"`
	Attached        bool      `json:"attached"`
	AttachedClients int       `json:"attached_clients"`
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
