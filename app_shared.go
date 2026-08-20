// Package main implements the backend functionality for the code search application.
package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/sirupsen/logrus"
)

// sanitizePath validates that filePath is non-empty and free of directory-
// traversal components. It checks both the ORIGINAL input and the cleaned path,
// because filepath.Clean resolves "/tmp/../etc/x" to "/etc/x" and would hide
// traversal intent; matching on path components (not substrings) keeps
// legitimate names like "foo..bar.txt" working. Returns the cleaned path.
// Shared by validatePathForEditor and validatePathForShowInFolder.
func (a *App) sanitizePath(filePath string) (string, error) {
	if filePath == "" {
		a.logWarn("Empty file path provided", logrus.Fields{})
		return "", fmt.Errorf("file path is required")
	}

	if containsDotDotComponent(filePath) {
		a.logError("Invalid file path contains directory traversal", nil, logrus.Fields{
			"filePath": filePath,
		})
		return "", fmt.Errorf("invalid file path: contains directory traversal")
	}

	cleanPath := filepath.Clean(filePath)

	// Defense in depth: a cleaned path should never retain a ".." component.
	if containsDotDotComponent(cleanPath) {
		a.logError("Invalid file path contains directory traversal", nil, logrus.Fields{
			"filePath":  filePath,
			"cleanPath": cleanPath,
		})
		return "", fmt.Errorf("invalid file path: contains directory traversal")
	}

	return cleanPath, nil
}

// validatePathForEditor checks that the given filePath is safe (no path traversal)
// and that the file actually exists. Returns the cleaned absolute path or an error.
// This logic is shared by the linux and windows implementations of openInEditor.
func (a *App) validatePathForEditor(filePath string) (string, error) {
	cleanPath, err := a.sanitizePath(filePath)
	if err != nil {
		return "", err
	}

	if _, err := os.Stat(cleanPath); os.IsNotExist(err) {
		a.logError("File does not exist", err, logrus.Fields{
			"filePath": cleanPath,
		})
		return "", fmt.Errorf("file does not exist: %s", cleanPath)
	}

	return cleanPath, nil
}

// validatePathForShowInFolder checks that the given filePath is safe (no path
// traversal) and that the parent directory exists. Returns the cleaned absolute
// directory path or an error. Shared by the linux and windows implementations.
func (a *App) validatePathForShowInFolder(filePath string) (string, error) {
	cleanPath, err := a.sanitizePath(filePath)
	if err != nil {
		return "", err
	}

	dir := filepath.Dir(cleanPath)
	absDir, err := filepath.Abs(dir)
	if err != nil {
		a.logError("Invalid directory path", err, logrus.Fields{
			"filePath": filePath,
			"dir":      dir,
		})
		return "", fmt.Errorf("invalid directory path: %w", err)
	}

	if _, err := os.Stat(absDir); os.IsNotExist(err) {
		a.logError("Directory does not exist", err, logrus.Fields{
			"absDir": absDir,
		})
		return "", fmt.Errorf("directory does not exist: %s", absDir)
	}

	return absDir, nil
}

// lookUpEditor checks whether an editor command is available in the system
// PATH and returns its resolved absolute path. Callers exec that resolved
// path rather than re-resolving by name, closing the TOCTOU window where the
// PATH changes between the check and the exec.
func (a *App) lookUpEditor(editor string) (string, error) {
	path, err := exec.LookPath(editor)
	if err != nil {
		a.logError("Editor not found in system PATH", err, logrus.Fields{
			"editor": editor,
		})
		return "", fmt.Errorf("editor '%s' not found in system PATH: %w", editor, err)
	}
	return path, nil
}

// runCommand starts an external command and returns any error from Start.
// Both the linux and darwin platform files use this so it lives here.
func runCommand(name string, args []string) error {
	return startAndReap(exec.Command(name, args...))
}

// startAndReap starts cmd and reaps it in a background goroutine. Start
// without Wait leaves every exited child as a zombie until the parent (the
// whole app) exits — each folder reveal or editor launch leaked one process
// entry for the app's lifetime. Waiting asynchronously keeps the call
// non-blocking (editors like code/nvim keep running) while still reaping
// short-lived helpers such as xdg-open.
func startAndReap(cmd *exec.Cmd) error {
	if err := cmd.Start(); err != nil {
		return err
	}
	go func() {
		if err := cmd.Wait(); err != nil {
			// Non-zero exits are expected for editors/helpers — silence
			// those. Anything else (fork failure, killed, missing lib)
			// indicates the child died abnormally after Start; surface it.
			var exitErr *exec.ExitError
			if !errors.As(err, &exitErr) {
				log.Printf("command %s wait failed: %v", cmd.Path, err)
			}
		}
	}()
	return nil
}

// appendPath returns a fresh slice with path appended after args. It never
// appends in place: the args slices come from the package-level
// editorBindings map (and getJetBrainsEditor), which are shared across all
// calls. A plain append(args, path) would write into the shared backing
// array whenever args has spare capacity, corrupting concurrent
// OpenInEditorByName calls. Copying is one small allocation per launch.
func appendPath(args []string, path string) []string {
	out := make([]string, 0, len(args)+1)
	out = append(out, args...)
	return append(out, path)
}
