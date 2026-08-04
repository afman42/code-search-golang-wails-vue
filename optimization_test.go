package main

import (
	"fmt"
	"regexp"
	"sync"
	"testing"
)

// cacheAndCompile compiles query via compileRegex and stores it in the cache
// under the key for a regex search (useRegex=true). Returns the stored pointer.
func cacheAndCompile(cache *LRUPatternCache, query string, caseSensitive bool) (*regexp.Regexp, error) {
	key := getPatternCacheKey(true, caseSensitive, query)
	re, err := compileRegex(query, caseSensitive)
	if err != nil {
		return nil, err
	}
	cache.Set(key, re)
	return re, nil
}

// TestLRUPatternCacheReuse verifies that compiling the same regex twice returns
// a cached instance (same pointer) rather than recompiling.
func TestLRUPatternCacheReuse(t *testing.T) {
	cache := NewLRUPatternCache(100)

	query := "test-pattern-unique-12345"

	first, err := cacheAndCompile(cache, query, false)
	if err != nil {
		t.Fatalf("First compile failed: %v", err)
	}

	second, ok := cache.Get(getPatternCacheKey(true, false, query))
	if !ok {
		t.Fatal("Expected cache hit after first compile, got miss")
	}

	// Same query should return the SAME compiled pointer (cache hit)
	if first != second {
		t.Error("Expected cached pattern to be reused, got different instances")
	}
}

// TestLRUPatternCacheCaseSensitivityIsolation ensures case-sensitive and
// case-insensitive variants of the same query are cached separately.
func TestLRUPatternCacheCaseSensitivityIsolation(t *testing.T) {
	cache := NewLRUPatternCache(100)

	query := "CaseTest"
	sensitiveKey := getPatternCacheKey(true, true, query)
	insensitiveKey := getPatternCacheKey(true, false, query)

	sensitive, err := cacheAndCompile(cache, query, true)
	if err != nil {
		t.Fatalf("Case-sensitive compile failed: %v", err)
	}
	insensitive, err := cacheAndCompile(cache, query, false)
	if err != nil {
		t.Fatalf("Case-insensitive compile failed: %v", err)
	}

	// Different case sensitivity => different cache entries => different pointers
	if sensitive == insensitive {
		t.Error("Case-sensitive and case-insensitive patterns should be distinct")
	}

	// Verify actual behavior differs
	if !insensitive.MatchString("casetest") {
		t.Error("Case-insensitive pattern should match lowercase")
	}
	if sensitive.MatchString("casetest") {
		t.Error("Case-sensitive pattern should NOT match lowercase")
	}

	// Both entries must be individually retrievable by their own keys.
	if _, ok := cache.Get(sensitiveKey); !ok {
		t.Error("Expected case-sensitive entry to be cached")
	}
	if _, ok := cache.Get(insensitiveKey); !ok {
		t.Error("Expected case-insensitive entry to be cached")
	}
}

// TestLRUPatternCacheEviction verifies the cache evicts entries once it grows
// past capacity, preventing unbounded memory growth.
func TestLRUPatternCacheEviction(t *testing.T) {
	cache := NewLRUPatternCache(10) // small cap to force eviction quickly

	// Insert 15 unique patterns to exceed capacity.
	for i := 0; i < 15; i++ {
		query := fmt.Sprintf("eviction-test-pattern-%d", i)
		if _, err := cacheAndCompile(cache, query, false); err != nil {
			t.Fatalf("Compile %d failed: %v", i, err)
		}
	}

	if int64(len(cache.cache)) != 10 {
		t.Errorf("Expected cache to hold exactly maxSize entries, got %d", len(cache.cache))
	}

	// The oldest entry should have been evicted, the newest retained.
	if _, ok := cache.Get(getPatternCacheKey(true, false, "eviction-test-pattern-0")); ok {
		t.Error("Expected oldest entry to be evicted once at capacity")
	}
	if _, ok := cache.Get(getPatternCacheKey(true, false, "eviction-test-pattern-14")); !ok {
		t.Error("Expected newest entry to remain in cache")
	}
}

// TestLRUPatternCacheLRUOrder verifies the LRU ordering: re-touching an entry
// keeps it alive while untouched entries get evicted first.
func TestLRUPatternCacheLRUOrder(t *testing.T) {
	cache := NewLRUPatternCache(2)

	keyA := getPatternCacheKey(true, false, "aaa")
	keyB := getPatternCacheKey(true, false, "bbb")
	for _, q := range []string{"aaa", "bbb"} {
		if _, err := cacheAndCompile(cache, q, false); err != nil {
			t.Fatal(err)
		}
	}

	// Touch A so B becomes the least-recently-used entry.
	if _, ok := cache.Get(keyA); !ok {
		t.Fatal("Expected A to be cached")
	}

	// Insert C, which evicts B (LRU), not A.
	if _, err := cacheAndCompile(cache, "ccc", false); err != nil {
		t.Fatal(err)
	}

	if _, ok := cache.Get(keyA); !ok {
		t.Error("Expected A to remain (it was recently used)")
	}
	if _, ok := cache.Get(keyB); ok {
		t.Error("Expected B to be evicted (least recently used)")
	}
}

// TestLRUPatternCacheConcurrency ensures the cache is safe under concurrent access.
func TestLRUPatternCacheConcurrency(t *testing.T) {
	cache := NewLRUPatternCache(100)

	var wg sync.WaitGroup
	errChan := make(chan error, 50)

	// 50 goroutines compiling overlapping patterns concurrently
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			query := fmt.Sprintf("concurrent-pattern-%d", id%10) // 10 unique, high contention
			if _, err := cacheAndCompile(cache, query, false); err != nil {
				errChan <- err
			}
		}(i)
	}

	wg.Wait()
	close(errChan)

	for err := range errChan {
		t.Errorf("Concurrent compile error: %v", err)
	}
}

// TestCompileRegexFlagCorrectness verifies compileRegex produces valid patterns
// with correct case sensitivity (regression test for the flag-prefix bug).
func TestCompileRegexFlagCorrectness(t *testing.T) {
	tests := []struct {
		name          string
		query         string
		caseSensitive bool
		testInput     string
		shouldMatch   bool
	}{
		{"case-insensitive matches uppercase", "hello", false, "HELLO", true},
		{"case-insensitive matches lowercase", "HELLO", false, "hello", true},
		{"case-sensitive rejects wrong case", "hello", true, "HELLO", false},
		{"case-sensitive matches exact", "hello", true, "hello", true},
		{"regex metacharacters work", "test.*end", false, "test middle end", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			re, err := compileRegex(tt.query, tt.caseSensitive)
			if err != nil {
				t.Fatalf("compileRegex failed: %v", err)
			}

			got := re.MatchString(tt.testInput)
			if got != tt.shouldMatch {
				t.Errorf("MatchString(%q) = %v, want %v", tt.testInput, got, tt.shouldMatch)
			}
		})
	}
}

// BenchmarkLRUPatternCacheHit measures the speedup from cache hits vs cold compile.
func BenchmarkLRUPatternCacheHit(b *testing.B) {
	cache := NewLRUPatternCache(100)
	query := "benchmark-pattern"
	if _, err := cacheAndCompile(cache, query, false); err != nil {
		b.Fatal(err)
	}
	key := getPatternCacheKey(true, false, query)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cache.Get(key)
	}
}

// BenchmarkLRUPatternCacheMiss measures the cost of filling the cache (Set).
func BenchmarkLRUPatternCacheMiss(b *testing.B) {
	cache := NewLRUPatternCache(100)
	for i := 0; i < b.N; i++ {
		query := fmt.Sprintf("miss-pattern-%d", i)
		if _, err := cacheAndCompile(cache, query, false); err != nil {
			b.Fatal(err)
		}
	}
}
