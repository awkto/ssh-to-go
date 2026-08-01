// Command stogo is the native terminal client for ssh-to-go.
//
// It talks to a running ssh-to-go server over the same HTTP API the web
// dashboard uses, and attaches to tmux sessions through the websocket
// relay — so a session opened in the browser can be picked up in a real
// terminal and vice versa.
package main

import (
	"fmt"
	"os"
)

// Version is stamped at build time via -ldflags.
var Version = "dev"

const usageText = `stogo — terminal client for ssh-to-go

Usage:
  stogo auth login [-url URL] [-name TOKEN_NAME]   authenticate with a server
  stogo auth logout                                revoke token and forget server
  stogo list | ls [-t|-a] [-o json]                list tmux sessions (-t by activity, default; -a by name)
  stogo connect | c <session>                      attach to a session (interactive)
  stogo offload <session>                          offload a session (stop it, keep it resumable)
  stogo kill <session>                             kill a session
  stogo status                                     server connectivity and summary
  stogo completion bash                            print bash completion script
  stogo version                                    print client version

Sessions may be addressed as NAME (unique across hosts), HOST/NAME, or the
numeric ID shown by "stogo list".

Config: ~/.config/stogo/config.json (overridable with STOGO_URL / STOGO_TOKEN
environment variables for headless use).
`

func usage() {
	fmt.Fprint(os.Stderr, usageText)
}

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}

	var err error
	switch os.Args[1] {
	case "auth":
		err = cmdAuth(os.Args[2:])
	case "list", "ls":
		err = cmdList(os.Args[2:])
	case "connect", "c":
		err = cmdConnect(os.Args[2:])
	case "offload":
		err = cmdOffload(os.Args[2:])
	case "kill":
		err = cmdKill(os.Args[2:])
	case "status":
		err = cmdStatus(os.Args[2:])
	case "completion":
		err = cmdCompletion(os.Args[2:])
	case "__sessions": // hidden: feeds bash completion
		err = cmdCompleteSessions()
	case "version", "-v", "--version":
		fmt.Println(Version)
	case "help", "-h", "--help":
		usage()
	default:
		fmt.Fprintf(os.Stderr, "stogo: unknown command %q\n\n", os.Args[1])
		usage()
		os.Exit(2)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "stogo: %v\n", err)
		os.Exit(1)
	}
}
