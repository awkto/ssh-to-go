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

func cmdList(args []string) error {
	fs := flag.NewFlagSet("list", flag.ContinueOnError)
	output := fs.String("o", "", "output format (json)")
	byTime := fs.Bool("t", false, "sort by last activity, most recent first (default)")
	byName := fs.Bool("a", false, "sort alphabetically by session name")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *byTime && *byName {
		return fmt.Errorf("-t and -a are mutually exclusive")
	}

	cfg, err := loadConfig()
	if err != nil {
		return err
	}
	sessions, err := newClient(cfg).sessions()
	if err != nil {
		return err
	}

	if *byName {
		sort.SliceStable(sessions, func(i, j int) bool {
			a, b := strings.ToLower(sessions[i].Session.Name), strings.ToLower(sessions[j].Session.Name)
			if a != b {
				return a < b
			}
			return sessions[i].HostName < sessions[j].HostName
		})
	} else {
		sort.SliceStable(sessions, func(i, j int) bool {
			a, b := sessions[i].Session.Activity, sessions[j].Session.Activity
			if !a.Equal(b) {
				return a.After(b)
			}
			return sessions[i].Session.Name < sessions[j].Session.Name
		})
	}

	ids := assignIDs(sessions)

	if *output == "json" {
		type row struct {
			ID int `json:"id"`
			hostSession
		}
		rows := make([]row, len(sessions))
		for i, hs := range sessions {
			rows[i] = row{ID: ids[sessionKey(hs)], hostSession: hs}
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(rows)
	}

	if len(sessions) == 0 {
		fmt.Println("No sessions.")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 2, 4, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tSESSION\tHOST\tWINDOWS\tCLIENTS\tACTIVITY")
	for _, hs := range sessions {
		clients := "-"
		if hs.Session.Attached {
			clients = fmt.Sprintf("%d", hs.Session.AttachedClients)
		}
		fmt.Fprintf(w, "%d\t%s\t%s\t%d\t%s\t%s\n",
			ids[sessionKey(hs)], hs.Session.Name, hs.HostName, hs.Session.Windows, clients,
			relTime(hs.Session.Activity))
	}
	return w.Flush()
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
