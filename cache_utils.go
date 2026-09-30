package main

import (
	"slices"
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
	// slices.SortFunc: inlines the comparator (no closure→interface call per
	// compare like sort.Slice) and performs fewer moves on sorted runs.
	slices.SortFunc(results, func(a, b SearchResult) int {
		if a.FilePath != b.FilePath {
			if a.FilePath < b.FilePath {
				return -1
			}
			return 1
		}
		if a.LineNum < b.LineNum {
			return -1
		}
		if a.LineNum > b.LineNum {
			return 1
		}
		return 0
	})
}

// sortFileReplacements orders staged file replacements the same way.
func sortFileReplacements(files []FileReplacement) {
	slices.SortFunc(files, func(a, b FileReplacement) int {
		if a.FilePath != b.FilePath {
			if a.FilePath < b.FilePath {
				return -1
			}
			return 1
		}
		if a.LineNum < b.LineNum {
			return -1
		}
		if a.LineNum > b.LineNum {
			return 1
		}
		return 0
	})
}
