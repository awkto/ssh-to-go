package relay

import "bytes"

// altBufferStripper is a stateful byte-stream filter that removes the
// xterm/VT escape sequences that would prevent a native client from
// scrolling its own local scrollback smoothly. Without this filter the
// client emulator (Android TerminalView, or xterm.js when it opts into
// mouse=off) ends up forwarding wheel events to tmux as mouse-button
// reports and tmux falls back to its 5-line-per-wheel copy-mode scroll,
// which the user perceives as jerky.
//
// Two families of sequences are stripped:
//
//   - Alt-screen buffer switches. TUI programs (claude code, vim, htop,
//     less, …) emit \x1b[?1049h on attach which flips the emulator to
//     the alt buffer — a buffer with NO scrollback by design — instantly
//     killing the user's ability to scroll into the history that was
//     just prefilled via tmux capture-pane. Sequences: \x1b[?47h/l,
//     \x1b[?1047h/l, \x1b[?1049h/l.
//
//   - Mouse-tracking enables. tmux with mouse mode on, and most TUIs,
//     send DECSET 1000/1002/1003/etc to ask the terminal to report wheel
//     and click events as escape codes. Stripping these means the client
//     never enters mouse-tracking mode, so its native scroll handler
//     (xterm.js viewport scrollbar, Android smooth-scroll view) keeps
//     working. Sequences: \x1b[?9h/l, ?1000–?1003 h/l, ?1005/?1006/?1015
//     h/l.
//
// Trade-offs: TUI apps' mouse interactions (vim with `:set mouse=a`,
// htop click-to-focus, etc.) stop working — acceptable for our
// scroll-first use case. When a TUI exits, the shell prompt isn't
// auto-restored; the TUI's last frame stays in scrollback. Cosmetic.
//
// Other CSI ? sequences pass through untouched.
type altBufferStripper struct {
	// pending holds the tail of the previous write when it might be the
	// start of a sequence we'd want to strip; we resume scanning from
	// there on the next Process call so sequences split across reads
	// still match.
	pending []byte
}

// maxSeqLen is the longest sequence we strip ("\x1b[?1049h" / "...l" = 8 bytes).
const maxSeqLen = 8

var stripSequences = [][]byte{
	// Alt-screen buffer switches.
	[]byte("\x1b[?47h"),
	[]byte("\x1b[?47l"),
	[]byte("\x1b[?1047h"),
	[]byte("\x1b[?1047l"),
	[]byte("\x1b[?1049h"),
	[]byte("\x1b[?1049l"),
	// Mouse-tracking enable/disable.
	[]byte("\x1b[?9h"),
	[]byte("\x1b[?9l"),
	[]byte("\x1b[?1000h"),
	[]byte("\x1b[?1000l"),
	[]byte("\x1b[?1001h"),
	[]byte("\x1b[?1001l"),
	[]byte("\x1b[?1002h"),
	[]byte("\x1b[?1002l"),
	[]byte("\x1b[?1003h"),
	[]byte("\x1b[?1003l"),
	[]byte("\x1b[?1005h"),
	[]byte("\x1b[?1005l"),
	[]byte("\x1b[?1006h"),
	[]byte("\x1b[?1006l"),
	[]byte("\x1b[?1015h"),
	[]byte("\x1b[?1015l"),
}

// Process consumes a chunk of bytes from the upstream stream and returns
// the bytes that should be forwarded to the client. If the trailing bytes
// could be the start of a sequence we want to strip, they're held back
// until the next call.
func (s *altBufferStripper) Process(in []byte) []byte {
	var buf []byte
	if len(s.pending) > 0 {
		buf = make([]byte, 0, len(s.pending)+len(in))
		buf = append(buf, s.pending...)
		buf = append(buf, in...)
		s.pending = nil
	} else {
		buf = in
	}

	out := make([]byte, 0, len(buf))
	i := 0
	for i < len(buf) {
		if buf[i] != 0x1b {
			out = append(out, buf[i])
			i++
			continue
		}

		// Possible start of an ESC sequence. Look for an exact full match
		// among stripSequences; if found, drop it.
		matched := false
		for _, seq := range stripSequences {
			if i+len(seq) <= len(buf) && bytes.Equal(buf[i:i+len(seq)], seq) {
				i += len(seq)
				matched = true
				break
			}
		}
		if matched {
			continue
		}

		// Could the bytes from i still grow into one of the strip
		// sequences? If so, hold them back for the next Process call.
		remaining := len(buf) - i
		if remaining < maxSeqLen {
			if mightBeStripPrefix(buf[i:]) {
				s.pending = append(s.pending, buf[i:]...)
				return out
			}
		}

		// Not a strippable sequence and not a deferrable partial: emit ESC
		// and continue. The next iteration will handle whatever follows.
		out = append(out, buf[i])
		i++
	}
	return out
}

// mightBeStripPrefix returns true if any of stripSequences starts with the
// given bytes — i.e., this could grow into a full match.
func mightBeStripPrefix(b []byte) bool {
	if len(b) == 0 {
		return false
	}
	for _, seq := range stripSequences {
		if len(b) < len(seq) && bytes.HasPrefix(seq, b) {
			return true
		}
	}
	return false
}
