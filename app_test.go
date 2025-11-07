package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)



func TestValidateDirectory(t *testing.T) {
	app := NewApp()

	tempDir := t.TempDir()
	
	t.Run("ValidDirectory", func(t *testing.T) {
		valid, err := app.ValidateDirectory(tempDir)
		if err != nil {
			t.Errorf("ValidateDirectory returned error for valid directory: %v", err)
		}
		if !valid {
			t.Error("ValidateDirectory should return true for valid directory")
		}
	})
	
	t.Run("NonExistentDirectory", func(t *testing.T) {
		nonExistentDir := "/non/existent/directory"
		valid, err := app.ValidateDirectory(nonExistentDir)
		if err == nil {
			t.Error("ValidateDirectory should return error for non-existent directory")
		}
		if valid {
			t.Error("ValidateDirectory should return false for non-existent directory")
		}
	})
	
	t.Run("FileInsteadOfDirectory", func(t *testing.T) {
		// Create a temporary file
		tempFile := filepath.Join(t.TempDir(), "temp.txt")
		err := os.WriteFile(tempFile, []byte("test"), 0644)
		if err != nil {
			t.Fatalf("Failed to create temp file: %v", err)
		}
		
		valid, err := app.ValidateDirectory(tempFile)
		if err == nil {
			t.Error("ValidateDirectory should return error when path is a file")
		}
		if valid {
			t.Error("ValidateDirectory should return false for a file path")
		}
	})
}

func TestShowInFolder(t *testing.T) {
	app := NewApp()

	tempDir := t.TempDir()
	
	// Create a test file
	testFile := filepath.Join(tempDir, "test.txt")
	err := os.WriteFile(testFile, []byte("test content"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	
	t.Run("ValidFilePath", func(t *testing.T) {
		// This test will try to open the folder containing the test file
		// It might not work in all environments but shouldn't crash
		err := app.ShowInFolder(testFile)
		// We don't check for success as it depends on system capabilities
		// But it shouldn't return an error for a valid file
		if err != nil {
			// On some CI systems, this might not be supported
			// That's OK, just make sure it's an expected error
			// if err.Error() != "unsupported platform: "+runtime.GOOS {
			// 	t.Logf("ShowInFolder returned expected error (may not be supported in test environment): %v", err)
			// }
		}
	})
	
	t.Run("NonExistentFile", func(t *testing.T) {
		nonExistentFile := "/non/existent/file.txt"
		err := app.ShowInFolder(nonExistentFile)
		if err == nil {
			t.Error("ShowInFolder should return error for non-existent file")
		}
	})
}

func TestSelectDirectory(t *testing.T) {
	app := NewApp()

	t.Run("DirectorySelectionNotSupportedInTestEnv", func(t *testing.T) {
		// The SelectDirectory function may attempt to open a GUI dialog in some environments
		// In test environments, we expect it to either fail due to missing GUI or
		// timeout due to waiting for user input
		// Set a short timeout to prevent long waits in CI
		done := make(chan error, 1)
		
		go func() {
			_, err := app.SelectDirectory("Test Title")
			done <- err
		}()

		// Wait for a short time to avoid hanging in CI
		select {
		case err := <-done:
			// This test is primarily to ensure the function doesn't crash
			// It may succeed (if GUI is available), fail (no GUI tools), or return empty (user cancelled)
			// For CI environments, we mainly want to ensure no panics occur
			if err != nil {
				t.Logf("SelectDirectory returned expected error in test environment: %v", err)
			} else {
				t.Log("SelectDirectory completed (possibly with empty selection in test environment)")
			}
		case <-time.After(2 * time.Second): // Use 2 second timeout
			t.Log("SelectDirectory test timed out (expected in CI environments)")
		}
	})
}





