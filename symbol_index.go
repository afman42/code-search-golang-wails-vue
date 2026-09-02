package main

import (
	"crypto/sha1"
	"encoding/hex"
	"io/fs"
	"path/filepath"
	"sort"
	"strconv"
	"sync"
	"time"
)

// ---------------------------------------------------------------------------
// Persistent symbol index
//
// GetAllSymbols walks the entire source tree on every call — which for
// SearchSymbols means a full rescan per keystroke. This cache stores the
// extracted symbols keyed by a directory fingerprint (path + size + modtime
// of every supported source file). When the fingerprint is unchanged the
// cached symbols are returned instantly; when files change only the diff is
// rescanned.
//
// The cache lives in-memory for the process lifetime and is guarded by a
// mutex. It is intentionally simple (no on-disk persistence) because the
// fingerprint walk is cheap relative to symbol extraction.
// ---------------------------------------------------------------------------

// symbolIndexEntry holds the cached symbols for one directory.
type symbolIndexEntry struct {
	fingerprint string
	symbols     []SymbolInfo
	createdAt   time.Time
}

// maxSymbolIndexEntries caps how many directories are cached simultaneously.
const maxSymbolIndexEntries = 8

// symbolIndexCache is a thread-safe LRU-ish cache of extracted symbols per
// directory. It is stored on the App struct so it survives across Wails
// binding calls.
type symbolIndexCache struct {
	mu      sync.RWMutex
	entries map[string]*symbolIndexEntry
}

func newSymbolIndexCache() *symbolIndexCache {
	return &symbolIndexCache{entries: make(map[string]*symbolIndexEntry)}
}

// computeDirectoryFingerprint builds a deterministic hash of all supported
// source files under `directory` (path + size + modtime). Two calls return
// the same hash iff the set of source files and their metadata are unchanged.
//
// It deliberately walks the caller's RAW path while the cache key is
// normalized (symbolCacheKey): the fingerprint is content-derived (staleness),
// the key is identity-derived (which tree). Since the hash includes the walked
// path strings it inherits the caller's spelling, so an alternate spelling of
// the same tree is a miss that re-indexes and overwrites the one normalized
// entry — never a second entry eating the slot budget. That is the intended
// behavior; do not "fix" it by absolutizing the walk root here.
func computeDirectoryFingerprint(directory string) string {
	type fingerprintFile struct {
		path    string
		size    int64
		modTime int64
	}

	var files []fingerprintFile
	_ = filepath.WalkDir(directory, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			if shouldSkipDirForSymbolScan(d) {
				return filepath.SkipDir
			}
			return nil
		}
		if !isSymbolSupportedExtension(path) {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return nil
		}
		files = append(files, fingerprintFile{
			path:    path,
			size:    info.Size(),
			modTime: info.ModTime().UnixNano(),
		})
		return nil
	})

	// Sort by path for determinism.
	sort.Slice(files, func(i, j int) bool { return files[i].path < files[j].path })

	h := sha1.New()
	for _, f := range files {
		h.Write([]byte(f.path))
		h.Write([]byte{0})
		h.Write([]byte(strconv.FormatInt(f.size, 10)))
		h.Write([]byte{0})
		h.Write([]byte(strconv.FormatInt(f.modTime, 10)))
		h.Write([]byte{0})
	}
	return hex.EncodeToString(h.Sum(nil))
}

// symbolCacheKey canonicalizes a directory into the cache key. Without it
// `/a/b` and `./b` (from `/a`) are two keys for one tree, so a handful of
// relative/absolute spellings burn the whole maxSymbolIndexEntries budget and
// evict live entries. Applied inside get/set (never at the callsites) so no
// caller can bypass it. Mirrors collectionCacheKey in collection_index.go,
// including leaving the raw path in place when Abs fails (no cwd available) —
// a stable-but-unnormalized key still works, it just may duplicate.
func symbolCacheKey(directory string) string {
	if abs, err := filepath.Abs(directory); err == nil {
		directory = abs
	}
	return filepath.Clean(directory)
}

// get returns the cached symbols for a directory if the fingerprint matches.
// Returned slice is a copy so callers can sort/filter without corrupting cache.
func (c *symbolIndexCache) get(directory, fingerprint string) ([]SymbolInfo, bool) {
	key := symbolCacheKey(directory)
	c.mu.RLock()
	defer c.mu.RUnlock()
	entry, ok := c.entries[key]
	if !ok || entry.fingerprint != fingerprint {
		return nil, false
	}
	out := make([]SymbolInfo, len(entry.symbols))
	copy(out, entry.symbols)
	return out, true
}

// set stores symbols for a directory, evicting the oldest entry when the
// cache exceeds maxSymbolIndexEntries.
func (c *symbolIndexCache) set(directory, fingerprint string, symbols []SymbolInfo) {
	key := symbolCacheKey(directory)
	c.mu.Lock()
	defer c.mu.Unlock()

	// Only evict when inserting a NEW directory key; re-indexing an existing
	// directory just overwrites its entry and must not evict a live one.
	if _, exists := c.entries[key]; !exists && len(c.entries) >= maxSymbolIndexEntries {
		// Evict the oldest entry (simple eviction — not true LRU, but
		// sufficient for a desktop app where users rarely switch between
		// more than a handful of directories).
		var oldestKey string
		var oldestTime time.Time
		for k, v := range c.entries {
			if oldestKey == "" || v.createdAt.Before(oldestTime) {
				oldestKey = k
				oldestTime = v.createdAt
			}
		}
		delete(c.entries, oldestKey)
	}

	c.entries[key] = &symbolIndexEntry{
		fingerprint: fingerprint,
		symbols:     symbols,
		createdAt:   time.Now(),
	}
}

// ClearSymbolCache removes all cached symbol indices. Exposed as a Wails
// binding so the frontend can force a re-index after large changes.
func (a *App) ClearSymbolCache() {
	if a.symbolIndex != nil {
		a.symbolIndex.mu.Lock()
		a.symbolIndex.entries = make(map[string]*symbolIndexEntry)
		a.symbolIndex.mu.Unlock()
	}
}

// globalSymbolIndex is set by the App binding methods (app_symbols.go) so
// the standalone GetAllSymbolsWithProgress function can access the cache
// without needing an App receiver. In unit tests it stays nil (no caching).
var globalSymbolIndex *symbolIndexCache
