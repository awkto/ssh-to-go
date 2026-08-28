package relay

import "bytes"

// mouseModeFilter is a stateful byte-stream filter that decides which
// mouse-tracking DECSET escape sequences reach the client. The goal:
// line-oriented shell output keeps the client's native (local scrollback)
// wheel scrolling, while fullscreen TUIs get real mouse events.
//
// The rule is alt-buffer-aware:
//
//   - Mouse-tracking ENABLES (\x1b[?9h, ?1000–?1003h, ?1005/?1006/?1015h)
//     arriving while the pane is on the NORMAL buffer are stripped. Without
//     this the client would enter mouse-tracking mode and forward wheel
//     events to the pane as mouse reports (tmux then falls back to its
//     5-line-per-wheel copy-mode scroll, which the user perceives as jerky),
//     instead of scrolling its own scrollback smoothly.
//
//   - While the pane is on the ALTERNATE buffer (a fullscreen TUI: opencode,
//     vim, htop, less, …), mouse-tracking sequences PASS THROUGH. The alt
//     buffer has no scrollback, so xterm.js's fallback for a wheel event
//     with no mouse mode is to translate it into arrow keys — which a TUI
//     like opencode interprets as prompt-history navigation instead of
//     scrolling its feed. Letting the TUI's mouse enables through means the
//     wheel reaches the app as real mouse reports, exactly like a native
//     terminal, and the app scrolls its own content. This is the correct
//     contract for a fullscreen app.
//
//   - Mouse-tracking DISABLES always pass through: they are a no-op on a
//     client that never enabled tracking, and they guarantee tracking is
//     switched off when the TUI asks for it.
//
//   - Enables stripped on the normal buffer are remembered and REPLAYED
//     when the app switches to the alt buffer, covering apps that enable
//     mouse reporting before (or without an observable order against) the
//     ?1049h switch.
//
//   - When the alt buffer is exited, disables are synthesized for any modes
//     still live on the client. A TUI that is killed (or crashes) before
//     sending its own disables must not leave the client stuck in mouse
//     mode on the normal buffer — that would silently kill local scrolling
//     for the shell the user lands back in.
//
// Alt-screen buffer switches themselves (\x1b[?47h/l, ?1047h/l, ?1049h/l)
// are NOT stripped, and this is deliberate. Fullscreen TUIs repaint their
// whole frame IN PLACE using absolute cursor addressing and partial
// line-clears, on the assumption they own an alternate screen buffer. When
// we stripped the ?1049h switch, all that in-place drawing landed on the
// client's normal (scrolling) buffer, where successive frames stacked on
// top of each other instead of replacing — producing duplicated lines,
// box-rules struck through text, and dropped characters (issue #59).
// Passing the switch through lets xterm.js / TerminalView follow the app
// into its alt buffer and render each frame cleanly, exactly like a native
// tmux client. control.go additionally syncs the alt-buffer state (and now
// the pane's mouse flags) on attach, so a TUI that was ALREADY running
// when the browser connects is entered correctly.
//
// Other CSI ? sequences pass through untouched.
type mouseModeFilter struct {
	// pending holds the tail of the previous write when it might be the
	// start of a sequence we track; we resume scanning from there on the
	// next Process call so sequences split across reads still match.
	pending []byte

	// alt is true while the pane is on the alternate screen buffer.
	alt bool

	// stripped are mouse-enable sequences dropped while on the normal
	// buffer, deduped. Replayed on alt-buffer entry.
	stripped [][]byte

	// live are mouse modes currently enabled on the client (passed through
	// or replayed). Force-disabled on alt-buffer exit.
	live [][]byte
}

// maxSeqLen is the longest sequence we track ("\x1b[?1049h" = 8 bytes).
const maxSeqLen = 8

var (
	altEnterSeqs = [][]byte{
		[]byte("\x1b[?47h"),
		[]byte("\x1b[?1047h"),
		[]byte("\x1b[?1049h"),
	}
	altExitSeqs = [][]byte{
		[]byte("\x1b[?47l"),
		[]byte("\x1b[?1047l"),
		[]byte("\x1b[?1049l"),
	}
	mouseEnableSeqs = [][]byte{
		[]byte("\x1b[?9h"),
		[]byte("\x1b[?1000h"),
		[]byte("\x1b[?1001h"),
		[]byte("\x1b[?1002h"),
		[]byte("\x1b[?1003h"),
		[]byte("\x1b[?1005h"),
		[]byte("\x1b[?1006h"),
		[]byte("\x1b[?1015h"),
		[]byte("\x1b[?1016h"),
	}
	mouseDisableSeqs = [][]byte{
		[]byte("\x1b[?9l"),
		[]byte("\x1b[?1000l"),
		[]byte("\x1b[?1001l"),
		[]byte("\x1b[?1002l"),
		[]byte("\x1b[?1003l"),
		[]byte("\x1b[?1005l"),
		[]byte("\x1b[?1006l"),
		[]byte("\x1b[?1015l"),
		[]byte("\x1b[?1016l"),
	}
)

// enableForDisable maps a mouse disable sequence to its enable counterpart
// (same index in mouseEnableSeqs/mouseDisableSeqs).
func enableForDisable(disable []byte) []byte {
	for i, d := range mouseDisableSeqs {
		if bytes.Equal(d, disable) {
			return mouseEnableSeqs[i]
		}
	}
	return nil
}

func disableForEnable(enable []byte) []byte {
	for i, e := range mouseEnableSeqs {
		if bytes.Equal(e, enable) {
			return mouseDisableSeqs[i]
		}
	}
	return nil
}

func containsSeq(list [][]byte, seq []byte) bool {
	for _, s := range list {
		if bytes.Equal(s, seq) {
			return true
		}
	}
	return false
}

func removeSeq(list [][]byte, seq []byte) [][]byte {
	out := list[:0]
	for _, s := range list {
		if !bytes.Equal(s, seq) {
			out = append(out, s)
		}
	}
	return out
}

// Process consumes a chunk of bytes from the upstream stream and returns
// the bytes that should be forwarded to the client. If the trailing bytes
// could be the start of a sequence we track, they're held back until the
// next call.
func (s *mouseModeFilter) Process(in []byte) []byte {
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

		seq, kind := matchSeq(buf[i:])
		if kind != seqNone {
			i += len(seq)
			switch kind {
			case seqAltEnter:
				s.alt = true
				out = append(out, seq...)
				// An app that enabled mouse tracking before switching
				// buffers had those enables stripped; replay them now the
				// client can honour them.
				for _, en := range s.stripped {
					out = append(out, en...)
					s.live = appendLive(s.live, en)
				}
				s.stripped = nil
			case seqAltExit:
				out = append(out, seq...)
				// Guarantee mouse tracking dies with the alt buffer, even
				// if the app was killed before sending its own disables.
				for _, en := range s.live {
					out = append(out, disableForEnable(en)...)
				}
				s.live = nil
				s.stripped = nil
				s.alt = false
			case seqMouseEnable:
				if s.alt {
					out = append(out, seq...)
					s.live = appendLive(s.live, seq)
				} else if !containsSeq(s.stripped, seq) {
					s.stripped = append(s.stripped, append([]byte(nil), seq...))
				}
			case seqMouseDisable:
				// Always pass: no-op when tracking was never enabled, and
				// it keeps the client's state machine in step with the app.
				out = append(out, seq...)
				s.live = removeSeq(s.live, enableForDisable(seq))
				s.stripped = removeSeq(s.stripped, enableForDisable(seq))
			}
			continue
		}

		// Could the bytes from i still grow into a tracked sequence? If so,
		// hold them back for the next Process call.
		remaining := len(buf) - i
		if remaining < maxSeqLen {
			if mightBeTrackedPrefix(buf[i:]) {
				s.pending = append(s.pending, buf[i:]...)
				return out
			}
		}

		// Not a tracked sequence and not a deferrable partial: emit ESC and
		// continue. The next iteration will handle whatever follows.
		out = append(out, buf[i])
		i++
	}
	return out
}

func appendLive(live [][]byte, en []byte) [][]byte {
	if containsSeq(live, en) {
		return live
	}
	return append(live, append([]byte(nil), en...))
}

type seqKind int

const (
	seqNone seqKind = iota
	seqAltEnter
	seqAltExit
	seqMouseEnable
	seqMouseDisable
)

// matchSeq returns the tracked sequence at the start of b (which begins
// with ESC), or seqNone. Longer sequences are listed first within each
// family so e.g. ?1049h wins over any shorter shared prefix.
func matchSeq(b []byte) ([]byte, seqKind) {
	for _, seq := range altEnterSeqs {
		if len(b) >= len(seq) && bytes.Equal(b[:len(seq)], seq) {
			return seq, seqAltEnter
		}
	}
	for _, seq := range altExitSeqs {
		if len(b) >= len(seq) && bytes.Equal(b[:len(seq)], seq) {
			return seq, seqAltExit
		}
	}
	for _, seq := range mouseEnableSeqs {
		if len(b) >= len(seq) && bytes.Equal(b[:len(seq)], seq) {
			return seq, seqMouseEnable
		}
	}
	for _, seq := range mouseDisableSeqs {
		if len(b) >= len(seq) && bytes.Equal(b[:len(seq)], seq) {
			return seq, seqMouseDisable
		}
	}
	return nil, seqNone
}

// mightBeTrackedPrefix returns true if any tracked sequence starts with the
// given bytes — i.e., this could grow into a full match.
func mightBeTrackedPrefix(b []byte) bool {
	if len(b) == 0 {
		return false
	}
	families := [][][]byte{altEnterSeqs, altExitSeqs, mouseEnableSeqs, mouseDisableSeqs}
	for _, fam := range families {
		for _, seq := range fam {
			if len(b) < len(seq) && bytes.HasPrefix(seq, b) {
				return true
			}
		}
	}
	return false
}
