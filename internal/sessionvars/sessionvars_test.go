package sessionvars

import (
	"testing"
	"time"
)

var testVars = Vars{
	Name: "api-refactor",
	Now:  time.Date(2026, 7, 31, 14, 5, 0, 0, time.UTC),
}

func TestExpand(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		// The two things the feature exists for.
		{"name in a path", "~/sessions/$name", "~/sessions/api-refactor"},
		{"name in a command", "claude --name $name", "claude --name api-refactor"},
		{"date in a path", "~/logs/$date", "~/logs/2026-07-31"},

		// Braces: the reason ${name} is supported at all is a suffix that
		// would otherwise be swallowed into the variable name.
		{"braced name", "~/sessions/${name}-work", "~/sessions/api-refactor-work"},
		{"braced date", "${date}.log", "2026-07-31.log"},

		{"both, repeated", "$name/$date/$name", "api-refactor/2026-07-31/api-refactor"},
		{"mid-word after a dash", "$name-COPY", "api-refactor-COPY"},
		{"trailing slash", "~/x/$name/", "~/x/api-refactor/"},

		// Everything below is the "leave the shell alone" rule. Each of
		// these is a string somebody could plausibly have in a launch
		// command today, and each must survive byte for byte.
		{"shell variable", "cd $HOME && claude", "cd $HOME && claude"},
		{"PATH", "PATH=$PATH:/opt/bin claude", "PATH=$PATH:/opt/bin claude"},
		{"positional", "sh -c 'echo $1'", "sh -c 'echo $1'"},
		{"pid", "echo $$", "echo $$"},
		{"command substitution", "echo $(whoami)", "echo $(whoami)"},
		{"braced unknown", "echo ${FOO}", "echo ${FOO}"},
		{"unterminated brace", "echo ${name", "echo ${name"},
		{"bare dollar", "echo $ x", "echo $ x"},
		{"trailing dollar", "echo $", "echo $"},

		// The word-boundary rule. $nameless is one unknown word — expanding
		// its prefix would rewrite the middle of somebody's variable.
		{"longer name", "$nameless", "$nameless"},
		{"name with underscore", "$name_dir", "$name_dir"},
		{"name with digit", "$name2", "$name2"},
		{"date prefix", "$dateformat", "$dateformat"},

		// Escapes. \$name is ours to unescape; \$HOME belongs to the shell.
		{"escaped name", `claude --name \$name`, "claude --name $name"},
		{"escaped braced name", `echo \${name}`, "echo ${name}"},
		{"escaped shell variable", `echo \$HOME`, `echo \$HOME`},
		{"backslash not before a dollar", `echo a\b`, `echo a\b`},

		{"empty", "", ""},
		{"no variables at all", "claude --resume", "claude --resume"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := Expand(tc.in, testVars); got != tc.want {
				t.Errorf("Expand(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

// $name is the sanitized name because that is the name tmux and the registry
// actually use. Expanding the typed name would put a space in the path — the
// exact thing sanitizeSessionName exists to prevent.
func TestExpandUsesTheNameItIsGiven(t *testing.T) {
	got := Expand("~/sessions/$name", Vars{Name: "my-session", Now: testVars.Now})
	if want := "~/sessions/my-session"; got != want {
		t.Errorf("Expand = %q, want %q", got, want)
	}
}

// A caller that forgets Now should get today, not the year 1 — a path like
// ~/logs/0001-01-01 is a silent bug you only notice much later.
func TestZeroTimeMeansToday(t *testing.T) {
	got := Expand("$date", Vars{Name: "x"})
	if want := time.Now().Format(dateLayout); got != want {
		t.Errorf("Expand($date) with a zero Now = %q, want %q", got, want)
	}
}

func TestHasVars(t *testing.T) {
	cases := []struct {
		in   string
		want bool
	}{
		{"~/sessions/$name", true},
		{"${date}", true},
		{"claude --name $name --resume", true},
		{"", false},
		{"claude", false},
		{"cd $HOME", false},
		{"echo $$", false},
		{"$nameless", false},
		// An escaped variable is not a use of it: the whole point of the
		// escape is that the string stays literal, so there is no template
		// worth remembering and Duplicate should copy it verbatim.
		{`claude --name \$name`, false},
	}
	for _, tc := range cases {
		if got := HasVars(tc.in); got != tc.want {
			t.Errorf("HasVars(%q) = %v, want %v", tc.in, got, tc.want)
		}
	}
}

// The rule that protects every existing launch command: if there is nothing
// of ours in the string, the caller gets the same bytes back.
func TestVariableFreeStringsAreUntouched(t *testing.T) {
	for _, s := range []string{
		"claude",
		"cd $HOME && exec $SHELL -l",
		`awk '{print $1, $NF}' /var/log/x`,
		"echo ${UNSET:-default}",
		"docker run -e FOO=$FOO $IMAGE",
	} {
		if got := Expand(s, testVars); got != s {
			t.Errorf("Expand(%q) = %q, want it unchanged", s, got)
		}
	}
}
