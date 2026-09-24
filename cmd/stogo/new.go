package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"strings"

	"golang.org/x/term"

	"github.com/awkto/ssh-to-go/internal/sessionvars"
)

// cmdNew creates a session through prefilled prompts — name (when not
// given as an argument), host, directory, launch command — where Enter
// accepts the remembered default, so a repeat run is a name and
// Enter-Enter-Enter. It then attaches; -bg skips that. Flags answer prompts
// ahead of time; -y (or piped stdin) accepts every default unprompted. Only
// interactively given answers update the remembered defaults, so scripts
// never silently retrain them.
func cmdNew(args []string) error {
	fs := flag.NewFlagSet("new", flag.ContinueOnError)
	hostFlag := fs.String("host", "", "target host (default: remembered, then server default, then sole host)")
	dirFlag := fs.String("dir", "", "working directory ($name/$date expand server-side)")
	cmdFlag := fs.String("cmd", "", `launch command ("-" for none)`)
	attachFlag := fs.Bool("attach", false, "connect after creating (the default; kept for old scripts)")
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

	cfg, err := loadConfig()
	if err != nil {
		return err
	}
	c := newClient(cfg)

	interactive := !*yes && term.IsTerminal(int(os.Stdin.Fd()))
	reader := bufio.NewReader(os.Stdin)

	if rawName == "" {
		if !interactive {
			return fmt.Errorf("usage: stogo new [-host H] [-dir D] [-cmd C] [-bg] [-y] <name>")
		}
		for rawName == "" {
			if rawName, err = promptLine(reader, "name: "); err != nil {
				return err
			}
		}
	}
	name := sanitizeName(rawName)

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
	// The host list serves the host prompt and, afterwards, the check for an
	// offloaded session already tracked under this name.
	hosts, err := c.hosts()
	if err != nil {
		return err
	}
	// The host prompt is skipped when -host answered it or there is nothing
	// to choose between.
	pickedHost := false
	if *hostFlag == "" && (interactive || host == "") {
		switch {
		case len(hosts) == 0:
			return fmt.Errorf("no hosts configured on the server")
		case len(hosts) == 1:
			host = hosts[0].Config.Name
		case !interactive:
			return fmt.Errorf("multiple hosts configured — pass -host (one of: %s)", strings.Join(hostNames(hosts), ", "))
		default:
			if host, err = pickHost(reader, hosts, host); err != nil {
				return err
			}
			pickedHost = true
		}
	}

	if name != rawName {
		fmt.Printf("session: %s (from %q, on %s)\n", name, rawName, host)
	} else {
		fmt.Printf("session: %s (on %s)\n", name, host)
	}

	// An offloaded session of this name is not a collision, it is the thing
	// being asked for: bring it back with its recorded directory and command
	// instead of asking for new ones the server would refuse anyway.
	if entry, ok := findOffloaded(hosts, host, name); ok {
		if pickedHost {
			if err := updateNewDefaults(func(nd *newDefaults) { nd.Host = host }); err != nil {
				fmt.Fprintf(os.Stderr, "warning: could not remember defaults: %v\n", err)
			}
		}
		return resumeSession(c, host, entry, !*bgFlag)
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

	attach := !*bgFlag

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
	if pickedHost || cmdAnswered {
		err := updateNewDefaults(func(nd *newDefaults) {
			if pickedHost {
				nd.Host = host
			}
			if cmdAnswered {
				remembered := command
				nd.Command = &remembered
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

// pickHost asks which host to use, Enter accepting def. An answer may be a
// full host name or any unambiguous prefix of one; anything else re-asks
// rather than quietly creating the session somewhere unintended. When def
// is empty or no longer exists, the first online host stands in — an
// offline default would turn Enter-Enter-Enter into a guaranteed failure.
func pickHost(reader *bufio.Reader, hosts []hostState, def string) (string, error) {
	known := false
	for _, h := range hosts {
		if h.Config.Name == def {
			known = true
			break
		}
	}
	if !known {
		def = hosts[0].Config.Name
		for _, h := range hosts {
			if h.Online {
				def = h.Config.Name
				break
			}
		}
	}

	var others []string
	for _, h := range hosts {
		if h.Config.Name == def {
			continue
		}
		label := h.Config.Name
		if !h.Online {
			label += " (offline)"
		}
		others = append(others, label)
	}
	prompt := fmt.Sprintf("host    [%s] (or: %s): ", def, strings.Join(others, ", "))

	for {
		line, err := promptLine(reader, prompt)
		if err != nil {
			return "", err
		}
		if line == "" {
			return def, nil
		}
		if match, ok := matchHost(hosts, line); ok {
			return match, nil
		}
		fmt.Fprintf(os.Stderr, "no single host matches %q\n", line)
	}
}

// matchHost resolves an exact host name, or failing that a prefix shared by
// exactly one host.
func matchHost(hosts []hostState, in string) (string, bool) {
	var prefixed []string
	for _, h := range hosts {
		if h.Config.Name == in {
			return in, true
		}
		if strings.HasPrefix(h.Config.Name, in) {
			prefixed = append(prefixed, h.Config.Name)
		}
	}
	if len(prefixed) == 1 {
		return prefixed[0], true
	}
	return "", false
}

func hostNames(hosts []hostState) []string {
	names := make([]string, 0, len(hosts))
	for _, h := range hosts {
		names = append(names, h.Config.Name)
	}
	return names
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
