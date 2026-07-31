package tmux

import (
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestRemotePathExpandsTildeOutsideQuotes(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"", ""},
		{"  ", ""},
		{"~", `"$HOME"`},
		{"~/", `"$HOME"`},
		{"~/sessions/api", `"$HOME"/'sessions/api'`},
		{"/srv/work", `'/srv/work'`},
		{"/srv/my work", `'/srv/my work'`},
		// A literal $ in a path must not be expanded by the remote shell.
		{"/srv/$USER", `'/srv/$USER'`},
		{"~/it's", `"$HOME"/'it'\''s'`},
	}
	for _, c := range cases {
		if got := remotePath(c.in); got != c.want {
			t.Errorf("remotePath(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestBuildCreateCmdPlain(t *testing.T) {
	got := buildCreateCmd("work", CreateOptions{})
	if strings.Contains(got, "mkdir") {
		t.Errorf("no cwd should mean no mkdir: %q", got)
	}
	if strings.Contains(got, "send-keys") {
		t.Errorf("no command should mean no send-keys: %q", got)
	}
	if !strings.Contains(got, `new-session -d -s "work"`) {
		t.Errorf("missing new-session: %q", got)
	}
}

func TestBuildCreateCmdCreateDirRunsBeforeTmux(t *testing.T) {
	got := buildCreateCmd("work", CreateOptions{Cwd: "~/sessions/api", CreateDir: true})
	mkdir := strings.Index(got, "mkdir -p")
	tmuxIdx := strings.Index(got, "tmux ")
	if mkdir < 0 || tmuxIdx < 0 || mkdir > tmuxIdx {
		t.Fatalf("mkdir must precede tmux: %q", got)
	}
	// && so a failed mkdir aborts instead of starting the session in $HOME.
	if !strings.Contains(got, `mkdir -p "$HOME"/'sessions/api' && tmux`) {
		t.Errorf("mkdir not chained with &&: %q", got)
	}
	if !strings.Contains(got, `-c "$HOME"/'sessions/api'`) {
		t.Errorf("missing -c working dir: %q", got)
	}
}

func TestBuildCreateCmdWithoutCreateDirHasNoMkdir(t *testing.T) {
	got := buildCreateCmd("work", CreateOptions{Cwd: "~/sessions/api"})
	if strings.Contains(got, "mkdir") {
		t.Errorf("CreateDir=false must not mkdir: %q", got)
	}
	if !strings.Contains(got, `-c "$HOME"/'sessions/api'`) {
		t.Errorf("missing -c working dir: %q", got)
	}
}

func TestBuildCreateCmdSendsCommandAfterCreate(t *testing.T) {
	got := buildCreateCmd("work", CreateOptions{Cwd: "~/p", Command: "claude --resume"})
	newSess := strings.Index(got, "new-session")
	send := strings.Index(got, "send-keys")
	if newSess < 0 || send < 0 || send < newSess {
		t.Fatalf("send-keys must follow new-session: %q", got)
	}
	if !strings.Contains(got, `send-keys -t "work" 'claude --resume' Enter`) {
		t.Errorf("command not single-quoted for the pane: %q", got)
	}
}

// The launch command must wait for the shell to draw its prompt before
// typing (issue #75): send-keys chained straight into the create fires
// before bash owns the tty, and the TUI then draws over the prompt line.
func TestBuildCreateCmdWaitsForPromptBeforeCommand(t *testing.T) {
	got := buildCreateCmd("work", CreateOptions{Command: "claude"})
	wait := strings.Index(got, "cursor_x")
	send := strings.Index(got, "send-keys")
	if wait < 0 {
		t.Fatalf("missing cursor_x prompt wait: %q", got)
	}
	if send < wait {
		t.Errorf("send-keys must come after the prompt wait: %q", got)
	}
	// The wait must be OUTSIDE the atomic tmux "\;" chain — a plain shell
	// sequence after it. "\; send-keys" would put it back in the race.
	if strings.Contains(got, `\; send-keys`) {
		t.Errorf("send-keys is still chained into the tmux invocation: %q", got)
	}
	// Timeout path sends anyway: the send-keys is joined with ";" not "&&".
	if !strings.Contains(got, `done; tmux send-keys`) {
		t.Errorf("send-keys must run even when the wait times out: %q", got)
	}
}

func TestBuildCreateCmdCommandStaysLiteral(t *testing.T) {
	// The command is typed into the pane, so the OUTER shell must not
	// expand it — $(...) and backticks stay text.
	got := buildCreateCmd("work", CreateOptions{Command: "echo $(whoami) `id`"})
	if !strings.Contains(got, `'echo $(whoami) `+"`id`"+`'`) {
		t.Errorf("command not kept literal: %q", got)
	}
}

func TestBuildCreateCmdBlankCommandIgnored(t *testing.T) {
	if got := buildCreateCmd("work", CreateOptions{Command: "   "}); strings.Contains(got, "send-keys") {
		t.Errorf("whitespace-only command should be dropped: %q", got)
	}
}

func TestBuildCreateCmdMousePerSession(t *testing.T) {
	got := buildCreateCmd("work", CreateOptions{Mouse: true})
	// Per-session -t, never -g: a global write would leak mouse mode to
	// sessions ssh-to-go did not create (issue #78).
	if !strings.Contains(got, `set-option -t "work" mouse on`) {
		t.Errorf("missing per-session mouse option: %q", got)
	}
	if strings.Contains(got, "-g mouse") {
		t.Errorf("mouse must not be set globally: %q", got)
	}
	newSess := strings.Index(got, "new-session")
	mouse := strings.Index(got, "mouse on")
	if mouse < newSess {
		t.Errorf("mouse option must follow new-session so the session exists: %q", got)
	}
}

func TestBuildCreateCmdMouseOffByDefault(t *testing.T) {
	if got := buildCreateCmd("work", CreateOptions{}); strings.Contains(got, "mouse") {
		t.Errorf("Mouse=false must not touch the mouse option: %q", got)
	}
}

func TestBuildCreateCmdKeepsHistoryLimitFirst(t *testing.T) {
	got := buildCreateCmd("work", CreateOptions{HistoryLimit: 50000, Cwd: "~/p", CreateDir: true})
	hist := strings.Index(got, "history-limit")
	newSess := strings.Index(got, "new-session")
	if hist < 0 || hist > newSess {
		t.Fatalf("history-limit must precede new-session so the pane inherits it: %q", got)
	}
}

// TestBuildCreateCmdAgainstRealTmux runs the generated command against a real
// tmux on an isolated socket: the directory must be created, the pane must
// start there, and the command must land in the shell.
func TestBuildCreateCmdAgainstRealTmux(t *testing.T) {
	if _, err := exec.LookPath("tmux"); err != nil {
		t.Skip("tmux not installed")
	}
	sock := "s2gtest" + strconv.Itoa(os.Getpid())
	base := t.TempDir()
	dir := filepath.Join(base, "made", "by", "create")
	defer exec.Command("tmux", "-L", sock, "kill-server").Run()

	cmd := buildCreateCmd("createtest", CreateOptions{
		Cwd:          dir,
		CreateDir:    true,
		Command:      "echo hello-from-command",
		HistoryLimit: 1000,
		Mouse:        true,
	})
	// The remote runs this through a shell; -L keeps it off the user's server.
	cmd = strings.ReplaceAll(cmd, "tmux ", "tmux -L "+sock+" ")
	if out, err := exec.Command("sh", "-c", cmd).CombinedOutput(); err != nil {
		t.Fatalf("run %q: %v\n%s", cmd, err, out)
	}

	if _, err := os.Stat(dir); err != nil {
		t.Errorf("CreateDir did not make %s: %v", dir, err)
	}
	paneDir, err := exec.Command("tmux", "-L", sock, "display-message", "-p", "-t", "createtest", "#{pane_current_path}").Output()
	if err != nil {
		t.Fatalf("query pane path: %v", err)
	}
	if got := strings.TrimSpace(string(paneDir)); got != dir {
		t.Errorf("pane started in %q, want %q", got, dir)
	}

	// The shell needs a moment to consume the sent keys and echo the output.
	var pane string
	for i := 0; i < 40; i++ {
		out, _ := exec.Command("tmux", "-L", sock, "capture-pane", "-p", "-t", "createtest").Output()
		pane = string(out)
		if strings.Contains(pane, "hello-from-command") {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	if !strings.Contains(pane, "hello-from-command") {
		t.Errorf("command never ran in the pane; capture:\n%s", pane)
	}
	// The session must OUTLIVE the command — that is the point of send-keys.
	if err := exec.Command("tmux", "-L", sock, "has-session", "-t", "createtest").Run(); err != nil {
		t.Errorf("session died with the command: %v", err)
	}

	// Mouse mode must land on THIS session and stay off the server default.
	sessMouse, _ := exec.Command("tmux", "-L", sock, "show-options", "-t", "createtest", "mouse").Output()
	if !strings.Contains(string(sessMouse), "mouse on") {
		t.Errorf("session mouse option = %q, want mouse on", sessMouse)
	}
	globalMouse, _ := exec.Command("tmux", "-L", sock, "show-options", "-g", "mouse").Output()
	if strings.Contains(string(globalMouse), "mouse on") {
		t.Errorf("global mouse option leaked on: %q", globalMouse)
	}
}

// tmux quietly falls back to $HOME when -c names a missing directory, so
// without CreateDir we must check the path ourselves and fail instead.
func TestBuildCreateCmdWithoutCreateDirGuardsMissingPath(t *testing.T) {
	got := buildCreateCmd("work", CreateOptions{Cwd: "/srv/gone"})
	if !strings.Contains(got, "[ -d '/srv/gone' ]") {
		t.Errorf("missing existence guard: %q", got)
	}
	guard := strings.Index(got, "[ -d")
	tmuxIdx := strings.Index(got, "tmux ")
	if guard < 0 || guard > tmuxIdx {
		t.Errorf("guard must precede tmux: %q", got)
	}
}

func TestCreateGuardRejectsMissingDirWithRealTmux(t *testing.T) {
	if _, err := exec.LookPath("tmux"); err != nil {
		t.Skip("tmux not installed")
	}
	sock := "s2gguard" + strconv.Itoa(os.Getpid())
	defer exec.Command("tmux", "-L", sock, "kill-server").Run()
	missing := filepath.Join(t.TempDir(), "not", "there")

	cmd := buildCreateCmd("guardtest", CreateOptions{Cwd: missing})
	cmd = strings.ReplaceAll(cmd, "tmux ", "tmux -L "+sock+" ")
	out, err := exec.Command("sh", "-c", cmd).CombinedOutput()
	if err == nil {
		t.Fatalf("expected failure for missing dir, got success: %s", out)
	}
	if !strings.Contains(string(out), "working directory does not exist") {
		t.Errorf("unhelpful error: %s", out)
	}
	if exec.Command("tmux", "-L", sock, "has-session", "-t", "guardtest").Run() == nil {
		t.Error("session was created despite the missing directory")
	}
}

func TestGuardFailureExtractsCleanMessage(t *testing.T) {
	// What sshutil.Exec actually produces: the whole command echoed back.
	raw := `exec "{ [ -d '/srv/gone' ] || { echo 'working directory does not exist: '/srv/gone >&2; exit 3; }; } && tmux new-session ..." : Process exited with status 3 (output: working directory does not exist: /srv/gone
)`
	got := guardFailure(raw)
	if got != "working directory does not exist: /srv/gone" {
		t.Errorf("guardFailure = %q", got)
	}
	if guardFailure("some unrelated ssh failure") != "" {
		t.Error("guardFailure must not match unrelated errors")
	}
}
