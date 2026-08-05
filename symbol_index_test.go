package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestSymbolIndexCache verifies that the persistent symbol index returns
// cached results when the directory fingerprint is unchanged, and rescans
// when a file changes.
func TestSymbolIndexCache(t *testing.T) {
	cache := newSymbolIndexCache()
	globalSymbolIndex = cache
	defer func() { globalSymbolIndex = nil }()

	tempDir := t.TempDir()
	goFile := filepath.Join(tempDir, "main.go")
	os.WriteFile(goFile, []byte("package main\nfunc Foo() {}\n"), 0o644)

	// First call — cache miss, should extract and store.
	symbols1 := GetAllSymbolsWithProgress(tempDir, 1000, nil)
	if len(symbols1) != 1 || symbols1[0].Name != "Foo" {
		t.Fatalf("expected 1 symbol 'Foo', got %+v", symbols1)
	}

	// Verify the cache was populated.
	cache.mu.RLock()
	entry, ok := cache.entries[tempDir]
	cache.mu.RUnlock()
	if !ok {
		t.Fatal("expected cache entry after first extraction")
	}
	if entry.fingerprint == "" {
		t.Fatal("expected non-empty fingerprint")
	}

	// Second call — cache hit (fingerprint unchanged), should return instantly.
	symbols2 := GetAllSymbolsWithProgress(tempDir, 1000, nil)
	if len(symbols2) != 1 || symbols2[0].Name != "Foo" {
		t.Fatalf("cached call returned wrong symbols: %+v", symbols2)
	}

	// Modify the file — fingerprint changes, cache should miss and rescan.
	// Use a different mtime so the fingerprint differs.
	os.WriteFile(goFile, []byte("package main\nfunc Bar() {}\n"), 0o644)
	// Ensure mtime advances (some filesystems have coarse resolution).
	newTime := time.Now().Add(2 * time.Second)
	os.Chtimes(goFile, newTime, newTime)

	symbols3 := GetAllSymbolsWithProgress(tempDir, 1000, nil)
	if len(symbols3) != 1 || symbols3[0].Name != "Bar" {
		t.Fatalf("after file change, expected 'Bar', got %+v", symbols3)
	}
}

// TestClearSymbolCache verifies the Wails-bound ClearSymbolCache method
// empties the cache.
func TestClearSymbolCache(t *testing.T) {
	cache := newSymbolIndexCache()
	globalSymbolIndex = cache
	defer func() { globalSymbolIndex = nil }()

	app := &App{symbolIndex: cache}

	tempDir := t.TempDir()
	os.WriteFile(filepath.Join(tempDir, "main.go"), []byte("package main\nfunc Foo() {}\n"), 0o644)

	GetAllSymbolsWithProgress(tempDir, 1000, nil)
	if len(cache.entries) != 1 {
		t.Fatalf("expected 1 cache entry, got %d", len(cache.entries))
	}

	app.ClearSymbolCache()
	if len(cache.entries) != 0 {
		t.Fatalf("expected 0 entries after clear, got %d", len(cache.entries))
	}
}
