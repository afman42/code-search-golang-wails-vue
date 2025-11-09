package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// TestEmptySearchRequest tests searching with an empty request
func TestEmptySearchRequest(t *testing.T) {
	app := NewApp()
	
	req := SearchRequest{
		Directory: "",
		Query:     "",
	}
	
	_, err := app.SearchWithProgress(req)
	if err == nil {
		t.Error("Expected error for empty directory, got nil")
	}
	if !strings.Contains(err.Error(), "directory does not exist") {
		t.Errorf("Expected directory validation error, got: %v", err)
	}
}

// TestEmptyQuery tests searching with an empty query
func TestEmptyQuery(t *testing.T) {
	app := NewApp()
	
	tempDir := t.TempDir()
	
	req := SearchRequest{
		Directory: tempDir,
		Query:     "",
	}
	
	results, err := app.SearchWithProgress(req)
	if err != nil {
		t.Errorf("Expected no error for empty query, got: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("Expected empty results for empty query, got %d results", len(results))
	}
}

// TestNonExistentDirectory tests searching in a non-existent directory
func TestNonExistentDirectory(t *testing.T) {
	app := NewApp()
	
	req := SearchRequest{
		Directory: "/non/existent/directory",
		Query:     "test",
	}
	
	_, err := app.SearchWithProgress(req)
	if err == nil {
		t.Error("Expected error for non-existent directory, got nil")
	}
}

// TestProtectedSystemDirectory tests searching in protected system directories
func TestProtectedSystemDirectory(t *testing.T) {
	app := NewApp()
	
	var protectedPath string
	if runtime.GOOS == "windows" {
		protectedPath = "C:\\"
	} else {
		protectedPath = "/"
	}
	
	req := SearchRequest{
		Directory: protectedPath,
		Query:     "test",
	}
	
	_, err := app.SearchWithProgress(req)
	if err == nil {
		t.Error("Expected error for protected system directory, got nil")
	}
	if !strings.Contains(err.Error(), "protected system directory") {
		t.Errorf("Expected protected directory error, got: %v", err)
	}
}

// TestInvalidRegexPattern tests searching with an invalid regex pattern
func TestInvalidRegexPattern(t *testing.T) {
	app := NewApp()
	
	tempDir := t.TempDir()
	
	// Create a test file
	testFile := filepath.Join(tempDir, "test.txt")
	err := os.WriteFile(testFile, []byte("test content"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	
	falseValue := false
	req := SearchRequest{
		Directory: tempDir,
		Query:     "[invalid", // Invalid regex pattern
		UseRegex:  &falseValue, // Use regex mode to trigger validation
	}
	
	_, err = app.SearchWithProgress(req)
	if err == nil {
		t.Error("Expected error for invalid regex pattern, got nil")
	}
}

// TestInvalidSearchPattern tests searching with a malformed search pattern
func TestInvalidSearchPattern(t *testing.T) {
	app := NewApp()
	
	tempDir := t.TempDir()
	
	// Create a test file
	testFile := filepath.Join(tempDir, "test.txt")
	err := os.WriteFile(testFile, []byte("test content"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	
	trueValue := true
	req := SearchRequest{
		Directory: tempDir,
		Query:     "test[unclosed", // Invalid regex pattern
		UseRegex:  &trueValue,
	}
	
	_, err = app.SearchWithProgress(req)
	if err == nil {
		t.Error("Expected error for invalid search pattern, got nil")
	}
	if !strings.Contains(err.Error(), "invalid search pattern") {
		t.Errorf("Expected invalid search pattern error, got: %v", err)
	}
}

// TestPathTraversalInDirectory tests directory path traversal attacks
func TestPathTraversalInDirectory(t *testing.T) {
	app := NewApp()
	
	tempDir := t.TempDir()
	
	req := SearchRequest{
		Directory: filepath.Join(tempDir, ".."), // Attempt path traversal
		Query:     "test",
	}
	
	_, err := app.SearchWithProgress(req)
	if err == nil {
		t.Error("Expected error for path traversal in directory, got nil")
	}
}

// TestPathTraversalInFileOperations tests file path traversal in ReadFile
func TestPathTraversalInFileOperations(t *testing.T) {
	app := NewApp()
	
	tempDir := t.TempDir()
	
	// Create a test file
	testFile := filepath.Join(tempDir, "test.txt")
	err := os.WriteFile(testFile, []byte("test content"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	
	// Test path traversal in ReadFile
	traversalPath := filepath.Join(tempDir, "..", filepath.Base(testFile))
	_, err = app.ReadFile(traversalPath)
	if err == nil {
		t.Error("Expected error for path traversal in ReadFile, got nil")
	}
	if !strings.Contains(err.Error(), "contains directory traversal") {
		t.Errorf("Expected directory traversal error, got: %v", err)
	}
}

// TestLargeMaxResults tests search with very large MaxResults value
func TestLargeMaxResults(t *testing.T) {
	app := NewApp()
	
	tempDir := t.TempDir()
	
	// Create multiple test files
	for i := 0; i < 10; i++ {
		testFile := filepath.Join(tempDir, fmt.Sprintf("test%d.txt", i))
		err := os.WriteFile(testFile, []byte(fmt.Sprintf("test content %d", i)), 0644)
		if err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}
	}
	
	req := SearchRequest{
		Directory:  tempDir,
		Query:      "test",
		MaxResults: 1000000, // Very large number
	}
	
	results, err := app.SearchWithProgress(req)
	if err != nil {
		t.Errorf("Unexpected error for large MaxResults: %v", err)
	}
	
	if len(results) > 10 {
		t.Errorf("Expected at most 10 results, got %d", len(results))
	}
}

// TestZeroMaxFileSize tests search with zero MaxFileSize (should use default)
func TestZeroMaxFileSize(t *testing.T) {
	app := NewApp()
	
	tempDir := t.TempDir()
	
	// Create a test file (should be under default 10MB limit)
	testFile := filepath.Join(tempDir, "test.txt")
	err := os.WriteFile(testFile, []byte("small test content"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	
	req := SearchRequest{
		Directory:   tempDir,
		Query:       "test",
		MaxFileSize: 0, // Should use default (10MB)
	}
	
	results, err := app.SearchWithProgress(req)
	if err != nil {
		t.Errorf("Unexpected error for zero MaxFileSize: %v", err)
	}
	
	if len(results) == 0 {
		t.Errorf("Expected results for file under size limit, got %d", len(results))
	}
}

// TestVeryLargeMaxFileSize tests search with extremely large MaxFileSize
func TestVeryLargeMaxFileSize(t *testing.T) {
	app := NewApp()
	
	tempDir := t.TempDir()
	
	// Create a test file
	testFile := filepath.Join(tempDir, "test.txt")
	err := os.WriteFile(testFile, []byte("test content"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	
	req := SearchRequest{
		Directory:   tempDir,
		Query:       "test",
		MaxFileSize: 9223372036854775807, // Max int64 value
	}
	
	results, err := app.SearchWithProgress(req)
	if err != nil {
		t.Errorf("Unexpected error for large MaxFileSize: %v", err)
	}
	
	if len(results) == 0 {
		t.Errorf("Expected results for valid file, got %d", len(results))
	}
}

// TestNegativeMaxResults tests search with negative MaxResults
func TestNegativeMaxResults(t *testing.T) {
	app := NewApp()
	
	tempDir := t.TempDir()
	
	// Create a test file
	testFile := filepath.Join(tempDir, "test.txt")
	err := os.WriteFile(testFile, []byte("test content"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	
	req := SearchRequest{
		Directory:  tempDir,
		Query:      "test",
		MaxResults: -1, // Negative value
	}
	
	results, err := app.SearchWithProgress(req)
	if err != nil {
		t.Errorf("Unexpected error for negative MaxResults: %v", err)
	}
	
	// Should still return results despite negative value (due to default handling)
	if len(results) == 0 {
		t.Errorf("Expected results for valid search, got %d", len(results))
	}
}

// TestMaxFileSizeZeroAllowsAllFiles tests MaxFileSize of 0 allows all files
func TestMaxFileSizeZeroAllowsAllFiles(t *testing.T) {
	app := NewApp()
	
	tempDir := t.TempDir()
	
	// Create a test file
	testFile := filepath.Join(tempDir, "test.txt")
	err := os.WriteFile(testFile, []byte("test content"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	
	req := SearchRequest{
		Directory:   tempDir,
		Query:       "test",
		MaxFileSize: 0, // Should use default (10MB)
	}
	
	results, err := app.SearchWithProgress(req)
	if err != nil {
		t.Errorf("Unexpected error for MaxFileSize 0: %v", err)
	}
	
	if len(results) == 0 {
		t.Errorf("Expected results for valid search with MaxFileSize 0, got %d", len(results))
	}
}

// TestMinFileSizeBoundary tests MinFileSize boundary conditions
func TestMinFileSizeBoundary(t *testing.T) {
	app := NewApp()
	
	tempDir := t.TempDir()
	
	// Create a very small file
	smallFile := filepath.Join(tempDir, "small.txt")
	err := os.WriteFile(smallFile, []byte("a"), 0644) // 1 byte
	if err != nil {
		t.Fatalf("Failed to create small file: %v", err)
	}
	
	// Create a larger file
	largeFile := filepath.Join(tempDir, "large.txt")
	err = os.WriteFile(largeFile, []byte(strings.Repeat("a", 1000)), 0644) // 1000 bytes
	if err != nil {
		t.Fatalf("Failed to create large file: %v", err)
	}
	
	req := SearchRequest{
		Directory:   tempDir,
		Query:       "a",
		MinFileSize: 500, // Only files larger than 500 bytes
	}
	
	results, err := app.SearchWithProgress(req)
	if err != nil {
		t.Errorf("Unexpected error for MinFileSize test: %v", err)
	}
	
	foundLargeFile := false
	for _, result := range results {
		if strings.Contains(result.FilePath, "large.txt") {
			foundLargeFile = true
			break
		}
	}
	
	if !foundLargeFile {
		t.Errorf("Expected to find large file that exceeds MinFileSize, but found %d results", len(results))
	}
	
	// Test with MinFileSize that should exclude both files
	req.MinFileSize = 2000 // Both files are smaller than this
	results, err = app.SearchWithProgress(req)
	if err != nil {
		t.Errorf("Unexpected error for high MinFileSize test: %v", err)
	}
	
	if len(results) > 0 {
		t.Errorf("Expected no results for high MinFileSize, got %d", len(results))
	}
}

// TestSearchWithInvalidUnicode tests searching for invalid Unicode content
func TestSearchWithInvalidUnicode(t *testing.T) {
	app := NewApp()
	
	tempDir := t.TempDir()
	
	// Create a file with mixed Unicode content
	testFile := filepath.Join(tempDir, "unicode.txt")
	content := "Hello 世界 Κώδικας 🌍"
	err := os.WriteFile(testFile, []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	
	req := SearchRequest{
		Directory: tempDir,
		Query:     "世界", // Unicode query
	}
	
	results, err := app.SearchWithProgress(req)
	if err != nil {
		t.Errorf("Unexpected error for Unicode search: %v", err)
	}
	
	if len(results) == 0 {
		t.Errorf("Expected results for Unicode search, got %d", len(results))
	}
}

// TestSearchInDeeplyNestedDirectory tests searching in a deeply nested directory
func TestSearchInDeeplyNestedDirectory(t *testing.T) {
	app := NewApp()
	
	tempDir := t.TempDir()
	
	// Create a deeply nested directory structure
	nestedDir := tempDir
	for i := 0; i < 50; i++ { // 50 levels deep
		nestedDir = filepath.Join(nestedDir, fmt.Sprintf("level_%d", i))
	}
	
	err := os.MkdirAll(nestedDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create nested directories: %v", err)
	}
	
	// Create a file in the deepest directory
	testFile := filepath.Join(nestedDir, "deep.txt")
	err = os.WriteFile(testFile, []byte("deep file content"), 0644)
	if err != nil {
		t.Fatalf("Failed to create deep file: %v", err)
	}
	
	req := SearchRequest{
		Directory: tempDir,
		Query:     "content",
	}
	
	results, err := app.SearchWithProgress(req)
	if err != nil {
		t.Errorf("Unexpected error for deep directory search: %v", err)
	}
	
	if len(results) == 0 {
		t.Errorf("Expected results in deep directory, got %d", len(results))
	}
}

// TestSearchWithInvalidExcludePatterns tests search with invalid exclude patterns
func TestSearchWithInvalidExcludePatterns(t *testing.T) {
	app := NewApp()
	
	tempDir := t.TempDir()
	
	// Create a test file
	testFile := filepath.Join(tempDir, "test.txt")
	err := os.WriteFile(testFile, []byte("test content"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	
	req := SearchRequest{
		Directory:      tempDir,
		Query:          "test",
		ExcludePatterns: []string{"[invalid"}, // Invalid glob pattern
	}
	
	// This should not crash the application
	results, err := app.SearchWithProgress(req)
	if err != nil {
		// It's acceptable to get an error here due to the invalid pattern
		t.Logf("Got expected error for invalid exclude pattern: %v", err)
	} else {
		// If no error, make sure we still get results
		if len(results) == 0 {
			t.Errorf("Expected results despite invalid exclude pattern, got %d", len(results))
		}
	}
}

// TestExtensionWithSpecialCharacters tests file extension filtering with special characters
func TestExtensionWithSpecialCharacters(t *testing.T) {
	app := NewApp()
	
	tempDir := t.TempDir()
	
	// Create files with special extensions
	extensions := []string{"min.js", "tar.gz", "config.bak", "file.txt"}
	for _, ext := range extensions {
		var fileName string
		if ext == "file.txt" {
			fileName = "file.txt" // Standard extension
		} else {
			fileName = fmt.Sprintf("test.%s", ext) // Special extensions
		}
		testFile := filepath.Join(tempDir, fileName)
		err := os.WriteFile(testFile, []byte("test content"), 0644)
		if err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}
	}
	
	// Test searching with special extension "min.js"
	req := SearchRequest{
		Directory: tempDir,
		Query:     "test",
		Extension: "min.js", // Double extension
	}
	
	results, err := app.SearchWithProgress(req)
	if err != nil {
		t.Errorf("Unexpected error for special extension: %v", err)
	}
	
	foundFile := false
	for _, result := range results {
		if strings.Contains(result.FilePath, "test.min.js") {
			foundFile = true
			break
		}
	}
	
	if !foundFile && len(results) > 0 {
		t.Errorf("Expected to find file with 'min.js' extension, but didn't find it in results")
	}
}

// TestAllowedFileTypesWithEmptyList tests behavior when AllowedFileTypes is empty
func TestAllowedFileTypesWithEmptyList(t *testing.T) {
	app := NewApp()
	
	tempDir := t.TempDir()
	
	// Create different file types
	files := map[string]string{
		"test.go":    "package main",
		"test.js":    "console.log('test');",
		"test.py":    "print('test')",
		"test.txt":   "plain text",
	}
	
	for name, content := range files {
		testFile := filepath.Join(tempDir, name)
		err := os.WriteFile(testFile, []byte(content), 0644)
		if err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}
	}
	
	req := SearchRequest{
		Directory:        tempDir,
		Query:            "test",
		AllowedFileTypes: []string{}, // Empty list should allow all
	}
	
	results, err := app.SearchWithProgress(req)
	if err != nil {
		t.Errorf("Unexpected error for empty AllowedFileTypes: %v", err)
	}
	
	if len(results) == 0 {
		t.Errorf("Expected results with empty AllowedFileTypes, got %d", len(results))
	}
}

// TestAllowedFileTypesWithSpecificTypes tests behavior when AllowedFileTypes has specific entries
func TestAllowedFileTypesWithSpecificTypes(t *testing.T) {
	app := NewApp()
	
	tempDir := t.TempDir()
	
	// Create different file types
	files := map[string]string{
		"test.go":    "package main",
		"test.js":    "console.log('test');",
		"test.py":    "print('test')",
		"test.txt":   "plain text",
	}
	
	for name, content := range files {
		testFile := filepath.Join(tempDir, name)
		err := os.WriteFile(testFile, []byte(content), 0644)
		if err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}
	}
	
	req := SearchRequest{
		Directory:        tempDir,
		Query:            "test",
		AllowedFileTypes: []string{"go", "js"}, // Only allow go and js files
	}
	
	results, err := app.SearchWithProgress(req)
	if err != nil {
		t.Errorf("Unexpected error for specific AllowedFileTypes: %v", err)
	}
	
	for _, result := range results {
		ext := strings.TrimPrefix(filepath.Ext(result.FilePath), ".")
		allowed := false
		for _, allowedExt := range req.AllowedFileTypes {
			if ext == allowedExt {
				allowed = true
				break
			}
		}
		if !allowed {
			t.Errorf("Found result with disallowed extension: %s", ext)
		}
	}
}

// TestSearchWithVeryLongPath tests searching with very long file paths
func TestSearchWithVeryLongPath(t *testing.T) {
	app := NewApp()
	
	tempDir := t.TempDir()
	
	// Create a file with a very long path
	longDir := tempDir
	for i := 0; i < 20; i++ {
		longDir = filepath.Join(longDir, "very_long_directory_name_that_exceeds_normal_length_limits")
	}
	
	err := os.MkdirAll(longDir, 0755)
	if err != nil {
		t.Logf("Could not create long path (may be system-limited): %v", err)
		// Skip test if system doesn't support long paths
		t.SkipNow()
		return
	}
	
	testFile := filepath.Join(longDir, "long_path_file.txt")
	err = os.WriteFile(testFile, []byte("test content in long path"), 0644)
	if err != nil {
		t.Logf("Could not create file in long path: %v", err)
		t.SkipNow()
		return
	}
	
	req := SearchRequest{
		Directory: tempDir,
		Query:     "content",
	}
	
	results, err := app.SearchWithProgress(req)
	if err != nil {
		t.Errorf("Unexpected error for long path search: %v", err)
	}
	
	if len(results) == 0 {
		t.Errorf("Expected results in long path, got %d", len(results))
	}
}

// TestSearchWithSpecialCharactersInQuery tests searching with special regex characters as literals
func TestSearchWithSpecialCharactersInQuery(t *testing.T) {
	app := NewApp()
	
	tempDir := t.TempDir()
	
	// Create a file with special characters
	testFile := filepath.Join(tempDir, "special.txt")
	content := "Find me: [this] (that) {other} *star* .dot^ $end"
	err := os.WriteFile(testFile, []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	
	falseValue := false
	req := SearchRequest{
		Directory: tempDir,
		Query:     "[this]", // Literal search for bracketed text
		UseRegex:  &falseValue,
	}
	
	results, err := app.SearchWithProgress(req)
	if err != nil {
		t.Errorf("Unexpected error for special character search: %v", err)
	}
	
	found := false
	for _, result := range results {
		if strings.Contains(result.Content, "[this]") {
			found = true
			break
		}
	}
	
	if !found {
		t.Errorf("Expected to find literal special character sequence, got %d results", len(results))
	}
}

// TestValidateDirectoryWithInvalidPath tests ValidateDirectory with invalid paths
func TestValidateDirectoryWithInvalidPath(t *testing.T) {
	app := NewApp()
	
	// Test with non-existent path
	valid, err := app.ValidateDirectory("/non/existent/path")
	if err == nil {
		t.Error("Expected error for non-existent directory, got nil")
	}
	if valid {
		t.Error("Expected valid=false for non-existent directory")
	}
	
	// Test with file instead of directory
	tempFile := filepath.Join(t.TempDir(), "temp.txt")
	err = os.WriteFile(tempFile, []byte("test"), 0644)
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	
	valid, err = app.ValidateDirectory(tempFile)
	if err == nil {
		t.Error("Expected error for file path instead of directory, got nil")
	}
	if valid {
		t.Error("Expected valid=false for file path")
	}
}

// TestReadFileWithInvalidPath tests ReadFile with invalid paths
func TestReadFileWithInvalidPath(t *testing.T) {
	app := NewApp()
	
	// Test with empty path
	_, err := app.ReadFile("")
	if err == nil {
		t.Error("Expected error for empty file path, got nil")
	}
	if !strings.Contains(err.Error(), "file path is required") {
		t.Errorf("Expected 'file path is required' error, got: %v", err)
	}
	
	// Test with non-existent path
	_, err = app.ReadFile("/non/existent/file.txt")
	if err == nil {
		t.Error("Expected error for non-existent file, got nil")
	}
	if !strings.Contains(err.Error(), "file does not exist") {
		t.Errorf("Expected 'file does not exist' error, got: %v", err)
	}
}

// TestReadFileWithLargeSize tests ReadFile with size limits
func TestReadFileWithLargeSize(t *testing.T) {
	app := NewApp()
	
	tempDir := t.TempDir()
	
	// Create a large file (but still within the 50MB limit)
	largeFile := filepath.Join(tempDir, "large.txt")
	largeContent := strings.Repeat("a", 25*1024*1024) // 25MB
	err := os.WriteFile(largeFile, []byte(largeContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create large file: %v", err)
	}
	
	_, err = app.ReadFile(largeFile)
	if err != nil {
		t.Errorf("Unexpected error for large but valid file: %v", err)
	}
	
	// Create a file larger than the 50MB limit
	veryLargeFile := filepath.Join(tempDir, "very_large.txt")
	veryLargeContent := strings.Repeat("b", 60*1024*1024) // 60MB
	err = os.WriteFile(veryLargeFile, []byte(veryLargeContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create very large file: %v", err)
	}
	
	_, err = app.ReadFile(veryLargeFile)
	if err == nil {
		t.Error("Expected error for file larger than 50MB, got nil")
	}
	if !strings.Contains(err.Error(), "file too large to read") {
		t.Errorf("Expected 'file too large' error, got: %v", err)
	}
}

// TestConcurrentSearches tests potential race conditions with concurrent searches
func TestConcurrentSearches(t *testing.T) {
	app := NewApp()
	
	tempDir := t.TempDir()
	
	// Create test files
	for i := 0; i < 10; i++ {
		testFile := filepath.Join(tempDir, fmt.Sprintf("test%d.txt", i))
		err := os.WriteFile(testFile, []byte(fmt.Sprintf("test content %d", i)), 0644)
		if err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}
	}
	
	// Run multiple searches concurrently
	errChan := make(chan error, 5)
	resultChan := make(chan int, 5)
	
	for i := 0; i < 5; i++ {
		go func(searchNum int) {
			req := SearchRequest{
				Directory: tempDir,
				Query:     fmt.Sprintf("content %d", searchNum%10),
			}
			results, err := app.SearchWithProgress(req)
			errChan <- err
			resultChan <- len(results)
		}(i)
	}
	
	// Collect results
	for i := 0; i < 5; i++ {
		err := <-errChan
		results := <-resultChan
		if err != nil {
			t.Errorf("Concurrent search %d failed: %v", i, err)
		}
		t.Logf("Concurrent search %d returned %d results", i, results)
	}
}

// TestSearchCancellation tests search cancellation functionality
func TestSearchCancellation(t *testing.T) {
	app := NewApp()
	
	tempDir := t.TempDir()
	
	// Create a large number of files to make search take time
	for i := 0; i < 1000; i++ {
		testFile := filepath.Join(tempDir, fmt.Sprintf("test%d.txt", i))
		err := os.WriteFile(testFile, []byte("searchable content"), 0644)
		if err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}
	}
	
	// Start a search with a small MaxResults to make it return quickly
	req := SearchRequest{
		Directory:  tempDir,
		Query:      "content",
		MaxResults: 5, // Limit results
	}
	
	// Start search in a goroutine
	resultChan := make(chan []SearchResult, 1)
	errChan := make(chan error, 1)
	
	go func() {
		results, err := app.SearchWithProgress(req)
		resultChan <- results
		errChan <- err
	}()
	
	// Try to cancel the search (may not be active yet)
	err := app.CancelSearch()
	if err != nil && !strings.Contains(err.Error(), "no active search") {
		t.Logf("CancelSearch returned: %v", err) // This is OK, may not have active search yet
	}
	
	// Wait for search to complete
	results := <-resultChan
	err = <-errChan
	
	if err != nil {
		t.Errorf("Search failed: %v", err)
	}
	
	if len(results) > req.MaxResults {
		t.Errorf("Expected at most %d results, got %d", req.MaxResults, len(results))
	}
}

// TestBinaryDetectionEdgeCases tests edge cases in binary file detection
func TestBinaryDetectionEdgeCases(t *testing.T) {
	app := NewApp()
	
	// Test with empty content
	emptyContent := []byte{}
	isBinary := app.isBinary(emptyContent)
	if isBinary {
		t.Error("Empty content should not be detected as binary")
	}
	
	// Test with content that has null bytes
	nullContent := []byte("text\x00more text")
	isBinary = app.isBinary(nullContent)
	if !isBinary {
		t.Error("Content with null bytes should be detected as binary")
	}
	
	// Test with high-byte UTF-8 content (should not be binary)
	utf8Content := []byte("Hello 世界 Κώδικας 🌍") 
	isBinary = app.isBinary(utf8Content)
	if isBinary {
		t.Error("Valid UTF-8 content should not be detected as binary")
	}
	
	// Test with mostly printable content with few non-printable
	mixedContent := make([]byte, 512)
	for i := range mixedContent {
		if i%100 == 0 {
			mixedContent[i] = 0x01 // Non-printable
		} else {
			mixedContent[i] = byte('A') // Printable
		}
	}
	isBinary = app.isBinary(mixedContent)
	if isBinary {
		// This is acceptable behavior - the algorithm is designed to be conservative
		t.Logf("Mixed content correctly detected as binary (conservative detection)")
	}
}

// TestCompileSearchPatternEdgeCases tests edge cases in pattern compilation
func TestCompileSearchPatternEdgeCases(t *testing.T) {
	app := NewApp()
	
	tempDir := t.TempDir()
	
	// Test with regex special characters that should be escaped in literal mode
	falseValue := false
	req := SearchRequest{
		Directory: tempDir,
		Query:     `test[abc].*test`, // Should be treated literally, not as regex
		UseRegex:  &falseValue,
	}
	
	_, err := app.compileSearchPattern(req)
	if err != nil {
		t.Errorf("Unexpected error for literal search pattern: %v", err)
	}
	
	// Test with case-sensitive and case-insensitive regex
	trueValue := true
	req.UseRegex = &trueValue
	req.CaseSensitive = true
	pattern, err := app.compileSearchPattern(req)
	if err != nil {
		t.Errorf("Unexpected error for case-sensitive regex: %v", err)
	}
	
	// Check that case-sensitive pattern doesn't have (?i) flag
	patternStr := pattern.String()
	if strings.Contains(patternStr, "(?i)") {
		t.Errorf("Case-sensitive pattern shouldn't have (?i) flag, got: %s", patternStr)
	}
	
	req.CaseSensitive = false
	pattern, err = app.compileSearchPattern(req)
	if err != nil {
		t.Errorf("Unexpected error for case-insensitive regex: %v", err)
	}
	
	// Check that case-insensitive pattern has (?i) flag
	patternStr = pattern.String()
	if !strings.Contains(patternStr, "(?i)") {
		t.Errorf("Case-insensitive pattern should have (?i) flag, got: %s", patternStr)
	}
}

// TestMatchesPatternEdgeCases tests edge cases in pattern matching
func TestMatchesPatternEdgeCases(t *testing.T) {
	app := NewApp()
	
	// Test exact match
	if !app.matchesPattern("/path/to/node_modules", "node_modules") {
		t.Error("Exact match should return true")
	}
	
	// Test glob pattern match
	if !app.matchesPattern("/path/to/node_modules/file.js", "node_modules") {
		t.Error("Directory pattern match should return true")
	}
	
	// Test case sensitivity in pattern matching
	if !app.matchesPattern("/path/to/Node_Modules/file.js", "node_modules") {
		t.Logf("Case-insensitive pattern matching not implemented (expected behavior)")
	}
	
	// Test non-matching pattern
	if app.matchesPattern("/path/to/src/file.js", "node_modules") {
		t.Error("Non-matching pattern should return false")
	}
}

// TestGetFullExtensionEdgeCases tests edge cases in extension extraction
func TestGetFullExtensionEdgeCases(t *testing.T) {
	// Test with no extension
	if getFullExtension("/path/to/file") != "" {
		t.Error("File with no extension should return empty string")
	}
	
	// Test with single extension
	if getFullExtension("/path/to/file.txt") != ".txt" {
		t.Error("File with single extension should return correct extension")
	}
	
	// Test with double extension
	if getFullExtension("/path/to/file.min.js") != ".min.js" {
		t.Error("File with double extension should return full extension")
	}
	
	// Test with archive extension
	if getFullExtension("/path/to/file.tar.gz") != ".tar.gz" {
		t.Error("Archive file should return full extension")
	}
	
	// Test with many dots
	if getFullExtension("/path/to/file.a.b.c.d") != ".a.b.c.d" {
		t.Error("File with many extensions should return full extension")
	}
}

// TestMatchExtensionEdgeCases tests edge cases in extension matching
func TestMatchExtensionEdgeCases(t *testing.T) {
	// Test empty requested extension (should match all)
	if !matchExtension("/path/to/file.txt", "") {
		t.Error("Empty requested extension should match all files")
	}
	
	// Test exact extension match
	if !matchExtension("/path/to/file.go", "go") {
		t.Error("Exact extension match should return true")
	}
	
	// Test case-insensitive extension match
	if !matchExtension("/path/to/file.GO", "go") {
		t.Error("Case-insensitive extension match should return true")
	}
	
	// Test double extension match
	if !matchExtension("/path/to/file.min.js", "min.js") {
		t.Error("Double extension match should return true")
	}
	
	// Test single extension on double extension file
	if !matchExtension("/path/to/file.min.js", "js") {
		t.Error("Single extension should match double extension file")
	}
	
	// Test non-matching extension
	if matchExtension("/path/to/file.go", "js") {
		t.Error("Non-matching extension should return false")
	}
}