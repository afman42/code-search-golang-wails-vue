package main

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"sync"

	"github.com/sirupsen/logrus"
)

// walkDirectoryTree walks the directory tree and returns two slices:
//
//   - textCandidates: files that passed all cheap filters (extension, size,
//     exclude patterns) and are either known-text extensions OR need a
//     binary probe. Each entry carries the absolute path and size so the
//     worker pool doesn't re-stat.
//
//   - binaryCheckCandidates: the subset of textCandidates that still need
//     the 512-byte binary probe (i.e. their extension is NOT in the
//     known-text set). Files with known-text extensions are in
//     textCandidates but NOT in binaryCheckCandidates, so the parallel
//     probe only opens files it actually needs to check.
//
// This two-phase split is the core of the collection optimizations:
//
//	Phase 1 (walkDirectoryTree): single-threaded directory walk that applies
//	cheap filters (extension, size, exclude) and skips known-text files past
//	the binary probe. No file I/O for known-text extensions — saves one
//	open+read+close syscall per known-text file.
//
//	Phase 2 (probeBinaryInParallel): worker pool that opens only the
//	unknown-extension files and runs the 512-byte binary check in parallel.
//	On a multi-core machine this turns N sequential open+read+close
//	operations into N/numWorkers parallel ones.
func (a *App) walkDirectoryTree(ctx context.Context, req SearchRequest, debug bool) (textCandidates []fileMeta, binaryCheckCandidates []fileMeta, stats collectStats, err error) {
	// Compute the absolute base directory and the current working directory
	// ONCE, before the walk starts. The previous implementation called
	// filepath.Abs(path) on EVERY file inside the WalkDir callback, which
	// does an os.Getwd() syscall (cached after the first call, but still
	// string work per call). For 2000 files that's 2000 redundant calls.
	//
	// filepath.WalkDir roots all paths at req.Directory:
	//   - If req.Directory is absolute, all path values are absolute.
	//   - If req.Directory is relative, all path values are relative to CWD.
	//
	// So we check once whether req.Directory is absolute and compute the
	// CWD once for the relative case. In the callback, resolving absPath
	// becomes a cheap filepath.Clean or filepath.Join — no per-file syscall.
	absBaseDir, err := filepath.Abs(req.Directory)
	if err != nil {
		return nil, nil, collectStats{}, err
	}
	absBaseDir = filepath.Clean(absBaseDir)
	// Resolve symlinks on the base dir so a symlink like /tmp/link -> /etc
	// doesn't let the prefix check be bypassed. If EvalSymlinks fails
	// (e.g. path doesn't exist yet in tests), keep the cleaned path.
	if resolved, evalErr := filepath.EvalSymlinks(absBaseDir); evalErr == nil {
		absBaseDir = resolved
	}
	dirIsAbs := filepath.IsAbs(req.Directory)

	// Only need the CWD if req.Directory is relative. Skip the os.Getwd
	// syscall entirely for the common case of an absolute search directory.
	var cwd string
	if !dirIsAbs {
		cwd, err = os.Getwd()
		if err != nil {
			return nil, nil, collectStats{}, err
		}
	}

	// The prefix used for the cheap traversal check (Opt 2). We append a
	// separator so that "/home/user/project" doesn't match
	// "/home/user/project-backup". The previous implementation called
	// filepath.Rel(baseDir, absPath) per file — a string allocation + path
	// computation per file just to check for "..". A prefix check with a
	// separator-terminated base is equivalent and allocation-free.
	prefixCheck := absBaseDir + string(filepath.Separator)

	// Nested .gitignore support, gated so RespectGitignore=false costs
	// nothing: with the flag off the stack is nil, no ignore file is opened
	// and no matcher is compiled. Rooted at req.Directory (not absBaseDir)
	// because the stack matches the walk's own path spelling, and even
	// constructed it reads nothing until a path is tested.
	var ignores *ignoreStack
	if req.RespectGitignore {
		ignores = newIgnoreStack(req.Directory)
	}

	wc := &walkCtx{
		req:         req,
		ignores:     ignores,
		debug:       debug,
		absBaseDir:  absBaseDir,
		prefixCheck: prefixCheck,
		dirIsAbs:    dirIsAbs,
		cwd:         cwd,
		ctx:         ctx,
		app:         a,
	}
	err = filepath.WalkDir(req.Directory, wc.handleEntry)
	textCandidates = wc.textOut
	binaryCheckCandidates = wc.binaryOut
	stats = wc.stats
	return textCandidates, binaryCheckCandidates, stats, err
}

// walkCtx carries the per-walk state that the WalkDir callback closes over.
// Bundling it into a struct keeps the closure variables explicit and the
// per-entry pipeline (dir handling → filter → classify) readable.
type walkCtx struct {
	req         SearchRequest
	ignores     *ignoreStack
	debug       bool
	absBaseDir  string
	prefixCheck string
	dirIsAbs    bool
	cwd         string
	ctx         context.Context
	app         *App

	textOut  []fileMeta
	binaryOut []fileMeta
	stats    collectStats
}

func (w *walkCtx) handleEntry(path string, d fs.DirEntry, walkErr error) error {
	if w.ctx.Err() != nil {
		return filepath.SkipAll
	}
	if walkErr != nil {
		if w.debug {
			w.app.logDebug("Skipping file/directory due to access error", logrus.Fields{
				"path":  path,
				"error": walkErr.Error(),
			})
		}
		return nil
	}
	if d.IsDir() {
		return w.handleDir(path, d)
	}
	return w.handleFile(path, d)
}

func (w *walkCtx) handleDir(path string, d fs.DirEntry) error {
	// Skip hidden directories that start with a dot (e.g., .git, .vscode)
	if strings.HasPrefix(d.Name(), ".") {
		if w.debug {
			w.app.logDebug("Skipping hidden directory", logrus.Fields{
				"directory": path,
			})
		}
		w.stats.dirsSkipped++
		return filepath.SkipDir
	}
	// Prune ignored directories instead of walking them and dropping their
	// files one by one — mirrors git, which never descends into an ignored
	// directory, and means a pruned subtree costs no ReadDir and no ignore-file reads.
	if w.ignores != nil && w.ignores.ignoresDir(path) {
		if w.debug {
			w.app.logDebug("Skipping gitignored directory", logrus.Fields{
				"directory": path,
			})
		}
		w.stats.dirsSkipped++
		return filepath.SkipDir
	}
	return nil
}

func (w *walkCtx) handleFile(path string, d fs.DirEntry) error {
	// --- Opt 1: Compute absPath without per-file filepath.Abs ---
	var absPath string
	if w.dirIsAbs || filepath.IsAbs(path) {
		absPath = filepath.Clean(path)
	} else {
		absPath = filepath.Join(w.cwd, path)
	}

	// --- Opt 2: Prefix check instead of filepath.Rel ---
	if absPath != w.absBaseDir && !strings.HasPrefix(absPath, w.prefixCheck) {
		if w.debug {
			w.app.logDebug("Skipping file due to path traversal detection", logrus.Fields{
				"path":    path,
				"absPath": absPath,
				"baseDir": w.absBaseDir,
			})
		}
		w.stats.filesSkipped++
		return nil
	}

	// --- File extension filter ---
	if w.req.Extension != "" {
		if !matchExtension(path, w.req.Extension) {
			if w.debug {
				w.app.logDebug("Skipping file due to extension filter", logrus.Fields{
					"path":      path,
					"extension": w.req.Extension,
				})
			}
			w.stats.filesSkipped++
			return nil
		}
	}

	// --- File type allow-list ---
	if len(w.req.AllowedFileTypes) > 0 {
		isAllowed := false
		for _, allowedExt := range w.req.AllowedFileTypes {
			if matchExtension(path, allowedExt) {
				isAllowed = true
				break
			}
		}
		if !isAllowed {
			if w.debug {
				w.app.logDebug("Skipping file due to allowed types filter", logrus.Fields{
					"path":         path,
					"allowedTypes": w.req.AllowedFileTypes,
				})
			}
			w.stats.filesSkipped++
			return nil
		}
	}

	// --- Symlink guard ---
	if d.Type()&fs.ModeSymlink != 0 {
		if w.debug {
			w.app.logDebug("Skipping symlink", logrus.Fields{"path": path})
		}
		w.stats.filesSkipped++
		return nil
	}

	// --- File size filters ---
	fileInfo, err := d.Info()
	if err != nil {
		if w.debug {
			w.app.logDebug("Skipping file due to info error", logrus.Fields{
				"path":  path,
				"error": err.Error(),
			})
		}
		return nil
	}
	if fileInfo.Size() > w.req.MaxFileSize {
		if w.debug {
			w.app.logDebug("Skipping large file due to size limit", logrus.Fields{
				"path":     path,
				"fileSize": fileInfo.Size(),
				"maxSize":  w.req.MaxFileSize,
			})
		}
		w.stats.filesSkipped++
		return nil
	}
	if fileInfo.Size() < w.req.MinFileSize {
		if w.debug {
			w.app.logDebug("Skipping small file due to size filter", logrus.Fields{
				"path":     path,
				"fileSize": fileInfo.Size(),
				"minSize":  w.req.MinFileSize,
			})
		}
		w.stats.filesSkipped++
		return nil
	}

	// --- Exclude patterns ---
	for _, patternStr := range w.req.ExcludePatterns {
		if patternStr != "" && w.app.matchesPattern(path, patternStr) {
			if w.debug {
				w.app.logDebug("Skipping file due to exclude pattern", logrus.Fields{
					"path":        path,
					"excludePath": patternStr,
				})
			}
			w.stats.filesSkipped++
			return nil
		}
	}

	// --- Nested .gitignore filter ---
	if w.ignores != nil && w.ignores.ignoresFile(path) {
		if w.debug {
			w.app.logDebug("Skipping gitignored file", logrus.Fields{
				"path": path,
			})
		}
		w.stats.filesSkipped++
		return nil
	}

	// --- Opt 3: Skip binary probe for known-text extensions ---
	meta := fileMeta{absPath: absPath, size: fileInfo.Size()}
	if w.req.IncludeBinary {
		w.textOut = append(w.textOut, meta)
		w.stats.filesCollected++
		return nil
	}
	if isKnownTextExtension(path) {
		w.textOut = append(w.textOut, meta)
		w.stats.filesCollected++
		return nil
	}
	// Unknown extension — defer the binary probe to the parallel worker pool.
	w.binaryOut = append(w.binaryOut, meta)
	return nil
}

// probeBinaryInParallel runs the 512-byte binary detection probe on each
// candidate file in parallel using a worker pool sized to the CPU count.
// Files that pass (are text) are appended to textCandidates; files that
// fail (are binary) are counted as skipped.
//
// This is Opt 4: the previous implementation ran the binary probe
// sequentially inside the WalkDir callback, so each file's open+read+close
// had to finish before the walk could advance to the next file. By
// separating the probe from the walk, we can run N/numWorkers probes in
// parallel, which on a multi-core machine is a linear speedup of the
// collection phase.
//
// The function respects context cancellation: if ctx is cancelled (e.g.
// the user cancelled the search), remaining probes are abandoned.
func (a *App) probeBinaryInParallel(ctx context.Context, candidates []fileMeta, debug bool) (textFiles []fileMeta, skipped int) {
	if len(candidates) == 0 {
		return nil, 0
	}

	// Default to a background context if the caller passed nil (e.g. in
	// tests). Without this, the select on ctx.Done() panics.
	if ctx == nil {
		ctx = context.Background()
	}

	numWorkers := runtime.NumCPU()
	if numWorkers < 2 {
		numWorkers = 2
	}
	if numWorkers > len(candidates) {
		numWorkers = len(candidates)
	}

	// Channel of work (file indices) and channel of results.
	type probeResult struct {
		meta   fileMeta
		isText bool
	}
	workChan := make(chan fileMeta, len(candidates))
	resultChan := make(chan probeResult, len(candidates))

	// Launch workers that pull from workChan and push to resultChan.
	var wg sync.WaitGroup
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			// Reuse a 512-byte buffer per worker from the pool. Each
			// worker keeps its borrowed buffer for the duration of its
			// lifetime, returning it to the pool only when the worker
			// exits. This avoids per-file allocation while keeping the
			// pool small (numWorkers buffers, not one per file).
			bufPtr := binaryCheckBufPool.Get().(*[]byte)
			defer binaryCheckBufPool.Put(bufPtr)
			buffer := (*bufPtr)[:cap(*bufPtr)]

			for {
				select {
				case <-ctx.Done():
					return
				case meta, ok := <-workChan:
					if !ok {
						return
					}
					isText := probeIsText(meta.absPath, buffer, debug, a)
					select {
					case resultChan <- probeResult{meta: meta, isText: isText}:
					case <-ctx.Done():
						return
					}
				}
			}
		}()
	}

	// Feed all candidates into the work channel in a separate goroutine.
	go func() {
		defer close(workChan)
		for _, meta := range candidates {
			select {
			case <-ctx.Done():
				return
			case workChan <- meta:
			}
		}
	}()

	// Close resultChan once all workers are done.
	go func() {
		wg.Wait()
		close(resultChan)
	}()

	// Collect results.
	textFiles = make([]fileMeta, 0, len(candidates))
	for res := range resultChan {
		if res.isText {
			textFiles = append(textFiles, res.meta)
		} else {
			skipped++
		}
	}

	return textFiles, skipped
}

// probeIsText opens the file, reads the first 512 bytes, and reports
// whether the content appears to be text. The buffer is borrowed from the
// caller (the per-worker buffer) to avoid allocation. If the file can't be
// opened or read, it's treated as non-text (skipped) — the safe default.
func probeIsText(path string, buffer []byte, debug bool, a *App) bool {
	file, err := os.Open(path)
	if err != nil {
		a.logWarn("Skipping file due to read error for binary check", logrus.Fields{
			"path":  path,
			"error": err.Error(),
		})
		return false
	}
	n, readErr := file.Read(buffer)
	closeErr := file.Close()
	if readErr != nil {
		// Read failure is NOT evidence the file is text: with n==0 the
		// content scan below would pass trivially and the file would be
		// searched. Treat it as skipped and say so instead of silently
		// misclassifying.
		a.logWarn("Skipping file due to read error for binary check", logrus.Fields{
			"path":  path,
			"error": readErr.Error(),
		})
		return false
	}
	_ = closeErr
	if n > 0 && a.isBinary(buffer[:n]) {
		if debug {
			a.logDebug("Skipping binary file", logrus.Fields{
				"path": path,
			})
		}
		return false
	}
	return true
}

// collectFilesToProcess walks the directory tree and collects all files to
// process based on search criteria. This is the public entry point used by
// SearchWithProgress.
//
// The collection is now two-phase for performance:
//
//  1. walkDirectoryTree — single-threaded walk that applies cheap filters
//     (extension, size, exclude patterns, nested .gitignore) and splits
//     files into:
//     - textCandidates: known-text extensions or IncludeBinary=true
//     - binaryCheckCandidates: unknown extensions needing a binary probe
//
//  2. probeBinaryInParallel — worker pool that runs the 512-byte binary
//     detection on binaryCheckCandidates in parallel. Files that pass are
//     added to the final list.
//
// On a 2000-file tree of .go/.ts files (all known-text), Phase 2 is empty
// and the walk is the only cost. On a mixed tree with unknown extensions,
// Phase 2 parallelizes the binary probes across CPU cores.
func (a *App) collectFilesToProcess(ctx context.Context, req SearchRequest, pattern *regexp.Regexp) ([]fileMeta, error) {
	debug := a.logger != nil && a.logger.IsLevelEnabled(logrus.DebugLevel)

	// Persistent collection cache: repeat searches with unchanged directory
	// and unchanged cheap filters skip the walk + binary probe entirely.
	// The cache is bypassed when a.collectionIndex is nil (unit tests that
	// construct App directly) — same pattern as globalSymbolIndex.
	cacheKey := ""
	fingerprint := ""
	if a.collectionIndex != nil {
		cacheKey = collectionCacheKey(req)
		fingerprint = computeCollectionFingerprint(req.Directory)
		if cached, ok := a.collectionIndex.get(cacheKey, fingerprint); ok {
			a.logDebug("Collection cache hit", logrus.Fields{
				"directory": req.Directory,
				"files":     len(cached),
			})
			return cached, nil
		}
	}

	textCandidates, binaryCandidates, stats, err := a.walkDirectoryTree(ctx, req, debug)
	if err != nil {
		a.logError("Error during file walk", err, logrus.Fields{
			"directory": req.Directory,
		})
		return nil, err
	}

	// Run the binary probe in parallel on the unknown-extension files.
	// Thread the search context through so a user cancel aborts remaining
	// probes too — without it, cancelling a search on a large tree leaves
	// the collection phase running to completion.
	var binarySkipped int
	var probedText []fileMeta
	if len(binaryCandidates) > 0 {
		probedText, binarySkipped = a.probeBinaryInParallel(ctx, binaryCandidates, debug)
		stats.filesSkipped += binarySkipped
	}

	// Merge: known-text candidates + probed-text files.
	allFiles := make([]fileMeta, 0, len(textCandidates)+len(probedText))
	allFiles = append(allFiles, textCandidates...)
	allFiles = append(allFiles, probedText...)
	stats.filesCollected = len(allFiles)

	// No .gitignore post-pass: the walk applies the nested ignore stack
	// inline (see walkDirectoryTree), which also prunes ignored directories
	// and keeps ignored files out of the binary-probe phase entirely.
	// Filtering here instead would re-probe files git never looks at.

	// Store the fully filtered result under the request's filter key, so a
	// later request with the same directory + filters is served from cache.
	// Trees over maxCachedFiles skip caching to bound memory.
	if a.collectionIndex != nil && len(allFiles) <= maxCachedFiles {
		a.collectionIndex.set(cacheKey, fingerprint, allFiles)
	}

	a.logInfo("File collection completed", logrus.Fields{
		"filesProcessed":     stats.filesCollected,
		"filesSkipped":       stats.filesSkipped,
		"dirsSkipped":        stats.dirsSkipped,
		"binaryProbesRun":    len(binaryCandidates),
		"binaryFilesSkipped": binarySkipped,
		"textExtShortlisted": len(textCandidates),
		"gitignoreFiltered":  req.RespectGitignore,
		"directory":          req.Directory,
	})

	return allFiles, nil
}
