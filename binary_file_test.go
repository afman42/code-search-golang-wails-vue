package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestBinaryFileFiltering tests the IncludeBinary functionality
func TestBinaryFileFiltering(t *testing.T) {
	app := NewApp()

	tempDir := t.TempDir()

	// Create a text file with search term
	textFile := filepath.Join(tempDir, "text.txt")
	textContent := "This is a text file with test pattern inside"
	err := os.WriteFile(textFile, []byte(textContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create text file: %v", err)
	}

	// Create a binary file with the same search term
	// Use content that includes null bytes to make it clearly binary
	binaryFile := filepath.Join(tempDir, "binary.dat")
	binaryContent := []byte("This is a binary file with test pattern inside\x00more binary data")
	err = os.WriteFile(binaryFile, binaryContent, 0644)
	if err != nil {
		t.Fatalf("Failed to create binary file: %v", err)
	}

	t.Run("IncludeBinaryTrue", func(t *testing.T) {
		// Test with IncludeBinary=true, should find matches in both files
		req := SearchRequest{
			Directory:     tempDir,
			Query:         "test pattern",
			Extension:     "",
			IncludeBinary: true, // Include binary files
		}

		results, err := app.SearchWithProgress(req)
		if err != nil {
			t.Fatalf("SearchWithProgress failed: %v", err)
		}

		// Should find results in both files
		foundText := false
		foundBinary := false
		for _, result := range results {
			if strings.Contains(result.FilePath, "text.txt") {
				foundText = true
			}
			if strings.Contains(result.FilePath, "binary.dat") {
				foundBinary = true
			}
		}

		if !foundText {
			t.Error("Expected to find result in text file when IncludeBinary=true")
		}
		if !foundBinary {
			t.Error("Expected to find result in binary file when IncludeBinary=true")
		}
	})

	t.Run("IncludeBinaryFalse", func(t *testing.T) {
		// Test with IncludeBinary=false, should find matches only in text files
		req := SearchRequest{
			Directory:     tempDir,
			Query:         "test pattern",
			Extension:     "",
			IncludeBinary: false, // Do NOT include binary files
		}

		results, err := app.SearchWithProgress(req)
		if err != nil {
			t.Fatalf("SearchWithProgress failed: %v", err)
		}

		// Should find results only in text file, not in binary file
		foundText := false
		foundBinary := false
		for _, result := range results {
			if strings.Contains(result.FilePath, "text.txt") {
				foundText = true
			}
			if strings.Contains(result.FilePath, "binary.dat") {
				foundBinary = true
			}
		}

		if !foundText {
			t.Error("Expected to find result in text file when IncludeBinary=false")
		}
		if foundBinary {
			t.Error("Should not find result in binary file when IncludeBinary=false")
		}
	})

	t.Run("BinaryDetection", func(t *testing.T) {
		// Test the isBinary function directly
		binaryWithNulls := []byte("some text\x00binary content")
		if !app.isBinary(binaryWithNulls) {
			t.Error("isBinary should return true for content with null bytes")
		}

		textWithoutNulls := []byte("just regular text content")
		if app.isBinary(textWithoutNulls) {
			t.Error("isBinary should return false for text-only content")
		}

		// Test with high percentage of non-printable characters
		mostlyBinary := []byte{1, 2, 3, 4, 5, 't', 'e', 's', 't'} // majority are non-printable
		if !app.isBinary(mostlyBinary) {
			t.Error("isBinary should return true for content with high percentage of non-printable characters")
		}
	})
}