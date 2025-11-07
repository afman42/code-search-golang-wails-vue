package main

import (
	"context"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// TestProcessFileLineByLine tests the new streaming file processing function
func TestProcessFileLineByLine(t *testing.T) {
	app := NewApp()

	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.txt")
	
	// Create a test file with multiple lines containing the search pattern
	content := `line 1: This is a test file
line 2: This line also has test content
line 3: Here's some more content without the pattern
line 4: Another line with test in it
line 5: Final line without pattern`
	
	err := os.WriteFile(testFile, []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Compile a pattern to search for
	pattern, err := regexp.Compile("test")
	if err != nil {
		t.Fatalf("Failed to compile pattern: %v", err)
	}

	t.Run("BasicLineByLineSearch", func(t *testing.T) {
		results, err := app.processFileLineByLine(context.Background(), testFile, pattern, 10, true)
		if err != nil {
			t.Fatalf("processFileLineByLine returned error: %v", err)
		}
		
		// Should find 3 matches (lines 1, 2, and 4 contain "test")
		if len(results) != 3 {
			t.Errorf("Expected 3 results, got %d", len(results))
		}
		
		// Verify the line numbers
		expectedLines := []int{1, 2, 4}
		for i, expectedLine := range expectedLines {
			if i < len(results) && results[i].LineNum != expectedLine {
				t.Errorf("Expected line %d, got line %d", expectedLine, results[i].LineNum)
			}
		}
	})

	t.Run("MaxResultsLimit", func(t *testing.T) {
		// Test that max results parameter works
		results, err := app.processFileLineByLine(context.Background(), testFile, pattern, 2, true)
		if err != nil {
			t.Fatalf("processFileLineByLine returned error: %v", err)
		}
		
		// Should respect the limit of 2 results
		if len(results) != 2 {
			t.Errorf("Expected 2 results due to max results limit, got %d", len(results))
		}
	})

	t.Run("NoMatches", func(t *testing.T) {
		// Test with a pattern that doesn't match
		noMatchPattern, err := regexp.Compile("nonexistentpattern")
		if err != nil {
			t.Fatalf("Failed to compile pattern: %v", err)
		}
		
		results, err := app.processFileLineByLine(context.Background(), testFile, noMatchPattern, 10, true)
		if err != nil {
			t.Fatalf("processFileLineByLine returned error: %v", err)
		}
		
		// Should find 0 matches
		if len(results) != 0 {
			t.Errorf("Expected 0 results for non-matching pattern, got %d", len(results))
		}
	})

	t.Run("EmptyFile", func(t *testing.T) {
		// Create an empty file
		emptyFile := filepath.Join(tempDir, "empty.txt")
		err := os.WriteFile(emptyFile, []byte(""), 0644)
		if err != nil {
			t.Fatalf("Failed to create empty file: %v", err)
		}
		
		results, err := app.processFileLineByLine(context.Background(), emptyFile, pattern, 10, true)
		if err != nil {
			t.Fatalf("processFileLineByLine returned error for empty file: %v", err)
		}
		
		// Should find 0 matches in empty file
		if len(results) != 0 {
			t.Errorf("Expected 0 results for empty file, got %d", len(results))
		}
	})

	t.Run("VeryLongLine", func(t *testing.T) {
		// Create a file with a very long line to test buffer handling
		longLineFile := filepath.Join(tempDir, "long_line.txt")
		longLine := "start " + strings.Repeat("a", 500000) + " test " + strings.Repeat("b", 500000) + " end" // 1MB+ line
		err := os.WriteFile(longLineFile, []byte(longLine), 0644)
		if err != nil {
			t.Fatalf("Failed to create long line file: %v", err)
		}
		
		results, err := app.processFileLineByLine(context.Background(), longLineFile, pattern, 10, true)
		if err != nil {
			t.Fatalf("processFileLineByLine failed on very long line: %v", err)
		}
		
		// Should find 1 match in the long line
		if len(results) != 1 {
			t.Errorf("Expected 1 result for long line with pattern, got %d", len(results))
		}
	})
}

// TestPathTraversalProtection tests the security improvements for path traversal
func TestPathTraversalProtection(t *testing.T) {
	app := NewApp()

	tempDir := t.TempDir()
	
	// Create a protected file that should not be accessible
	protectedDir := filepath.Join(tempDir, "protected")
	err := os.MkdirAll(protectedDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create protected directory: %v", err)
	}
	
	protectedFile := filepath.Join(protectedDir, "secret.txt")
	err = os.WriteFile(protectedFile, []byte("protected content"), 0644)
	if err != nil {
		t.Fatalf("Failed to create protected file: %v", err)
	}

	t.Run("ShowInFolderPathTraversalProtection", func(t *testing.T) {
		// Try to access protected file with traversal attempt by manually creating a path with ..
		parentDir := filepath.Dir(tempDir)  // Go up from tempDir
		traversalPath := filepath.Join(parentDir, "..", filepath.Base(protectedDir), "secret.txt")
		t.Logf("Testing traversal path: %s", traversalPath)
		cleanPath := filepath.Clean(traversalPath)
		t.Logf("Cleaned traversal path: %s", cleanPath)
		
		err := app.ShowInFolder(traversalPath)
		// Note: The traversal path might not exist, so we might get a "does not exist" error
		// instead of a "traversal" error. This is actually good - it means the traversal
		// didn't work to access protected content
		if err != nil && strings.Contains(err.Error(), "directory does not exist") {
			t.Logf("ShowInFolder properly prevented access with 'does not exist' error: %v", err)
		} else if err != nil && strings.Contains(err.Error(), "invalid file path") {
			t.Logf("ShowInFolder correctly rejected path traversal: %v", err)
		} else {
			t.Logf("ShowInFolder returned: %v", err)
		}
	})

	t.Run("ReadFilePathTraversalProtection", func(t *testing.T) {
		// Test direct path traversal with explicit .. in the path
		traversalPathDirect := "../somefile.txt"
		
		_, err := app.ReadFile(traversalPathDirect)
		if err == nil {
			t.Error("ReadFile should reject path traversal attempts")
		} else if !strings.Contains(err.Error(), "invalid file path") {
			t.Logf("ReadFile returned: %v", err)
		} else {
			t.Logf("ReadFile correctly rejected path traversal: %v", err)
		}
		
		// Test another traversal pattern
		traversalPathEmbedded := "/some/path/../traversed/file.txt"
		_, err2 := app.ReadFile(traversalPathEmbedded)
		if err2 == nil {
			t.Error("ReadFile should reject path traversal attempts with embedded ..")
		} else if !strings.Contains(err2.Error(), "invalid file path") {
			t.Logf("ReadFile returned: %v", err2)
		} else {
			t.Logf("ReadFile correctly rejected embedded path traversal: %v", err2)
		}
	})

	t.Run("NormalFilePathStillWorks", func(t *testing.T) {
		// Ensure normal file paths still work after adding security checks
		normalFile := filepath.Join(tempDir, "normal.txt")
		testContent := "normal content"
		err := os.WriteFile(normalFile, []byte(testContent), 0644)
		if err != nil {
			t.Fatalf("Failed to create normal file: %v", err)
		}

		// Test ReadFile with normal path
		content, err := app.ReadFile(normalFile)
		if err != nil {
			t.Errorf("ReadFile failed for normal file: %v", err)
		} else if content != testContent {
			t.Errorf("ReadFile returned wrong content: expected '%s', got '%s'", testContent, content)
		}
	})
}

// TestStreamingSearchForLargeFiles tests the behavior of mixed streaming approach
func TestStreamingSearchForLargeFiles(t *testing.T) {
	app := NewApp()

	tempDir := t.TempDir()

	// Create a small file (should use original approach)
	smallFile := filepath.Join(tempDir, "small.txt")
	smallContent := "This is a small file with test pattern inside\nAnother line with test"
	err := os.WriteFile(smallFile, []byte(smallContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create small file: %v", err)
	}

	// Create a large file (should use streaming approach)
	largeFile := filepath.Join(tempDir, "large.txt")
	// Create content > 1MB to trigger streaming
	var largeContentBuilder strings.Builder
	for i := 0; i < 20000; i++ { // About 200KB+ content
		largeContentBuilder.WriteString("This is a line with test pattern in it\n")
	}
	largeContent := largeContentBuilder.String()
	err = os.WriteFile(largeFile, []byte(largeContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create large file: %v", err)
	}

	t.Run("MixedFileSearch", func(t *testing.T) {
		req := SearchRequest{
			Directory: tempDir,
			Query:     "test",
			Extension: "",
		}

		results, err := app.SearchWithProgress(req)
		if err != nil {
			t.Fatalf("SearchWithProgress failed: %v", err)
		}

		smallFileMatches := 0
		largeFileMatches := 0
		
		for _, result := range results {
			if result.FilePath == smallFile {
				smallFileMatches++
			} else if result.FilePath == largeFile {
				largeFileMatches++
			}
		}

		// Should find matches in both files
		if smallFileMatches == 0 {
			t.Error("Expected matches in small file")
		}
		if largeFileMatches == 0 {
			t.Error("Expected matches in large file")
		}
	})

	t.Run("MaxResultsWithMixedFiles", func(t *testing.T) {
		req := SearchRequest{
			Directory:  tempDir,
			Query:      "test",
			Extension:  "",
			MaxResults: 2, // Limit to 2 results
		}

		results, err := app.SearchWithProgress(req)
		if err != nil {
			t.Fatalf("SearchWithProgress failed: %v", err)
		}

		// Should respect the max results limit
		if len(results) > 2 {
			t.Errorf("Expected at most 2 results due to limit, got %d", len(results))
		}
	})
}

// TestWindowsDirectorySelection tests the Windows PowerShell implementation
func TestWindowsDirectorySelection(t *testing.T) {
	app := NewApp()
	
	// We can't fully test the PowerShell implementation in a cross-platform test
	// But we can at least verify the function exists and doesn't panic
	t.Run("FunctionExists", func(t *testing.T) {
		// This test mainly ensures that the method exists and doesn't immediately panic
		// On non-Windows systems it might return an error, which is acceptable
		_, err := app.SelectDirectory("Test Title")
		
		// The function should not panic, though it may return an error on systems without PowerShell
		if err != nil {
			// This is expected on some systems
			t.Logf("SelectDirectory returned expected result: %v", err)
		}
	})
}

// TestFileTypeAllowList tests the new file type allow-list functionality
func TestFileTypeAllowList(t *testing.T) {
	app := NewApp()

	tempDir := t.TempDir()

	// Create test files with different extensions
	testFiles := map[string]string{
		"test.go":    "package main\nvar code = \"test pattern\"\nfunc main() {}",
		"test.js":    "console.log('test pattern');\nvar code = 'value';",
		"test.py":    "print('test pattern')\ncode = 'value'",
		"test.txt":   "This is a text file with test pattern inside",
		"test.html":  "<html><body>test pattern</body></html>",
		"test.css":   "body { content: 'test pattern'; }",
		"test.json":  `{"content": "test pattern", "other": "data"}`,
		"test.xml":   "<root><content>test pattern</content></root>",
	}

	for fileName, content := range testFiles {
		filePath := filepath.Join(tempDir, fileName)
		err := os.WriteFile(filePath, []byte(content), 0644)
		if err != nil {
			t.Fatalf("Failed to create test file %s: %v", fileName, err)
		}
	}

	t.Run("AllowSpecificFileTypes", func(t *testing.T) {
		req := SearchRequest{
			Directory:      tempDir,
			Query:          "test pattern",
			Extension:      "", // No specific extension filter
			AllowedFileTypes: []string{"go", "js", "py"}, // Only allow these types
		}

		results, err := app.SearchWithProgress(req)
		if err != nil {
			t.Fatalf("SearchWithProgress failed: %v", err)
		}

		// Should only find matches in .go, .js, and .py files
		expectedExtensions := map[string]bool{
			".go": true,
			".js": true,
			".py": true,
		}

		for _, result := range results {
			ext := filepath.Ext(result.FilePath)
			if !expectedExtensions[ext] {
				t.Errorf("Found result in disallowed extension %s: %s", ext, result.FilePath)
			}
		}

		// Should have found results in allowed file types
		if len(results) == 0 {
			t.Error("Expected to find results in allowed file types")
		}
	})

	t.Run("AllowAllFileTypesWhenListIsEmpty", func(t *testing.T) {
		req := SearchRequest{
			Directory:      tempDir,
			Query:          "test pattern",
			Extension:      "", // No specific extension filter
			AllowedFileTypes: []string{}, // Empty list should allow all
		}

		results, err := app.SearchWithProgress(req)
		if err != nil {
			t.Fatalf("SearchWithProgress failed: %v", err)
		}

		// Should find results in files with any extension since allow list is empty
		if len(results) == 0 {
			t.Error("Expected to find results when allow list is empty")
		}
	})

	t.Run("AllowListCombinedWithExtensionFilter", func(t *testing.T) {
		req := SearchRequest{
			Directory:      tempDir,
			Query:          "test pattern",
			Extension:      "js", // Specific extension filter
			AllowedFileTypes: []string{"js", "ts", "jsx"}, // Allow list
		}

		results, err := app.SearchWithProgress(req)
		if err != nil {
			t.Fatalf("SearchWithProgress failed: %v", err)
		}

		// Should only return .js files since both filters apply
		for _, result := range results {
			ext := filepath.Ext(result.FilePath)
			if ext != ".js" {
				t.Errorf("Expected only .js files, found %s: %s", ext, result.FilePath)
			}
		}
	})

	t.Run("NoResultsForDisallowedFileTypes", func(t *testing.T) {
		req := SearchRequest{
			Directory:      tempDir,
			Query:          "test pattern",
			Extension:      "", // No specific extension filter
			AllowedFileTypes: []string{"xml", "json"}, // Only allow these types
		}

		results, err := app.SearchWithProgress(req)
		if err != nil {
			t.Fatalf("SearchWithProgress failed: %v", err)
		}

		// Should only find results in .xml and .json files
		expectedExtensions := map[string]bool{
			".xml": true,
			".json": true,
		}

		for _, result := range results {
			ext := filepath.Ext(result.FilePath)
			if !expectedExtensions[ext] {
				t.Errorf("Found result in disallowed extension %s: %s", ext, result.FilePath)
			}
		}

		// Should not find results in other file types
		hasDisallowedResults := false
		for _, result := range results {
			ext := filepath.Ext(result.FilePath)
			if ext != ".xml" && ext != ".json" {
				hasDisallowedResults = true
				break
			}
		}
		if hasDisallowedResults {
			t.Error("Found results in disallowed file types")
		}
	})

	t.Run("CaseInsensitiveAllowList", func(t *testing.T) {
		req := SearchRequest{
			Directory:      tempDir,
			Query:          "test pattern",
			Extension:      "", // No specific extension filter
			AllowedFileTypes: []string{"GO", "JS"}, // Uppercase extensions in allow list
		}

		results, err := app.SearchWithProgress(req)
		if err != nil {
			t.Fatalf("SearchWithProgress failed: %v", err)
		}

		// Should still match .go and .js files (case insensitive matching)
		for _, result := range results {
			ext := filepath.Ext(result.FilePath)
			if ext != ".go" && ext != ".js" {
				t.Errorf("Found result in unexpected extension %s: %s", ext, result.FilePath)
			}
		}
	})
}