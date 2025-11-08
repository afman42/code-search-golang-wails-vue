//go:build linux

// Package main implements the backend functionality for the code search application.
// It provides functions for searching through code files, validating directories,
// and interacting with the system's file manager.
package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/sirupsen/logrus"
)

// ShowInFolder opens the containing folder of the given file path in the system's file manager.
// This function is cross-platform and works on Windows and Linux.
// It takes a file path and opens the parent directory containing that file.
func (a *App) ShowInFolder(filePath string) error {
	a.logDebug("Opening file location in folder", logrus.Fields{
		"filePath": filePath,
	})

	// Sanitize the input path to prevent directory traversal attacks
	cleanPath := filepath.Clean(filePath)

	// Check if the clean path still contains parent directory references at the start
	// which would indicate an attempt to access directories outside the expected scope
	if strings.HasPrefix(cleanPath, "../") || strings.Contains(cleanPath, "/../") || strings.HasSuffix(cleanPath, "/..") {
		a.logError("Invalid file path contains directory traversal", nil, logrus.Fields{
			"filePath": filePath,
		})
		return fmt.Errorf("invalid file path: contains directory traversal")
	}

	// Get the directory containing the file by taking the parent directory of the file path
	dir := filepath.Dir(cleanPath)

	// Validate that the directory path is absolute and properly formed
	absDir, err := filepath.Abs(dir)
	if err != nil {
		a.logError("Invalid directory path", err, logrus.Fields{
			"filePath": filePath,
			"dir":      dir,
		})
		return fmt.Errorf("invalid directory path: %v", err)
	}

	// Ensure the directory exists before attempting to open it
	if _, err := os.Stat(absDir); os.IsNotExist(err) {
		a.logError("Directory does not exist", err, logrus.Fields{
			"absDir": absDir,
		})
		return fmt.Errorf("directory does not exist: %s", absDir)
	}

	// Determine the OS and run appropriate command to open the file manager
	var cmd string
	var args []string

	switch runtime.GOOS {
	case "linux":
		// On Linux, use 'xdg-open' command to open the directory (works with most desktop environments)
		cmd = "xdg-open"
		args = []string{absDir}
	case "darwin":
		// On macOS, this function will be implemented in appDarwin.go
		a.logError("macOS folder opening not implemented in this build", nil, logrus.Fields{
			"filePath": filePath,
		})
		return fmt.Errorf("macOS folder opening not implemented in this build")
	default:
		a.logError("Unsupported platform for ShowInFolder", nil, logrus.Fields{
			"platform": runtime.GOOS,
		})
		return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}

	// Execute the command to open the file manager
	// Use Start() instead of Run() to avoid blocking the application
	a.logDebug("Executing command to open folder", logrus.Fields{
		"command": cmd,
		"args":    args,
	})
	
	command := exec.Command(cmd, args...)
	err = command.Start()
	if err != nil {
		a.logError("Failed to open folder", err, logrus.Fields{
			"command": cmd,
			"args":    args,
		})
		return fmt.Errorf("failed to open folder: %v", err)
	}

	a.logDebug("Successfully opened folder", logrus.Fields{
		"directory": absDir,
	})
	return nil
}

// openInEditor is a helper function to open a file in a specific editor
func (a *App) openInEditor(filePath string, editor string, args []string) error {
	a.logDebug("Opening file in editor", logrus.Fields{
		"filePath": filePath,
		"editor":   editor,
		"args":     args,
	})

	// Sanitize the input path to prevent directory traversal attacks
	cleanPath := filepath.Clean(filePath)

	// Check if the clean path still contains parent directory references at the start
	// which would indicate an attempt to access directories outside the expected scope
	if strings.HasPrefix(cleanPath, "../") || strings.Contains(cleanPath, "/../") || strings.HasSuffix(cleanPath, "/..") {
		a.logError("Invalid file path contains directory traversal", nil, logrus.Fields{
			"filePath": filePath,
		})
		return fmt.Errorf("invalid file path: contains directory traversal")
	}

	// Validate that the file exists before attempting to open it
	if _, err := os.Stat(cleanPath); os.IsNotExist(err) {
		a.logError("File does not exist", err, logrus.Fields{
			"filePath": cleanPath,
		})
		return fmt.Errorf("file does not exist: %s", cleanPath)
	}

	// Check if editor command is available in system PATH
	_, err := exec.LookPath(editor)
	if err != nil {
		a.logError("Editor not found in system PATH", err, logrus.Fields{
			"editor": editor,
		})
		return fmt.Errorf("editor '%s' not found in system PATH: %v", editor, err)
	}

	// Build the command arguments
	finalArgs := append(args, cleanPath)

	// Execute the command to open the file in the editor
	a.logDebug("Executing command to open file in editor", logrus.Fields{
		"command": editor,
		"args":    finalArgs,
	})
	
	command := exec.Command(editor, finalArgs...)
	err = command.Start()
	if err != nil {
		a.logError("Failed to open file in editor", err, logrus.Fields{
			"editor": editor,
			"args":   finalArgs,
		})
		return fmt.Errorf("failed to open file in %s: %v", editor, err)
	}

	a.logDebug("Successfully opened file in editor", logrus.Fields{
		"editor":   editor,
		"filePath": filePath,
	})
	return nil
}

// OpenInDefaultEditor opens a file in the system's default editor
func (a *App) OpenInDefaultEditor(filePath string) error {
	a.logDebug("Opening file in default editor", logrus.Fields{
		"filePath": filePath,
	})

	var cmd string
	var args []string

	switch runtime.GOOS {
	case "linux":
		cmd = "xdg-open"
		args = []string{filePath}
	default:
		a.logError("Unsupported platform for OpenInDefaultEditor", nil, logrus.Fields{
			"platform": runtime.GOOS,
		})
		return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}

	a.logDebug("Executing command to open file in default editor", logrus.Fields{
		"command": cmd,
		"args":    args,
	})
	
	command := exec.Command(cmd, args...)
	err := command.Start()
	if err != nil {
		a.logError("Failed to open file in default editor", err, logrus.Fields{
			"command": cmd,
			"args":    args,
		})
		return fmt.Errorf("failed to open file in default editor: %v", err)
	}

	a.logDebug("Successfully opened file in default editor", logrus.Fields{
		"filePath": filePath,
	})
	return nil
}
