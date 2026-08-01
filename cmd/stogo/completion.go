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
// is unique across hosts, host/name otherwise. All failures are silent; a
// broken login must not spew errors into the middle of a tab press.
func cmdCompleteSessions() error {
	cfg, err := loadConfig()
	if err != nil {
		return nil
	}
	sessions, err := newClient(cfg).sessions()
	if err != nil {
		return nil
	}
	counts := map[string]int{}
	for _, hs := range sessions {
		counts[hs.Session.Name]++
	}
	for _, hs := range sessions {
		if counts[hs.Session.Name] == 1 {
			fmt.Println(hs.Session.Name)
		} else {
			fmt.Println(hs.HostName + "/" + hs.Session.Name)
		}
	}
	return nil
}
