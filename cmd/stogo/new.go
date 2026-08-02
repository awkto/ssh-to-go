package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"

	"golang.org/x/term"

	"github.com/awkto/ssh-to-go/internal/sessionvars"
)

// cmdNew creates a session through three prefilled prompts — directory,
// launch command, connect-or-not — where Enter accepts the remembered
// default, so a repeat run is Enter-Enter-Enter. Flags answer prompts ahead
// of time; -y (or piped stdin) accepts every default unprompted. Only
// interactively typed answers update the remembered defaults, so scripts
// never silently retrain them.
func cmdNew(args []string) error {
	fs := flag.NewFlagSet("new", flag.ContinueOnError)
	hostFlag := fs.String("host", "", "target host (default: remembered, then server default, then sole host)")
	dirFlag := fs.String("dir", "", "working directory ($name/$date expand server-side)")
	cmdFlag := fs.String("cmd", "", `launch command ("-" for none)`)
	attachFlag := fs.Bool("attach", false, "connect to the session after creating it")
	bgFlag := fs.Bool("bg", false, "create in the background, don't connect")
	yes := fs.Bool("y", false, "accept all defaults without prompting")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *attachFlag && *bgFlag {
		return fmt.Errorf("-attach and -bg are mutually exclusive")
	}
	// Multi-word names need no quoting: everything after the flags is the
	// name, and the server collapses the spaces to dashes anyway.
	rawName := strings.TrimSpace(strings.Join(fs.Args(), " "))
	if rawName == "" {
		return fmt.Errorf("usage: stogo new [-host H] [-dir D] [-cmd C] [-attach|-bg] [-y] <name>")
	}
	name := sanitizeName(rawName)

	cfg, err := loadConfig()
	if err != nil {
		return err
	}
	c := newClient(cfg)

	interactive := !*yes && term.IsTerminal(int(os.Stdin.Fd()))
	reader := bufio.NewReader(os.Stdin)

	nd := cfg.New
	if nd == nil {
		nd = &newDefaults{}
	}

	// Server settings feed the host and directory defaults. Best-effort: a
	// server predating the endpoint just means empty defaults.
	st, _ := c.settings()

	host := *hostFlag
	if host == "" {
		host = nd.Host
	}
	if host == "" {
		host = st.DefaultHost
	}
	pickedHost := false
	if host == "" {
		hosts, err := c.hosts()
		if err != nil {
			return err
		}
		switch len(hosts) {
		case 0:
			return fmt.Errorf("no hosts configured on the server")
		case 1:
			host = hosts[0].Config.Name
		default:
			if !interactive {
				var names []string
				for _, h := range hosts {
					names = append(names, h.Config.Name)
				}
				return fmt.Errorf("multiple hosts configured — pass -host (one of: %s)", strings.Join(names, ", "))
			}
			host = pickHost(reader, hosts)
			pickedHost = true
		}
	}

	if name != rawName {
		fmt.Printf("session: %s (from %q, on %s)\n", name, rawName, host)
	} else {
		fmt.Printf("session: %s (on %s)\n", name, host)
	}

	// Directory. The value sent to the server may be a template — display
	// it expanded (same rules, same package) so the prompt shows the real
	// path the session will get.
	vars := sessionvars.Vars{Name: name}
	dir := *dirFlag
	if dir == "" {
		def := nd.Dir
		if def == "" {
			base := st.NewSessionDir
			if base == "" {
				base = "~/sessions/"
			}
			def = joinDir(base, dirSlug(rawName))
		}
		dir = def
		if interactive {
			shown := sessionvars.Expand(def, vars)
			if shown != def {
				shown = def + " → " + shown
			}
			line, err := promptLine(reader, fmt.Sprintf("dir     [%s]: ", shown))
			if err != nil {
				return err
			}
			if line != "" {
				dir = line
			}
		}
	}

	// Launch command. First run seeds the default from the server's
	// recent-commands list — the same chips the web form offers.
	command := ""
	cmdAnswered := false
	switch {
	case *cmdFlag == "-":
		// none
	case *cmdFlag != "":
		command = *cmdFlag
	default:
		var def string
		if nd.Command != nil {
			def = *nd.Command
		} else if rc, err := c.recentCommands(); err == nil && len(rc) > 0 {
			def = rc[0].Command
		}
		command = def
		if interactive {
			shown := "none"
			if def != "" {
				shown = def
			}
			line, err := promptLine(reader, fmt.Sprintf("command [%s] ('-' for none): ", shown))
			if err != nil {
				return err
			}
			switch line {
			case "":
			case "-":
				command = ""
			default:
				command = line
			}
			cmdAnswered = true
		}
	}

	attach := true
	if nd.Attach != nil {
		attach = *nd.Attach
	}
	attachAnswered := false
	switch {
	case *attachFlag:
		attach = true
	case *bgFlag:
		attach = false
	default:
		if interactive {
			hint := "[Y/n]"
			if !attach {
				hint = "[y/N]"
			}
			line, err := promptLine(reader, fmt.Sprintf("connect now? %s ", hint))
			if err != nil {
				return err
			}
			switch strings.ToLower(line) {
			case "y", "yes":
				attach = true
			case "n", "no":
				attach = false
			}
			attachAnswered = true
		}
	}

	if err := c.createSession(host, createSessionReq{
		Name:      rawName,
		Cwd:       dir,
		CreateDir: true,
		Command:   command,
	}); err != nil {
		return err
	}

	// Remember only now that the session really exists — a failed create
	// should not retrain the defaults.
	if pickedHost || cmdAnswered || attachAnswered {
		err := updateNewDefaults(func(nd *newDefaults) {
			if pickedHost {
				nd.Host = host
			}
			if cmdAnswered {
				remembered := command
				nd.Command = &remembered
			}
			if attachAnswered {
				a := attach
				nd.Attach = &a
			}
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "warning: could not remember defaults: %v\n", err)
		}
	}

	if attach {
		fmt.Printf("created %s/%s — attaching…\n", host, name)
		return attachSession(c, host, name)
	}
	fmt.Printf("created %s/%s — connect with: stogo connect %s\n", host, name, name)
	return nil
}

// pickHost shows a numbered list and reads a choice by number or name. The
// default (Enter, or unparseable input) is the first online host — an
// offline default would turn Enter-Enter-Enter into a guaranteed failure.
func pickHost(reader *bufio.Reader, hosts []hostState) string {
	def := 0
	for i, h := range hosts {
		if h.Online {
			def = i
			break
		}
	}
	fmt.Println("hosts:")
	for i, h := range hosts {
		status := ""
		if !h.Online {
			status = " (offline)"
		}
		fmt.Printf("  %d. %s%s\n", i+1, h.Config.Name, status)
	}
	line, err := promptLine(reader, fmt.Sprintf("host [%d = %s]: ", def+1, hosts[def].Config.Name))
	if err != nil || line == "" {
		return hosts[def].Config.Name
	}
	if n, err := strconv.Atoi(line); err == nil && n >= 1 && n <= len(hosts) {
		return hosts[n-1].Config.Name
	}
	for _, h := range hosts {
		if h.Config.Name == line {
			return line
		}
	}
	fmt.Fprintf(os.Stderr, "no host %q — using %s\n", line, hosts[def].Config.Name)
	return hosts[def].Config.Name
}

func promptLine(reader *bufio.Reader, prompt string) (string, error) {
	fmt.Print(prompt)
	line, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(line), nil
}

// sanitizeName mirrors the server's sanitizeSessionName: trim, and collapse
// each internal whitespace run into a single dash. Kept in sync so the name
// shown (and used for $name and the attach) is the name tmux really gets.
func sanitizeName(name string) string {
	name = strings.TrimSpace(name)
	var b strings.Builder
	b.Grow(len(name))
	prevSpace := false
	for _, r := range name {
		if r == ' ' || r == '\t' {
			if !prevSpace {
				b.WriteByte('-')
				prevSpace = true
			}
			continue
		}
		b.WriteRune(r)
		prevSpace = false
	}
	return b.String()
}

// dirSlug mirrors the web form's nsDirSlug so the CLI derives the same
// directory the dashboard would: lowercase, runs of anything outside
// [a-z0-9._-] become one dash, dashes collapsed, trimmed of leading and
// trailing dashes/dots, capped at 40 chars.
func dirSlug(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	var b strings.Builder
	b.Grow(len(s))
	prevDash := false
	for _, r := range s {
		ok := (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '.' || r == '_'
		if !ok {
			if prevDash {
				continue
			}
			b.WriteByte('-')
			prevDash = true
			continue
		}
		b.WriteRune(r)
		prevDash = false
	}
	out := strings.Trim(b.String(), "-.")
	if len(out) > 40 {
		out = strings.TrimRight(out[:40], "-.")
	}
	return out
}

// joinDir appends a slug to a base directory, tolerating a base with or
// without a trailing slash. An empty slug (a name with no usable chars)
// leaves the base alone.
func joinDir(base, slug string) string {
	if slug == "" {
		return base
	}
	if strings.HasSuffix(base, "/") {
		return base + slug
	}
	return base + "/" + slug
}
