package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReadFile(t *testing.T) {
	app := NewApp()

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
	
	t.Run("ReadExistingFile", func(t *testing.T) {
		filePath := filepath.Join(tempDir, "test1.go")
		content, err := app.ReadFile(filePath)
		
		if err != nil {
			t.Fatalf("ReadFile returned error: %v", err)
		}
		
		if content == "" {
			t.Error("ReadFile returned empty content")
		}
		
		expected := "package main\n\nimport \"fmt\"\n\nfunc main() {\n\tfmt.Println(\"hello world\")\n}"
		if content != expected {
			t.Errorf("ReadFile returned unexpected content. Got: %s, Expected: %s", content, expected)
		}
	})
	
	t.Run("ReadNonExistentFile", func(t *testing.T) {
		nonExistentFile := "/non/existent/file.txt"
		_, err := app.ReadFile(nonExistentFile)
		
		if err == nil {
			t.Error("ReadFile should return error for non-existent file")
		}
		
		if err != nil && err.Error() == "" {
			t.Error("ReadFile should return meaningful error message for non-existent file")
		}
	})
	
	t.Run("ReadEmptyFilePath", func(t *testing.T) {
		_, err := app.ReadFile("")
		
		if err == nil {
			t.Error("ReadFile should return error for empty file path")
		}
		
		if err != nil && err.Error() == "" {
			t.Error("ReadFile should return meaningful error message for empty file path")
		}
	})
	
	t.Run("ReadDifferentFileTypes", func(t *testing.T) {
		// Test reading JavaScript file
		jsFile := filepath.Join(tempDir, "test2.js")
		jsContent, err := app.ReadFile(jsFile)
		
		if err != nil {
			t.Fatalf("ReadFile returned error for JS file: %v", err)
		}
		
		if jsContent == "" {
			t.Error("ReadFile returned empty content for JS file")
		}
		
		expectedJS := "console.log('hello world');\nconsole.log('test');"
		if jsContent != expectedJS {
			t.Errorf("ReadFile returned unexpected content for JS file. Got: %s, Expected: %s", jsContent, expectedJS)
		}
		
		// Test reading text file
		txtFile := filepath.Join(tempDir, "test3.txt")
		txtContent, err := app.ReadFile(txtFile)
		
		if err != nil {
			t.Fatalf("ReadFile returned error for TXT file: %v", err)
		}
		
		if txtContent == "" {
			t.Error("ReadFile returned empty content for TXT file")
		}
		
		expectedTxt := "This is a test file with hello world content"
		if txtContent != expectedTxt {
			t.Errorf("ReadFile returned unexpected content for TXT file. Got: %s, Expected: %s", txtContent, expectedTxt)
		}
	})
}