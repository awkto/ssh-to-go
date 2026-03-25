package tmux

import (
	"strconv"
	"strings"
	"time"
)

type Session struct {
	Name     string    `json:"name"`
	Windows  int       `json:"windows"`
	Created  time.Time `json:"created"`
	Attached bool      `json:"attached"`
}

// Format string for tmux list-sessions -F
const ListFormat = "#{session_name}\t#{session_windows}\t#{session_created}\t#{session_attached}"

// ParseSessions parses output from `tmux list-sessions -F <ListFormat>`.
func ParseSessions(output string) []Session {
	var sessions []Session
	for _, line := range strings.Split(strings.TrimSpace(output), "\n") {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "\t", 4)
		if len(parts) < 4 {
			continue
		}

		windows, _ := strconv.Atoi(parts[1])
		createdUnix, _ := strconv.ParseInt(parts[2], 10, 64)
		attachedCount, _ := strconv.Atoi(parts[3])

		sessions = append(sessions, Session{
			Name:     parts[0],
			Windows:  windows,
			Created:  time.Unix(createdUnix, 0),
			Attached: attachedCount > 0,
		})
	}
	return sessions
}
