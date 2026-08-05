package main

import (
	"crypto/sha1"
	"encoding/hex"
	"io/fs"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
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
func computeDirectoryFingerprint(directory string) string {
	extensions := []string{".go", ".ts", ".tsx", ".js", ".vue"}
	isSupported := func(path string) bool {
		ext := strings.ToLower(filepath.Ext(path))
		for _, e := range extensions {
			if ext == e {
				return true
			}
		}
		return false
	}

	type fileMeta struct {
		path    string
		size    int64
		modTime int64
	}

	var files []fileMeta
	_ = filepath.WalkDir(directory, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			name := strings.ToLower(d.Name())
			if name == "node_modules" || name == ".git" || name == "vendor" ||
				name == "build" || name == "dist" || name == "bin" {
				return filepath.SkipDir
			}
			return nil
		}
		if !isSupported(path) {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return nil
		}
		files = append(files, fileMeta{
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
		h.Write([]byte(strconv.FormatInt(f.modTime, 10)))
	}
	return hex.EncodeToString(h.Sum(nil))
}

// get returns the cached symbols for a directory if the fingerprint matches.
func (c *symbolIndexCache) get(directory, fingerprint string) ([]SymbolInfo, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	entry, ok := c.entries[directory]
	if !ok || entry.fingerprint != fingerprint {
		return nil, false
	}
	return entry.symbols, true
}

// set stores symbols for a directory, evicting the oldest entry when the
// cache exceeds maxSymbolIndexEntries.
func (c *symbolIndexCache) set(directory, fingerprint string, symbols []SymbolInfo) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if len(c.entries) >= maxSymbolIndexEntries {
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

	c.entries[directory] = &symbolIndexEntry{
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
