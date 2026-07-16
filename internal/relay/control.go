package relay

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/awkto/ssh-to-go/internal/sshutil"
	"golang.org/x/crypto/ssh"
	"nhooyr.io/websocket"
)

// Control-mode relay.
//
// Instead of attaching a screen-painting tmux client (which repaints the
// visible pane on attach and on every resize, duplicating or clipping the
// scrollback the browser already holds), we attach in tmux control mode
// (`tmux -C`). Control mode is a line protocol: tmux emits `%output` events
// containing only NEW pane output — never a repaint, never a status bar,
// never copy-mode. The browser's emulator therefore owns the scrollback and
// scrolls natively.
//
// History is prefilled by issuing `capture-pane -p -e -J -S -` over the
// control channel right after attach. Because the tmux server processes
// control commands and pane output in one ordered event loop, every byte of
// pane output is either inside the capture reply or arrives as a `%output`
// event after it — never both. We drop `%output` events until the capture
// reply has been forwarded, which makes the history/live seam exact.
// `-J` joins wrapped lines so the browser re-wraps history at its own width
// instead of inheriting the old pane width (the "cut off" artifact).
//
// Known trade-offs (acceptable for the web terminal):
//   - tmux prefix-key bindings (C-b ...) don't work: input is injected with
//     send-keys, which bypasses client key bindings.
//   - Only the active pane is streamed; split panes from an external attach
//     will not render side by side.

// controlCmdKind tells the parser what to do with a command's reply block.
// Control mode answers commands in FIFO order, one %begin/%end block each.
type controlCmdKind int

const (
	cmdConnect controlCmdKind = iota // reply to the new-session on the command line
	cmdIgnore                        // reply carries nothing we need (send-keys, refresh-client, ...)
	cmdMeta                          // display-message reply: "<client_name>,<pane_id>"
	cmdPane                          // display-message reply: "<pane_id>" (active window changed)
	cmdAlt                           // display-message reply: "<alternate_on>" (0/1)
	cmdHistory                       // capture-pane reply: scrollback prefill
)

// controlParser consumes the control-mode line stream. It is driven from a
// single goroutine; only the command queue is shared with the stdin writer.
type controlParser struct {
	mu    sync.Mutex
	queue []controlCmdKind

	inBlock    bool
	blockTime  string
	blockNum   string
	blockKind  controlCmdKind
	blockLines []string

	// historyDone gates %output forwarding: anything tmux emitted before the
	// capture-pane reply finished is already inside the prefill.
	historyDone bool
	// paneID is the active pane whose %output we forward ("" = forward all).
	paneID string

	emit            func(data []byte)       // terminal bytes for the browser
	onMeta          func(clientName string) // client_name learned (may be "")
	onWindowChanged func()                  // active window changed; re-query pane
}

func (p *controlParser) pushCmd(kind controlCmdKind) {
	p.mu.Lock()
	p.queue = append(p.queue, kind)
	p.mu.Unlock()
}

func (p *controlParser) popCmd() controlCmdKind {
	p.mu.Lock()
	defer p.mu.Unlock()
	if len(p.queue) == 0 {
		return cmdIgnore
	}
	k := p.queue[0]
	p.queue = p.queue[1:]
	return k
}

func (p *controlParser) feedLine(line string) {
	if p.inBlock {
		if isBlockEnd(line, p.blockTime, p.blockNum) {
			p.inBlock = false
			p.dispatchBlock(strings.HasPrefix(line, "%error"))
			p.blockLines = nil
			return
		}
		p.blockLines = append(p.blockLines, line)
		return
	}

	switch {
	case strings.HasPrefix(line, "%begin "):
		f := strings.Fields(line)
		p.blockTime, p.blockNum = "", ""
		if len(f) >= 3 {
			p.blockTime, p.blockNum = f[1], f[2]
		}
		p.blockKind = p.popCmd()
		p.inBlock = true
		p.blockLines = nil

	case strings.HasPrefix(line, "%output "):
		rest := line[len("%output "):]
		sp := strings.IndexByte(rest, ' ')
		if sp < 0 {
			return
		}
		pane := rest[:sp]
		if !p.historyDone {
			return // already contained in the capture-pane prefill
		}
		if p.paneID != "" && pane != p.paneID {
			return
		}
		if data := unescapeControl(rest[sp+1:]); len(data) > 0 && p.emit != nil {
			p.emit(data)
		}

	case strings.HasPrefix(line, "%window-pane-changed "):
		// "%window-pane-changed @win %pane" — follow the active pane.
		f := strings.Fields(line)
		if len(f) >= 3 {
			p.paneID = f[2]
		}

	case strings.HasPrefix(line, "%session-window-changed "):
		if p.onWindowChanged != nil {
			p.onWindowChanged()
		}

	default:
		// %exit precedes EOF; %session-changed, %window-add, %layout-change,
		// etc. don't affect a single-pane byte stream.
	}
}

// isBlockEnd matches %end/%error against the %begin identifiers so pane
// content lines that merely start with "%end" can't terminate a block.
func isBlockEnd(line, blockTime, blockNum string) bool {
	if !strings.HasPrefix(line, "%end ") && !strings.HasPrefix(line, "%error ") {
		return false
	}
	f := strings.Fields(line)
	return len(f) >= 3 && f[1] == blockTime && f[2] == blockNum
}

func (p *controlParser) dispatchBlock(isErr bool) {
	lines := p.blockLines
	switch p.blockKind {
	case cmdHistory:
		p.historyDone = true
		if isErr {
			log.Printf("control relay: capture-pane failed: %s", strings.Join(lines, " | "))
			return
		}
		// Drop the unused rows at the bottom of the visible pane so the
		// cursor lands right after the last real line (the prompt).
		end := len(lines)
		for end > 0 && strings.TrimRight(lines[end-1], " ") == "" {
			end--
		}
		if end == 0 {
			return
		}
		payload := strings.Join(lines[:end], "\r\n") + "\x1b[0m"
		if p.emit != nil {
			p.emit([]byte(payload))
		}

	case cmdMeta:
		name := ""
		if !isErr && len(lines) > 0 {
			if i := strings.LastIndex(lines[0], ","); i >= 0 {
				name = lines[0][:i]
				if pane := lines[0][i+1:]; pane != "" {
					p.paneID = pane
				}
			}
		}
		if p.onMeta != nil {
			p.onMeta(name)
		}

	case cmdPane:
		if !isErr && len(lines) > 0 && strings.HasPrefix(lines[0], "%") {
			p.paneID = strings.TrimSpace(lines[0])
		}

	case cmdAlt:
		// If the pane's app was already in the alternate screen buffer when
		// we attached (e.g. Claude Code already running), enter the alt
		// buffer on the client BEFORE the capture-pane frame is painted, so
		// the app's in-place, absolute-positioned live repaints land in the
		// alt buffer and align instead of stacking on the normal buffer.
		// This reply is queued ahead of cmdHistory, so ordering holds.
		if !isErr && len(lines) > 0 && strings.TrimSpace(lines[0]) == "1" && p.emit != nil {
			p.emit([]byte("\x1b[?1049h"))
		}

	default: // cmdConnect, cmdIgnore
		if isErr {
			log.Printf("control relay: command error: %s", strings.Join(lines, " | "))
		}
	}
}

// unescapeControl decodes the visual escaping tmux applies to %output data:
// non-printable and non-ASCII bytes as \ooo octal, backslash as \\ (plus the
// C-style forms for robustness across vis() flag variations).
func unescapeControl(s string) []byte {
	out := make([]byte, 0, len(s))
	for i := 0; i < len(s); {
		c := s[i]
		if c != '\\' || i+1 >= len(s) {
			out = append(out, c)
			i++
			continue
		}
		n := s[i+1]
		if n >= '0' && n <= '7' && i+3 < len(s) && isOctalDigit(s[i+2]) && isOctalDigit(s[i+3]) {
			v := int(n-'0')<<6 | int(s[i+2]-'0')<<3 | int(s[i+3]-'0')
			out = append(out, byte(v))
			i += 4
			continue
		}
		switch n {
		case '\\':
			out = append(out, '\\')
		case 'n':
			out = append(out, '\n')
		case 'r':
			out = append(out, '\r')
		case 't':
			out = append(out, '\t')
		default:
			out = append(out, '\\', n)
		}
		i += 2
	}
	return out
}

func isOctalDigit(b byte) bool { return b >= '0' && b <= '7' }

// hexWords renders bytes as the space-separated hex arguments send-keys -H expects.
func hexWords(b []byte) string {
	var sb strings.Builder
	sb.Grow(len(b) * 3)
	for i, c := range b {
		if i > 0 {
			sb.WriteByte(' ')
		}
		fmt.Fprintf(&sb, "%02x", c)
	}
	return sb.String()
}

// tmuxQuote single-quotes a name for a control-mode command line. Session
// names are sanitized at creation; embedded single quotes are stripped since
// tmux has no in-quote escape for them.
func tmuxQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "") + "'"
}

// relayControlMode bridges a WebSocket to a tmux session via control mode.
// Mirrors RelayWithOptions' lifecycle (kick channel, close codes, teardown)
// but speaks the -C line protocol instead of relaying a PTY byte stream.
func relayControlMode(ctx context.Context, ws *websocket.Conn, client *ssh.Client, sessionName, windowSize string, historyLimit int) error {
	// No PTY: control mode is a line protocol over pipes. A PTY would add
	// input echo and CRLF translation that corrupt parsing.
	session, err := client.NewSession()
	if err != nil {
		return fmt.Errorf("session: %w", err)
	}
	defer session.Close()

	stdin, err := session.StdinPipe()
	if err != nil {
		return fmt.Errorf("stdin pipe: %w", err)
	}
	stdout, err := session.StdoutPipe()
	if err != nil {
		return fmt.Errorf("stdout pipe: %w", err)
	}
	stderr, err := session.StderrPipe()
	if err != nil {
		return fmt.Errorf("stderr pipe: %w", err)
	}

	if windowSize == "" {
		windowSize = "largest"
	}

	// Ensure the session exists with deep scrollback BEFORE attaching, so the
	// control client's new-session -A just attaches to it. history-limit must
	// be set in the SAME tmux invocation that creates the pane (a standalone
	// `set -g` runs in a server that exits before the next command). If the
	// session already exists this is a no-op. Best-effort; done outside the
	// control stream so it can't disturb the connect-reply block ordering.
	if historyLimit > 0 {
		q := tmuxQuote(sessionName)
		_, _ = sshutil.Exec(client, fmt.Sprintf(
			`tmux has-session -t %s 2>/dev/null || tmux set-option -g history-limit %d \; new-session -d -s %s`,
			q, historyLimit, q))
	}

	cmd := fmt.Sprintf(`exec tmux -C new-session -A -s %q`, sessionName)
	if err := session.Start(cmd); err != nil {
		return fmt.Errorf("start tmux -C: %w", err)
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	stripper := &altBufferStripper{}
	parser := &controlParser{}

	var stdinMu sync.Mutex
	sendCmd := func(kind controlCmdKind, command string) {
		// Queue the expected reply before writing so the reader can never
		// see a block with no matching queue entry.
		parser.pushCmd(kind)
		stdinMu.Lock()
		defer stdinMu.Unlock()
		_, _ = io.WriteString(stdin, command+"\n")
	}

	target := tmuxQuote(sessionName)
	metaCh := make(chan string, 1)
	wasKicked := false

	parser.emit = func(data []byte) {
		out := stripper.Process(data)
		if len(out) == 0 {
			return
		}
		_ = ws.Write(ctx, websocket.MessageBinary, out)
	}
	parser.onMeta = func(clientName string) {
		select {
		case metaCh <- clientName:
		default:
		}
	}
	parser.onWindowChanged = func() {
		sendCmd(cmdPane, fmt.Sprintf(`display-message -p -t %s "#{pane_id}"`, target))
	}

	// The reply to the connect command (new-session -A on the tmux command
	// line) is the first block on the wire.
	parser.pushCmd(cmdConnect)

	sendCmd(cmdIgnore, fmt.Sprintf("set-option window-size %s", windowSize))
	sendCmd(cmdMeta, fmt.Sprintf(`display-message -p -t %s "#{client_name},#{pane_id}"`, target))
	// Query alt-buffer state BEFORE the history capture so, if a fullscreen
	// TUI is already running, we can enter the client's alt buffer ahead of
	// painting the captured frame (see cmdAlt in dispatchBlock).
	sendCmd(cmdAlt, fmt.Sprintf(`display-message -p -t %s "#{alternate_on}"`, target))
	sendCmd(cmdHistory, fmt.Sprintf("capture-pane -p -e -J -S - -E - -t %s", target))

	var wg sync.WaitGroup

	// Kick watcher: register under our tmux client_name (the PTY pipeline's
	// tty path equivalent) once display-message answers.
	wg.Add(1)
	go func() {
		defer wg.Done()
		var clientName string
		select {
		case clientName = <-metaCh:
		case <-ctx.Done():
			return
		}
		if clientName == "" {
			return
		}
		ttyMsg, _ := json.Marshal(map[string]string{"type": "tty", "tty": clientName})
		_ = ws.Write(ctx, websocket.MessageText, ttyMsg)

		kickCh := RegisterKickCh(clientName)
		defer UnregisterKickCh(clientName)
		select {
		case <-kickCh:
			wasKicked = true
			kickMsg, _ := json.Marshal(map[string]string{"type": "kicked"})
			_ = ws.Write(ctx, websocket.MessageText, kickMsg)
		case <-ctx.Done():
		}
	}()

	// Control stream -> parser -> WebSocket.
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer cancel()
		br := bufio.NewReaderSize(stdout, 64*1024)
		for {
			line, err := br.ReadString('\n')
			if len(line) > 0 {
				parser.feedLine(strings.TrimRight(line, "\r\n"))
			}
			if err != nil {
				return
			}
		}
	}()

	// stderr -> WebSocket so failures ("tmux: command not found", version
	// errors) are visible in the browser instead of a silent blank screen.
	wg.Add(1)
	go func() {
		defer wg.Done()
		buf := make([]byte, 4096)
		for {
			n, err := stderr.Read(buf)
			if n > 0 {
				msg := strings.ReplaceAll(string(buf[:n]), "\n", "\r\n")
				_ = ws.Write(ctx, websocket.MessageBinary, []byte(msg))
			}
			if err != nil {
				return
			}
		}
	}()

	// WebSocket -> control commands. Keystrokes become send-keys -H so the
	// bytes reach the pane exactly as typed; resize becomes refresh-client -C
	// (a control client's size is whatever it declares, no PTY involved).
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer cancel()
		for {
			typ, data, err := ws.Read(ctx)
			if err != nil {
				return
			}
			if typ == websocket.MessageText {
				var msg resizeMsg
				if json.Unmarshal(data, &msg) == nil && msg.Type == "resize" {
					if msg.Cols > 0 && msg.Rows > 0 {
						sendCmd(cmdIgnore, fmt.Sprintf("refresh-client -C %dx%d", msg.Cols, msg.Rows))
					}
					continue
				}
			}
			if len(data) == 0 {
				continue
			}
			const chunk = 512
			for off := 0; off < len(data); off += chunk {
				end := off + chunk
				if end > len(data) {
					end = len(data)
				}
				sendCmd(cmdIgnore, "send-keys -t "+target+" -H "+hexWords(data[off:end]))
			}
		}
	}()

	// Same teardown contract as the PTY pipeline: if the WebSocket drops,
	// close the SSH session so the remote tmux -C client exits.
	waitCh := make(chan error, 1)
	go func() {
		waitCh <- session.Wait()
	}()

	select {
	case err = <-waitCh:
	case <-ctx.Done():
		session.Close()
		err = <-waitCh
	}

	cancel()
	wg.Wait()

	if err != nil {
		log.Printf("control relay session ended: %v", err)
	}

	closeCode := websocket.StatusCode(4000) // session ended
	closeMsg := "session ended"
	if wasKicked {
		closeCode = 4001
		closeMsg = "detached by another client"
	}
	closeCtx, closeCancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer closeCancel()
	ws.Close(closeCode, closeMsg)
	<-closeCtx.Done()

	return nil
}
