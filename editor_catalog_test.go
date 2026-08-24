package main

import (
	"os"
	"reflect"
	"testing"
)

// TestEditorCatalogConsistency guards the invariants detectAvailableEditors,
// OpenInEditorByName, and getJetBrainsEditor all rely on. editorCatalog is
// the single source of truth for editor detection and launching (#18): if
// one of these breaks, detection totals, launch dispatch, or JetBrains
// routing silently go wrong.
func TestEditorCatalogConsistency(t *testing.T) {
	if len(editorCatalog) != 22 {
		t.Errorf("expected 22 catalog rows, got %d", len(editorCatalog))
	}

	seenKeys := make(map[string]bool)
	seenCommands := make(map[string]bool)
	for _, e := range editorCatalog {
		if e.key == "" {
			t.Error("catalog row with empty key")
		}
		if seenKeys[e.key] {
			t.Errorf("duplicate catalog key %q", e.key)
		}
		seenKeys[e.key] = true

		if e.command == "" {
			t.Errorf("catalog key %q has empty command", e.key)
		}
		if seenCommands[e.command] {
			t.Errorf("duplicate catalog command %q (key %q)", e.command, e.key)
		}
		seenCommands[e.command] = true

		if e.displayName == "" {
			t.Errorf("catalog key %q has empty displayName (progress events would be blank)", e.key)
		}

		if e.key == "JetBrains" || e.key == "SystemDefault" {
			t.Errorf("%q must not appear in editorCatalog (special-cased outside it)", e.key)
		}

		// Every setter must write an existing EditorAvailability field so
		// detection results actually reach GetEditorDetectionStatus.
		app := NewApp()
		app.editorsMu.Lock()
		e.set(app, true)
		field := reflect.ValueOf(app.availableEditors).FieldByName(e.key)
		app.editorsMu.Unlock()
		if !field.IsValid() {
			t.Errorf("catalog key %q has no matching EditorAvailability field", e.key)
		} else if !field.Bool() {
			t.Errorf("setter for %q did not set availableEditors.%s = true", e.key, e.key)
		}
	}

	// Every JetBrains switch arm must resolve through the catalog.
	for _, key := range []string{"GoLand", "PyCharm", "WebStorm", "PhpStorm", "IntelliJ", "CLion", "Rider"} {
		if catalogEntry(key) == nil {
			t.Errorf("getJetBrainsEditor routes to %q but it is missing from editorCatalog", key)
		}
	}

	routes := []struct{ file, want string }{
		{"main.go", "goland"},
		{"script.py", "pycharm"},
		{"app.js", "webstorm"},
		{"index.php", "phpstorm"},
		{"Main.java", "idea"},
		{"main.cpp", "clion"},
		{"Program.cs", "rider"},
		{"notes.md", "idea"}, // generic default
	}
	app := NewApp()
	for _, r := range routes {
		path := t.TempDir() + "/" + r.file
		if err := os.WriteFile(path, []byte("x"), 0644); err != nil {
			t.Fatalf("write %s: %v", r.file, err)
		}
		editor, args := app.getJetBrainsEditor(path)
		if editor != r.want {
			t.Errorf("%s: expected %q, got %q", r.file, r.want, editor)
		}
		if len(args) != 0 {
			t.Errorf("%s: expected no extra args, got %v", r.file, args)
		}
	}
}
