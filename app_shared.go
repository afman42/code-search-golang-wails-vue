// Package main implements the backend functionality for the code search application.
package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/sirupsen/logrus"
)

// validatePathForEditor checks that the given filePath is safe (no path traversal)
// and that the file actually exists. Returns the cleaned absolute path or an error.
// This logic is shared by the linux and windows implementations of openInEditor.
func (a *App) validatePathForEditor(filePath string) (string, error) {
	cleanPath := filepath.Clean(filePath)

	if strings.HasPrefix(cleanPath, "../") ||
		strings.Contains(cleanPath, "/../") ||
		strings.HasSuffix(cleanPath, "/..") {
		a.logError("Invalid file path contains directory traversal", nil, logrus.Fields{
			"filePath": filePath,
		})
		return "", fmt.Errorf("invalid file path: contains directory traversal")
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
	cleanPath := filepath.Clean(filePath)

	if strings.HasPrefix(cleanPath, "../") ||
		strings.Contains(cleanPath, "/../") ||
		strings.HasSuffix(cleanPath, "/..") {
		a.logError("Invalid file path contains directory traversal", nil, logrus.Fields{
			"filePath": filePath,
		})
		return "", fmt.Errorf("invalid file path: contains directory traversal")
	}

	dir := filepath.Dir(cleanPath)
	absDir, err := filepath.Abs(dir)
	if err != nil {
		a.logError("Invalid directory path", err, logrus.Fields{
			"filePath": filePath,
			"dir":      dir,
		})
		return "", fmt.Errorf("invalid directory path: %v", err)
	}

	if _, err := os.Stat(absDir); os.IsNotExist(err) {
		a.logError("Directory does not exist", err, logrus.Fields{
			"absDir": absDir,
		})
		return "", fmt.Errorf("directory does not exist: %s", absDir)
	}

	return absDir, nil
}

// lookUpEditor checks whether an editor command is available in the system PATH.
func (a *App) lookUpEditor(editor string) error {
	_, err := exec.LookPath(editor)
	if err != nil {
		a.logError("Editor not found in system PATH", err, logrus.Fields{
			"editor": editor,
		})
		return fmt.Errorf("editor '%s' not found in system PATH: %v", editor, err)
	}
	return nil
}

// runCommand starts an external command and returns any error from Start.
// Both the linux and windows platform files use this so it lives here.
func runCommand(name string, args []string) error {
	return exec.Command(name, args...).Start()
}
