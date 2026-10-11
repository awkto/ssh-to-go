package relay

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// The set-buffer line must hand tmux the text verbatim. Run it through a
// private tmux server and read the buffer back, so the escaping is checked
// against tmux's real parser rather than a model of it.
func TestPasteCmdsRoundTrip(t *testing.T) {
	if _, err := exec.LookPath("tmux"); err != nil {
		t.Skip("tmux not installed")
	}
	sock := filepath.Join(t.TempDir(), "s")
	tm := func(args ...string) (string, error) {
		cmd := exec.Command("tmux", append([]string{"-S", sock, "-f", "/dev/null"}, args...)...)
		cmd.Env = append(os.Environ(), "TMUX=")
		out, err := cmd.CombinedOutput()
		return string(out), err
	}
	if out, err := tm("new-session", "-d", "-s", "t"); err != nil {
		t.Fatalf("start tmux: %v %s", err, out)
	}
	defer tm("kill-server")

	cases := map[string]string{
		"/home/u/.ssh-to-go/paste/20261012-001128-213eff.png ":       "/home/u/.ssh-to-go/paste/20261012-001128-213eff.png ",
		`quote " back \ dollar $HOME tilde ~ hash #{pane_id} ; semi`: `quote " back \ dollar $HOME tilde ~ hash #{pane_id} ; semi`,
		"single ' quote, café":                                       "single ' quote, café",
		"line1\nline2\r\x1b[31m":                                     "line1line2[31m", // control chars dropped
	}
	for in, want := range cases {
		cmds := pasteCmds("t", in)
		if len(cmds) != 2 || !strings.HasPrefix(cmds[1], "paste-buffer -p -d ") {
			t.Fatalf("unexpected commands %q", cmds)
		}
		// Control-mode command lines go through the same parser as a
		// config file, so source one holding the line.
		f := filepath.Join(t.TempDir(), "cmd.conf")
		os.WriteFile(f, []byte(cmds[0]+"\n"), 0o600)
		if out, err := tm("source-file", f); err != nil {
			t.Fatalf("%q: set-buffer failed: %v %s", in, err, out)
		}
		got, err := tm("show-buffer", "-b", "ssh-to-go-paste")
		if err != nil {
			t.Fatalf("%q: show-buffer: %v %s", in, err, got)
		}
		if got != want {
			t.Errorf("%q: buffer holds %q, want %q", in, got, want)
		}
	}

	if cmds := pasteCmds("t", "\r\n"); cmds != nil {
		t.Errorf("text with nothing printable should produce no commands, got %q", cmds)
	}
}
