package sessionid

import "testing"

func TestAssignStableAndUnique(t *testing.T) {
	s, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	a := s.Assign("hostA", "web")
	b := s.Assign("hostB", "web") // same name, different host
	c := s.Assign("hostA", "db")
	if a < minID || a > maxID {
		t.Fatalf("id %d out of range", a)
	}
	if a == b || a == c || b == c {
		t.Fatalf("ids not unique: %d %d %d", a, b, c)
	}
	if got := s.Assign("hostA", "web"); got != a {
		t.Fatalf("reassignment changed id: %d != %d", got, a)
	}
}

func TestPersistAcrossReopen(t *testing.T) {
	dir := t.TempDir()
	s1, _ := NewStore(dir)
	a := s1.Assign("h", "one")
	b := s1.Assign("h", "two")

	s2, err := NewStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	if got := s2.Assign("h", "one"); got != a {
		t.Fatalf("id lost across reopen: %d != %d", got, a)
	}
	if got := s2.Assign("h", "two"); got != b {
		t.Fatalf("id lost across reopen: %d != %d", got, b)
	}
}

func TestRenameKeepsID(t *testing.T) {
	dir := t.TempDir()
	s, _ := NewStore(dir)
	a := s.Assign("h", "old")
	s.Rename("h", "old", "new")
	if got := s.Assign("h", "new"); got != a {
		t.Fatalf("rename lost id: %d != %d", got, a)
	}
	if got := s.Assign("h", "old"); got == a {
		t.Fatalf("old name still resolves to the moved id %d", a)
	}

	// Survives reopen under the new name too.
	s2, _ := NewStore(dir)
	if got := s2.Assign("h", "new"); got != a {
		t.Fatalf("renamed id lost across reopen: %d != %d", got, a)
	}
}

func TestRenameOntoExistingNameDropsStale(t *testing.T) {
	s, _ := NewStore(t.TempDir())
	a := s.Assign("h", "keep")
	s.Assign("h", "victim")
	s.Rename("h", "keep", "victim")
	if got := s.Assign("h", "victim"); got != a {
		t.Fatalf("rename-over did not carry id: %d != %d", got, a)
	}
}
