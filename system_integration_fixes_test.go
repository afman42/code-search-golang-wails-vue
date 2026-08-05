package main

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// TestReadFileAcceptsShellMetacharacters verifies that ReadFile no longer
// rejects file paths containing shell metacharacters like |, &, ;, `, and
// $(...). The previous implementation had an over-aggressive "command
// injection character" filter that rejected legitimate Unix filenames
// (e.g. "foo$(bar).txt", "a;b.txt") even though ReadFile never passes the
// path to a shell (#14). The filter has been removed; path traversal is
// still handled by the containsDotDotComponent + filepath.Clean checks,
// and null-byte injection is still blocked.
func TestReadFileAcceptsShellMetacharacters(t *testing.T) {
	app := NewApp()
	tempDir := t.TempDir()

	// Each of these names is valid on Unix filesystems and was previously
	// rejected by the command-injection char filter. We create the file,
	// then confirm ReadFile reads it successfully.
	cases := []string{
		"pipe|file.txt",
		"ampersand&file.txt",
		"semicolon;file.txt",
		"dollar$(paren).txt",
	}
	for _, name := range cases {
		full := filepath.Join(tempDir, name)
		if err := os.WriteFile(full, []byte("content for "+name), 0o644); err != nil {
			// Some platforms (Windows) don't allow these chars in names.
			// Skip the case instead of failing — the test is about the
			// Go-side filter, not the OS's naming rules.
			if runtime.GOOS == "windows" {
				t.Logf("skipping %q on Windows (filesystem doesn't allow the char)", name)
				continue
			}
			t.Fatalf("creating %q: %v", name, err)
		}
		content, err := app.ReadFile(full)
		if err != nil {
			t.Errorf("ReadFile(%q) failed: %v — the command-injection char filter should have been removed (#14)", name, err)
			continue
		}
		if content != "content for "+name {
			t.Errorf("ReadFile(%q) returned %q, expected %q", name, content, "content for "+name)
		}
	}
}

// TestReadFileStillRejectsNullBytes verifies that the null-byte injection
// check (which IS a real security concern) is still in place after removing
// the over-aggressive char filter (#14).
func TestReadFileStillRejectsNullBytes(t *testing.T) {
	app := NewApp()
	_, err := app.ReadFile("foo\x00bar.txt")
	if err == nil {
		t.Error("expected ReadFile to reject null bytes in file path, got nil error")
	}
}

// TestReadFileStillRejectsPathTraversal verifies that the path-traversal
// checks (containsDotDotComponent + filepath.Clean) still reject ../ paths
// after the char filter removal (#14). We build the path with string
// concatenation instead of filepath.Join because filepath.Join would clean
// the ".." away before ReadFile ever sees it — and the first check in
// ReadFile is on the raw input (before Clean), so we need the ".." to
// survive into the call.
func TestReadFileStillRejectsPathTraversal(t *testing.T) {
	app := NewApp()
	tempDir := t.TempDir()
	// Create a real file so the only failure is the traversal check, not
	// a missing-file error.
	realFile := filepath.Join(tempDir, "real.txt")
	if err := os.WriteFile(realFile, []byte("x"), 0o644); err != nil {
		t.Fatalf("creating real file: %v", err)
	}
	// Build a path that contains a literal ".." component. Using string
	// concatenation (NOT filepath.Join) so the ".." is preserved into the
	// ReadFile call, where the first containsDotDotComponent check fires.
	traversalPath := tempDir + string(filepath.Separator) + ".." + string(filepath.Separator) + filepath.Base(tempDir) + string(filepath.Separator) + "real.txt"
	_, err := app.ReadFile(traversalPath)
	if err == nil {
		t.Error("expected ReadFile to reject a path containing a .. component, got nil error")
	}
}

// TestOpenInEditorByNameUnknownEditor verifies that the generic dispatcher
// (#18) rejects an unknown binding name instead of silently doing nothing
// or panicking.
func TestOpenInEditorByNameUnknownEditor(t *testing.T) {
	app := NewApp()
	err := app.OpenInEditorByName("DefinitelyNotAnEditor", "/tmp/some-file.txt")
	if err == nil {
		t.Error("expected OpenInEditorByName to reject an unknown binding name, got nil error")
	}
}

// TestOpenInEditorByNameJetBrainsRoutesByExtension verifies that the
// "JetBrains" binding name routes to the appropriate JetBrains IDE based
// on file extension via getJetBrainsEditor, rather than looking up a single
// command in editorBindings (which has no "JetBrains" key — it's handled
// as a special case in OpenInEditorByName).
func TestOpenInEditorByNameJetBrainsRoutesByExtension(t *testing.T) {
	app := NewApp()
	tmpFile := t.TempDir() + "/main.go"
	if err := os.WriteFile(tmpFile, []byte("package main"), 0644); err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}

	// A .go file must route to goland. If nvim/goland isn't in PATH the call
	// errors at LookPath, but the routing decision (goland vs idea/pycharm)
	// happens before that — so we verify via the routing helper directly.
	editor, args := app.getJetBrainsEditor(tmpFile)
	if editor != "goland" {
		t.Errorf("expected .go to route to goland, got %q", editor)
	}
	if len(args) != 0 {
		t.Errorf("expected no extra args for goland, got %v", args)
	}

	// The "JetBrains" name must NOT be in editorBindings — it's dispatched
	// via the special case, not the map. If someone adds it to the map,
	// the special case would be dead code.
	if _, ok := editorBindings["JetBrains"]; ok {
		t.Error("editorBindings must not contain \"JetBrains\" — it is handled by a special case in OpenInEditorByName")
	}
}

// TestEditorBindingsHasNoDuplicateCommands verifies that no two binding
// entries silently point at the same editor command. Duplicates would mean
// one of the OpenInX wrappers is redundant — likely a copy-paste mistake.
// (Note: this is a sanity check, not a hard requirement — if two binding
// names legitimately share a command, add them to the allowlist here.)
func TestEditorBindingsHasNoDuplicateCommands(t *testing.T) {
	seen := make(map[string]string) // command -> first binding name that used it
	for name, binding := range editorBindings {
		if prev, dup := seen[binding.command]; dup {
			t.Errorf("editor binding %q and %q both point at command %q — likely a copy-paste mistake (#18)", prev, name, binding.command)
		}
		seen[binding.command] = name
	}
}

// TestCountEditorsFromSnapshot verifies that the snapshot-based editor
// counter (#20) matches the lock-based countAvailableEditors. If they
// diverge, GetEditorDetectionStatus would report a different total than
// the frontend expects.
//
// Note: SystemDefault is intentionally NOT counted by either function —
// it's a derived "always true" flag set unconditionally in
// detectAvailableEditors, not a detected editor. The availableEditorFields
// list excludes it for that reason.
func TestCountEditorsFromSnapshot(t *testing.T) {
	app := NewApp()
	// Set a known set of available editors under lock so both counters
	// observe the same state.
	app.editorsMu.Lock()
	app.availableEditors = EditorAvailability{
		VSCode:        true,
		Sublime:       true,
		Neovim:        true,
		Vim:           true,
		SystemDefault: true, // not counted — see comment above
	}
	snapshot := app.availableEditors
	app.editorsMu.Unlock()

	gotSnapshot := countEditorsFromSnapshot(snapshot)
	gotLocked := app.countAvailableEditors()
	if gotSnapshot != gotLocked {
		t.Errorf("countEditorsFromSnapshot=%d != countAvailableEditors=%d", gotSnapshot, gotLocked)
	}
	// We set 4 real editor fields true (VSCode, Sublime, Neovim, Vim).
	// SystemDefault is not in availableEditorFields, so it's not counted.
	if gotSnapshot != 4 {
		t.Errorf("expected 4 available editors, got %d", gotSnapshot)
	}
}

// TestGetEditorDetectionStatusTotalMatchesCount verifies that
// GetEditorDetectionStatus reports a totalAvailable that's consistent with
// the snapshot it returns — i.e. the redundant second lock acquisition
// that #20 removed didn't also remove the count consistency.
func TestGetEditorDetectionStatusTotalMatchesCount(t *testing.T) {
	app := NewApp()
	app.editorsMu.Lock()
	app.availableEditors = EditorAvailability{
		VSCode:        true,
		VSCodium:      true,
		SystemDefault: true, // not counted
	}
	app.editorsMu.Unlock()

	status := app.GetEditorDetectionStatus()
	total, ok := status["totalAvailable"].(int)
	if !ok {
		t.Fatalf("totalAvailable is not an int: %T", status["totalAvailable"])
	}
	// 2 real editors (VSCode, VSCodium). SystemDefault is not counted.
	if total != 2 {
		t.Errorf("expected totalAvailable=2, got %d", total)
	}
}
