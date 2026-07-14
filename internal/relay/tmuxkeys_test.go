package relay

import "testing"

// The MCP interactive-TUI tools rely on these exported wrappers producing the
// exact same encoding as the control-mode relay's internal helpers.
func TestExportedWrappers(t *testing.T) {
	if HexWords([]byte("hi\r")) != hexWords([]byte("hi\r")) {
		t.Error("HexWords must delegate to hexWords")
	}
	if got := HexWords([]byte("Enter")); got != "45 6e 74 65 72" {
		t.Errorf("HexWords(\"Enter\") = %q; literal text must be hex, not a key name", got)
	}
	if TmuxQuote("my'sess") != tmuxQuote("my'sess") {
		t.Error("TmuxQuote must delegate to tmuxQuote")
	}
}
