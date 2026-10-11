package main

import "testing"

func TestNormalizeTagMatchesServer(t *testing.T) {
	for in, want := range map[string]string{
		" Work ":        "work",
		"Side  Project": "side-project",
		"":              "",
	} {
		if got := normalizeTag(in); got != want {
			t.Errorf("normalizeTag(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestTagHueFallsBackToWebHash(t *testing.T) {
	if got := tagHue("work", map[string]int{"work": 150}); got != 150 {
		t.Fatalf("stored hue ignored: %d", got)
	}
	// Same hash as tagHue() in web/static/app/sessions.jsx: "ab" =
	// (97*31 + 98) % 360.
	if got := tagHue("ab", nil); got != (97*31+98)%360 {
		t.Fatalf("fallback hue = %d", got)
	}
}

func TestTagRGBGreenForHue150(t *testing.T) {
	r, g, b := tagRGB("x", map[string]int{"x": 150})
	if !(g > r && g > b) {
		t.Fatalf("hue 150 should be green, got %d,%d,%d", r, g, b)
	}
}
