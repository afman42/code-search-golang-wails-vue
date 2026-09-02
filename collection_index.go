package main

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"hash"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

// ---------------------------------------------------------------------------
// Persistent file-collection cache
//
// collectFilesToProcess re-walks the directory tree and re-runs the binary
// probe on every search. For repeat searches against an unchanged directory
// this cache skips both: the walked file list is stored keyed by
// (directory, filter-set) and validated by a metadata fingerprint. When the
// fingerprint is unchanged the cached list is returned directly.
//
// The cache key includes the request's cheap-filter dimension (extension,
// allowed types, exclude patterns, size bounds, includeBinary,
// respectGitignore) so a cached entry is only ever reused for a request that
// would produce the identical collection. Typing a new query with unchanged
// filters — the dominant repeat case — is a hit. Changing a filter is a miss
// that re-walks and overwrites that key.
//
// Intentionally in-memory for the process lifetime (no on-disk persistence),
// mirroring symbol_index.go.
//
// ponytail: a hit still pays one fingerprint (metadata) walk to detect
// staleness, so the saved work is the binary-probe phase and re-filtering,
// not the directory traversal — marginal on pure known-text trees, real on
// mixed-extension trees. Upgrade: fsnotify-based invalidation to drop the
// per-search fingerprint walk, only if that walk becomes the measured
// bottleneck.
// ---------------------------------------------------------------------------

// collectionEntry holds the collected (already probed/filtered) file list for
// one directory + filter-set, valid while fingerprint is unchanged.
type collectionEntry struct {
	fingerprint string
	files       []fileMeta
	createdAt   time.Time
}

// maxCollectionEntries caps how many directory+filter combinations are cached
// simultaneously. Matches maxSymbolIndexEntries.
const maxCollectionEntries = 8

// maxCachedFiles bounds the size of a cached entry: trees larger than this
// fall through to the uncached path to bound memory.
const maxCachedFiles = 200_000

// collectionCache is a thread-safe LRU-ish cache of collected file lists.
type collectionCache struct {
	mu      sync.RWMutex
	entries map[string]*collectionEntry // key = directory + filter-set
}

func newCollectionCache() *collectionCache {
	return &collectionCache{entries: make(map[string]*collectionEntry)}
}

// joinLenPrefixed encodes strings so no two element sets collide: each
// element is prefixed with its length, so ["a,b"] and ["a","b"] produce
// different encodings. A plain comma join would conflate them and serve
// wrong cached results.
func joinLenPrefixed(parts []string) string {
	var sb strings.Builder
	for _, p := range parts {
		sb.WriteString(strconv.Itoa(len(p)))
		sb.WriteByte(':')
		sb.WriteString(p)
		sb.WriteByte(';')
	}
	return sb.String()
}

// collectionCacheKey folds the collection-relevant request fields into a
// canonical string so different filter sets never share an entry. Slices are
// sorted so order doesn't matter. Matching-affecting fields (query, regex,
// case, fuzzy, contextLines, maxResults) deliberately excluded — they don't
// change which files get collected.
func collectionCacheKey(req SearchRequest) string {
	allowed := append([]string(nil), req.AllowedFileTypes...)
	sort.Strings(allowed)
	excludes := append([]string(nil), req.ExcludePatterns...)
	sort.Strings(excludes)
	// Absolutize the directory so relative and absolute spellings of the
	// same tree share one cache entry instead of duplicating.
	dir := req.Directory
	if abs, err := filepath.Abs(dir); err == nil {
		dir = abs
	}
	dir = filepath.Clean(dir)
	return fmt.Sprintf("%s\x00%s\x00%s\x00%s\x00%d\x00%d\x00%t\x00%t",
		dir,
		req.Extension,
		joinLenPrefixed(allowed),
		joinLenPrefixed(excludes),
		req.MinFileSize,
		req.MaxFileSize,
		req.IncludeBinary,
		req.RespectGitignore,
	)
}

// computeCollectionFingerprint builds a deterministic hash of every file
// under `directory` (path + size + modtime), using the same always-on skip
// rules as the collection walk (hidden directories, symlinks), plus every
// ignore file that can change which files the collection keeps. Two calls
// return the same hash iff the file set and metadata are unchanged. Per-
// request filters are intentionally absent — they live in the cache key.
func computeCollectionFingerprint(directory string) string {
	h := sha1.New()

	// WalkDir visits entries in deterministic lexical order (os.ReadDir
	// sorts by name), so hashing in visit order needs no sort or fMeta
	// slice. Per-request filters are absent — they live in the cache key.
	_ = filepath.WalkDir(directory, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			if strings.HasPrefix(d.Name(), ".") {
				return filepath.SkipDir
			}
			return nil
		}
		if d.Type()&fs.ModeSymlink != 0 {
			// A symlinked .gitignore is still read (os.ReadFile follows the
			// link) by the nested ignore stack, so it has to be hashed even
			// though symlinks are otherwise not collected. Stat follows the
			// link and reports the target's metadata — one extra syscall,
			// only for the rare symlinked ignore file.
			if d.Name() == gitignoreFileName {
				if info, statErr := os.Stat(path); statErr == nil {
					hashPathMeta(h, path, info)
				}
			}
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return nil
		}
		// Regular .gitignore files need no special case here: hidden FILES
		// are not skipped (only hidden directories), so every nested
		// .gitignore the walk passes is already folded in by path + size +
		// modtime. Editing sub/pkg/.gitignore therefore invalidates the
		// cached collection even though the file set is unchanged, which it
		// must — the nested stack would produce a different result.
		hashPathMeta(h, path, info)
		return nil
	})

	// .git/info/exclude is the one ignore source the walk cannot reach: it
	// lives under the hidden .git directory, which is pruned above. Stat it
	// directly so editing it invalidates cached collections too.
	//
	// Note on collectionCacheKey: it already carries RespectGitignore, and
	// that remains sufficient. The flag only selects WHETHER ignore
	// filtering runs (so on/off never share an entry), while every change to
	// the ignore rules themselves lands in this fingerprint.
	exclude := filepath.Join(directory, ".git", "info", "exclude")
	if info, err := os.Stat(exclude); err == nil {
		hashPathMeta(h, exclude, info)
	}
	return hex.EncodeToString(h.Sum(nil))
}

// hashPathMeta folds one path's identity and metadata into the running hash.
// Single source of truth for the fingerprint's field order, so the walk and
// the ignore-file additions can never drift apart.
func hashPathMeta(h hash.Hash, path string, info fs.FileInfo) {
	h.Write([]byte(path))
	h.Write([]byte{0})
	h.Write([]byte(strconv.FormatInt(info.Size(), 10)))
	h.Write([]byte{0})
	h.Write([]byte(strconv.FormatInt(info.ModTime().UnixNano(), 10)))
	h.Write([]byte{0})
}

// get returns the cached file list for a key if the fingerprint matches.
// The returned slice is a copy so callers can append/sort without
// corrupting the cache entry. Copy cost is small vs walk+probe savings.
func (c *collectionCache) get(key, fingerprint string) ([]fileMeta, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	entry, ok := c.entries[key]
	if !ok || entry.fingerprint != fingerprint {
		return nil, false
	}
	out := make([]fileMeta, len(entry.files))
	copy(out, entry.files)
	return out, true
}

// set stores files for a key, evicting the oldest entry when the cache
// exceeds maxCollectionEntries (simple eviction, not true LRU — sufficient
// for a desktop app that searches a handful of directories/filter combos).
func (c *collectionCache) set(key, fingerprint string, files []fileMeta) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if _, exists := c.entries[key]; !exists && len(c.entries) >= maxCollectionEntries {
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

	c.entries[key] = &collectionEntry{
		fingerprint: fingerprint,
		files:       files,
		createdAt:   time.Now(),
	}
}
