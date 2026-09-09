package main

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/sirupsen/logrus"
	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

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
