package main

import (
	_ "embed"
	"fmt"
)

//go:embed completion.bash
var bashCompletion string

func cmdCompletion(args []string) error {
	if len(args) != 1 || args[0] != "bash" {
		return fmt.Errorf("usage: stogo completion bash")
	}
	fmt.Print(bashCompletion)
	return nil
}

// cmdCompleteSessions backs tab completion (hidden `__sessions` command): it
// prints one completable identifier per line — the bare session name when it
// is unique across hosts, host/name otherwise. With no argument it lists
// active sessions; "offloaded" lists the resumable ones instead and "all"
// lists both. All failures are silent; a broken login must not spew errors
// into the middle of a tab press.
func cmdCompleteSessions(args []string) error {
	scope := "active"
	if len(args) > 0 {
		scope = args[0]
	}
	cfg, err := loadConfig()
	if err != nil {
		return nil
	}
	c := newClient(cfg)

	type ident struct{ host, name string }
	var ids []ident
	if scope != "offloaded" {
		sessions, err := c.sessions()
		if err != nil {
			return nil
		}
		for _, hs := range sessions {
			ids = append(ids, ident{hs.HostName, hs.Session.Name})
		}
	}
	if scope != "active" {
		hosts, err := c.hosts()
		if err != nil {
			return nil
		}
		for _, h := range hosts {
			for _, m := range h.MissingSessions {
				ids = append(ids, ident{h.Config.Name, m.Name})
			}
		}
	}

	counts := map[string]int{}
	for _, id := range ids {
		counts[id.name]++
	}
	for _, id := range ids {
		if counts[id.name] == 1 {
			fmt.Println(id.name)
		} else {
			fmt.Println(id.host + "/" + id.name)
		}
	}
	return nil
}
