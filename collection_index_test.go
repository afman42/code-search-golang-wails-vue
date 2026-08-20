package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// collectionCacheKey
// ---------------------------------------------------------------------------

func TestCollectionCacheKeySorting(t *testing.T) {
	// Keys with the same fields in different order must produce the same key.
	req1 := SearchRequest{
		Directory:        "/tmp",
		AllowedFileTypes: []string{".go", ".ts"},
		ExcludePatterns:  []string{"node_modules", ".git"},
		MinFileSize:      100,
		MaxFileSize:      10485760,
		IncludeBinary:    false,
		RespectGitignore: true,
	}
	req2 := req1
	req2.AllowedFileTypes = []string{".ts", ".go"}
	req2.ExcludePatterns = []string{".git", "node_modules"}

	k1 := collectionCacheKey(req1)
	k2 := collectionCacheKey(req2)
	if k1 != k2 {
		t.Errorf("cache keys should be order-independent:\n  %q\n  %q", k1, k2)
	}
}

func TestCollectionCacheKeyExcludesQuery(t *testing.T) {
	// Query changes must NOT affect the cache key (same collection regardless).
	req1 := SearchRequest{Directory: "/tmp", Query: "hello"}
	req2 := SearchRequest{Directory: "/tmp", Query: "world"}

	k1 := collectionCacheKey(req1)
	k2 := collectionCacheKey(req2)
	if k1 != k2 {
		t.Error("cache key should not include query")
	}
}

func TestCollectionCacheKeyDiffersOnFilter(t *testing.T) {
	req1 := SearchRequest{Directory: "/tmp", Extension: ".go"}
	req2 := SearchRequest{Directory: "/tmp", Extension: ".ts"}

	if collectionCacheKey(req1) == collectionCacheKey(req2) {
		t.Error("different extensions should produce different cache keys")
	}
}

func TestCollectionCacheKeyNoSliceCollision(t *testing.T) {
	// ["a,b"] and ["a","b"] must NOT collide — a comma join would conflate
	// them and serve wrong cached results.
	req1 := SearchRequest{Directory: "/tmp", AllowedFileTypes: []string{"a,b"}}
	req2 := SearchRequest{Directory: "/tmp", AllowedFileTypes: []string{"a", "b"}}

	if collectionCacheKey(req1) == collectionCacheKey(req2) {
		t.Error("different slice element splits must produce different cache keys")
	}
}

// ---------------------------------------------------------------------------
// computeCollectionFingerprint
// ---------------------------------------------------------------------------

func TestComputeCollectionFingerprintChangesOnEdit(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "a.txt")
	if err := os.WriteFile(path, []byte("hello"), 0644); err != nil {
		t.Fatal(err)
	}

	fp1 := computeCollectionFingerprint(dir)

	// Modify the file with different size so the fingerprint changes even on
	// coarse-modtime filesystems.
	if err := os.WriteFile(path, []byte("world and more content"), 0644); err != nil {
		t.Fatal(err)
	}

	fp2 := computeCollectionFingerprint(dir)
	if fp1 == fp2 {
		t.Error("fingerprint should change after file content modification")
	}
}

func TestComputeCollectionFingerprintStable(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "a.txt"), []byte("hello"), 0644); err != nil {
		t.Fatal(err)
	}

	fp1 := computeCollectionFingerprint(dir)
	fp2 := computeCollectionFingerprint(dir)
	if fp1 != fp2 {
		t.Error("fingerprint should be stable for unchanged directory")
	}
}

// ---------------------------------------------------------------------------
// collectionCache
// ---------------------------------------------------------------------------

func TestCollectionCacheMissPopulate(t *testing.T) {
	cc := newCollectionCache()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "a.txt"), []byte("hello"), 0644); err != nil {
		t.Fatal(err)
	}

	key := collectionCacheKey(SearchRequest{Directory: dir})
	fp := computeCollectionFingerprint(dir)

	// Cold miss.
	if _, ok := cc.get(key, fp); ok {
		t.Fatal("expected cache miss on first get")
	}

	// Populate.
	files := []fileMeta{{absPath: filepath.Join(dir, "a.txt"), size: 5}}
	cc.set(key, fp, files)

	// Warm hit.
	got, ok := cc.get(key, fp)
	if !ok {
		t.Fatal("expected cache hit after set")
	}
	if len(got) != 1 || got[0].absPath != files[0].absPath {
		t.Error("cache returned wrong data")
	}
}

func TestCollectionCacheEviction(t *testing.T) {
	cc := newCollectionCache()
	dir := t.TempDir()

	// Insert maxCollectionEntries + 1.
	for i := 0; i < maxCollectionEntries+1; i++ {
		key := collectionCacheKey(SearchRequest{Directory: dir, Extension: fmt.Sprintf(".ext%d", i)})
		cc.set(key, "fp", []fileMeta{{absPath: "dummy", size: 1}})
	}

	if len(cc.entries) > maxCollectionEntries {
		t.Errorf("expected at most %d entries after eviction, got %d", maxCollectionEntries, len(cc.entries))
	}
}

func TestCollectionCacheStaleFingerprint(t *testing.T) {
	cc := newCollectionCache()
	dir := t.TempDir()
	path := filepath.Join(dir, "a.txt")
	if err := os.WriteFile(path, []byte("hello"), 0644); err != nil {
		t.Fatal(err)
	}

	key := collectionCacheKey(SearchRequest{Directory: dir})
	fpOld := computeCollectionFingerprint(dir)
	cc.set(key, fpOld, []fileMeta{{absPath: path, size: 5}})

	// Modify file (different size) → fingerprint changes.
	if err := os.WriteFile(path, []byte("world and more"), 0644); err != nil {
		t.Fatal(err)
	}
	fpNew := computeCollectionFingerprint(dir)

	if _, ok := cc.get(key, fpOld); !ok {
		t.Error("old fingerprint should still match if cache entry stored with it")
	}
	if _, ok := cc.get(key, fpNew); ok {
		t.Error("cache should miss when fingerprint doesn't match entry")
	}
}

// ---------------------------------------------------------------------------
// Integration: collection cache in collectFilesToProcess
// ---------------------------------------------------------------------------

func TestCollectFilesToProcessCachedResult(t *testing.T) {
	app := NewApp()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "a.txt"), []byte("hello"), 0644); err != nil {
		t.Fatal(err)
	}

	req := SearchRequest{
		Directory:   dir,
		Query:       "hello",
		MaxFileSize: 10485760,
	}

	// First call: cold miss, populates cache.
	first, err := app.collectFilesToProcess(context.Background(), req, nil)
	if err != nil {
		t.Fatal(err)
	}

	// Second call: should hit cache, return identical.
	second, err := app.collectFilesToProcess(context.Background(), req, nil)
	if err != nil {
		t.Fatal(err)
	}

	if len(first) != len(second) {
		t.Fatalf("cached result length mismatch: %d vs %d", len(first), len(second))
	}
	for i := range first {
		if first[i].absPath != second[i].absPath {
			t.Errorf("item %d: path mismatch: %s vs %s", i, first[i].absPath, second[i].absPath)
		}
	}
}

func TestCollectFilesToProcessCacheBypassOnDifferentFilter(t *testing.T) {
	app := NewApp()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "a.go"), []byte("package main"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "a.txt"), []byte("hello"), 0644); err != nil {
		t.Fatal(err)
	}

	req1 := SearchRequest{
		Directory:        dir,
		Query:            "hello",
		MaxFileSize:      10485760,
		AllowedFileTypes: []string{".go"},
	}
	req2 := SearchRequest{
		Directory:        dir,
		Query:            "hello",
		MaxFileSize:      10485760,
		AllowedFileTypes: []string{".txt"},
	}

	// Different filter → different cache key → both populate.
	got1, err := app.collectFilesToProcess(context.Background(), req1, nil)
	if err != nil {
		t.Fatal(err)
	}
	got2, err := app.collectFilesToProcess(context.Background(), req2, nil)
	if err != nil {
		t.Fatal(err)
	}

	if len(got1) != 1 || len(got2) != 1 {
		t.Fatalf("expected 1 file each, got %d and %d", len(got1), len(got2))
	}
	if !strings.HasSuffix(got1[0].absPath, ".go") {
		t.Error("expected .go file for req1")
	}
	if !strings.HasSuffix(got2[0].absPath, ".txt") {
		t.Error("expected .txt file for req2")
	}
}
