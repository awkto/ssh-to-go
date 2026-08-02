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
