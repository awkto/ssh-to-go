package main

import (
	"flag"
	"fmt"
	"os"
)

// cmdResume brings an offloaded session back — the server recreates it in
// the directory and with the launch command it recorded — and attaches,
// unless -bg. It is the CLI counterpart of the dashboard's Recreate button.
func cmdResume(args []string) error {
	fs := flag.NewFlagSet("resume", flag.ContinueOnError)
	bgFlag := fs.Bool("bg", false, "recreate in the background, don't connect")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 1 {
		return fmt.Errorf("usage: stogo resume [-bg] <session>")
	}
	cfg, err := loadConfig()
	if err != nil {
		return err
	}
	c := newClient(cfg)

	host, entry, err := c.resolveOffloaded(fs.Arg(0))
	if err != nil {
		return err
	}
	return resumeSession(c, host, entry, !*bgFlag)
}

// resumeSession recreates one offloaded session and, when attach is set,
// replaces the process with the attach command. Shared by resume, connect
// (when asked for a name that is offloaded) and new (when the requested
// name is already tracked as offloaded).
func resumeSession(c *apiClient, host string, entry offloadedSession, attach bool) error {
	fmt.Printf("resuming %s/%s (dir %s, command %s)\n", host, entry.Name, dash(entry.WorkingDir), dash(entry.Command))
	res, err := c.recreateSession(host, entry.Name)
	if err != nil {
		return err
	}
	if attach {
		fmt.Fprintf(os.Stderr, "recreated %s/%s — attaching…\n", host, res.Name)
		return attachSession(c, host, res.Name)
	}
	fmt.Printf("recreated %s/%s — connect with: stogo connect %s\n", host, res.Name, res.Name)
	return nil
}
