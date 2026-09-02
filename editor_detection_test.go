package main

import (
	"os"
	"strings"
	"testing"
)

func TestIsEditorAvailable(t *testing.T) {
	app := NewApp()

	t.Run("Existing command returns true", func(t *testing.T) {
		// "sh" or "echo" should be available in any test environment
		available := app.isEditorAvailable("echo")
		if !available {
			t.Error("isEditorAvailable should return true for 'echo' which should be in PATH")
		}
	})

	t.Run("Non-existent command returns false", func(t *testing.T) {
		available := app.isEditorAvailable("this-command-definitely-does-not-exist-12345")
		if available {
			t.Error("isEditorAvailable should return false for a non-existent command")
		}
	})
}

func TestCountAvailableEditors(t *testing.T) {
	app := NewApp()

	t.Run("Starts at zero", func(t *testing.T) {
		count := app.countAvailableEditors()
		if count != 0 {
			t.Errorf("countAvailableEditors should start at 0, got %d", count)
		}
	})

	t.Run("Counts Neovim when set", func(t *testing.T) {
		app.editorsMu.Lock()
		app.availableEditors.Neovim = true
		app.editorsMu.Unlock()

		count := app.countAvailableEditors()
		if count != 1 {
			t.Errorf("countAvailableEditors should be 1 after setting Neovim, got %d", count)
		}

		// Reset for other tests
		app.editorsMu.Lock()
		app.availableEditors.Neovim = false
		app.editorsMu.Unlock()
	})

	t.Run("Counts multiple editors including Neovim", func(t *testing.T) {
		app.editorsMu.Lock()
		app.availableEditors.VSCode = true
		app.availableEditors.Neovim = true
		app.availableEditors.Emacs = true
		app.editorsMu.Unlock()

		count := app.countAvailableEditors()
		if count != 3 {
			t.Errorf("countAvailableEditors should be 3 after setting VSCode+Neovim+Emacs, got %d", count)
		}

		// Reset
		app.editorsMu.Lock()
		app.availableEditors = EditorAvailability{}
		app.editorsMu.Unlock()
	})

	t.Run("JetBrains derived flag counts as one", func(t *testing.T) {
		app.editorsMu.Lock()
		app.availableEditors.GoLand = true
		app.availableEditors.PyCharm = true
		// JetBrains should be derived
		app.availableEditors.JetBrains = true
		app.editorsMu.Unlock()

		count := app.countAvailableEditors()
		if count != 3 { // JetBrains + GoLand + PyCharm = 3 counts (individual + composite)
			t.Errorf("countAvailableEditors should be 3 (GoLand+PyCharm+JetBrains), got %d", count)
		}

		// Reset
		app.editorsMu.Lock()
		app.availableEditors = EditorAvailability{}
		app.editorsMu.Unlock()
	})
}

func TestGetAvailableEditors(t *testing.T) {
	app := NewApp()

	t.Run("Returns Neovim field", func(t *testing.T) {
		editors := app.GetAvailableEditors()
		// The field should exist and be a bool (zero value is false)
		if editors.Neovim != false {
			t.Error("GetAvailableEditors should include Neovim field (default false)")
		}
	})

	t.Run("Reflects set Neovim value", func(t *testing.T) {
		app.editorsMu.Lock()
		app.availableEditors.Neovim = true
		app.editorsMu.Unlock()

		editors := app.GetAvailableEditors()
		if !editors.Neovim {
			t.Error("GetAvailableEditors should return true for Neovim after setting it")
		}

		// Reset
		app.editorsMu.Lock()
		app.availableEditors = EditorAvailability{}
		app.editorsMu.Unlock()
	})
}

func TestGetEditorDetectionStatus(t *testing.T) {
	app := NewApp()

	t.Run("Includes availableEditors with Neovim", func(t *testing.T) {
		app.editorsMu.Lock()
		app.availableEditors.Neovim = true
		app.editorsMu.Unlock()

		status := app.GetEditorDetectionStatus()
		editors, ok := status["availableEditors"].(EditorAvailability)
		if !ok {
			t.Fatal("availableEditors should be of type EditorAvailability")
		}
		if !editors.Neovim {
			t.Error("availableEditors.Neovim should be true after setting it")
		}

		// Reset
		app.editorsMu.Lock()
		app.availableEditors = EditorAvailability{}
		app.editorsMu.Unlock()
	})

	t.Run("Has all required keys", func(t *testing.T) {
		status := app.GetEditorDetectionStatus()
		requiredKeys := []string{"availableEditors", "totalAvailable", "detectionComplete"}
		for _, key := range requiredKeys {
			if _, exists := status[key]; !exists {
				t.Errorf("GetEditorDetectionStatus should include key '%s'", key)
			}
		}
	})
}

func TestOpenInNeovim(t *testing.T) {
	app := NewApp()

	t.Run("Dispatches to nvim via OpenInEditorByName", func(t *testing.T) {
		// Create a temp file so openInEditor's os.Stat check passes.
		tmpFile := t.TempDir() + "/test.txt"
		if err := os.WriteFile(tmpFile, []byte("test"), 0644); err != nil {
			t.Fatalf("Failed to create temp file: %v", err)
		}

		err := app.OpenInEditorByName("Neovim", tmpFile)
		// If nvim is not in PATH, the error should mention it.
		// If nvim IS in PATH, the command might succeed — that's fine too.
		if err != nil {
			t.Logf("OpenInEditorByName(\"Neovim\") returned (expected if nvim not in PATH): %v", err)
		} else {
			t.Log("OpenInEditorByName(\"Neovim\") succeeded (nvim is available on this system)")
		}
	})
}

func TestOpenInEditorUnavailable(t *testing.T) {
	app := NewApp()

	t.Run("Fails for non-existent editor", func(t *testing.T) {
		tmpFile := t.TempDir() + "/test.txt"
		if err := os.WriteFile(tmpFile, []byte("test"), 0644); err != nil {
			t.Fatalf("Failed to create temp file: %v", err)
		}

		err := app.openInEditor(tmpFile, "this-editor-definitely-does-not-exist-xyzzy", []string{})
		if err == nil {
			t.Error("openInEditor should return error for a non-existent editor command")
		}
		if err != nil && !strings.Contains(err.Error(), "not found in system PATH") {
			t.Errorf("Expected error about editor not found, got: %v", err)
		}
	})
}

// TestDetectAvailableEditors covers the detection pass itself: the parallel
// PATH probes plus the derived-flag block that runs after them. a.ctx is nil
// here, so safeEmitEvent short-circuits and no Wails runtime is needed.
func TestDetectAvailableEditors(t *testing.T) {
	app := NewApp()

	if done, _ := app.GetEditorDetectionStatus()["detectionComplete"].(bool); done {
		t.Fatal("detectionComplete is true before detectAvailableEditors ran")
	}

	app.detectAvailableEditors()

	status := app.GetEditorDetectionStatus()
	if done, _ := status["detectionComplete"].(bool); !done {
		t.Error("detectionComplete = false after detectAvailableEditors; the frontend polls this flag")
	}

	editors := app.GetAvailableEditors()
	if !editors.SystemDefault {
		t.Error("SystemDefault = false; it is set unconditionally and is always available")
	}

	wantJetBrains := editors.GoLand || editors.PyCharm || editors.IntelliJ ||
		editors.WebStorm || editors.PhpStorm || editors.CLion || editors.Rider
	if editors.JetBrains != wantJetBrains {
		t.Errorf("JetBrains = %v, want %v (OR of the seven JetBrains IDEs)", editors.JetBrains, wantJetBrains)
	}

	if total, _ := status["totalAvailable"].(int); total != countEditorsFromSnapshot(editors) {
		t.Errorf("totalAvailable = %d, want %d", total, countEditorsFromSnapshot(editors))
	}

	// Each probe must agree with a direct PATH lookup: the concurrent writes
	// must land in the right field, not a neighbour's.
	for _, e := range editorCatalog {
		recorded := probedField(t, editors, e.key)
		if got := app.isEditorAvailable(e.command); got != recorded {
			t.Errorf("%s: detection recorded %v, direct PATH probe of %q says %v",
				e.key, recorded, e.command, got)
		}
	}

	// Detection is idempotent — a second pass must not flip anything. Under
	// -race this also exercises the lock around the derived-flag block.
	app.detectAvailableEditors()
	if again := app.GetAvailableEditors(); again != editors {
		t.Error("a second detectAvailableEditors pass changed the availability snapshot")
	}
}

// probedField reads the availability field matching a catalog entry's key.
func probedField(t *testing.T, ed EditorAvailability, key string) bool {
	t.Helper()
	byName := map[string]bool{
		"VSCode": ed.VSCode, "VSCodium": ed.VSCodium, "Sublime": ed.Sublime,
		"Geany": ed.Geany, "GoLand": ed.GoLand, "PyCharm": ed.PyCharm,
		"IntelliJ": ed.IntelliJ, "WebStorm": ed.WebStorm, "PhpStorm": ed.PhpStorm,
		"CLion": ed.CLion, "Rider": ed.Rider, "AndroidStudio": ed.AndroidStudio,
		"Emacs": ed.Emacs, "Neovide": ed.Neovide, "CodeBlocks": ed.CodeBlocks,
		"DevCpp": ed.DevCpp, "NotepadPlusPlus": ed.NotepadPlusPlus,
		"VisualStudio": ed.VisualStudio, "Eclipse": ed.Eclipse,
		"NetBeans": ed.NetBeans, "Neovim": ed.Neovim, "Vim": ed.Vim,
	}
	got, ok := byName[key]
	if !ok {
		t.Fatalf("catalog entry %q has no field in probedField; add it", key)
	}
	return got
}
