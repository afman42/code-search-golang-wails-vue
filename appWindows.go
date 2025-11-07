//go:build windows

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
	"syscall"
)

// SelectDirectory opens a native directory selection dialog and returns the selected path.
// This function implements cross-platform directory selection using system dialogs:
// - On Linux: Tries multiple options in order of preference (zenity, kdialog, yad)
// - On Windows: Uses PowerShell to show a native folder browser dialog
// - On macOS: Uses AppleScript to show a native dialog (in the macOS-specific file)
func (a *App) SelectDirectory(title string) (string, error) {
	var cmd string
	var args []string

	switch runtime.GOOS {
	case "windows":
		// On Windows, use PowerShell to show a folder browser dialog
		// Using System.Windows.Forms.FolderBrowserDialog for native Windows experience
		script := `
        Add-Type -AssemblyName System.Windows.Forms
        $folderBrowser = New-Object System.Windows.Forms.FolderBrowserDialog
        $folderBrowser.Description = "` + title + `"
        $result = $folderBrowser.ShowDialog()
        if ($result -eq [System.Windows.Forms.DialogResult]::OK) {
            Write-Output $folderBrowser.SelectedPath
        } else {
            Write-Output ""
        }
        `
		cmd = "powershell"
		args = []string{"-Command", script}
	case "darwin":
		// On macOS, this function will be implemented in appDarwin.go with AppleScript
		return "", fmt.Errorf("macOS directory selection not implemented in this build")
	default:
		return "", fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}

	// Execute the command to show the directory picker
	command := exec.Command(cmd, args...)
	command.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: 0x08000000,
	}
	output, err := command.Output()
	if err != nil {
		// Check if the user cancelled the dialog (exit code 1 for zenity, etc.)
		if exitError, ok := err.(*exec.ExitError); ok {
			if exitError.ExitCode() == 1 {
				// User cancelled the dialog - return empty string but no error
				return "", nil
			}
		}
		return "", fmt.Errorf("failed to show directory picker: %v", err)
	}

	// Clean up the output (remove trailing newline)
	path := strings.TrimSpace(string(output))
	if path == "" {
		// User cancelled the dialog
		return "", nil
	}

	return path, nil
}

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
	case "windows":
		// On Windows, use 'cmd /c start' to open the directory
		cmd = "cmd"
		args = []string{"/c", "start", absDir}
	case "darwin":
		// On macOS, this function will be implemented in appDarwin.go
		return fmt.Errorf("macOS folder opening not implemented in this build")
	default:
		return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}

	// Execute the command to open the file manager
	// Use Start() instead of Run() to avoid blocking the application
	command := exec.Command(cmd, args...)
	command.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: 0x08000000,
	}
	err = command.Start()
	if err != nil {
		return fmt.Errorf("failed to open folder: %v", err)
	}

	return nil
}
