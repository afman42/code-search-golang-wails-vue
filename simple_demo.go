package main

import (
	"fmt"
	"os"
	"path/filepath"
)

func demoSearch() {
	// Create a simple test to verify search works in a real scenario
	app := NewApp()

	// Create a test directory with some content
	tempDir := os.TempDir()
	testDir := filepath.Join(tempDir, "search_test")
	os.RemoveAll(testDir) // Clean up if exists
	os.MkdirAll(testDir, 0755)

	// Create a test file
	testFile := filepath.Join(testDir, "sample.go")
	testContent := `package main

import "fmt"

func main() {
    fmt.Println("Hello, World!")
    // This is another line with hello
}`
	
	err := os.WriteFile(testFile, []byte(testContent), 0644)
	if err != nil {
		fmt.Printf("Error creating test file: %v\n", err)
		return
	}

	// Perform search
	req := SearchRequest{
		Directory:     testDir,
		Query:         "Hello",
		Extension:     "go",
		CaseSensitive: false,
	}

	fmt.Printf("Searching for '%s' in %s\n", req.Query, req.Directory)
	results, err := app.SearchCode(req)
	if err != nil {
		fmt.Printf("Error during search: %v\n", err)
		return
	}

	fmt.Printf("Found %d results:\n", len(results))
	for _, result := range results {
		fmt.Printf("  File: %s, Line: %d, Content: '%s'\n", 
			result.FilePath, result.LineNum, result.Content)
	}

	// Clean up
	os.RemoveAll(testDir)
}