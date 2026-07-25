package api

import "testing"

func TestCopyBaseStripsChain(t *testing.T) {
	cases := map[string]string{
		"foo":           "foo",
		"foo-COPY":      "foo",
		"foo-COPY2":     "foo",
		"foo-COPY17":    "foo",
		"foo-COPYcat":   "foo-COPYcat", // not a suffix we generated
		"my-COPY-thing": "my-COPY-thing",
		"COPY":          "COPY",
		"claude-COPY3":  "claude",
		"a-COPY2-COPY":  "a-COPY2", // one level per duplicate, which is enough
	}
	for in, want := range cases {
		if got := copyBase(in); got != want {
			t.Errorf("copyBase(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestCopyNameNumbering(t *testing.T) {
	cases := []struct {
		name  string
		base  string
		taken []string
		want  string
	}{
		{"first copy", "foo", []string{"foo"}, "foo-COPY"},
		{"second copy", "foo", []string{"foo", "foo-COPY"}, "foo-COPY2"},
		{"third copy", "foo", []string{"foo", "foo-COPY", "foo-COPY2"}, "foo-COPY3"},
		{
			// The whole reason for scanning the highest rather than the
			// first free: deleting the middle of a chain must not make the
			// next duplicate collide with the end of it.
			"gap in the chain", "foo",
			[]string{"foo", "foo-COPY", "foo-COPY3"},
			"foo-COPY4",
		},
		{"other sessions ignored", "foo", []string{"bar-COPY2", "foobar-COPY9"}, "foo-COPY"},
		{"nothing taken", "foo", nil, "foo-COPY"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := copyName(c.base, c.taken); got != c.want {
				t.Errorf("copyName(%q, %v) = %q, want %q", c.base, c.taken, got, c.want)
			}
		})
	}
}

// Duplicating a copy continues the chain instead of nesting suffixes.
func TestCopyOfACopyContinuesTheChain(t *testing.T) {
	taken := []string{"foo", "foo-COPY", "foo-COPY2"}
	if got := copyName(copyBase("foo-COPY"), taken); got != "foo-COPY3" {
		t.Errorf("duplicating foo-COPY = %q, want foo-COPY3", got)
	}
}
