package main

import "testing"

func TestImageType(t *testing.T) {
	cases := map[string]string{
		"TARGETS\nimage/jpeg\nimage/png\ntext/html": "image/png",
		"text/html\nimage/webp\nimage/jpeg":         "image/webp",
		"text/plain\nUTF8_STRING":                   "",
		"":                                          "",
	}
	for in, want := range cases {
		if got := imageType(in); got != want {
			t.Errorf("imageType(%q) = %q, want %q", in, got, want)
		}
	}
}
