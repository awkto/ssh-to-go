package relay

// Exported wrappers so other packages (notably the MCP interactive-TUI tools:
// send_keys / read_pane / wait_for_pane) can reuse the exact send-keys hex
// encoding and session-name quoting the control-mode relay already uses,
// instead of reimplementing them and risking a subtly different contract.

// HexWords renders bytes as the space-separated hex arguments `send-keys -H`
// expects. See hexWords.
func HexWords(b []byte) string { return hexWords(b) }

// TmuxQuote single-quotes a tmux target (session) name for a command line.
// See tmuxQuote.
func TmuxQuote(s string) string { return tmuxQuote(s) }
