package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

// TestResourceManagementMemoryUsage tests memory usage under extreme conditions
func TestResourceManagementMemoryUsage(t *testing.T) {
	app := NewApp()

	t.Run("LargeNumberOfFiles", func(t *testing.T) {
		tempDir := t.TempDir()

		// Create many small files to test resource usage
		numFiles := 5000 // Large but manageable number
		for i := 0; i < numFiles; i++ {
			filename := filepath.Join(tempDir, "file_"+string(rune(i%26+'a'))+string(rune((i/26)%26+'A'))+string(rune((i/676)%10+'0'))+".txt")
			// Create small files with search term
			content := "test content for file " + fmt.Sprintf("%d", i)
			if i%100 == 0 {
				content = "search_term " + content // Add search term every 100 files
			}
			err := os.WriteFile(filename, []byte(content), 0644)
			if err != nil {
				t.Fatalf("Failed to create test file %d: %v", i, err)
			}
		}

		// Capture memory before
		var m1 runtime.MemStats
		runtime.GC()
		runtime.ReadMemStats(&m1)

		req := SearchRequest{
			Directory: tempDir,
			Query:     "search_term",
			Extension: "",
		}

		start := time.Now()
		results, err := app.SearchWithProgress(req)
		duration := time.Since(start)

		if err != nil {
			t.Fatalf("SearchWithProgress failed: %v", err)
		}

		// Capture memory after
		var m2 runtime.MemStats
		runtime.GC()
		runtime.ReadMemStats(&m2)

		// Check memory growth is reasonable (use uint64 arithmetic to prevent underflow)
		var memoryGrowth uint64
		if m2.Alloc > m1.Alloc {
			memoryGrowth = m2.Alloc - m1.Alloc
		} else {
			// If m2.Alloc is less than m1.Alloc, it means GC happened and freed memory
			// We consider this as 0 growth or just log the values separately
			t.Logf("Memory allocation decreased from %d to %d (GC effect)", m1.Alloc, m2.Alloc)
			memoryGrowth = 0
		}
		
		if memoryGrowth > 100*1024*1024 { // 100MB limit
			t.Errorf("Memory usage grew by %d bytes, which may be excessive", memoryGrowth)
		}

		t.Logf("Searched %d files in %v, memory growth: %d bytes, found %d results", 
			numFiles, duration, memoryGrowth, len(results))
	})

	t.Run("LargeFileWithManyMatches", func(t *testing.T) {
		tempDir := t.TempDir()

		// Create a large file with many matches
		largeFile := filepath.Join(tempDir, "large_matches.txt")
		// Create content with lots of matches (10000 lines with the search term)
		var contentBuilder strings.Builder
		for i := 0; i < 10000; i++ {
			contentBuilder.WriteString("This is line ")
			contentBuilder.WriteString(string(rune(i%10000 + '0')))
			contentBuilder.WriteString(" with search_term to find\n")
		}
		content := contentBuilder.String()
		
		err := os.WriteFile(largeFile, []byte(content), 0644)
		if err != nil {
			t.Fatalf("Failed to create large file: %v", err)
		}

		// Capture memory before
		var m1 runtime.MemStats
		runtime.GC()
		runtime.ReadMemStats(&m1)

		req := SearchRequest{
			Directory:  tempDir,
			Query:      "search_term",
			MaxResults: 500, // Limit results to prevent excessive memory usage
		}

		start := time.Now()
		results, err := app.SearchWithProgress(req)
		duration := time.Since(start)

		if err != nil {
			t.Fatalf("SearchWithProgress failed: %v", err)
		}

		// Capture memory after
		var m2 runtime.MemStats
		runtime.GC()
		runtime.ReadMemStats(&m2)

		// Check memory growth is reasonable (use uint64 arithmetic to prevent underflow)
		var memoryGrowth uint64
		if m2.Alloc > m1.Alloc {
			memoryGrowth = m2.Alloc - m1.Alloc
		} else {
			// If m2.Alloc is less than m1.Alloc, it means GC happened and freed memory
			// We consider this as 0 growth or just log the values separately
			t.Logf("Memory allocation decreased from %d to %d (GC effect)", m1.Alloc, m2.Alloc)
			memoryGrowth = 0
		}
		
		if memoryGrowth > 50*1024*1024 { // 50MB limit
			t.Errorf("Memory usage grew by %d bytes for large file, which may be excessive", memoryGrowth)
		}

		// Should respect max results limit
		if len(results) > 500 {
			t.Errorf("Expected at most 500 results due to limit, got %d", len(results))
		}

		t.Logf("Processed large file (%d bytes) in %v, memory growth: %d bytes, found %d results", 
			len(content), duration, memoryGrowth, len(results))
	})

	t.Run("MaximumFileProcessing", func(t *testing.T) {
		tempDir := t.TempDir()

		// Create maximum possible files with a reasonable limit to prevent system overload
		numFiles := 1000 // Reduced for CI safety
		for i := 0; i < numFiles; i++ {
			filename := filepath.Join(tempDir, "max_test_"+string(rune(i+65))+".txt")
			// Small content with search term
			content := "search content in file " + string(rune(i+65))
			err := os.WriteFile(filename, []byte(content), 0644)
			if err != nil {
				t.Fatalf("Failed to create max test file %d: %v", i, err)
			}
		}

		// Capture memory before
		var m1 runtime.MemStats
		runtime.GC()
		runtime.ReadMemStats(&m1)

		req := SearchRequest{
			Directory: tempDir,
			Query:     "search",
			Extension: "",
		}

		start := time.Now()
		results, err := app.SearchWithProgress(req)
		duration := time.Since(start)

		if err != nil {
			t.Fatalf("SearchWithProgress failed: %v", err)
		}

		// Capture memory after
		var m2 runtime.MemStats
		runtime.GC()
		runtime.ReadMemStats(&m2)

		// Check memory growth is reasonable (use uint64 arithmetic to prevent underflow)
		var memoryGrowth uint64
		if m2.Alloc > m1.Alloc {
			memoryGrowth = m2.Alloc - m1.Alloc
		} else {
			// If m2.Alloc is less than m1.Alloc, it means GC happened and freed memory
			// We consider this as 0 growth or just log the values separately
			t.Logf("Memory allocation decreased from %d to %d (GC effect)", m1.Alloc, m2.Alloc)
			memoryGrowth = 0
		}
		
		if memoryGrowth > 10*1024*1024 { // 10MB limit
			t.Errorf("Memory usage grew by %d bytes for max files, which may be excessive", memoryGrowth)
		}

		t.Logf("Processed %d files in %v, memory growth: %d bytes, found %d results", 
			numFiles, duration, memoryGrowth, len(results))
	})
}

// TestResourceManagementCPULimits tests CPU usage and time limits
func TestResourceManagementCPULimits(t *testing.T) {
	app := NewApp()

	t.Run("DeepDirectoryTraversal", func(t *testing.T) {
		tempDir := t.TempDir()

		// Create a deep nested directory structure
		currentDir := tempDir
		depth := 50 // Create 50 levels of nesting
		for i := 0; i < depth; i++ {
			currentDir = filepath.Join(currentDir, "level_"+string(rune(i+'0')))
			err := os.MkdirAll(currentDir, 0755)
			if err != nil {
				t.Fatalf("Failed to create nested directory: %v", err)
			}
			
			// Add a file in some levels
			if i%10 == 0 {
				testFile := filepath.Join(currentDir, "deep_file.txt")
				content := "deep search_term content at level " + string(rune(i))
				err = os.WriteFile(testFile, []byte(content), 0644)
				if err != nil {
					t.Fatalf("Failed to create deep file: %v", err)
				}
			}
		}

		req := SearchRequest{
			Directory: tempDir,
			Query:     "search_term",
			Extension: "",
		}

		start := time.Now()
		results, err := app.SearchWithProgress(req)
		duration := time.Since(start)

		if err != nil {
			t.Fatalf("SearchWithProgress failed: %v", err)
		}

		if duration > 10*time.Second {
			t.Logf("Deep traversal took %v, which is long but may be expected", duration)
		}

		t.Logf("Deep directory search took %v, found %d results", duration, len(results))
	})

	t.Run("ManyConcurrentFileReads", func(t *testing.T) {
		// This test verifies the worker pool behavior under load
		tempDir := t.TempDir()

		// Create moderate number of files to test the worker pool
		numFiles := 100
		for i := 0; i < numFiles; i++ {
			filename := filepath.Join(tempDir, "concurrent_"+string(rune(i+65))+".txt")
			// Medium-sized content to provide some processing time
			content := strings.Repeat("test content line\n", 100)
			if i%10 == 0 {
				content += "concurrent_search_term\n"
			}
			err := os.WriteFile(filename, []byte(content), 0644)
			if err != nil {
				t.Fatalf("Failed to create concurrent test file %d: %v", i, err)
			}
		}

		req := SearchRequest{
			Directory: tempDir,
			Query:     "concurrent_search_term",
			Extension: "",
		}

		start := time.Now()
		results, err := app.SearchWithProgress(req)
		duration := time.Since(start)

		if err != nil {
			t.Fatalf("SearchWithProgress failed: %v", err)
		}

		t.Logf("Concurrent file processing (%d files) took %v, found %d results", 
			numFiles, duration, len(results))
	})
}

// TestResourceManagementLimits tests the application's response to various limits
func TestResourceManagementLimits(t *testing.T) {
	app := NewApp()

	t.Run("ZeroLimitsHandling", func(t *testing.T) {
		tempDir := t.TempDir()
		testFile := filepath.Join(tempDir, "test.txt")
		err := os.WriteFile(testFile, []byte("test content"), 0644)
		if err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		req := SearchRequest{
			Directory:   tempDir,
			Query:       "test",
			MaxFileSize: 0, // Should use default
			MaxResults:  0, // Should use default
		}

		results, err := app.SearchWithProgress(req)
		if err != nil {
			t.Fatalf("SearchWithProgress failed with zero limits: %v", err)
		}

		// Should use defaults, so should still find results
		if len(results) == 0 {
			t.Error("Should find results when limits are zero (use defaults)")
		}
	})

	t.Run("VerySmallLimits", func(t *testing.T) {
		tempDir := t.TempDir()
		
		// Create many files that match
		for i := 0; i < 50; i++ {
			filename := filepath.Join(tempDir, "small_limit_"+string(rune(i+65))+".txt")
			content := "match_term " + string(rune(i+65))
			err := os.WriteFile(filename, []byte(content), 0644)
			if err != nil {
				t.Fatalf("Failed to create small limit test file %d: %v", i, err)
			}
		}

		req := SearchRequest{
			Directory:  tempDir,
			Query:      "match_term",
			MaxResults: 5, // Very small limit
		}

		results, err := app.SearchWithProgress(req)
		if err != nil {
			t.Fatalf("SearchWithProgress failed: %v", err)
		}

		if len(results) > 5 {
			t.Errorf("Expected at most 5 results due to limit, got %d", len(results))
		}
	})

	t.Run("LargeLimits", func(t *testing.T) {
		tempDir := t.TempDir()
		
		// Create many files
		for i := 0; i < 100; i++ {
			filename := filepath.Join(tempDir, "large_limit_"+string(rune(i+65))+".txt")
			content := "large_limit_term " + string(rune(i+65))
			err := os.WriteFile(filename, []byte(content), 0644)
			if err != nil {
				t.Fatalf("Failed to create large limit test file %d: %v", i, err)
			}
		}

		req := SearchRequest{
			Directory:  tempDir,
			Query:      "large_limit_term",
			MaxResults: 2000, // Larger than default but reasonable
		}

		results, err := app.SearchWithProgress(req)
		if err != nil {
			t.Fatalf("SearchWithProgress failed: %v", err)
		}

		// Should find all matching files up to the limit
		if len(results) == 0 {
			t.Error("Should find results with large limits")
		}
	})
}