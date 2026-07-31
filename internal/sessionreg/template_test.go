package sessionreg

import "testing"

// The templates have to reach disk: they are what Duplicate reads, and a
// duplicate is usually made days after the session was created — long after
// any in-memory copy is gone.
func TestTemplatesRoundTrip(t *testing.T) {
	dir := t.TempDir()
	s, err := NewStore(dir)
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	if err := s.AddSession("h1", "api", Attrs{
		WorkingDir:         "/home/me/sessions/api",
		Command:            "claude --name api",
		WorkingDirTemplate: "~/sessions/$name",
		CommandTemplate:    "claude --name $name",
	}); err != nil {
		t.Fatalf("add: %v", err)
	}

	reopened, err := NewStore(dir)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	e, ok := reopened.Get("h1", "api")
	if !ok {
		t.Fatal("entry missing after reload")
	}
	if e.WorkingDirTemplate != "~/sessions/$name" || e.CommandTemplate != "claude --name $name" {
		t.Errorf("templates did not round-trip: %+v", e)
	}
	// The expansions are what Recreate replays, and must survive alongside.
	if e.WorkingDir != "/home/me/sessions/api" || e.Command != "claude --name api" {
		t.Errorf("expanded values did not round-trip: %+v", e)
	}
}

// A session created without variables must look on disk exactly as it did
// before the feature existed — an absent field, not an empty one.
func TestNoTemplateMeansNoTemplate(t *testing.T) {
	s := newTestStore(t)
	if err := s.AddSession("h1", "work", Attrs{WorkingDir: "/srv", Command: "vim"}); err != nil {
		t.Fatalf("add: %v", err)
	}
	e, _ := s.Get("h1", "work")
	if e.WorkingDirTemplate != "" || e.CommandTemplate != "" {
		t.Errorf("templates invented for a variable-free session: %+v", e)
	}
}

// Same empty-means-leave-it-alone rule as Command: the poller's cwd refresh
// and the rename path don't know the templates and must not erase them.
func TestReAddingKeepsTheTemplates(t *testing.T) {
	s := newTestStore(t)
	_ = s.AddSession("h1", "api", Attrs{
		WorkingDir:         "/home/me/sessions/api",
		Command:            "claude --name api",
		WorkingDirTemplate: "~/sessions/$name",
		CommandTemplate:    "claude --name $name",
	})
	// What the poller does when it observes the session's real directory.
	if err := s.Add("h1", "api", "/home/me/sessions/api/src"); err != nil {
		t.Fatalf("add: %v", err)
	}

	e, _ := s.Get("h1", "api")
	if e.WorkingDirTemplate != "~/sessions/$name" || e.CommandTemplate != "claude --name $name" {
		t.Errorf("a cwd refresh wiped the templates: %+v", e)
	}
	if e.WorkingDir != "/home/me/sessions/api/src" {
		t.Errorf("the refresh did not land: %+v", e)
	}
}

// A rename must carry the templates with the entry, or the renamed session's
// next duplicate silently falls back to copying the old name.
func TestRenameKeepsTheTemplates(t *testing.T) {
	s := newTestStore(t)
	_ = s.AddSession("h1", "api", Attrs{
		WorkingDir:         "/home/me/sessions/api",
		CommandTemplate:    "claude --name $name",
		WorkingDirTemplate: "~/sessions/$name",
	})
	if _, err := s.Rename("h1", "api", "api2"); err != nil {
		t.Fatalf("rename: %v", err)
	}
	e, ok := s.Get("h1", "api2")
	if !ok {
		t.Fatal("renamed entry missing")
	}
	if e.CommandTemplate != "claude --name $name" || e.WorkingDirTemplate != "~/sessions/$name" {
		t.Errorf("templates lost in the rename: %+v", e)
	}
}
