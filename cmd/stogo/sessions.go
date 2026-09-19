package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/url"
	"os"
	"sort"
	"strings"
	"text/tabwriter"
	"time"
)

// listRow is one line of `stogo list`: a live tmux session, or an offloaded
// one the server still tracks (and can recreate) but tmux no longer runs.
// Exactly one of Session / Offloaded is set, matching State.
type listRow struct {
	HostName  string            `json:"host_name"`
	State     string            `json:"state"` // "active" | "offloaded"
	Session   *tmuxSession      `json:"session,omitempty"`
	Offloaded *offloadedSession `json:"offloaded,omitempty"`
}

func (r listRow) name() string {
	if r.Offloaded != nil {
		return r.Offloaded.Name
	}
	return r.Session.Name
}

// when is the sort/display timestamp: tmux activity for a live session, the
// last time the poller saw it for an offloaded one.
func (r listRow) when() time.Time {
	if r.Offloaded != nil {
		if !r.Offloaded.LastSeenAt.IsZero() {
			return r.Offloaded.LastSeenAt
		}
		return r.Offloaded.CreatedAt
	}
	return r.Session.Activity
}

const listUsage = "usage: stogo list [active|offloaded|all] [-t|-a] [-o json]"

func cmdList(args []string) error {
	fs := flag.NewFlagSet("list", flag.ContinueOnError)
	output := fs.String("o", "", "output format (json)")
	byTime := fs.Bool("t", false, "sort by last activity, most recent first (default)")
	byName := fs.Bool("a", false, "sort alphabetically by session name")
	if err := fs.Parse(args); err != nil {
		return err
	}
	// The filter word may sit before or after the flags ("list all -a",
	// "list -a all"); flag stops at the first positional, so parse the
	// remainder again.
	filter := "active"
	if rest := fs.Args(); len(rest) > 0 {
		filter = rest[0]
		if err := fs.Parse(rest[1:]); err != nil {
			return err
		}
		if fs.NArg() > 0 {
			return fmt.Errorf("%s", listUsage)
		}
	}
	if filter != "active" && filter != "offloaded" && filter != "all" {
		return fmt.Errorf("unknown filter %q\n%s", filter, listUsage)
	}
	if *byTime && *byName {
		return fmt.Errorf("-t and -a are mutually exclusive")
	}

	cfg, err := loadConfig()
	if err != nil {
		return err
	}
	c := newClient(cfg)

	var rows []listRow
	if filter != "offloaded" {
		sessions, err := c.sessions()
		if err != nil {
			return err
		}
		for i := range sessions {
			rows = append(rows, listRow{HostName: sessions[i].HostName, State: "active", Session: &sessions[i].Session})
		}
	}
	if filter != "active" {
		// Offloaded sessions ride along on the hosts endpoint — the same
		// missing_sessions the dashboard's Resumable list is built from.
		hosts, err := c.hosts()
		if err != nil {
			return err
		}
		for _, h := range hosts {
			for i := range h.MissingSessions {
				rows = append(rows, listRow{HostName: h.Config.Name, State: "offloaded", Offloaded: &h.MissingSessions[i]})
			}
		}
	}

	// Active sessions always sort ahead of offloaded ones; -t/-a order
	// within each group.
	sort.SliceStable(rows, func(i, j int) bool {
		a, b := rows[i], rows[j]
		if a.State != b.State {
			return a.State == "active"
		}
		if *byName {
			an, bn := strings.ToLower(a.name()), strings.ToLower(b.name())
			if an != bn {
				return an < bn
			}
			return a.HostName < b.HostName
		}
		if at, bt := a.when(), b.when(); !at.Equal(bt) {
			return at.After(bt)
		}
		return a.name() < b.name()
	})

	if *output == "json" {
		if rows == nil {
			rows = []listRow{}
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(rows)
	}

	if len(rows) == 0 {
		switch filter {
		case "offloaded":
			fmt.Println("No offloaded sessions.")
		case "all":
			fmt.Println("No sessions.")
		default:
			fmt.Println("No active sessions.")
		}
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 2, 4, 2, ' ', 0)
	switch filter {
	case "offloaded":
		fmt.Fprintln(w, "SESSION\tHOST\tDIR\tCOMMAND\tLAST SEEN")
		for _, r := range rows {
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
				r.name(), r.HostName, dash(r.Offloaded.WorkingDir), dash(r.Offloaded.Command),
				offloadedWhen(r))
		}
	case "all":
		fmt.Fprintln(w, "ID\tSESSION\tHOST\tSTATE\tWINDOWS\tCLIENTS\tACTIVITY")
		for _, r := range rows {
			if r.Offloaded != nil {
				fmt.Fprintf(w, "-\t%s\t%s\toffloaded\t-\t-\t%s\n", r.name(), r.HostName, offloadedWhen(r))
				continue
			}
			fmt.Fprintf(w, "%s\t%s\t%s\tactive\t%d\t%s\t%s\n",
				sessionID(r.Session), r.name(), r.HostName, r.Session.Windows,
				sessionClients(r.Session), relTime(r.Session.Activity))
		}
	default:
		fmt.Fprintln(w, "ID\tSESSION\tHOST\tWINDOWS\tCLIENTS\tACTIVITY")
		for _, r := range rows {
			fmt.Fprintf(w, "%s\t%s\t%s\t%d\t%s\t%s\n",
				sessionID(r.Session), r.name(), r.HostName, r.Session.Windows,
				sessionClients(r.Session), relTime(r.Session.Activity))
		}
	}
	return w.Flush()
}

// sessionID renders the short ID. An old server sends none; "-" beats a
// column of zeros.
func sessionID(s *tmuxSession) string {
	if s.ID > 0 {
		return fmt.Sprintf("%d", s.ID)
	}
	return "-"
}

func sessionClients(s *tmuxSession) string {
	if s.Attached {
		return fmt.Sprintf("%d", s.AttachedClients)
	}
	return "-"
}

// offloadedWhen marks sessions the idle sweeper put to sleep, so one that
// went away overnight doesn't read as something you did.
func offloadedWhen(r listRow) string {
	t := relTime(r.when())
	if r.Offloaded.AutoOffloaded {
		t += " (auto)"
	}
	return t
}

func dash(s string) string {
	if s == "" {
		return "-"
	}
	return s
}

func cmdStatus(args []string) error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}
	c := newClient(cfg)

	var me struct {
		Authenticated bool   `json:"authenticated"`
		NoAuth        bool   `json:"no_auth"`
		Version       string `json:"version"`
	}
	if err := c.do("GET", "/api/me", nil, &me); err != nil {
		return fmt.Errorf("server %s unreachable or rejected credentials: %w", cfg.URL, err)
	}

	fmt.Printf("Server:   %s (%s)\n", cfg.URL, strings.TrimPrefix(me.Version, "v"))
	switch {
	case me.NoAuth:
		fmt.Println("Auth:     disabled on server")
	case cfg.TokenName != "":
		fmt.Printf("Auth:     ok (token %q)\n", cfg.TokenName)
	default:
		fmt.Println("Auth:     ok")
	}

	hosts, err := c.hosts()
	if err != nil {
		return err
	}
	online, total, sessions := 0, len(hosts), 0
	for _, h := range hosts {
		if h.Online {
			online++
		}
		sessions += len(h.Sessions)
	}
	fmt.Printf("Hosts:    %d/%d online\n", online, total)
	fmt.Printf("Sessions: %d\n", sessions)
	for _, h := range hosts {
		if !h.Online && h.Error != "" {
			fmt.Printf("  offline: %s (%s)\n", h.Config.Name, h.Error)
		}
	}
	return nil
}

func cmdOffload(args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("usage: stogo offload <session>")
	}
	cfg, err := loadConfig()
	if err != nil {
		return err
	}
	c := newClient(cfg)
	host, session, err := c.resolveSession(args[0])
	if err != nil {
		return err
	}
	if err := c.do("POST", sessionPath(host, session)+"/offload", nil, nil); err != nil {
		return err
	}
	fmt.Printf("Offloaded %s/%s — resume it later from the dashboard or by recreating it\n", host, session)
	return nil
}

func cmdKill(args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("usage: stogo kill <session>")
	}
	cfg, err := loadConfig()
	if err != nil {
		return err
	}
	c := newClient(cfg)
	host, session, err := c.resolveSession(args[0])
	if err != nil {
		return err
	}
	if err := c.do("DELETE", sessionPath(host, session), nil, nil); err != nil {
		return err
	}
	fmt.Printf("Killed %s/%s\n", host, session)
	return nil
}

func sessionPath(host, session string) string {
	return "/api/hosts/" + url.PathEscape(host) + "/sessions/" + url.PathEscape(session)
}

// relTime renders a compact "how long ago" for table output.
func relTime(t time.Time) string {
	if t.IsZero() {
		return "-"
	}
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return "now"
	case d < time.Hour:
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	default:
		return fmt.Sprintf("%dd ago", int(d.Hours()/24))
	}
}
