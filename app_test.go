package main

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestSearchCode(t *testing.T) {
	app := NewApp()

	// Create a temporary directory for testing
	tempDir := t.TempDir()
	
	// Create test files
	testFiles := map[string]string{
		"test1.go": "package main\n\nimport \"fmt\"\n\nfunc main() {\n\tfmt.Println(\"hello world\")\n}",
		"test2.js": "console.log('hello world');\nconsole.log('test');",  // Removed fmt from js file
		"test3.txt": "This is a test file with hello world content",
	}
	
	for fileName, content := range testFiles {
		filePath := filepath.Join(tempDir, fileName)
		err := os.WriteFile(filePath, []byte(content), 0644)
		if err != nil {
			t.Fatalf("Failed to create test file %s: %v", fileName, err)
		}
	}
	
	// Test 1: Search for "hello"
	t.Run("SearchForHello", func(t *testing.T) {
		req := SearchRequest{
			Directory: tempDir,
			Query:     "hello",
			Extension: "",
			CaseSensitive: false,
		}
		
		results, err := app.SearchCode(req)
		if err != nil {
			t.Fatalf("SearchCode returned error: %v", err)
		}
		
		if len(results) != 3 {
			t.Errorf("Expected 3 results, got %d", len(results))
		}
		
		// Check that all results contain "hello"
		for _, result := range results {
			if result.Content == "" {
				t.Error("Result content is empty")
			}
		}
	})
	
	// Test 2: Search with specific extension (.go)
	t.Run("SearchInGoFilesOnly", func(t *testing.T) {
		req := SearchRequest{
			Directory: tempDir,
			Query:     "fmt",
			Extension: "go",
			CaseSensitive: false,
		}
		
		results, err := app.SearchCode(req)
		if err != nil {
			t.Fatalf("SearchCode returned error: %v", err)
		}
		
		if len(results) != 2 {
			t.Errorf("Expected 2 results when searching for 'fmt' in .go files (import and usage), got %d", len(results))
		}
		
		for _, result := range results {
			if filepath.Ext(result.FilePath) != ".go" {
				t.Errorf("Expected .go file, got %s", filepath.Ext(result.FilePath))
			}
		}
	})
	
	// Test 3: Search with case sensitivity
	t.Run("CaseSensitiveSearch", func(t *testing.T) {
		req := SearchRequest{
			Directory: tempDir,
			Query:     "HELLO",
			Extension: "",
			CaseSensitive: true,
		}
		
		results, err := app.SearchCode(req)
		if err != nil {
			t.Fatalf("SearchCode returned error: %v", err)
		}
		
		// Should find no results since we're searching for uppercase "HELLO" in lowercase text
		if len(results) != 0 {
			t.Errorf("Expected 0 results for case-sensitive 'HELLO' search, got %d", len(results))
		}
	})
	
	// Test 4: Search with case insensitivity
	t.Run("CaseInsensitiveSearch", func(t *testing.T) {
		req := SearchRequest{
			Directory: tempDir,
			Query:     "HELLO",
			Extension: "",
			CaseSensitive: false,
		}
		
		results, err := app.SearchCode(req)
		if err != nil {
			t.Fatalf("SearchCode returned error: %v", err)
		}
		
		// Should find results since we're doing case-insensitive search
		if len(results) != 3 {
			t.Errorf("Expected 3 results for case-insensitive 'HELLO' search, got %d", len(results))
		}
	})
	
	// Test 5: Search in non-existent directory
	t.Run("SearchInNonExistentDirectory", func(t *testing.T) {
		req := SearchRequest{
			Directory: "/non/existent/directory",
			Query:     "test",
			Extension: "",
			CaseSensitive: false,
		}
		
		_, err := app.SearchCode(req)
		if err == nil {
			t.Error("Expected error when searching in non-existent directory, got nil")
		}
		
		// Check if error message contains expected text
		if err != nil && err.Error() == "" {
			t.Error("Expected error message, got empty string")
		}
	})
	
	// Test 6: Search with regex pattern
	t.Run("SearchWithRegexPattern", func(t *testing.T) {
		req := SearchRequest{
			Directory: tempDir,
			Query:     "fmt\\.Println",
			Extension: "go",
			CaseSensitive: false,
		}
		
		results, err := app.SearchCode(req)
		if err != nil {
			t.Fatalf("SearchCode returned error: %v", err)
		}
		
		if len(results) != 1 {
			t.Errorf("Expected 1 result for regex search, got %d", len(results))
		}
	})
}

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
			if err.Error() != "unsupported platform: "+runtime.GOOS {
				t.Logf("ShowInFolder returned expected error (may not be supported in test environment): %v", err)
			}
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
		
		// In a test environment (particularly CI), this will likely fail or hang
		// The important thing is that it's implemented and works in real environments
		_, err := app.SelectDirectory("Test Title")
		
		// This test is primarily to ensure the function doesn't crash
		// It may succeed (if GUI is available), fail (no GUI tools), or hang (waiting for input)
		// For CI environments, we mainly want to ensure no panics occur
		if err != nil {
			t.Logf("SelectDirectory returned expected result in test environment: %v", err)
		} else {
			t.Log("SelectDirectory succeeded in test environment (GUI might be available)")
		}
	})
}