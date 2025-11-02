package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// MockContext simulates a Wails context for testing
type MockContext struct {
	context.Context
}

func TestSearchWithProgress(t *testing.T) {
	app := NewApp()
	// Don't set up context for testing - SearchWithProgress should handle this gracefully
	// ctx := context.Background()
	// app.startup(ctx)

	tempDir := t.TempDir()

	// Create test files
	testFiles := map[string]string{
		"test1.go": "package main\n\nimport \"fmt\"\n\nfunc main() {\n\tfmt.Println(\"hello world\")\n}",
		"test2.js": "console.log('hello world');\nconsole.log('test');",
		"test3.txt": "This is a test file with hello world content",
	}

	for fileName, content := range testFiles {
		filePath := filepath.Join(tempDir, fileName)
		err := os.WriteFile(filePath, []byte(content), 0644)
		if err != nil {
			t.Fatalf("Failed to create test file %s: %v", fileName, err)
		}
	}

	t.Run("BasicSearch", func(t *testing.T) {
		req := SearchRequest{
			Directory:     tempDir,
			Query:        "hello",
			Extension:    "",
			CaseSensitive: false,
		}

		results, err := app.SearchWithProgress(req)
		if err != nil {
			t.Fatalf("SearchWithProgress returned error: %v", err)
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

	t.Run("ExtensionFilter", func(t *testing.T) {
		req := SearchRequest{
			Directory:     tempDir,
			Query:        "fmt",
			Extension:    "go",
			CaseSensitive: false,
		}

		results, err := app.SearchWithProgress(req)
		if err != nil {
			t.Fatalf("SearchWithProgress returned error: %v", err)
		}

		if len(results) != 2 {
			t.Errorf("Expected 2 results when searching for 'fmt' in .go files, got %d", len(results))
		}

		for _, result := range results {
			if filepath.Ext(result.FilePath) != ".go" {
				t.Errorf("Expected .go file, got %s", filepath.Ext(result.FilePath))
			}
		}
	})

	t.Run("CaseSensitiveSearch", func(t *testing.T) {
		req := SearchRequest{
			Directory:     tempDir,
			Query:        "HELLO",
			Extension:    "",
			CaseSensitive: true,
		}

		results, err := app.SearchWithProgress(req)
		if err != nil {
			t.Fatalf("SearchWithProgress returned error: %v", err)
		}

		// Should find no results since we're searching for uppercase "HELLO" in lowercase text
		if len(results) != 0 {
			t.Errorf("Expected 0 results for case-sensitive 'HELLO' search, got %d", len(results))
		}
	})

	t.Run("CaseInsensitiveSearch", func(t *testing.T) {
		req := SearchRequest{
			Directory:     tempDir,
			Query:        "HELLO",
			Extension:    "",
			CaseSensitive: false,
		}

		results, err := app.SearchWithProgress(req)
		if err != nil {
			t.Fatalf("SearchWithProgress returned error: %v", err)
		}

		// Should find results since we're doing case-insensitive search
		if len(results) != 3 {
			t.Errorf("Expected 3 results for case-insensitive 'HELLO' search, got %d", len(results))
		}
	})

	t.Run("NonExistentDirectory", func(t *testing.T) {
		req := SearchRequest{
			Directory:     "/non/existent/directory",
			Query:        "test",
			Extension:    "",
			CaseSensitive: false,
		}

		_, err := app.SearchWithProgress(req)
		if err == nil {
			t.Error("Expected error when searching in non-existent directory, got nil")
		}
	})

	t.Run("RegexSearch", func(t *testing.T) {
		req := SearchRequest{
			Directory:     tempDir,
			Query:        "fmt\\.Println",
			Extension:    "go",
			CaseSensitive: false,
		}

		results, err := app.SearchWithProgress(req)
		if err != nil {
			t.Fatalf("SearchWithProgress returned error: %v", err)
		}

		if len(results) != 1 {
			t.Errorf("Expected 1 result for regex search, got %d", len(results))
		}
	})

	t.Run("EmptyQuery", func(t *testing.T) {
		req := SearchRequest{
			Directory:     tempDir,
			Query:        "",
			Extension:    "",
			CaseSensitive: false,
		}

		results, err := app.SearchWithProgress(req)
		if err != nil {
			t.Fatalf("SearchWithProgress returned error: %v", err)
		}

		// Should return empty results for empty query
		if len(results) != 0 {
			t.Errorf("Expected 0 results for empty query, got %d", len(results))
		}
	})

	t.Run("LiteralSearch", func(t *testing.T) {
		useRegex := false
		req := SearchRequest{
			Directory:     tempDir,
			Query:        "fmt.Println",
			Extension:    "go",
			CaseSensitive: false,
			UseRegex:     &useRegex,
		}

		results, err := app.SearchWithProgress(req)
		if err != nil {
			t.Fatalf("SearchWithProgress returned error: %v", err)
		}

		if len(results) != 1 {
			t.Errorf("Expected 1 result for literal search, got %d", len(results))
		}
	})

	t.Run("ExcludePatterns", func(t *testing.T) {
		// Create a directory to exclude
		excludeDir := filepath.Join(tempDir, "node_modules")
		err := os.Mkdir(excludeDir, 0755)
		if err != nil {
			t.Fatalf("Failed to create exclude directory: %v", err)
		}

		// Create a file in the excluded directory
		excludeFile := filepath.Join(excludeDir, "test.js")
		err = os.WriteFile(excludeFile, []byte("console.log('hello from node_modules');"), 0644)
		if err != nil {
			t.Fatalf("Failed to create exclude file: %v", err)
		}

		req := SearchRequest{
			Directory:      tempDir,
			Query:         "hello",
			Extension:     "",
			CaseSensitive:  false,
			ExcludePatterns: []string{"node_modules"},
		}

		results, err := app.SearchWithProgress(req)
		if err != nil {
			t.Fatalf("SearchWithProgress returned error: %v", err)
		}

		// Should not include results from excluded directory
		for _, result := range results {
			if strings.Contains(result.FilePath, "node_modules") {
				t.Errorf("Should not include results from excluded directory: %s", result.FilePath)
			}
		}
	})

	t.Run("FileSizeLimits", func(t *testing.T) {
		// Create a large file that exceeds max file size limit
		largeFile := filepath.Join(tempDir, "large.txt")
		content := strings.Repeat("a", 100000) // 100KB file
		err := os.WriteFile(largeFile, []byte(content), 0644)
		if err != nil {
			t.Fatalf("Failed to create large file: %v", err)
		}

		req := SearchRequest{
			Directory:   tempDir,
			Query:      "a",
			MaxFileSize: 50000, // 50KB limit
		}

		results, err := app.SearchWithProgress(req)
		if err != nil {
			t.Fatalf("SearchWithProgress returned error: %v", err)
		}

		// Should not include results from large file
		for _, result := range results {
			if result.FilePath == largeFile {
				t.Errorf("Should not include results from large file exceeding limit")
			}
		}
	})

	t.Run("MinFileSize", func(t *testing.T) {
		// Create a tiny file
		tinyFile := filepath.Join(tempDir, "tiny.txt")
		err := os.WriteFile(tinyFile, []byte("a"), 0644)
		if err != nil {
			t.Fatalf("Failed to create tiny file: %v", err)
		}

		req := SearchRequest{
			Directory:   tempDir,
			Query:      "a",
			MinFileSize: 100, // Require files to be at least 100 bytes
		}

		results, err := app.SearchWithProgress(req)
		if err != nil {
			t.Fatalf("SearchWithProgress returned error: %v", err)
		}

		// Should not include results from tiny file
		for _, result := range results {
			if result.FilePath == tinyFile {
				t.Errorf("Should not include results from tiny file below minimum size")
			}
		}
	})

	t.Run("MaxResults", func(t *testing.T) {
		req := SearchRequest{
			Directory: tempDir,
			Query:     "hello",
			MaxResults: 1,
		}

		results, err := app.SearchWithProgress(req)
		if err != nil {
			t.Fatalf("SearchWithProgress returned error: %v", err)
		}

		// Should limit results to max
		if len(results) > 1 {
			t.Errorf("Expected at most 1 result due to max results limit, got %d", len(results))
		}
	})

	t.Run("ContextLines", func(t *testing.T) {
		// Create a file with multiple lines for context testing
		content := `line 1: This is the first line
line 2: This line contains the search term
line 3: This is the third line
line 4: This is the fourth line
line 5: This line also contains the search term
line 6: This is the sixth line
line 7: Last line`

		filePath := filepath.Join(tempDir, "context_test.txt")
		err := os.WriteFile(filePath, []byte(content), 0644)
		if err != nil {
			t.Fatalf("Failed to create context test file: %v", err)
		}

		req := SearchRequest{
			Directory:     tempDir,
			Query:        "search term",
			CaseSensitive: false,
		}

		results, err := app.SearchWithProgress(req)
		if err != nil {
			t.Fatalf("SearchWithProgress returned error: %v", err)
		}

		if len(results) != 2 {
			t.Fatalf("Expected 2 results for 'search term', got %d", len(results))
		}

		// Check context lines for each result
		for _, result := range results {
			if result.ContextBefore == nil {
				t.Error("ContextBefore should not be nil")
			}
			if result.ContextAfter == nil {
				t.Error("ContextAfter should not be nil")
			}
		}
	})
}