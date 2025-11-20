package main

import (
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

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
		{"Atom", "atom", func(available bool) { a.availableEditors.Atom = available }},
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
	}

	// Check each editor and emit progress events
	totalEditors := len(editorsToCheck)
	for i, editor := range editorsToCheck {
		available := a.isEditorAvailable(editor.command)
		editor.setter(available)

		// Emit progress event for each editor checked
		progress := float32(i+1) / float32(totalEditors) * 100
		a.safeEmitEvent("editor-detection-progress", map[string]interface{}{
			"editor":    editor.name,
			"available": available,
			"progress":  progress,
			"total":     totalEditors,
			"completed": i + 1,
			"message":   fmt.Sprintf("Checking %s... %s", editor.name, map[bool]string{true: "✓", false: "✗"}[available]),
		})
	}

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

	// Emit completion event
	a.safeEmitEvent("editor-detection-complete", map[string]interface{}{
		"message":    "Editor detection complete!",
		"status":     "completed",
		"totalFound": a.countAvailableEditors(),
	})
}

// countAvailableEditors returns the number of available editors
func (a *App) countAvailableEditors() int {
	count := 0
	ed := a.availableEditors
	if ed.VSCode {
		count++
	}
	if ed.VSCodium {
		count++
	}
	if ed.Sublime {
		count++
	}
	if ed.Atom {
		count++
	}
	if ed.JetBrains {
		count++
	}
	if ed.Geany {
		count++
	}
	if ed.GoLand {
		count++
	}
	if ed.PyCharm {
		count++
	}
	if ed.IntelliJ {
		count++
	}
	if ed.WebStorm {
		count++
	}
	if ed.PhpStorm {
		count++
	}
	if ed.CLion {
		count++
	}
	if ed.Rider {
		count++
	}
	if ed.AndroidStudio {
		count++
	}
	if ed.Emacs {
		count++
	}
	if ed.Neovide {
		count++
	}
	if ed.CodeBlocks {
		count++
	}
	if ed.DevCpp {
		count++
	}
	if ed.NotepadPlusPlus {
		count++
	}
	if ed.VisualStudio {
		count++
	}
	if ed.Eclipse {
		count++
	}
	if ed.NetBeans {
		count++
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
	return a.availableEditors
}

// GetEditorDetectionStatus returns the current status of editor detection
func (a *App) GetEditorDetectionStatus() map[string]interface{} {
	return map[string]interface{}{
		"availableEditors":  a.availableEditors,
		"totalAvailable":    a.countAvailableEditors(),
		"detectionComplete": true, // By the time this is called, detection is complete at startup
	}
}

// GetDirectoryContents returns a list of all directory paths in the specified path.
// This function recursively walks the directory tree and collects all directories.
func (a *App) GetDirectoryContents(path string) ([]string, error) {
	var items []string

	// Walk the directory tree and collect all directories
	err := filepath.WalkDir(path, func(itemPath string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil // Skip unreadable items and continue
		}
		if d.IsDir() {
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

// ReadFile reads the content of a file and returns it as a string.
// This function is used by the frontend to read file contents for display in the modal.
func (a *App) ReadFile(filePath string) (string, error) {
	a.logDebug("Reading file", logrus.Fields{
		"filePath": filePath,
	})

	// Validate input
	if filePath == "" {
		a.logWarn("Empty file path provided", logrus.Fields{})
		return "", fmt.Errorf("file path is required")
	}

	// Check for potential path traversal patterns in the original filePath before cleaning
	// This catches cases where paths were constructed using traversal components like tempDir/../filename
	// Even if filepath.Join resolves these, our security check needs to detect the original intent
	if strings.Contains(filePath, "/../") || strings.Contains(filePath, "\\..") ||
	   strings.Contains(filePath, "../") || strings.Contains(filePath, "..\\") {
		a.logError("Invalid file path contains directory traversal", nil, logrus.Fields{
			"filePath": filePath,
		})
		return "", fmt.Errorf("invalid file path: contains directory traversal")
	}

	// Sanitize the input path to prevent directory traversal attacks
	cleanPath := filepath.Clean(filePath)

	// Validate that the path does not contain traversal sequences
	// Enhanced check to catch more types of traversal attempts
	if strings.Contains(cleanPath, "..") {
		a.logError("Invalid file path contains directory traversal", nil, logrus.Fields{
			"filePath": filePath,
			"cleanPath": cleanPath,
		})
		return "", fmt.Errorf("invalid file path: contains directory traversal")
	}

	// Additional security check: prevent null byte injection
	if strings.Contains(cleanPath, "\x00") {
		a.logError("Invalid file path contains null bytes", nil, logrus.Fields{
			"filePath": filePath,
		})
		return "", fmt.Errorf("invalid file path: contains null bytes")
	}

	// Additional security check: prevent command injection characters
	for _, dangerousChar := range []string{"|", "&", ";", "`", "$("} {
		if strings.Contains(cleanPath, dangerousChar) {
			a.logError("Invalid file path contains command injection characters", nil, logrus.Fields{
				"filePath": filePath,
				"char":     dangerousChar,
			})
			return "", fmt.Errorf("invalid file path: contains command injection characters")
		}
	}

	// Check if file exists
	if _, err := os.Stat(cleanPath); os.IsNotExist(err) {
		a.logWarn("File does not exist", logrus.Fields{
			"filePath": cleanPath,
		})
		return "", fmt.Errorf("file does not exist: %s", cleanPath)
	}

	// Read file content with size limit to prevent memory issues
	fileInfo, err := os.Stat(cleanPath)
	if err != nil {
		a.logError("Failed to get file info", err, logrus.Fields{
			"filePath": cleanPath,
		})
		return "", fmt.Errorf("failed to get file info: %v", err)
	}

	// Limit file size to prevent memory issues (e.g., 50MB)
	maxReadSize := int64(50 * 1024 * 1024) // 50MB
	if fileInfo.Size() > maxReadSize {
		a.logWarn("File too large to read", logrus.Fields{
			"filePath": cleanPath,
			"fileSize": fileInfo.Size(),
			"maxSize":  maxReadSize,
		})
		return "", fmt.Errorf("file too large to read: %s (size: %d, max: %d)", cleanPath, fileInfo.Size(), maxReadSize)
	}

	// Read file content
	content, err := os.ReadFile(cleanPath)
	if err != nil {
		a.logError("Failed to read file", err, logrus.Fields{
			"filePath": cleanPath,
		})
		return "", fmt.Errorf("failed to read file: %v", err)
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

// OpenInVSCode opens a file in VSCode editor
func (a *App) OpenInVSCode(filePath string) error {
	return a.openInEditor(filePath, "code", []string{"--goto"})
}

// OpenInVSCodium opens a file in VSCodium editor
func (a *App) OpenInVSCodium(filePath string) error {
	return a.openInEditor(filePath, "codium", []string{"--goto"})
}

// OpenInSublime opens a file in Sublime Text editor
func (a *App) OpenInSublime(filePath string) error {
	return a.openInEditor(filePath, "subl", []string{})
}

// OpenInAtom opens a file in Atom editor
func (a *App) OpenInAtom(filePath string) error {
	return a.openInEditor(filePath, "atom", []string{})
}

// OpenInJetBrains opens a file in the appropriate JetBrains IDE based on file type
func (a *App) OpenInJetBrains(filePath string) error {
	// Determine the appropriate JetBrains IDE based on file extension
	editor, args := a.getJetBrainsEditor(filePath)
	return a.openInEditor(filePath, editor, args)
}

// OpenInGeany opens a file in Geany editor
func (a *App) OpenInGeany(filePath string) error {
	return a.openInEditor(filePath, "geany", []string{})
}

// OpenInNeovim opens a file in Neovim editor
func (a *App) OpenInNeovim(filePath string) error {
	return a.openInEditor(filePath, "nvim", []string{})
}

// OpenInVim opens a file in Vim editor
func (a *App) OpenInVim(filePath string) error {
	return a.openInEditor(filePath, "vim", []string{})
}

// OpenInGoland opens a file in GoLand editor
func (a *App) OpenInGoland(filePath string) error {
	return a.openInEditor(filePath, "goland", []string{})
}

// OpenInPyCharm opens a file in PyCharm editor
func (a *App) OpenInPyCharm(filePath string) error {
	return a.openInEditor(filePath, "pycharm", []string{})
}

// OpenInIntelliJ opens a file in IntelliJ IDEA editor
func (a *App) OpenInIntelliJ(filePath string) error {
	return a.openInEditor(filePath, "idea", []string{})
}

// OpenInWebStorm opens a file in WebStorm editor
func (a *App) OpenInWebStorm(filePath string) error {
	return a.openInEditor(filePath, "webstorm", []string{})
}

// OpenInPhpStorm opens a file in PhpStorm editor
func (a *App) OpenInPhpStorm(filePath string) error {
	return a.openInEditor(filePath, "phpstorm", []string{})
}

// OpenInCLion opens a file in CLion editor
func (a *App) OpenInCLion(filePath string) error {
	return a.openInEditor(filePath, "clion", []string{})
}

// OpenInRider opens a file in Rider editor
func (a *App) OpenInRider(filePath string) error {
	return a.openInEditor(filePath, "rider", []string{})
}

// OpenInAndroidStudio opens a file in Android Studio editor
func (a *App) OpenInAndroidStudio(filePath string) error {
	return a.openInEditor(filePath, "studio", []string{})
}

// OpenInEmacs opens a file in Emacs editor
func (a *App) OpenInEmacs(filePath string) error {
	return a.openInEditor(filePath, "emacs", []string{})
}

// OpenInNeovide opens a file in Neovide editor
func (a *App) OpenInNeovide(filePath string) error {
	return a.openInEditor(filePath, "neovide", []string{})
}

// OpenInCodeBlocks opens a file in Code::Blocks editor
func (a *App) OpenInCodeBlocks(filePath string) error {
	return a.openInEditor(filePath, "codeblocks", []string{})
}

// OpenInDevCpp opens a file in Dev-C++ editor
func (a *App) OpenInDevCpp(filePath string) error {
	return a.openInEditor(filePath, "devcpp", []string{})
}

// OpenInNotepadPlusPlus opens a file in Notepad++ editor
func (a *App) OpenInNotepadPlusPlus(filePath string) error {
	return a.openInEditor(filePath, "notepad++", []string{})
}

// OpenInVisualStudio opens a file in Visual Studio editor
func (a *App) OpenInVisualStudio(filePath string) error {
	return a.openInEditor(filePath, "devenv", []string{"/edit"})
}

// OpenInEclipse opens a file in Eclipse IDE
func (a *App) OpenInEclipse(filePath string) error {
	return a.openInEditor(filePath, "eclipse", []string{})
}

// OpenInNetBeans opens a file in NetBeans IDE
func (a *App) OpenInNetBeans(filePath string) error {
	return a.openInEditor(filePath, "netbeans", []string{})
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