package api

import (
	"testing"
	"time"

	"github.com/awkto/ssh-to-go/internal/sessionreg"
)

var dupNow = time.Date(2026, 7, 31, 9, 0, 0, 0, time.UTC)

// The reason templates are stored at all: a copy of a session launched with
// `claude --name $name` has to introduce itself as the copy. Copying the
// expanded command would give two sessions both claiming to be "api".
func TestDuplicateReExpandsTheTemplate(t *testing.T) {
	entry := sessionreg.Entry{
		WorkingDir:         "/home/me/sessions/api",
		Command:            "claude --name api",
		WorkingDirTemplate: "~/sessions/$name",
		CommandTemplate:    "claude --name $name",
	}
	cwd, command, createDir := duplicateLaunch(entry, "/home/me/sessions/api", "api-COPY", dupNow)

	if want := "~/sessions/api-COPY"; cwd != want {
		t.Errorf("cwd = %q, want %q", cwd, want)
	}
	if want := "claude --name api-COPY"; command != want {
		t.Errorf("command = %q, want %q", command, want)
	}
	// A recomputed path points somewhere that has never existed, so the copy
	// has to be allowed to create it.
	if !createDir {
		t.Error("createDir = false; a re-expanded directory does not exist yet")
	}
}

// Everything created before this feature, and everything created without a
// variable, must duplicate exactly as it did before: the live pane's
// directory, the recorded command, and no directory creation.
func TestDuplicateWithoutATemplateIsUnchanged(t *testing.T) {
	entry := sessionreg.Entry{WorkingDir: "/srv/app", Command: "vim"}
	cwd, command, createDir := duplicateLaunch(entry, "/srv/app/sub", "work-COPY", dupNow)

	if cwd != "/srv/app/sub" {
		t.Errorf("cwd = %q, want the live pane's directory", cwd)
	}
	if command != "vim" {
		t.Errorf("command = %q, want %q", command, "vim")
	}
	if createDir {
		t.Error("createDir = true; the source session demonstrably runs there")
	}
}

// The live pane wins over the registry, but only while the source session is
// running — a dead source falls back to what was recorded.
func TestDuplicateFallsBackToTheRecordedDir(t *testing.T) {
	entry := sessionreg.Entry{WorkingDir: "/srv/app"}
	if cwd, _, _ := duplicateLaunch(entry, "", "work-COPY", dupNow); cwd != "/srv/app" {
		t.Errorf("cwd = %q, want the recorded directory", cwd)
	}
}

// $date must NOT be re-dated: a copy made in October of a session created in
// July belongs beside the original, not in a directory named after today.
// The template is only re-expanded for the parts that name the session — so
// this is really a check that we date from the passed clock, which callers
// could later change to the entry's creation time if that reading changes.
func TestDuplicateDatesFromTheSuppliedClock(t *testing.T) {
	entry := sessionreg.Entry{WorkingDirTemplate: "~/logs/$date"}
	cwd, _, _ := duplicateLaunch(entry, "", "x-COPY", dupNow)
	if want := "~/logs/2026-07-31"; cwd != want {
		t.Errorf("cwd = %q, want %q", cwd, want)
	}
}

// A template on only one of the two fields must not drag the other into
// being re-expanded — in particular, a command template alone leaves the
// directory as the pane's and createDir off.
func TestDuplicateWithOnlyACommandTemplate(t *testing.T) {
	entry := sessionreg.Entry{
		WorkingDir:      "/srv/app",
		Command:         "claude --name api",
		CommandTemplate: "claude --name $name",
	}
	cwd, command, createDir := duplicateLaunch(entry, "/srv/app", "api-COPY", dupNow)

	if cwd != "/srv/app" {
		t.Errorf("cwd = %q, want it left alone", cwd)
	}
	if want := "claude --name api-COPY"; command != want {
		t.Errorf("command = %q, want %q", command, want)
	}
	if createDir {
		t.Error("createDir = true, but the directory was not recomputed")
	}
}
