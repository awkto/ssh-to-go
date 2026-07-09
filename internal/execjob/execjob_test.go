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

func TestStartScriptEncodesCommand(t *testing.T) {
	cmd := "echo 'hi there'; curl http://x/$FOO && echo \"done\"\nsecond line"
	s := StartScript("abc123", cmd, 0)

	// The command must be transferred base64-encoded so arbitrary shell
	// metacharacters and newlines can't break out of the launcher.
	wantB64 := base64.StdEncoding.EncodeToString([]byte(cmd))
	if !strings.Contains(s, wantB64) {
		t.Fatalf("start script missing base64 command\nscript:\n%s", s)
	}
	// Raw dangerous fragments must NOT appear unencoded.
	if strings.Contains(s, "curl http://x/$FOO") {
		t.Fatalf("raw command leaked into script:\n%s", s)
	}
	// Uses a detached tmux session named with the job id.
	if !strings.Contains(s, "tmux new-session -d -s '"+SessionName("abc123")+"'") {
		t.Fatalf("start script missing detached tmux launch:\n%s", s)
	}
	// No timeout wrapper when timeoutSecs == 0.
	if strings.Contains(s, "timeout ") {
		t.Fatalf("unexpected timeout wrapper:\n%s", s)
	}
}

func TestStartScriptTimeout(t *testing.T) {
	s := StartScript("id1", "sleep 100", 30)
	runB64Line := ""
	for _, line := range strings.Split(s, "\n") {
		if strings.Contains(line, "run.sh") && strings.Contains(line, "base64 -d") {
			runB64Line = line
		}
	}
	if runB64Line == "" {
		t.Fatalf("no run.sh line found:\n%s", s)
	}
	// Decode the embedded runner and confirm it wraps cmd.sh in `timeout 30`.
	start := strings.Index(runB64Line, "'")
	end := strings.LastIndex(runB64Line, "'")
	dec, err := base64.StdEncoding.DecodeString(runB64Line[start+1 : end])
	if err != nil {
		t.Fatalf("decode runner: %v", err)
	}
	runner := string(dec)
	if !strings.Contains(runner, "timeout 30 bash \"$D/cmd.sh\"") {
		t.Fatalf("runner missing timeout wrapper:\n%s", runner)
	}
	if !strings.Contains(runner, `echo $? > "$D/rc"`) {
		t.Fatalf("runner missing exit-code capture:\n%s", runner)
	}
}

func TestParseStatus(t *testing.T) {
	rc0 := 0
	rc7 := 7
	cases := []struct {
		name string
		in   string
		want StatusResult
	}{
		{"running", "STATE=running\nRC=\n", StatusResult{Status: StatusRunning}},
		{"finished ok", "STATE=finished\nRC=0\n", StatusResult{Status: StatusFinished, ExitCode: &rc0}},
		{"finished err", "STATE=finished\nRC=7\n", StatusResult{Status: StatusFinished, ExitCode: &rc7}},
		{"gone", "STATE=gone\n", StatusResult{Status: StatusGone}},
		{"finished no rc yet", "STATE=finished\nRC=\n", StatusResult{Status: StatusFinished}},
		{"garbage rc", "STATE=finished\nRC=notanumber\n", StatusResult{Status: StatusFinished}},
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
