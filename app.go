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
)



// ShowInFolder opens the containing folder of the given file path in the system's file manager.
// This function is cross-platform and works on Windows and Linux.
// It takes a file path and opens the parent directory containing that file.
func (a *App) ShowInFolder(filePath string) error {
	// Sanitize the input path to prevent directory traversal attacks
	cleanPath := filepath.Clean(filePath)

	// Check if the clean path still contains parent directory references at the start
	// which would indicate an attempt to access directories outside the expected scope
	if strings.HasPrefix(cleanPath, "../") || strings.Contains(cleanPath, "/../") || strings.HasSuffix(cleanPath, "/..") {
		return fmt.Errorf("invalid file path: contains directory traversal")
	}

	// Get the directory containing the file by taking the parent directory of the file path
	dir := filepath.Dir(cleanPath)

	// Validate that the directory path is absolute and properly formed
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return fmt.Errorf("invalid directory path: %v", err)
	}

	// Ensure the directory exists before attempting to open it
	if _, err := os.Stat(absDir); os.IsNotExist(err) {
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
		return fmt.Errorf("macOS folder opening not implemented in this build")
	default:
		return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}

	// Execute the command to open the file manager
	// Use Start() instead of Run() to avoid blocking the application
	command := exec.Command(cmd, args...)
	err = command.Start()
	if err != nil {
		return fmt.Errorf("failed to open folder: %v", err)
	}

	return nil
}
