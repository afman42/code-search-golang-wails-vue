package main

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/sirupsen/logrus"
	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

// detectAvailableEditors checks which editors are available on the system
func (a *App) detectAvailableEditors() {
	// Emit event to notify frontend that editor detection is starting
	a.safeEmitEvent("editor-detection-start", map[string]interface{}{
		"message": "Detecting available code editors...",
		"status":  "scanning",
	})

	// Define editor commands to check with their display names
	editorsToCheck := []struct {
		name    string
		command string
		setter  func(bool)
	}{
		{"VSCode", "code", func(available bool) { a.availableEditors.VSCode = available }},
		{"VSCodium", "codium", func(available bool) { a.availableEditors.VSCodium = available }},
		{"Sublime Text", "subl", func(available bool) { a.availableEditors.Sublime = available }},
		{"Geany", "geany", func(available bool) { a.availableEditors.Geany = available }},
		{"GoLand", "goland", func(available bool) { a.availableEditors.GoLand = available }},
		{"PyCharm", "pycharm", func(available bool) { a.availableEditors.PyCharm = available }},
		{"IntelliJ", "idea", func(available bool) { a.availableEditors.IntelliJ = available }},
		{"WebStorm", "webstorm", func(available bool) { a.availableEditors.WebStorm = available }},
		{"PhpStorm", "phpstorm", func(available bool) { a.availableEditors.PhpStorm = available }},
		{"CLion", "clion", func(available bool) { a.availableEditors.CLion = available }},
		{"Rider", "rider", func(available bool) { a.availableEditors.Rider = available }},
		{"Android Studio", "studio", func(available bool) { a.availableEditors.AndroidStudio = available }},
		{"Emacs", "emacs", func(available bool) { a.availableEditors.Emacs = available }},
		{"Neovide", "neovide", func(available bool) { a.availableEditors.Neovide = available }},
		{"Code::Blocks", "codeblocks", func(available bool) { a.availableEditors.CodeBlocks = available }},
		{"Dev-C++", "devcpp", func(available bool) { a.availableEditors.DevCpp = available }},
		{"Notepad++", "notepad++", func(available bool) { a.availableEditors.NotepadPlusPlus = available }},
		{"Visual Studio", "devenv", func(available bool) { a.availableEditors.VisualStudio = available }},
		{"Eclipse", "eclipse", func(available bool) { a.availableEditors.Eclipse = available }},
		{"NetBeans", "netbeans", func(available bool) { a.availableEditors.NetBeans = available }},
		{"Neovim", "nvim", func(available bool) { a.availableEditors.Neovim = available }},
		{"Vim", "vim", func(available bool) { a.availableEditors.Vim = available }},
	}

	// Check each editor in parallel. Each probe is an independent exec.LookPath
	// (a PATH scan), so running them concurrently turns ~21 sequential scans into
	// roughly the cost of a single one. Results are written under editorsMu.
	totalEditors := len(editorsToCheck)
	var wg sync.WaitGroup
	var completed int32
	for _, editor := range editorsToCheck {
		wg.Add(1)
		go func(editor struct {
			name    string
			command string
			setter  func(bool)
		}) {
			defer wg.Done()
			available := a.isEditorAvailable(editor.command)

			a.editorsMu.Lock()
			editor.setter(available)
			a.editorsMu.Unlock()

			// Emit progress event for each editor checked
			done := atomic.AddInt32(&completed, 1)
			progress := float32(done) / float32(totalEditors) * 100
			a.safeEmitEvent("editor-detection-progress", map[string]interface{}{
				"editor":    editor.name,
				"available": available,
				"progress":  progress,
				"total":     totalEditors,
				"completed": int(done),
				"message":   fmt.Sprintf("Checking %s... %s", editor.name, map[bool]string{true: "✓", false: "✗"}[available]),
			})
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
	a.safeEmitEvent("editor-detection-complete", map[string]interface{}{
		"message":    "Editor detection complete!",
		"status":     "completed",
		"totalFound": a.countAvailableEditors(),
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

// countEditorsFromSnapshot counts the true fields of an EditorAvailability
// snapshot without taking the lock. Callers that already hold a snapshot
// (e.g. GetEditorDetectionStatus below) should call this directly to avoid
// re-acquiring editorsMu for a second time within the same call (#20).
func countEditorsFromSnapshot(ed EditorAvailability) int {
	count := 0
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
func (a *App) GetEditorDetectionStatus() map[string]interface{} {
	a.editorsMu.RLock()
	editors := a.availableEditors
	a.editorsMu.RUnlock()
	return map[string]interface{}{
		"availableEditors":  editors,
		"totalAvailable":    countEditorsFromSnapshot(editors),
		"detectionComplete": atomic.LoadInt32(&a.editorDetectionDone) == 1,
	}
}

// GetDirectoryContents returns a list of all directory paths in the specified path.
// This function recursively walks the directory tree and collects all directories.
// Hidden directories (dot-prefixed, e.g. .git, .vscode) are skipped, matching the
// search collection walk.
func (a *App) GetDirectoryContents(path string) ([]string, error) {
	var items []string

	// Walk the directory tree and collect all directories
	err := filepath.WalkDir(path, func(itemPath string, d fs.DirEntry, err error) error {
		if err != nil {
			// A directory that vanished mid-walk is not a problem; any other
			// error (permission, I/O) truncates the listing silently and
			// must surface instead of returning a partial tree as if
			// complete.
			if os.IsNotExist(err) {
				return nil
			}
			return err
		}
		if d.IsDir() {
			// Skip hidden directories that start with a dot (e.g., .git, .vscode)
			if strings.HasPrefix(d.Name(), ".") && itemPath != path {
				return filepath.SkipDir
			}
			items = append(items, itemPath) // Only add directories, not files
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return items, nil
}

// ValidateDirectory checks if a directory exists and is accessible for reading.
// This function is useful for validating user-provided directory paths before performing operations.
func (a *App) ValidateDirectory(path string) (bool, error) {
	a.logDebug("Validating directory", logrus.Fields{
		"directory": path,
	})

	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			a.logWarn("Directory does not exist", logrus.Fields{
				"directory": path,
			})
			return false, fmt.Errorf("directory does not exist: %s", path)
		}
		a.logError("Error accessing directory", err, logrus.Fields{
			"directory": path,
		})
		return false, err
	}

	if !info.IsDir() {
		a.logWarn("Path is not a directory", logrus.Fields{
			"directory": path,
			"fileInfo":  info.IsDir(),
		})
		return false, fmt.Errorf("path is not a directory: %s", path)
	}

	// Try to read the directory to ensure it's accessible
	_, err = os.ReadDir(path)
	if err != nil {
		a.logError("Directory is not accessible", err, logrus.Fields{
			"directory": path,
		})
		return false, fmt.Errorf("directory is not accessible: %s", path)
	}

	a.logDebug("Directory validation successful", logrus.Fields{
		"directory": path,
	})
	return true, nil
}

// containsDotDotComponent reports whether the given path contains a ".." path
// component, handling both Unix ("/") and Windows ("\") separators. Unlike a raw
// substring check for "..", this only flags genuine parent-directory components,
// so legitimate names such as "foo..bar.txt" are not rejected.
func containsDotDotComponent(path string) bool {
	// Normalize Windows separators so the split below works cross-platform.
	normalized := strings.ReplaceAll(path, "\\", "/")
	for _, segment := range strings.Split(normalized, "/") {
		if segment == ".." {
			return true
		}
	}
	return false
}

// ReadFile reads the content of a file and returns it as a string.
// This function is used by the frontend to read file contents for display in the modal.
func (a *App) ReadFile(filePath string) (string, error) {
	a.logDebug("Reading file", logrus.Fields{
		"filePath": filePath,
	})

	// Reuse the shared sanitizePath validation (empty, dot-dot, clean). This
	// replaces the previous inline re-implementation (DRY) and removes the
	// double os.Stat that opened a TOCTOU window (#19).
	cleanPath, err := a.sanitizePath(filePath)
	if err != nil {
		return "", err
	}

	// Additional char-level check: prevent null byte injection. The null-byte
	// check is the only char-level check that matters here — ReadFile never
	// passes the path to a shell, so shell metacharacters like |, &, ;, `,
	// and $(...) are NOT security issues and are valid in Unix filenames
	// (e.g. "foo$(bar).txt", "a;b.txt"). The previous filter rejected
	// legitimate files (#14). Path traversal is already handled by the
	// sanitizePath + containsDotDotComponent checks above.
	if strings.Contains(cleanPath, "\x00") {
		a.logError("Invalid file path contains null bytes", nil, logrus.Fields{
			"filePath": filePath,
		})
		return "", fmt.Errorf("invalid file path: contains null bytes")
	}

	// Check if file exists and get its size in one stat (not two like the
	// previous implementation — closes the TOCTOU window between the
	// existence check and the size check).
	fileInfo, err := os.Stat(cleanPath)
	if err != nil {
		if os.IsNotExist(err) {
			a.logWarn("File does not exist", logrus.Fields{
				"filePath": cleanPath,
			})
			return "", fmt.Errorf("file does not exist: %s", cleanPath)
		}
		a.logError("Failed to get file info", err, logrus.Fields{
			"filePath": cleanPath,
		})
		return "", fmt.Errorf("failed to get file info: %w", err)
	}

	// Limit file size to prevent memory issues (e.g., 50MB)
	const maxReadFileSize = 50 * 1024 * 1024 // 50MB
	if fileInfo.Size() > maxReadFileSize {
		a.logWarn("File too large to read", logrus.Fields{
			"filePath": cleanPath,
			"fileSize": fileInfo.Size(),
			"maxSize":  maxReadFileSize,
		})
		return "", fmt.Errorf("file too large to read: %s (size: %d, max: %d)", cleanPath, fileInfo.Size(), maxReadFileSize)
	}

	// Read file content using io.ReadAll with LimitReader for defense in
	// depth: a file that grew past maxReadFileSize between Stat and Read
	// is bounded and rejected.
	content, err := func() ([]byte, error) {
		f, err := os.Open(cleanPath)
		if err != nil {
			return nil, err
		}
		defer f.Close()
		// Read up to maxReadFileSize+1 so we can detect overflow.
		b, err := io.ReadAll(io.LimitReader(f, maxReadFileSize+1))
		if err != nil {
			return nil, err
		}
		if int64(len(b)) > maxReadFileSize {
			return nil, fmt.Errorf("file too large to read: %s (size: %d, max: %d)", cleanPath, len(b), maxReadFileSize)
		}
		return b, nil
	}()
	if err != nil {
		a.logError("Failed to read file", err, logrus.Fields{
			"filePath": cleanPath,
		})
		return "", fmt.Errorf("failed to read file: %w", err)
	}

	a.logDebug("Successfully read file", logrus.Fields{
		"filePath": cleanPath,
		"fileSize": len(content),
	})
	return string(content), nil
}

// SelectDirectory opens a native directory selection dialog and returns the selected path.
// This function uses the Wails runtime dialog to provide a native directory selection
// experience across all platforms (Windows, Linux, macOS).
func (a *App) SelectDirectory(title string) (string, error) {
	// Validate input parameters
	if title == "" {
		title = "Select Directory" // Use default title if none provided
	}

	// Check if we have a valid context
	if a.ctx == nil {
		a.logError("No valid context available for directory selection dialog", nil, logrus.Fields{})
		return "", fmt.Errorf("no valid context available for dialog - application may not be fully initialized")
	}

	a.logDebug("Opening directory selection dialog", logrus.Fields{
		"title": title,
	})

	// Prepare dialog options with the provided title
	dialogOptions := wailsRuntime.OpenDialogOptions{
		Title: title,
	}

	// Use Wails runtime OpenDirectoryDialog to show the native dialog
	selectedPath, err := wailsRuntime.OpenDirectoryDialog(a.ctx, dialogOptions)
	if err != nil {
		a.logError("Failed to open directory dialog", err, logrus.Fields{
			"title": title,
		})
		// Return any error that occurred during the dialog operation
		// This includes system-level errors but excludes user cancellation
		return "", fmt.Errorf("failed to open directory dialog: %w", err)
	}

	// If selectedPath is empty, the user cancelled the dialog
	if selectedPath == "" {
		a.logDebug("Directory selection dialog cancelled by user", logrus.Fields{})
	}

	// Return empty string with no error to indicate cancellation
	return selectedPath, nil
}

// editorBindings is the single source of truth for the command and args used
// to launch each editor. Adding a new editor is now one map entry plus one
// thin Wails-bound wrapper method (OpenInX) — previously the cmd/args were
// hardcoded in each wrapper method, so adding an editor meant touching
// scattered code (#18).
//
// The keys are the public "binding names" used by OpenInEditorByName and the
// OpenInX wrappers; the values are the executable name and the extra args
// passed before the file path.
var editorBindings = map[string]struct {
	command string
	args    []string
}{
	"VSCode":          {"code", []string{"--goto"}},
	"VSCodium":        {"codium", []string{"--goto"}},
	"Sublime":         {"subl", nil},
	"Geany":           {"geany", nil},
	"GoLand":          {"goland", nil},
	"PyCharm":         {"pycharm", nil},
	"IntelliJ":        {"idea", nil},
	"WebStorm":        {"webstorm", nil},
	"PhpStorm":        {"phpstorm", nil},
	"CLion":           {"clion", nil},
	"Rider":           {"rider", nil},
	"AndroidStudio":   {"studio", nil},
	"Emacs":           {"emacs", nil},
	"Neovide":         {"neovide", nil},
	"CodeBlocks":      {"codeblocks", nil},
	"DevCpp":          {"devcpp", nil},
	"NotepadPlusPlus": {"notepad++", nil},
	"VisualStudio":    {"devenv", []string{"/edit"}},
	"Eclipse":         {"eclipse", nil},
	"NetBeans":        {"netbeans", nil},
	"Neovim":          {"nvim", nil},
	"Vim":             {"vim", nil},
}

// OpenInEditorByName opens a file in the editor identified by the given
// binding name (a key in editorBindings). This is the sole Wails-bound
// dispatcher for named editors; the frontend calls it directly with a
// binding name from its editorBindingName map.
//
// "JetBrains" is a special case: rather than mapping to a single command,
// it routes to the appropriate JetBrains IDE (GoLand, PyCharm, IntelliJ,
// etc.) based on the file's extension via getJetBrainsEditor.
func (a *App) OpenInEditorByName(name string, filePath string) error {
	if name == "JetBrains" {
		editor, args := a.getJetBrainsEditor(filePath)
		return a.openInEditor(filePath, editor, args)
	}
	binding, ok := editorBindings[name]
	if !ok {
		return fmt.Errorf("unknown editor binding: %q", name)
	}
	return a.openInEditor(filePath, binding.command, binding.args)
}

// getJetBrainsEditor determines the appropriate JetBrains IDE based on file extension
func (a *App) getJetBrainsEditor(filePath string) (string, []string) {
	ext := strings.ToLower(filepath.Ext(filePath))

	switch ext {
	case ".go":
		return "goland", []string{}
	case ".py", ".pyw":
		return "pycharm", []string{}
	case ".js", ".ts", ".jsx", ".tsx", ".html", ".css", ".json":
		return "webstorm", []string{}
	case ".php", ".phtml", ".php3", ".php4", ".php5", ".php7", ".php8":
		return "phpstorm", []string{}
	case ".java", ".kt", ".kts", ".groovy":
		return "idea", []string{}
	case ".gradle":
		return "idea", []string{}
	case ".cpp", ".cxx", ".cc", ".c", ".h", ".hpp", ".hxx":
		return "clion", []string{}
	case ".cs":
		return "rider", []string{}
	case ".xml":
		return "idea", []string{}
	case ".yml", ".yaml", ".properties", ".sql", ".dart", ".md":
		// For generic files, use idea by default
		return "idea", []string{}
	default:
		// Default to idea for other file types
		return "idea", []string{}
	}
}
