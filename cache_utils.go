package main

import (
	"sort"
	"time"
)

// ---------------------------------------------------------------------------
// Shared cache + slice helpers
//
// Extracted from symbol_index.go, collection_index.go, polling_server.go and
// symbols.go, which each carried a hand-rolled copy of the same shapes:
// copy-on-read cache returns, oldest-entry eviction, and FilePath/LineNum
// result sorting.
// ---------------------------------------------------------------------------

// copySlice returns a copy of s so callers can sort/append/filter without
// corrupting the cached backing array. A nil input yields nil (callers that
// need a non-nil empty slice for the frontend should normalize themselves).
func copySlice[T any](s []T) []T {
	out := make([]T, len(s))
	copy(out, s)
	return out
}

// evictOldestKey removes the entry with the oldest creation time when a NEW
// key is about to exceed max entries. Callers must hold the write lock and
// must have already checked `_, exists := entries[key]; !exists` —
// re-indexing an existing key overwrites in place and must not evict a live
// entry. createdAt extracts the entry timestamp (entries are pointers in both
// caches, hence the func param instead of an interface).
func evictOldestKey[K comparable, V any](entries map[K]V, max int, createdAt func(V) time.Time) {
	var oldestKey K
	var oldestTime time.Time
	first := true
	for k, v := range entries {
		if t := createdAt(v); first || t.Before(oldestTime) {
			oldestKey, oldestTime = k, t
			first = false
		}
	}
	if !first {
		delete(entries, oldestKey)
	}
}

// sortSearchResults orders results deterministically by file path then line
// number, regardless of worker completion order.
func sortSearchResults(results []SearchResult) {
	sort.Slice(results, func(i, j int) bool {
		if results[i].FilePath != results[j].FilePath {
			return results[i].FilePath < results[j].FilePath
		}
		return results[i].LineNum < results[j].LineNum
	})
}

// sortFileReplacements orders staged file replacements the same way.
func sortFileReplacements(files []FileReplacement) {
	sort.Slice(files, func(i, j int) bool {
		if files[i].FilePath != files[j].FilePath {
			return files[i].FilePath < files[j].FilePath
		}
		return files[i].LineNum < files[j].LineNum
	})
}
