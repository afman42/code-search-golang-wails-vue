package main

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestSearchCodeDebug(t *testing.T) {
	app := NewApp()

	// Create a temporary directory for testing
	tempDir := t.TempDir()
	fmt.Printf("Created temp directory: %s\n", tempDir)
	
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
		fmt.Printf("Created test file: %s with content: %s\n", filePath, content)
	}
	
	// Test: Search for "hello"
	fmt.Println("Testing search for 'hello'...")
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
	
	fmt.Printf("SearchCode returned %d results\n", len(results))
	for i, result := range results {
		fmt.Printf("Result %d: Path=%s, Line=%d, Content='%s'\n", i+1, result.FilePath, result.LineNum, result.Content)
	}
	
	if len(results) != 3 {
		t.Errorf("Expected 3 results, got %d", len(results))
	} else {
		fmt.Println("SUCCESS: Found expected number of results")
	}
}