package relay

import "bytes"

// altBufferStripper is a stateful byte-stream filter that removes the
// mouse-tracking DECSET escape sequences that would otherwise make the
// client forward wheel events to tmux as mouse-button reports (tmux then
// falls back to its 5-line-per-wheel copy-mode scroll, which the user
// perceives as jerky).
//
// Only ONE family of sequences is stripped:
//
//   - Mouse-tracking enables. tmux with mouse mode on, and most TUIs,
//     send DECSET 1000/1002/1003/etc to ask the terminal to report wheel
//     and click events as escape codes. Stripping these means the client
//     never enters mouse-tracking mode, so its native scroll handler
//     (xterm.js viewport scrollbar, Android smooth-scroll view) keeps
//     working for line-oriented shell output. Sequences: \x1b[?9h/l,
//     ?1000–?1003 h/l, ?1005/?1006/?1015 h/l.
//
// Alt-screen buffer switches (\x1b[?47h/l, ?1047h/l, ?1049h/l) are NO
// LONGER stripped, and this is deliberate. Fullscreen TUIs (claude code,
// vim, htop, less, …) repaint their whole frame IN PLACE using absolute
// cursor addressing and partial line-clears, on the assumption they own an
// alternate screen buffer. When we stripped the ?1049h switch, all that
// in-place drawing landed on the client's normal (scrolling) buffer, where
// successive frames stacked on top of each other instead of replacing —
// producing duplicated lines, box-rules struck through text, and dropped
// characters (issue #59). Passing the switch through lets xterm.js /
// TerminalView follow the app into its alt buffer and render each frame
// cleanly, exactly like a native tmux client. control.go additionally
// syncs the alt-buffer state on attach (via #{alternate_on}) so a TUI that
// was ALREADY running when the browser connects is entered correctly.
//
// Trade-off: while a fullscreen app is active the client is in the alt
// buffer, which has no scrollback — wheel scrolls the app (via xterm's
// alternate-scroll → arrow keys), not local history. That matches native
// terminal behavior and is the correct contract for a fullscreen app.
//
// Other CSI ? sequences pass through untouched.
type altBufferStripper struct {
	// pending holds the tail of the previous write when it might be the
	// start of a sequence we'd want to strip; we resume scanning from
	// there on the next Process call so sequences split across reads
	// still match.
	pending []byte
}

// maxSeqLen is the longest sequence we strip ("\x1b[?1000h" / "...l" = 8 bytes).
const maxSeqLen = 8

var stripSequences = [][]byte{
	// Mouse-tracking enable/disable. Alt-screen buffer switches are
	// intentionally NOT stripped — see the type doc above.
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
