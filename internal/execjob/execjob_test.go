package execjob

import (
	"encoding/base64"
	"regexp"
	"strings"
	"testing"
	"time"
)

func TestNewIDUnique(t *testing.T) {
	re := regexp.MustCompile(`^[0-9a-f]{12}$`)
	seen := map[string]bool{}
	for i := 0; i < 1000; i++ {
		id := NewID()
		if !re.MatchString(id) {
			t.Fatalf("id %q is not 12 hex chars", id)
		}
		if seen[id] {
			t.Fatalf("duplicate id %q", id)
		}
		seen[id] = true
	}
}

func mustStartScript(t *testing.T, id string, spec RunSpec) string {
	t.Helper()
	s, err := StartScript(id, spec)
	if err != nil {
		t.Fatalf("StartScript: %v", err)
	}
	return s
}

// decodeEmbedded finds the launcher line that writes the given file and
// returns its decoded content.
func decodeEmbedded(t *testing.T, script, file string) string {
	t.Helper()
	for _, line := range strings.Split(script, "\n") {
		if strings.Contains(line, `> "$D/`+file+`"`) && strings.Contains(line, "base64 -d") {
			start := strings.Index(line, "'")
			end := strings.Index(line[start+1:], "'") + start + 1
			dec, err := base64.StdEncoding.DecodeString(line[start+1 : end])
			if err != nil {
				t.Fatalf("decode %s: %v", file, err)
			}
			return string(dec)
		}
	}
	t.Fatalf("no launcher line writes %s:\n%s", file, script)
	return ""
}

func TestStartScriptEncodesCommand(t *testing.T) {
	cmd := "echo 'hi there'; curl http://x/$FOO && echo \"done\"\nsecond line"
	s := mustStartScript(t, "abc123", RunSpec{Command: cmd})

	// The command must be transferred base64-encoded so arbitrary shell
	// metacharacters and newlines can't break out of the launcher.
	if got := decodeEmbedded(t, s, "cmd.sh"); got != cmd {
		t.Fatalf("cmd.sh roundtrip: got %q want %q", got, cmd)
	}
	// Raw dangerous fragments must NOT appear unencoded.
	if strings.Contains(s, "curl http://x/$FOO") {
		t.Fatalf("raw command leaked into script:\n%s", s)
	}
	// Uses a detached tmux session named with the job id.
	if !strings.Contains(s, "tmux new-session -d -s '"+SessionName("abc123")+"'") {
		t.Fatalf("start script missing detached tmux launch:\n%s", s)
	}
	// Artifacts must be private: umask before any file is written.
	if !strings.HasPrefix(s, "umask 077") {
		t.Fatalf("launcher must set umask 077 first:\n%s", s)
	}
}

func TestRunnerScriptHardening(t *testing.T) {
	s := mustStartScript(t, "id1", RunSpec{Command: "sleep 100", TimeoutSecs: 30})
	runner := decodeEmbedded(t, s, "run.sh")

	for _, want := range []string{
		"umask 077",                   // private artifacts
		`< "$IN"`,                     // stdin never the pane TTY
		"timeout -k 10 30",            // SIGKILL escalation
		`printf %s "$pid" > "$D/pid"`, // liveness for kill/crash detection
		`kill -KILL -"$pid"`,          // reap the process group on exit
		`mv "$D/rc.tmp" "$D/rc"`,      // atomic exit-code write
		`2> "$D/err"`,                 // stderr split from stdout
	} {
		if !strings.Contains(runner, want) {
			t.Fatalf("runner missing %q:\n%s", want, runner)
		}
	}
}

func TestRunnerScriptNoTimeoutKeepsGroupSemantics(t *testing.T) {
	// timeout 0 disables the clock but still runs the command in its own
	// process group, so cleanup on exit works even for unlimited jobs.
	s := mustStartScript(t, "id2", RunSpec{Command: "x", TimeoutSecs: 0})
	runner := decodeEmbedded(t, s, "run.sh")
	if !strings.Contains(runner, "timeout -k 10 0 bash") {
		t.Fatalf("unlimited jobs must still run under timeout 0:\n%s", runner)
	}
}

func TestStartScriptCwdEnvStdin(t *testing.T) {
	stdin := "line1\nline2"
	s := mustStartScript(t, "id3", RunSpec{
		Command: "pwd",
		Cwd:     "/tmp/some dir",
		Env:     map[string]string{"FOO": "with 'quote'", "BAR": "x"},
		Stdin:   &stdin,
	})
	if got := decodeEmbedded(t, s, "cwd"); got != "/tmp/some dir" {
		t.Fatalf("cwd roundtrip: %q", got)
	}
	env := decodeEmbedded(t, s, "env")
	if !strings.Contains(env, `export BAR='x'`) || !strings.Contains(env, `export FOO='with '\''quote'\'''`) {
		t.Fatalf("env file wrong:\n%s", env)
	}
	// Env values must never appear in cmd.sh.
	if cmd := decodeEmbedded(t, s, "cmd.sh"); strings.Contains(cmd, "quote") {
		t.Fatalf("env value leaked into cmd.sh: %q", cmd)
	}
	if got := decodeEmbedded(t, s, "stdin"); got != stdin {
		t.Fatalf("stdin roundtrip: %q", got)
	}
	// A missing cwd must fail the launch loudly.
	if !strings.Contains(s, "cwd not found") {
		t.Fatalf("launcher missing cwd validation:\n%s", s)
	}
}

func TestStartScriptRejectsBadEnvKey(t *testing.T) {
	_, err := StartScript("id4", RunSpec{
		Command: "x",
		Env:     map[string]string{"BAD-KEY; rm -rf /": "v"},
	})
	if err == nil {
		t.Fatal("expected error for invalid env key")
	}
}

func TestStartScriptGC(t *testing.T) {
	s := mustStartScript(t, "id5", RunSpec{Command: "x"})
	if !strings.Contains(s, "-mmin +1440") || !strings.Contains(s, "rm -rf") {
		t.Fatalf("launcher missing TTL GC:\n%s", s)
	}
	if !strings.Contains(s, `tmux has-session -t "stg-exec-$i"`) {
		t.Fatalf("GC must never prune live sessions:\n%s", s)
	}
}

func TestParseStatus(t *testing.T) {
	rc0 := 0
	rc7 := 7
	crashed := -1
	cases := []struct {
		name string
		in   string
		want StatusResult
	}{
		{"running", "STATE=running\n", StatusResult{Status: StatusRunning}},
		{"finished ok", "STATE=finished\nRC=0\n", StatusResult{Status: StatusFinished, ExitCode: &rc0}},
		{"finished err", "STATE=finished\nRC=7\n", StatusResult{Status: StatusFinished, ExitCode: &rc7}},
		{"gone", "STATE=gone\n", StatusResult{Status: StatusGone}},
		{"crashed", "STATE=crashed\n", StatusResult{Status: StatusCrashed, ExitCode: &crashed}},
		// A failure must never be mistakable for success: finished without a
		// parseable rc is reported as crashed with exit code -1, not rc 0.
		{"finished no rc", "STATE=finished\nRC=\n", StatusResult{Status: StatusCrashed, ExitCode: &crashed}},
		{"garbage rc", "STATE=finished\nRC=notanumber\n", StatusResult{Status: StatusCrashed, ExitCode: &crashed}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := ParseStatus(c.in)
			if got.Status != c.want.Status {
				t.Fatalf("status: got %q want %q", got.Status, c.want.Status)
			}
			switch {
			case got.ExitCode == nil && c.want.ExitCode == nil:
				// ok
			case got.ExitCode == nil || c.want.ExitCode == nil:
				t.Fatalf("exit code presence mismatch: got %v want %v", got.ExitCode, c.want.ExitCode)
			case *got.ExitCode != *c.want.ExitCode:
				t.Fatalf("exit code: got %d want %d", *got.ExitCode, *c.want.ExitCode)
			}
		})
	}
}

func b64lines(s string) string {
	// coreutils base64 wraps at 76 columns; emulate multi-line payloads.
	enc := base64.StdEncoding.EncodeToString([]byte(s))
	var out strings.Builder
	for i := 0; i < len(enc); i += 8 {
		end := i + 8
		if end > len(enc) {
			end = len(enc)
		}
		out.WriteString(enc[i:end] + "\n")
	}
	return out.String()
}

func TestParseOutput(t *testing.T) {
	raw := "OUT_TOTAL=1000\nOUT_BEGIN\n" + b64lines("hello stdout") +
		"OUT_END\nERR_TOTAL=3\nERR_BEGIN\n" + b64lines("err") + "ERR_END\n"
	res, err := ParseOutput(raw)
	if err != nil {
		t.Fatalf("ParseOutput: %v", err)
	}
	if res.Stdout != "hello stdout" || res.Stderr != "err" {
		t.Fatalf("streams: %q / %q", res.Stdout, res.Stderr)
	}
	if res.StdoutBytes != 1000 || res.StderrBytes != 3 {
		t.Fatalf("totals: %d / %d", res.StdoutBytes, res.StderrBytes)
	}
	// stdout on disk (1000) exceeds what came back (12) → truncated.
	if !res.Truncated() {
		t.Fatal("expected truncated")
	}
	res2, _ := ParseOutput("OUT_TOTAL=12\nOUT_BEGIN\n" + b64lines("hello stdout") +
		"OUT_END\nERR_TOTAL=0\nERR_BEGIN\nERR_END\n")
	if res2.Truncated() {
		t.Fatal("complete output must not report truncated")
	}
}

func TestOutputScriptModes(t *testing.T) {
	s := OutputScript("j1", OutputOpts{})
	if !strings.Contains(s, "tail -c 262144") {
		t.Fatalf("default byte cap missing:\n%s", s)
	}
	s = OutputScript("j1", OutputOpts{TailLines: 50})
	if !strings.Contains(s, "tail -n 50") {
		t.Fatalf("tail_lines missing:\n%s", s)
	}
	s = OutputScript("j1", OutputOpts{MaxBytes: 100 * 1024 * 1024})
	if !strings.Contains(s, "tail -c 4194304") {
		t.Fatalf("byte cap not clamped:\n%s", s)
	}
}

func TestParseJobList(t *testing.T) {
	cmdPrev := base64.StdEncoding.EncodeToString([]byte("echo hi"))
	out := strings.Join([]string{
		"JOB|aaa|finished|0|10|0|1700000100|" + cmdPrev,
		"JOB|bbb|running||5|2|1700000200|" + cmdPrev,
		"JOB|ccc|crashed||0|0|1700000000|" + cmdPrev,
		"garbage line",
	}, "\n")
	jobs := ParseJobList(out)
	if len(jobs) != 3 {
		t.Fatalf("expected 3 jobs, got %d", len(jobs))
	}
	// Newest first.
	if jobs[0].ID != "bbb" || jobs[1].ID != "aaa" || jobs[2].ID != "ccc" {
		t.Fatalf("wrong order: %v %v %v", jobs[0].ID, jobs[1].ID, jobs[2].ID)
	}
	if jobs[1].ExitCode == nil || *jobs[1].ExitCode != 0 {
		t.Fatalf("finished job exit code wrong: %v", jobs[1].ExitCode)
	}
	if jobs[2].ExitCode == nil || *jobs[2].ExitCode != -1 {
		t.Fatalf("crashed job must report -1: %v", jobs[2].ExitCode)
	}
	if jobs[0].CommandPreview != "echo hi" {
		t.Fatalf("command preview: %q", jobs[0].CommandPreview)
	}
}

func TestPollStatusReturnsOnTerminal(t *testing.T) {
	calls := 0
	run := func(script string) (string, error) {
		calls++
		if calls < 3 {
			return "STATE=running\n", nil
		}
		return "STATE=finished\nRC=0\n", nil
	}
	res, err := PollStatus(run, "x", 5*time.Second)
	if err != nil {
		t.Fatalf("PollStatus: %v", err)
	}
	if res.Status != StatusFinished || calls != 3 {
		t.Fatalf("status %q after %d calls", res.Status, calls)
	}
}

func TestPollStatusZeroWaitSingleCheck(t *testing.T) {
	calls := 0
	run := func(script string) (string, error) {
		calls++
		return "STATE=running\n", nil
	}
	res, _ := PollStatus(run, "x", 0)
	if res.Status != StatusRunning || calls != 1 {
		t.Fatalf("zero wait must check exactly once, got %d calls", calls)
	}
}

func TestKillParsesResult(t *testing.T) {
	run := func(script string) (string, error) { return "RESULT=killed\n", nil }
	res, err := Kill(run, "x", false)
	if err != nil || res != "killed" {
		t.Fatalf("Kill: %v %q", err, res)
	}
}

func TestStoreAddGetListNewestFirst(t *testing.T) {
	s := NewStore()
	base := time.Unix(1000, 0)
	for i := 0; i < 3; i++ {
		s.Add(&Job{ID: NewID(), CreatedAt: base.Add(time.Duration(i) * time.Minute)})
	}
	list := s.List()
	if len(list) != 3 {
		t.Fatalf("expected 3 jobs, got %d", len(list))
	}
	if !list[0].CreatedAt.After(list[1].CreatedAt) || !list[1].CreatedAt.After(list[2].CreatedAt) {
		t.Fatalf("jobs not newest-first: %v", list)
	}
	if _, ok := s.Get(list[0].ID); !ok {
		t.Fatalf("Get failed for known id")
	}
	if _, ok := s.Get("nope"); ok {
		t.Fatalf("Get returned a job for unknown id")
	}
}

func TestStorePrunesOldest(t *testing.T) {
	s := NewStore()
	base := time.Unix(0, 0)
	// Fill past the cap; oldest should be evicted, keeping maxJobs entries.
	for i := 0; i < maxJobs+50; i++ {
		s.Add(&Job{ID: NewID(), CreatedAt: base.Add(time.Duration(i) * time.Second)})
	}
	if got := len(s.List()); got != maxJobs {
		t.Fatalf("expected store capped at %d, got %d", maxJobs, got)
	}
}
