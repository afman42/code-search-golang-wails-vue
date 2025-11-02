package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSearchWithProgressExtended(t *testing.T) {
	app := NewApp()

	tempDir := t.TempDir()

	// Create test files including edge cases
	testFiles := map[string]string{
		"normal.go":             "package main\n\nimport \"fmt\"\n\nfunc main() {\n\tfmt.Println(\"hello world\")\n}",
		"large_file.go":         strings.Repeat("a", 10000000) + "hello" + strings.Repeat("b", 10000000), // 20MB+ file
		"empty_file.txt":        "",
		"unicode.txt":           "Привет мир! 你好世界! Hello, 世界!",
		"special_chars.go":      "var test = `file with special chars: $%^&*()`",
		"multiline.txt":         "line 1\nline 2 with hello\ntest hello again\nline 4",
		"binary_file.bin":       "\x00\x01\x02\x03\x04", // Binary content
		"no_match.txt":          "this file has no matches for our search query",
		"many_matches.txt":      strings.Repeat("hello\n", 50), // 50 matches to leave room for other files
		"regex_special.txt":     "This is a test for regex: [abc] and (group) and \\backslash\\",
		"symlink_test.go":       "package test", // Additional test file
		"file with spaces.go":   "package spaces\n// This file has spaces in name",
		"file-with-dashes.go":   "package dashes\n// This file has dashes in name",
		"file_with_underscores.go": "package underscores\n// This file has underscores in name",
	}

	for fileName, content := range testFiles {
		filePath := filepath.Join(tempDir, fileName)
		err := os.WriteFile(filePath, []byte(content), 0644)
		if err != nil {
			t.Fatalf("Failed to create test file %s: %v", fileName, err)
		}
	}

	// Create a nested directory structure for testing
	nestedDir := filepath.Join(tempDir, "nested", "subdir", "deep")
	err := os.MkdirAll(nestedDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create nested directory: %v", err)
	}
	
	// Create a file in the nested directory
	nestedFile := filepath.Join(nestedDir, "nested_file.go")
	err = os.WriteFile(nestedFile, []byte("package nested\n// This is in a nested directory with hello\nfunc helloFunc() { }"), 0644)
	if err != nil {
		t.Fatalf("Failed to create nested file: %v", err)
	}

	// Test 1: Search with no matches
	t.Run("SearchWithNoMatches", func(t *testing.T) {
		req := SearchRequest{
			Directory:     tempDir,
			Query:         "nonexistentpattern",
			Extension:     "",
			CaseSensitive: false,
		}

		results, err := app.SearchWithProgress(req)
		if err != nil {
			t.Fatalf("SearchWithProgress returned error: %v", err)
		}

		if len(results) != 0 {
			t.Errorf("Expected 0 results, got %d", len(results))
		}
	})

	// Test 2: Search with very large files (should be skipped)
	t.Run("SearchIgnoresLargeFiles", func(t *testing.T) {
		req := SearchRequest{
			Directory:     tempDir,
			Query:         "hello", // This exists in the large file
			Extension:     "",
			CaseSensitive: false,
		}

		results, err := app.SearchWithProgress(req)
		if err != nil {
			t.Fatalf("SearchWithProgress returned error: %v", err)
		}

		// The large file should be skipped, so we shouldn't get matches from it
		for _, result := range results {
			if filepath.Base(result.FilePath) == "large_file.go" {
				t.Errorf("Should not include results from large files")
			}
		}
	})

	// Test 3: Search in empty files
	t.Run("SearchInEmptyFiles", func(t *testing.T) {
		req := SearchRequest{
			Directory:     tempDir,
			Query:         "hello",
			Extension:     "",
			CaseSensitive: false,
		}

		results, err := app.SearchWithProgress(req)
		if err != nil {
			t.Fatalf("SearchWithProgress returned error: %v", err)
		}

		for _, result := range results {
			if result.FilePath == filepath.Join(tempDir, "empty_file.txt") {
				t.Errorf("Should not find matches in empty files")
			}
		}
	})

	// Test 4: Unicode character search
	t.Run("SearchWithUnicode", func(t *testing.T) {
		req := SearchRequest{
			Directory:     tempDir,
			Query:         "мир",
			Extension:     "",
			CaseSensitive: false,
		}

		results, err := app.SearchWithProgress(req)
		if err != nil {
			t.Fatalf("SearchWithProgress returned error: %v", err)
		}

		if len(results) == 0 {
			t.Errorf("Expected results for unicode search, got %d", len(results))
		}

		found := false
		for _, result := range results {
			if strings.Contains(result.Content, "мир") {
				found = true
				break
			}
		}
		if !found {
			t.Error("Expected to find unicode content in results")
		}
	})

	// Test 5: Test result truncation (limit of 1000)
	t.Run("SearchResultTruncation", func(t *testing.T) {
		// Create a separate directory with many matches to test truncation
		truncationDir := filepath.Join(t.TempDir(), "truncation_test")
		err := os.MkdirAll(truncationDir, 0755)
		if err != nil {
			t.Fatalf("Failed to create truncation test directory: %v", err)
		}
		
		// Create a file with many matches (>1000)
		truncationFile := filepath.Join(truncationDir, "truncation_test.txt")
		truncationContent := strings.Repeat("hello match\n", 1050)  // More than the limit
		err = os.WriteFile(truncationFile, []byte(truncationContent), 0644)
		if err != nil {
			t.Fatalf("Failed to create truncation test file: %v", err)
		}

		req := SearchRequest{
			Directory:     truncationDir,
			Query:         "hello",
			Extension:     "",
			CaseSensitive: false,
		}

		results, err := app.SearchWithProgress(req)
		if err != nil {
			t.Fatalf("SearchWithProgress returned error: %v", err)
		}

		if len(results) > 1000 {
			t.Errorf("Expected at most 1000 results due to limit, got %d", len(results))
		} else if len(results) == 1000 {
			t.Log("Result count is at the limit (1000), indicating proper truncation")
		} else {
			t.Logf("Got %d results (under limit of 1000)", len(results))
		}
	})

	// Test 6: Search with regex special characters as literal string
	t.Run("SearchRegexSpecialCharsLiteral", func(t *testing.T) {
		req := SearchRequest{
			Directory:     tempDir,
			Query:         "[abc]",
			Extension:     "",
			CaseSensitive: false,
		}

		results, err := app.SearchWithProgress(req)
		if err != nil {
			t.Fatalf("SearchWithProgress returned error: %v", err)
		}

		found := false
		for _, result := range results {
			if strings.Contains(result.Content, "[abc]") {
				found = true
				break
			}
		}
		if !found {
			t.Error("Expected to find literal [abc] pattern")
		}
	})

	// Test 7: Case sensitivity with unicode
	t.Run("CaseSensitivityWithUnicode", func(t *testing.T) {
		// Create a special test file with unicode case differences
		caseFile := filepath.Join(tempDir, "case_test.txt")
		err := os.WriteFile(caseFile, []byte("Привет\nПРИВЕТ\nпривет"), 0644)
		if err != nil {
			t.Fatalf("Failed to create case test file: %v", err)
		}

		req := SearchRequest{
			Directory:     tempDir,
			Query:         "ПРИВЕТ", // Uppercase
			Extension:     "",
			CaseSensitive: true,
		}

		results, err := app.SearchWithProgress(req)
		if err != nil {
			t.Fatalf("SearchWithProgress returned error: %v", err)
		}

		for _, result := range results {
			// All results should be exactly "ПРИВЕТ" (uppercase)
			if result.Content != "ПРИВЕТ" {
				t.Errorf("Expected only uppercase matches, found: %s", result.Content)
			}
		}

		// Now test case insensitive
		req.CaseSensitive = false
		results, err = app.SearchWithProgress(req)
		if err != nil {
			t.Fatalf("SearchWithProgress returned error: %v", err)
		}

		foundLower := false
		foundUpper := false
		for _, result := range results {
			if result.Content == "привет" {
				foundLower = true
			}
			if result.Content == "ПРИВЕТ" {
				foundUpper = true
			}
		}
		if !foundLower || !foundUpper {
			t.Error("Expected to find both uppercase and lowercase matches for case-insensitive search")
		}
	})

	// Test 8: Search in deeply nested directories
	t.Run("SearchInNestedDirectories", func(t *testing.T) {
		req := SearchRequest{
			Directory:     tempDir,
			Query:         "hello",
			Extension:     "",
			CaseSensitive: false,
		}

		results, err := app.SearchWithProgress(req)
		if err != nil {
			t.Fatalf("SearchWithProgress returned error: %v", err)
		}

		foundInNested := false
		for _, result := range results {
			// Check if any result comes from the nested directory structure
			if strings.Contains(result.FilePath, string(filepath.Separator)+"nested"+string(filepath.Separator)) ||
			   strings.Contains(result.FilePath, "nested_file.go") {
				foundInNested = true
				break
			}
		}
		if !foundInNested {
			t.Error("Expected to find matches in nested directories")
			// Print all results for debugging
			t.Logf("All results: %v", len(results))
			for _, result := range results {
				t.Logf("Result path: %s", result.FilePath)
			}
		}
	})

	// Test 9: Search with spaces in directory/file names
	t.Run("SearchInPathsWithSpaces", func(t *testing.T) {
		req := SearchRequest{
			Directory:     tempDir,
			Query:         "spaces",
			Extension:     "",
			CaseSensitive: false,
		}

		results, err := app.SearchWithProgress(req)
		if err != nil {
			t.Fatalf("SearchWithProgress returned error: %v", err)
		}

		found := false
		for _, result := range results {
			if strings.Contains(result.FilePath, "with spaces") {
				found = true
				break
			}
		}
		if !found {
			t.Error("Expected to find matches in files with spaces in names")
		}
	})

	// Test 10: Invalid regex pattern handling
	t.Run("InvalidRegexPatternHandling", func(t *testing.T) {
		// Test how the function handles invalid regex when case sensitivity is off
		// This will add (?i) to the pattern, which could interact with invalid patterns
		req := SearchRequest{
			Directory:     tempDir,
			Query:         "[invalid", // Invalid regex pattern
			Extension:     "",
			CaseSensitive: false,
		}

		_, err := app.SearchWithProgress(req)
		if err == nil {
			t.Error("Expected error for invalid regex pattern")
		}
	})

	// Test 11: Valid regex pattern handling
	t.Run("ValidRegexPatternHandling", func(t *testing.T) {
		req := SearchRequest{
			Directory:     tempDir,
			Query:         "hello", // Valid simple pattern
			Extension:     "",
			CaseSensitive: false,
		}

		results, err := app.SearchWithProgress(req)
		if err != nil {
			t.Fatalf("SearchWithProgress returned error for valid pattern: %v", err)
		}

		// There should be some results with "hello" in the test files
		if len(results) == 0 {
			t.Error("Expected results for valid pattern, got none")
		}
	})

	// Test 11: Multiline pattern matching
	t.Run("MultilineSearch", func(t *testing.T) {
		req := SearchRequest{
			Directory:     tempDir,
			Query:         "line 2 with hello",
			Extension:     "",
			CaseSensitive: false,
		}

		results, err := app.SearchWithProgress(req)
		if err != nil {
			t.Fatalf("SearchWithProgress returned error: %v", err)
		}

		found := false
		for _, result := range results {
			if strings.Contains(result.Content, "line 2 with hello") {
				found = true
				break
			}
		}
		if !found {
			t.Error("Expected to find multiline pattern")
		}
	})
}

func TestSearchWithProgressWithPermissions(t *testing.T) {
	app := NewApp()

	tempDir := t.TempDir()
	
	// Create a readable file
	readableFile := filepath.Join(tempDir, "readable.txt")
	err := os.WriteFile(readableFile, []byte("hello world"), 0644)
	if err != nil {
		t.Fatalf("Failed to create readable file: %v", err)
	}

	// Create a non-readable file (on systems that support it)
	nonReadableFile := filepath.Join(tempDir, "not_readable.txt")
	err = os.WriteFile(nonReadableFile, []byte("should not be readable"), 0000)
	if err != nil {
		t.Logf("Could not create non-readable file (may not be supported on this system): %v", err)
	}

	t.Run("SearchSkipsNonReadableFiles", func(t *testing.T) {
		req := SearchRequest{
			Directory:     tempDir,
			Query:         "should not be readable",
			Extension:     "",
			CaseSensitive: false,
		}

		results, err := app.SearchWithProgress(req)
		if err != nil {
			t.Fatalf("SearchWithProgress returned error: %v", err)
		}

		for _, result := range results {
			if strings.Contains(result.Content, "should not be readable") {
				t.Errorf("Should not include results from non-readable files")
			}
		}
	})
}

func TestSearchWithProgressSpecialFiles(t *testing.T) {
	app := NewApp()

	tempDir := t.TempDir()
	
	// Create test files
	testFile := filepath.Join(tempDir, "test.txt")
	err := os.WriteFile(testFile, []byte("hello world"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	t.Run("SearchInHiddenDirectories", func(t *testing.T) {
		// Create a hidden directory
		hiddenDir := filepath.Join(tempDir, ".hidden")
		err := os.Mkdir(hiddenDir, 0755)
		if err != nil {
			t.Fatalf("Failed to create hidden directory: %v", err)
		}
		
		// Create a file in hidden directory
		hiddenFile := filepath.Join(hiddenDir, "hidden.txt")
		err = os.WriteFile(hiddenFile, []byte("hidden hello world"), 0644)
		if err != nil {
			t.Fatalf("Failed to create hidden file: %v", err)
		}

		req := SearchRequest{
			Directory:     tempDir,
			Query:         "hello",
			Extension:     "",
			CaseSensitive: false,
		}

		results, err := app.SearchWithProgress(req)
		if err != nil {
			t.Fatalf("SearchWithProgress returned error: %v", err)
		}

		for _, result := range results {
			if strings.Contains(result.FilePath, ".hidden") {
				t.Errorf("Should not include results from hidden directories: %s", result.FilePath)
			}
		}
	})
}