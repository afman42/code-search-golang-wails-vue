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

// collectStats holds the counters gathered during the directory walk for
// logging at the end of collection. It's returned by walkDirectoryTree so
// the caller can log a single summary line without passing the App's logger
// deep into the walk.
type collectStats struct {
	filesCollected int
	filesSkipped   int
	dirsSkipped    int
}

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
func (a *App) walkDirectoryTree(req SearchRequest, debug bool) (textCandidates []fileMeta, binaryCheckCandidates []fileMeta, stats collectStats, err error) {
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

	err = filepath.WalkDir(req.Directory, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			if debug {
				a.logDebug("Skipping file/directory due to access error", logrus.Fields{
					"path":  path,
					"error": walkErr.Error(),
				})
			}
			return nil
		}

		// --- Directory handling (before the per-file optimization) ---
		if d.IsDir() {
			// Skip hidden directories that start with a dot (e.g., .git, .vscode)
			if strings.HasPrefix(d.Name(), ".") {
				if debug {
					a.logDebug("Skipping hidden directory", logrus.Fields{
						"directory": path,
					})
				}
				stats.dirsSkipped++
				return filepath.SkipDir
			}
			// If SearchSubdirs is false, skip all subdirectories beyond the root
			if !req.SearchSubdirs && path != req.Directory {
				stats.dirsSkipped++
				return filepath.SkipDir
			}
			return nil
		}

		// --- Opt 1: Compute absPath without per-file filepath.Abs ---
		// filepath.Abs(path) does an os.Getwd() syscall (cached after the
		// first call, but still string work per call). Since we already
		// know whether paths are absolute or relative (from req.Directory),
		// we can resolve absPath with a cheap Clean or Join against the
		// pre-computed CWD — no per-file syscall.
		var absPath string
		if dirIsAbs || filepath.IsAbs(path) {
			absPath = filepath.Clean(path)
		} else {
			// path is relative (rooted at req.Directory, which is also
			// relative to CWD). Join with CWD to get the absolute path.
			// We use Join (not Abs) because we already have the CWD.
			absPath = filepath.Join(cwd, path)
		}

		// --- Opt 2: Prefix check instead of filepath.Rel ---
		// The previous traversal check called filepath.Rel(baseDir, absPath)
		// per file, which allocates a string and does path arithmetic just
		// to detect ".." components. A prefix check against the
		// separator-terminated base directory is equivalent: if absPath
		// doesn't start with baseDir + separator, it's outside the search
		// scope (or trying to escape via symlinks). The only edge case is
		// absPath == absBaseDir itself (the root), which we allow.
		if absPath != absBaseDir && !strings.HasPrefix(absPath, prefixCheck) {
			if debug {
				a.logDebug("Skipping file due to path traversal detection", logrus.Fields{
					"path":    path,
					"absPath": absPath,
					"baseDir": absBaseDir,
				})
			}
			stats.filesSkipped++
			return nil
		}

		// --- File extension filter ---
		if req.Extension != "" {
			if !matchExtension(path, req.Extension) {
				if debug {
					a.logDebug("Skipping file due to extension filter", logrus.Fields{
						"path":      path,
						"extension": req.Extension,
					})
				}
				stats.filesSkipped++
				return nil
			}
		}

		// --- File type allow-list ---
		if len(req.AllowedFileTypes) > 0 {
			isAllowed := false
			for _, allowedExt := range req.AllowedFileTypes {
				if matchExtension(path, allowedExt) {
					isAllowed = true
					break
				}
			}
			if !isAllowed {
				if debug {
					a.logDebug("Skipping file due to allowed types filter", logrus.Fields{
						"path":         path,
						"allowedTypes": req.AllowedFileTypes,
					})
				}
				stats.filesSkipped++
				return nil
			}
		}

		// --- File size filters ---
		fileInfo, err := d.Info()
		if err != nil {
			if debug {
				a.logDebug("Skipping file due to info error", logrus.Fields{
					"path":  path,
					"error": err.Error(),
				})
			}
			return nil // Skip if we can't get file info
		}

		if fileInfo.Size() > req.MaxFileSize {
			if debug {
				a.logDebug("Skipping large file due to size limit", logrus.Fields{
					"path":     path,
					"fileSize": fileInfo.Size(),
					"maxSize":  req.MaxFileSize,
				})
			}
			stats.filesSkipped++
			return nil
		}

		if fileInfo.Size() < req.MinFileSize {
			if debug {
				a.logDebug("Skipping small file due to size filter", logrus.Fields{
					"path":     path,
					"fileSize": fileInfo.Size(),
					"minSize":  req.MinFileSize,
				})
			}
			stats.filesSkipped++
			return nil
		}

		// --- Exclude patterns ---
		for _, patternStr := range req.ExcludePatterns {
			if patternStr != "" && a.matchesPattern(path, patternStr) {
				if debug {
					a.logDebug("Skipping file due to exclude pattern", logrus.Fields{
						"path":        path,
						"excludePath": patternStr,
					})
				}
				stats.filesSkipped++
				return nil
			}
		}

		// --- Opt 3: Skip binary probe for known-text extensions ---
		// If the file has a known-text extension (.go, .ts, .py, .md, etc.),
		// it is NEVER binary, so we skip the open+read+close syscall
		// entirely. The file goes straight into textCandidates without
		// entering binaryCheckCandidates. On a tree of 2000 .go files this
		// saves 2000 syscalls.
		//
		// Unknown extensions (e.g. .dat, .bin, no extension) still get the
		// binary probe — the safe default.
		meta := fileMeta{absPath: absPath, size: fileInfo.Size()}

		if req.IncludeBinary {
			// User explicitly wants binary files searched — no probe needed.
			textCandidates = append(textCandidates, meta)
			stats.filesCollected++
			return nil
		}

		if isKnownTextExtension(path) {
			// Known text extension — skip the binary probe entirely.
			textCandidates = append(textCandidates, meta)
			stats.filesCollected++
			return nil
		}

		// Unknown extension — needs the binary probe. Defer the probe to
		// the parallel worker pool (Opt 4) by adding to
		// binaryCheckCandidates. The file is NOT in textCandidates yet;
		// it will be added there only if the probe says it's text.
		binaryCheckCandidates = append(binaryCheckCandidates, meta)
		return nil
	})

	return textCandidates, binaryCheckCandidates, stats, err
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
		if debug {
			a.logDebug("Skipping file due to read error for binary check", logrus.Fields{
				"path":  path,
				"error": err.Error(),
			})
		}
		return false
	}
	n, _ := file.Read(buffer)
	file.Close()
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
//     (extension, size, exclude patterns) and splits files into:
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
func (a *App) collectFilesToProcess(req SearchRequest, pattern *regexp.Regexp, baseDir string) ([]fileMeta, error) {
	debug := a.logger != nil && a.logger.IsLevelEnabled(logrus.DebugLevel)

	textCandidates, binaryCandidates, stats, err := a.walkDirectoryTree(req, debug)
	if err != nil {
		a.logError("Error during file walk", err, logrus.Fields{
			"directory": req.Directory,
		})
		return nil, err
	}

	// Run the binary probe in parallel on the unknown-extension files.
	// Use a background context so the probe completes even if the search
	// is cancelled mid-collection (the results are cheap and the cancel
	// will be checked by the search workers anyway).
	var binarySkipped int
	var probedText []fileMeta
	if len(binaryCandidates) > 0 {
		probedText, binarySkipped = a.probeBinaryInParallel(context.Background(), binaryCandidates, debug)
		stats.filesSkipped += binarySkipped
	}

	// Merge: known-text candidates + probed-text files.
	allFiles := make([]fileMeta, 0, len(textCandidates)+len(probedText))
	allFiles = append(allFiles, textCandidates...)
	allFiles = append(allFiles, probedText...)
	stats.filesCollected = len(allFiles)

	a.logInfo("File collection completed", logrus.Fields{
		"filesProcessed":      stats.filesCollected,
		"filesSkipped":        stats.filesSkipped,
		"dirsSkipped":         stats.dirsSkipped,
		"binaryProbesRun":     len(binaryCandidates),
		"binaryFilesSkipped":  binarySkipped,
		"textExtShortlisted":  len(textCandidates),
		"directory":           req.Directory,
	})

	return allFiles, nil
}
