package main

import (
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/sirupsen/logrus"
)

// ErrUnknownEditor is returned when no catalog entry matches a binding name.
var ErrUnknownEditor = errors.New("unknown editor binding")

// editorEntry describes one probeable editor: the binding key used by
// OpenInEditorByName, the display name shown in detection progress events,
// the launch command and args, and the availability flag its probe sets.
type editorEntry struct {
	key         string
	displayName string
	command     string
	args        []string
	terminal    bool
	set         func(a *App, available bool)
}

// editorCatalog is the single source of truth for editor detection and
// launching. Each row drives one availability probe, one progress event, and
// one OpenInEditorByName lookup, so command/args can no longer drift between
// detection and launch (#18).
//
// "JetBrains" and "SystemDefault" are intentionally absent: JetBrains is
// derived from the per-IDE probes (the OR in detectAvailableEditors plus the
// extension router in getJetBrainsEditor), and SystemDefault is always true.
//
// terminal=true means the editor needs a controlling TTY — these get wrapped
// in a terminal emulator before exec so they actually render when launched
// from the GUI app.
var editorCatalog = []editorEntry{
	{key: "VSCode", displayName: "VSCode", command: "code", args: []string{"--goto"}, terminal: false, set: func(a *App, available bool) { a.availableEditors.VSCode = available }},
	{key: "VSCodium", displayName: "VSCodium", command: "codium", args: []string{"--goto"}, terminal: false, set: func(a *App, available bool) { a.availableEditors.VSCodium = available }},
	{key: "Sublime", displayName: "Sublime Text", command: "subl", args: nil, terminal: false, set: func(a *App, available bool) { a.availableEditors.Sublime = available }},
	{key: "Geany", displayName: "Geany", command: "geany", args: nil, terminal: false, set: func(a *App, available bool) { a.availableEditors.Geany = available }},
	{key: "GoLand", displayName: "GoLand", command: "goland", args: nil, terminal: false, set: func(a *App, available bool) { a.availableEditors.GoLand = available }},
	{key: "PyCharm", displayName: "PyCharm", command: "pycharm", args: nil, terminal: false, set: func(a *App, available bool) { a.availableEditors.PyCharm = available }},
	{key: "IntelliJ", displayName: "IntelliJ", command: "idea", args: nil, terminal: false, set: func(a *App, available bool) { a.availableEditors.IntelliJ = available }},
	{key: "WebStorm", displayName: "WebStorm", command: "webstorm", args: nil, terminal: false, set: func(a *App, available bool) { a.availableEditors.WebStorm = available }},
	{key: "PhpStorm", displayName: "PhpStorm", command: "phpstorm", args: nil, terminal: false, set: func(a *App, available bool) { a.availableEditors.PhpStorm = available }},
	{key: "CLion", displayName: "CLion", command: "clion", args: nil, terminal: false, set: func(a *App, available bool) { a.availableEditors.CLion = available }},
	{key: "Rider", displayName: "Rider", command: "rider", args: nil, terminal: false, set: func(a *App, available bool) { a.availableEditors.Rider = available }},
	{key: "AndroidStudio", displayName: "Android Studio", command: "studio", args: nil, terminal: false, set: func(a *App, available bool) { a.availableEditors.AndroidStudio = available }},
	{key: "Emacs", displayName: "Emacs", command: "emacs", args: nil, terminal: false, set: func(a *App, available bool) { a.availableEditors.Emacs = available }},
	{key: "Neovide", displayName: "Neovide", command: "neovide", args: nil, terminal: false, set: func(a *App, available bool) { a.availableEditors.Neovide = available }},
	{key: "CodeBlocks", displayName: "Code::Blocks", command: "codeblocks", args: nil, terminal: false, set: func(a *App, available bool) { a.availableEditors.CodeBlocks = available }},
	{key: "DevCpp", displayName: "Dev-C++", command: "devcpp", args: nil, terminal: false, set: func(a *App, available bool) { a.availableEditors.DevCpp = available }},
	{key: "NotepadPlusPlus", displayName: "Notepad++", command: "notepad++", args: nil, terminal: false, set: func(a *App, available bool) { a.availableEditors.NotepadPlusPlus = available }},
	{key: "VisualStudio", displayName: "Visual Studio", command: "devenv", args: []string{"/edit"}, terminal: false, set: func(a *App, available bool) { a.availableEditors.VisualStudio = available }},
	{key: "Eclipse", displayName: "Eclipse", command: "eclipse", args: nil, terminal: false, set: func(a *App, available bool) { a.availableEditors.Eclipse = available }},
	{key: "NetBeans", displayName: "NetBeans", command: "netbeans", args: nil, terminal: false, set: func(a *App, available bool) { a.availableEditors.NetBeans = available }},
	{key: "Neovim", displayName: "Neovim", command: "nvim", args: nil, terminal: true, set: func(a *App, available bool) { a.availableEditors.Neovim = available }},
	{key: "Vim", displayName: "Vim", command: "vim", args: nil, terminal: true, set: func(a *App, available bool) { a.availableEditors.Vim = available }},
	{key: "Nano", displayName: "Nano", command: "nano", args: nil, terminal: true, set: func(a *App, available bool) { a.availableEditors.Nano = available }},
	{key: "Micro", displayName: "Micro", command: "micro", args: nil, terminal: true, set: func(a *App, available bool) { a.availableEditors.Micro = available }},
	{key: "Helix", displayName: "Helix", command: "helix", args: nil, terminal: true, set: func(a *App, available bool) { a.availableEditors.Helix = available }},
}

// detectAvailableEditors checks which editors are available on the system
func (a *App) detectAvailableEditors() {
	// Emit event to notify frontend that editor detection is starting
	a.safeEmitEvent("editor-detection-start", map[string]any{
		"message": "Detecting available code editors...",
		"status":  "scanning",
	})
	// Signal completion even on a panic below: without this a panicking
	// probe or emit would leave GetEditorDetectionStatus reporting
	// "in progress" forever. Idempotent with the explicit store after
	// the probes.
	defer atomic.StoreInt32(&a.editorDetectionDone, 1)

	// Check each editor in parallel. Each probe is an independent exec.LookPath
	// (a PATH scan), so running them concurrently turns ~21 sequential scans into
	// roughly the cost of a single one. Results are written under editorsMu.
	totalEditors := len(editorCatalog)
	var wg sync.WaitGroup
	var completed int32
	for _, editor := range editorCatalog {
		wg.Add(1)
		go func(e editorEntry) {
			defer wg.Done()
			a.probeEditor(e, totalEditors, &completed)
		}(editor)
	}
	wg.Wait()
	// Derived flags are computed after all probes complete, under the same lock.
	a.editorsMu.Lock()
	// JetBrains is available if any of the specific JetBrains editors are available
	a.availableEditors.JetBrains = a.availableEditors.GoLand ||
		a.availableEditors.PyCharm ||
		a.availableEditors.IntelliJ ||
		a.availableEditors.WebStorm ||
		a.availableEditors.PhpStorm ||
		a.availableEditors.CLion ||
		a.availableEditors.Rider

	// System default is conceptually always available
	a.availableEditors.SystemDefault = true
	a.editorsMu.Unlock()

	// Mark detection as complete AFTER all probes and derived flags are
	// settled, so GetEditorDetectionStatus returns an honest flag. It runs
	// in a background goroutine at startup, so the frontend can poll this
	// instead of trusting a hardcoded true.
	atomic.StoreInt32(&a.editorDetectionDone, 1)

	// Emit completion event
	a.safeEmitEvent("editor-detection-complete", map[string]any{
		"message":    "Editor detection complete!",
		"status":     "completed",
		"totalFound": a.countAvailableEditors(),
	})
}

// probeEditor probes one editor entry: LookPath availability, availability
// write under lock, and a progress emit. Split out so
// detectAvailableEditors stays readable.
func (a *App) probeEditor(e editorEntry, totalEditors int, completed *int32) {
	// A panicking probe (nil set fn, unexpected LookPath failure
	// mode) must not kill detection for the other ~20 editors.
	defer func() {
		if r := recover(); r != nil {
			a.warnErr("Editor probe panicked", nil, logrus.Fields{"editor": e.displayName, "panic": r})
		}
	}()
	available := a.isEditorAvailable(e.command)

	a.editorsMu.Lock()
	e.set(a, available)
	a.editorsMu.Unlock()

	// Emit progress event for each editor checked
	done := atomic.AddInt32(completed, 1)
	progress := float32(done) / float32(totalEditors) * 100
	a.safeEmitEvent("editor-detection-progress", map[string]any{
		"editor":    e.displayName,
		"available": available,
		"progress":  progress,
		"total":     totalEditors,
		"completed": int(done),
		"message":   fmt.Sprintf("Checking %s... %s", e.displayName, map[bool]string{true: "✓", false: "✗"}[available]),
	})
}

// countAvailableEditors returns the number of available editors. It takes a
// snapshot of the availability struct under the read lock and counts from
// that snapshot.
func (a *App) countAvailableEditors() int {
	a.editorsMu.RLock()
	ed := a.availableEditors
	a.editorsMu.RUnlock()
	return countEditorsFromSnapshot(ed)
}

// isEditorAvailable checks if an editor command is available in the system PATH
func (a *App) isEditorAvailable(editor string) bool {
	_, err := exec.LookPath(editor)
	return err == nil
}

// GetAvailableEditors returns information about which editors are available on the system
func (a *App) GetAvailableEditors() EditorAvailability {
	a.editorsMu.RLock()
	defer a.editorsMu.RUnlock()
	return a.availableEditors
}

// GetEditorDetectionStatus returns the current status of editor detection.
// The count is computed from the snapshot taken under the single RLock below
// (via countEditorsFromSnapshot), avoiding the redundant second RLock that
// the previous implementation incurred by calling countAvailableEditors
// after releasing the lock (#20). detectionComplete reflects whether the
// background detection goroutine has actually finished.
func (a *App) GetEditorDetectionStatus() map[string]any {
	a.editorsMu.RLock()
	editors := a.availableEditors
	a.editorsMu.RUnlock()
	return map[string]any{
		"availableEditors":  editors,
		"totalAvailable":    countEditorsFromSnapshot(editors),
		"detectionComplete": atomic.LoadInt32(&a.editorDetectionDone) == 1,
	}
}

// catalogEntry returns the editorCatalog row with the given binding key, or
// nil when no such editor exists.
func catalogEntry(key string) *editorEntry {
	for i := range editorCatalog {
		if editorCatalog[i].key == key {
			return &editorCatalog[i]
		}
	}
	return nil
}

// OpenInEditorByName opens a file in the editor identified by the given
// binding name (an editorCatalog key). This is the sole Wails-bound
// dispatcher for named editors; the frontend calls it directly with a
// binding name from its editorBindingName map.
//
// "JetBrains" is a special case: rather than mapping to a single command,
// it routes to the appropriate JetBrains IDE (GoLand, PyCharm, IntelliJ,
// etc.) based on the file's extension via getJetBrainsEditor.
func (a *App) OpenInEditorByName(name string, filePath string) error {
	if name == "JetBrains" {
		editor, args := a.getJetBrainsEditor(filePath)
		return a.openInEditor(filePath, editor, args, false)
	}
	entry := catalogEntry(name)
	if entry == nil {
		return fmt.Errorf("%w: %q", ErrUnknownEditor, name)
	}
	return a.openInEditor(filePath, entry.command, entry.args, entry.terminal)
}

// getJetBrainsEditor determines the appropriate JetBrains IDE based on the
// file extension. It resolves the IDE through editorCatalog so every launch
// command lives in exactly one table; the returned args are the catalog row's
// (empty for all JetBrains IDEs).
func (a *App) getJetBrainsEditor(filePath string) (string, []string) {
	ext := strings.ToLower(filepath.Ext(filePath))

	var key string
	switch ext {
	case ".go":
		key = "GoLand"
	case ".py", ".pyw":
		key = "PyCharm"
	case ".js", ".ts", ".jsx", ".tsx", ".html", ".css", ".json":
		key = "WebStorm"
	case ".php", ".phtml", ".php3", ".php4", ".php5", ".php7", ".php8":
		key = "PhpStorm"
	case ".java", ".kt", ".kts", ".groovy", ".gradle":
		key = "IntelliJ"
	case ".cpp", ".cxx", ".cc", ".c", ".h", ".hpp", ".hxx":
		key = "CLion"
	case ".cs":
		key = "Rider"
	default:
		// Generic and unknown file types default to IntelliJ (the old
		// switch routed .xml/.yml/.yaml/.properties/.sql/.dart/.md here too).
		key = "IntelliJ"
	}

	entry := catalogEntry(key)
	if entry == nil {
		// Unreachable while editorCatalog keeps every JetBrains row —
		// TestEditorCatalogConsistency guards that. Fail with a launch
		// error instead of guessing a command.
		return "", nil
	}
	return entry.command, entry.args
}

// countEditorsFromSnapshot counts the true fields of an EditorAvailability
// snapshot without taking the lock. Callers that already hold a snapshot
// (e.g. GetEditorDetectionStatus below) should call this directly to avoid
// re-acquiring editorsMu for a second time within the same call (#20).
func countEditorsFromSnapshot(ed EditorAvailability) int {
	var count int
	for _, ptr := range []*bool{
		&ed.VSCode, &ed.VSCodium, &ed.Sublime, &ed.JetBrains,
		&ed.Geany, &ed.GoLand, &ed.PyCharm, &ed.IntelliJ, &ed.WebStorm,
		&ed.PhpStorm, &ed.CLion, &ed.Rider, &ed.AndroidStudio, &ed.Emacs,
		&ed.Neovide, &ed.CodeBlocks, &ed.DevCpp, &ed.NotepadPlusPlus,
		&ed.VisualStudio, &ed.Eclipse, &ed.NetBeans, &ed.Neovim, &ed.Vim,
	} {
		if *ptr {
			count++
		}
	}
	return count
}
