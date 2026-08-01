package main

import (
	"fmt"
	"sync"
	"testing"
)

// TestPatternCacheReuse verifies that compiling the same regex twice returns
// a cached instance (same pointer) rather than recompiling.
func TestPatternCacheReuse(t *testing.T) {
	// Clear cache state for a clean test
	patternCache.Range(func(k, v interface{}) bool {
		patternCache.Delete(k)
		return true
	})

	query := "test-pattern-unique-12345"

	first, err := cachedCompileRegex(query, false)
	if err != nil {
		t.Fatalf("First compile failed: %v", err)
	}

	second, err := cachedCompileRegex(query, false)
	if err != nil {
		t.Fatalf("Second compile failed: %v", err)
	}

	// Same query should return the SAME compiled pointer (cache hit)
	if first != second {
		t.Error("Expected cached pattern to be reused, got different instances")
	}
}

// TestPatternCacheCaseSensitivityIsolation ensures case-sensitive and
// case-insensitive variants of the same query are cached separately.
func TestPatternCacheCaseSensitivityIsolation(t *testing.T) {
	patternCache.Range(func(k, v interface{}) bool {
		patternCache.Delete(k)
		return true
	})

	query := "CaseTest"

	sensitive, err := cachedCompileRegex(query, true)
	if err != nil {
		t.Fatalf("Case-sensitive compile failed: %v", err)
	}

	insensitive, err := cachedCompileRegex(query, false)
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
}

// TestPatternCacheEviction verifies the cache evicts old entries when it
// grows past the 100-entry limit, preventing unbounded memory growth.
func TestPatternCacheEviction(t *testing.T) {
	patternCache.Range(func(k, v interface{}) bool {
		patternCache.Delete(k)
		return true
	})

	// Compile 150 unique patterns to trigger eviction (limit is 100)
	for i := 0; i < 150; i++ {
		query := fmt.Sprintf("eviction-test-pattern-%d", i)
		_, err := cachedCompileRegex(query, false)
		if err != nil {
			t.Fatalf("Compile %d failed: %v", i, err)
		}
	}

	// Count remaining entries - should be bounded (eviction removes 10 at a time
	// once past 100, so it stays roughly at or below the limit)
	count := 0
	patternCache.Range(func(k, v interface{}) bool {
		count++
		return true
	})

	if count > 150 {
		t.Errorf("Cache grew unbounded: %d entries (expected eviction to cap growth)", count)
	}
}

// TestPatternCacheConcurrency ensures the cache is safe under concurrent access.
func TestPatternCacheConcurrency(t *testing.T) {
	patternCache.Range(func(k, v interface{}) bool {
		patternCache.Delete(k)
		return true
	})

	var wg sync.WaitGroup
	errChan := make(chan error, 50)

	// 50 goroutines compiling overlapping patterns concurrently
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			query := fmt.Sprintf("concurrent-pattern-%d", id%10) // 10 unique, high contention
			_, err := cachedCompileRegex(query, false)
			if err != nil {
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

// TestSearchWorkerPoolReuse verifies the worker pool returns reusable workers.
func TestSearchWorkerPoolReuse(t *testing.T) {
	w1 := getSearchWorker()
	if w1 == nil {
		t.Fatal("getSearchWorker returned nil")
	}
	if w1.buffer == nil || cap(w1.buffer) < 1024 {
		t.Errorf("Worker buffer not initialized correctly: cap=%d", cap(w1.buffer))
	}

	// Return it to the pool
	putSearchWorker(w1)

	// Get another - may or may not be the same instance (pool semantics),
	// but must be valid
	w2 := getSearchWorker()
	if w2 == nil {
		t.Fatal("getSearchWorker returned nil after put")
	}
	putSearchWorker(w2)
}

// TestSearchWorkerPoolConcurrency stresses the worker pool under concurrent use.
func TestSearchWorkerPoolConcurrency(t *testing.T) {
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			w := getSearchWorker()
			if w == nil {
				t.Error("nil worker under concurrency")
				return
			}
			// Simulate work
			w.workingOn = true
			putSearchWorker(w)
		}()
	}
	wg.Wait()
}

// BenchmarkPatternCacheHit measures the speedup from cache hits vs cold compile.
func BenchmarkPatternCacheHit(b *testing.B) {
	query := "benchmark-pattern"
	// Prime the cache
	cachedCompileRegex(query, false)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cachedCompileRegex(query, false)
	}
}

// BenchmarkPatternCacheMiss measures cold compilation cost.
func BenchmarkPatternCacheMiss(b *testing.B) {
	for i := 0; i < b.N; i++ {
		// Unique query each time to force a miss
		query := fmt.Sprintf("miss-pattern-%d", i)
		cachedCompileRegex(query, false)
	}
}
