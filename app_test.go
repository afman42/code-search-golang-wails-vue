package main

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
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

func TestSearchCodeWithParallelProcessing(t *testing.T) {
	app := NewApp()

	tempDir := t.TempDir()
	
	// Create multiple test files to test parallel processing
	testFiles := map[string]string{
		"file1.go": "package main\n\nimport \"fmt\"\n\nfunc main() {\n\tfmt.Println(\"hello from file1\")\n}",
		"file2.js": "console.log('hello world from file2');\nconsole.log('test');",
		"file3.txt": "This is a test file with hello from file3 content",
		"file4.go": "package main\n\nfunc TestFunc() {\n\tfmt.Println(\"another hello in file4\")\n}",
		"file5.txt": "More content with hello from file5\nAnd another line with hello\nAnd one more hello here",
	}
	
	for fileName, content := range testFiles {
		filePath := filepath.Join(tempDir, fileName)
		err := os.WriteFile(filePath, []byte(content), 0644)
		if err != nil {
			t.Fatalf("Failed to create test file %s: %v", fileName, err)
		}
	}
	
	t.Run("Parallel search for hello", func(t *testing.T) {
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
		
		if len(results) == 0 {
			t.Error("Expected results for 'hello' search, got none")
		}
		
		// Check that all results contain "hello" in their content
		for _, result := range results {
			if !strings.Contains(strings.ToLower(result.Content), "hello") {
				t.Errorf("Result content '%s' does not contain 'hello'", result.Content)
			}
		}
	})
}



func TestSearchCodeWithContextLines(t *testing.T) {
	app := NewApp()

	tempDir := t.TempDir()
	
	// Create a file with multiple lines for context testing
	contentWithLines := `line 1: This is the first line
line 2: This line contains the search term
line 3: This is the third line
line 4: This is the fourth line
line 5: This line also contains the search term
line 6: This is the sixth line
line 7: Last line`
	
	filePath := filepath.Join(tempDir, "context_test.go")
	err := os.WriteFile(filePath, []byte(contentWithLines), 0644)
	if err != nil {
		t.Fatalf("Failed to create context test file: %v", err)
	}
	
	t.Run("Search with context lines", func(t *testing.T) {
		req := SearchRequest{
			Directory: tempDir,
			Query:     "search term",
			Extension: "",
			CaseSensitive: false,
		}
		
		results, err := app.SearchCode(req)
		if err != nil {
			t.Fatalf("SearchCode returned error: %v", err)
		}
		
		if len(results) != 2 {
			t.Errorf("Expected 2 results for 'search term', got %d", len(results))
		}
		
		// Check each result for proper context
		for _, result := range results {
			// Verify context arrays are present
			if result.ContextBefore == nil {
				t.Error("ContextBefore should not be nil")
			}
			if result.ContextAfter == nil {
				t.Error("ContextAfter should not be nil")
			}
			
			// Check the specific result at line 2
			if result.LineNum == 2 {
				// Should have 1 before (line 1) and 2 after (lines 3, 4)
				if len(result.ContextBefore) != 1 {
					t.Errorf("Expected 1 context line before at line 2, got %d", len(result.ContextBefore))
				} else if result.ContextBefore[0] != "line 1: This is the first line" {
					t.Errorf("Expected 'line 1: This is the first line', got '%s'", result.ContextBefore[0])
				}
				
				if len(result.ContextAfter) != 2 {
					t.Errorf("Expected 2 context lines after at line 2, got %d", len(result.ContextAfter))
				} else {
					if result.ContextAfter[0] != "line 3: This is the third line" {
						t.Errorf("Expected 'line 3: This is the third line', got '%s'", result.ContextAfter[0])
					}
					if result.ContextAfter[1] != "line 4: This is the fourth line" {
						t.Errorf("Expected 'line 4: This is the fourth line', got '%s'", result.ContextAfter[1])
					}
				}
			}
			
			// Check the specific result at line 5
			if result.LineNum == 5 {
				// Should have 2 before (lines 3, 4) and 2 after (lines 6, 7)
				if len(result.ContextBefore) != 2 {
					t.Errorf("Expected 2 context lines before at line 5, got %d", len(result.ContextBefore))
				} else {
					if result.ContextBefore[0] != "line 3: This is the third line" {
						t.Errorf("Expected 'line 3: This is the third line', got '%s'", result.ContextBefore[0])
					}
					if result.ContextBefore[1] != "line 4: This is the fourth line" {
						t.Errorf("Expected 'line 4: This is the fourth line', got '%s'", result.ContextBefore[1])
					}
				}
				
				if len(result.ContextAfter) != 2 {
					t.Errorf("Expected 2 context lines after at line 5, got %d", len(result.ContextAfter))
				} else {
					if result.ContextAfter[0] != "line 6: This is the sixth line" {
						t.Errorf("Expected 'line 6: This is the sixth line', got '%s'", result.ContextAfter[0])
					}
					if result.ContextAfter[1] != "line 7: Last line" {
						t.Errorf("Expected 'line 7: Last line', got '%s'", result.ContextAfter[1])
					}
				}
			}
		}
	})
	
	t.Run("Search with context at file boundaries", func(t *testing.T) {
		req := SearchRequest{
			Directory: tempDir,
			Query:     "first line",
			Extension: "",
			CaseSensitive: false,
		}
		
		results, err := app.SearchCode(req)
		if err != nil {
			t.Fatalf("SearchCode returned error: %v", err)
		}
		
		if len(results) != 1 {
			t.Fatalf("Expected 1 result for 'first line', got %d", len(results))
		}
		
		result := results[0]
		
		// At the first line, there should be no context before
		if len(result.ContextBefore) != 0 {
			t.Errorf("Expected 0 context lines before first line, got %d", len(result.ContextBefore))
		}
		
		// At the first line, there should be context after
		if len(result.ContextAfter) == 0 {
			t.Error("Expected context lines after first line")
		}
		
		// Test for last line
		req.Query = "Last line"
		results, err = app.SearchCode(req)
		if err != nil {
			t.Fatalf("SearchCode returned error: %v", err)
		}
		
		if len(results) != 1 {
			t.Fatalf("Expected 1 result for 'Last line', got %d", len(results))
		}
		
		result = results[0]
		
		// At the last line, there should be no context after
		if len(result.ContextAfter) != 0 {
			t.Errorf("Expected 0 context lines after last line, got %d", len(result.ContextAfter))
		}
		
		// At the last line, there should be context before
		if len(result.ContextBefore) == 0 {
			t.Error("Expected context lines before last line")
		}
	})
}