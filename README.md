# Web Terminal Session Manager — Why This Needs to Exist

## The Problem

If you work across multiple machines — say a Linux workstation, a Windows laptop, and a cloud VM — there's no clean way to keep terminal sessions alive and accessible from wherever you happen to be. Reboot into a different OS? Session gone. Switch to your laptop on the couch? Start over. Running a long `claude code` vibe coding session you absolutely cannot afford to interrupt? Better hope your SSH connection holds.

**tmux and screen exist**, and they're great at what they do on a single machine. But they don't give you a unified view across hosts, they have no web interface, and reconnecting from a different device still means SSH-ing in manually and reattaching. It works, but it's duct tape.

**Apache Guacamole** technically solves this, but the setup complexity is brutal — Java, Tomcat, a database, guacd, and a mountain of configuration before you see a single terminal. Most people give up before they get there.

## What Exists Today

We surveyed the current landscape of open-source and self-hosted tools. Here's what's out there and where each falls short of the full vision:

| Tool | What It Does | What's Missing |
|------|-------------|----------------|
| **[tmate](https://tmate.io)** | tmux fork with instant terminal sharing via SSH or web URL. Supports named sessions for stable reconnection. | Designed for pair programming, not personal session management. No dashboard. Keeping sessions alive on headless machines can be unreliable. |
| **[ttyd](https://github.com/tsl0922/ttyd)** | Lightweight C-based tool that shares a terminal over the web via xterm.js and libwebsockets. Fast and stable. | One command per instance. No built-in session persistence or multi-machine management — you wire that up yourself with tmux. |
| **[GoTTY](https://github.com/sorenisanerd/gotty)** | Go-based tool that turns CLI programs into web applications. | Spawns a new process per client — no session resumption by default. Same tmux-wiring requirement as ttyd. |
| **[Sshwifty](https://github.com/nirui/sshwifty)** | Clean web-based SSH/Telnet client. Easy Docker deployment. | It's a connection client, not a session persistence layer. Doesn't manage or maintain sessions on the remote side. |
| **[ShellNGN](https://shellngn.com)** | Web SSH client with tabbed sessions, SFTP, and team features. Available as SaaS or self-hosted. | More of an SSH client replacement than a session orchestrator. No cross-machine session dashboard. |
| **[Apache Guacamole](https://guacamole.apache.org)** | Full-featured gateway for SSH, VNC, and RDP in the browser. | Notoriously complex to deploy. Java stack, Tomcat, PostgreSQL/MySQL, guacd daemon. Overkill for terminal-only use cases. |
| **[WeTTY](https://github.com/butlerx/wetty)** | Browser-based terminal using hterm over websockets. | Similar to ttyd/GoTTY — a transport layer, not a session manager. |

## The Gap

None of these tools combine **all** of the following:

1. **A web dashboard** showing all your active sessions across multiple machines in one place
2. **True session persistence** — sessions survive browser disconnects, network changes, and machine reboots
3. **Reconnect from anywhere** — open a URL from any device and pick up exactly where you left off
4. **Minimal setup** — a single binary or Docker container, not a multi-service Java stack
5. **Session management** — name, organize, tag, and kill sessions from the UI
6. **Multi-machine support** — a lightweight agent on each host, one central dashboard to rule them all

## Proposed Architecture

```
┌─────────────────────────────────────────────────┐
│                  Web Dashboard                   │
│          (xterm.js + session list UI)            │
└──────────────────────┬──────────────────────────┘
                       │ WebSocket
┌──────────────────────┴──────────────────────────┐
│               Central Server                     │
│     (session registry, auth, WS relay)           │
└───────┬──────────────┬──────────────┬───────────┘
        │              │              │
   ┌────┴────┐    ┌────┴────┐   ┌────┴────┐
   │ Agent   │    │ Agent   │   │ Agent   │
   │ + tmux  │    │ + tmux  │   │ + tmux  │
   │ (host1) │    │ (host2) │   │ (host3) │
   └─────────┘    └─────────┘   └─────────┘
```

- **Agent**: Lightweight daemon on each machine. Manages local tmux sessions, reports status back to the central server.
- **Central Server**: Session registry, WebSocket relay, authentication. Self-hostable as a single binary or container.
- **Web Dashboard**: Browser-based UI with xterm.js. Lists all sessions across all machines. Click to connect, sessions persist whether you're watching or not.

## Use Cases

- **Vibe coding with Claude Code** — Start a session on your desktop, continue on your laptop, never lose context
- **Multi-machine DevOps** — One dashboard for all your servers, dev VMs, and cloud instances
- **Pair programming / mentoring** — Share a persistent session URL with a teammate
- **Long-running tasks** — Deploy, build, or migrate without babysitting a terminal window

## Prior Art & Inspiration

This project would stand on the shoulders of excellent existing work:

- **tmux** — Battle-tested session persistence on the server side
- **xterm.js** — The standard for terminal emulation in the browser
- **ttyd** — Proof that a fast, minimal web terminal is achievable
- **tmate** — Demonstrated that relay-based terminal access works at scale
- **Guacamole** — Proved the concept, but showed that simplicity matters just as much as features

## Status

This project is in the idea/planning stage. Contributions, feedback, and architecture discussions are welcome.
