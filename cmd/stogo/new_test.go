package main

import "testing"

// sanitizeName must match the server's sanitizeSessionName — the CLI shows
// this name and attaches to it, so any drift means attaching to a session
// that doesn't exist.
func TestSanitizeName(t *testing.T) {
	cases := map[string]string{
		"bug hunt":       "bug-hunt",
		"  bug   hunt  ": "bug-hunt",
		"bug\thunt":      "bug-hunt",
		"plain":          "plain",
		"CamelCase":      "CamelCase",
		"a b c":          "a-b-c",
	}
	for in, want := range cases {
		if got := sanitizeName(in); got != want {
			t.Errorf("sanitizeName(%q) = %q, want %q", in, got, want)
		}
	}
}

// dirSlug must match the web form's nsDirSlug so CLI and dashboard derive
// the same directory for the same session name.
func TestDirSlug(t *testing.T) {
	cases := map[string]string{
		"bug hunt":      "bug-hunt",
		"Bug Hunt":      "bug-hunt",
		"a--b":          "a-b",
		"a -b":          "a-b",
		"..lead.trail.": "lead.trail",
		"héllo wörld":   "h-llo-w-rld",
		"under_score":   "under_score",
		"":              "",
		"!!!":           "",
		"0123456789012345678901234567890123456789extra": "0123456789012345678901234567890123456789",
	}
	for in, want := range cases {
		if got := dirSlug(in); got != want {
			t.Errorf("dirSlug(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestJoinDir(t *testing.T) {
	cases := []struct{ base, slug, want string }{
		{"~/sessions/", "foo", "~/sessions/foo"},
		{"~/sessions", "foo", "~/sessions/foo"},
		{"~/sessions/", "", "~/sessions/"},
	}
	for _, c := range cases {
		if got := joinDir(c.base, c.slug); got != c.want {
			t.Errorf("joinDir(%q, %q) = %q, want %q", c.base, c.slug, got, c.want)
		}
	}
}

// matchHost backs the host prompt: exact names win even when they prefix
// another host, and an ambiguous prefix must not pick one silently.
func TestMatchHost(t *testing.T) {
	var hosts []hostState
	for _, n := range []string{"pro", "prod-db", "lab"} {
		var h hostState
		h.Config.Name = n
		hosts = append(hosts, h)
	}
	cases := []struct {
		in, want string
		ok       bool
	}{
		{"pro", "pro", true},
		{"prod", "prod-db", true},
		{"l", "lab", true},
		{"pr", "", false},
		{"nope", "", false},
	}
	for _, c := range cases {
		got, ok := matchHost(hosts, c.in)
		if got != c.want || ok != c.ok {
			t.Errorf("matchHost(%q) = %q, %v; want %q, %v", c.in, got, ok, c.want, c.ok)
		}
	}
}

func TestFindOffloaded(t *testing.T) {
	var hosts []hostState
	pro := hostState{}
	pro.Config.Name = "pro"
	pro.MissingSessions = []offloadedSession{{Name: "bug-hunt", WorkingDir: "/x"}, {Name: "legacy name"}}
	lab := hostState{}
	lab.Config.Name = "lab"
	lab.MissingSessions = []offloadedSession{{Name: "other"}}
	hosts = append(hosts, pro, lab)

	cases := []struct {
		host, name, want string
		ok               bool
	}{
		{"pro", "bug-hunt", "bug-hunt", true},
		{"pro", "bug hunt", "bug-hunt", true},       // typed form sanitizes to the key
		{"pro", "legacy-name", "legacy name", true}, // legacy spaced entry
		{"pro", "other", "", false},                 // right name, wrong host
		{"lab", "other", "other", true},
		{"nope", "other", "", false},
	}
	for _, c := range cases {
		got, ok := findOffloaded(hosts, c.host, c.name)
		if ok != c.ok || got.Name != c.want {
			t.Errorf("findOffloaded(%q, %q) = (%q, %v), want (%q, %v)", c.host, c.name, got.Name, ok, c.want, c.ok)
		}
	}
}
