package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// Sentinel errors for replace operations. Messages match the historical
// strings so errors.Is works without breaking text assertions.
var (
	ErrReplaceLiteralOnly = errors.New("replace is literal-only")
	ErrQueryRequired      = errors.New("query is required")
	ErrReplaceCancelled   = errors.New("replace cancelled")
)

// ---------------------------------------------------------------------------
// Find & Replace
//
// Literal-only replace across the lines a search matches. The dry-run
// (Apply=false) computes the exact new line content and writes nothing; the
// apply (Apply=true) writes that same content atomically. Preview == Apply:
// both derive from the same matched (file, line) pairs, so what the user
// previews is exactly what gets written.
//
// No backups by design — the user's VCS is the undo path.
// ---------------------------------------------------------------------------

// stagedFile carries a file whose lines have been matched and replaced in
// memory, ready for atomic write-out.
type stagedFile struct {
	path  string
	lines [][]byte // full file, split on "\n"; replaced lines swapped in
	mode  os.FileMode
}

// progressFunc reports phased replace progress; force bypasses throttling.
type progressFunc func(phase string, processed, total int, currentFile string, force bool)

// replaceJob bundles the inputs stageReplacements needs per run.
type replaceJob struct {
	files    []fileMeta
	req      *ReplaceRequest
	pattern  *regexp.Regexp
	progress progressFunc
}

// ReplaceInFiles replaces the query match on each searched line with a literal
// replacement string. Dry-run by default; Apply=true commits the changes.
func (a *App) ReplaceInFiles(req ReplaceRequest) (ReplaceResult, error) {
	if req.Search.UseRegex {
		return ReplaceResult{}, fmt.Errorf("%w; disable regex search to replace", ErrReplaceLiteralOnly)
	}
	if req.Search.FuzzySearch {
		return ReplaceResult{}, fmt.Errorf("%w; disable fuzzy search to replace", ErrReplaceLiteralOnly)
	}
	if req.Search.Query == "" {
		return ReplaceResult{}, ErrQueryRequired
	}

	validated, err := a.validateAndSetDefaults(req.Search)
	if err != nil {
		return ReplaceResult{}, fmt.Errorf("validate search: %w", err)
	}
	req.Search = validated

	a.logDebug("Starting replace operation", logrus.Fields{
		"queryLength":       len(req.Search.Query),
		"apply":             req.Apply,
		"replacementLength": len(req.Replacement),
	})

	pattern, err := a.compileSearchPattern(req.Search)
	if err != nil {
		return ReplaceResult{}, fmt.Errorf("compile pattern: %w", err)
	}

	ctx, cancel, cancelHandle := a.createSearchContext()
	defer func() {
		a.clearSearchCancel(cancelHandle)
		cancel()
	}()

	filesToProcess, err := a.collectReplaceFiles(ctx, req.Search, pattern)
	if err != nil {
		return ReplaceResult{}, fmt.Errorf("collect replace files: %w", err)
	}

	result := ReplaceResult{Files: []FileReplacement{}}
	var lastEmit time.Time
	emitProgress := progressFunc(func(phase string, processed, total int, currentFile string, force bool) {
		if !force && time.Since(lastEmit) <= progressEmitInterval {
			return
		}
		lastEmit = time.Now()
		a.safeEmitEvent("replace-progress", &ReplaceProgress{
			Phase:          phase,
			ProcessedFiles: processed,
			TotalFiles:     total,
			CurrentFile:    currentFile,
			FilesChanged:   result.FilesChanged,
			LinesChanged:   result.LinesChanged,
		})
	})

	if ctx.Err() != nil {
		a.logInfo("Replace cancelled during file collection", logrus.Fields{
			"queryLength": len(req.Search.Query),
		})
		emitProgress("cancelled", 0, 0, "", true)
		return ReplaceResult{}, fmt.Errorf("%w during file collection (no files written): %w", ErrReplaceCancelled, ctx.Err())
	}

	staged := a.stageReplacements(ctx, replaceJob{
		files:    filesToProcess,
		req:      &req,
		pattern:  pattern,
		progress: emitProgress,
	}, &result)
	result.FilesChanged = len(staged)

	if req.Apply {
		if err := a.applyReplacements(ctx, staged, &result, emitProgress); err != nil {
			return ReplaceResult{}, fmt.Errorf("apply replacements: %w", err)
		}
		a.logInfo("Replace applied", logrus.Fields{
			"filesChanged": result.FilesChanged,
			"linesChanged": result.LinesChanged,
		})
	}

	emitProgress("complete", result.FilesChanged, result.FilesChanged, "", true)
	sortFileReplacements(result.Files)
	return result, nil
}

// collectReplaceFiles gathers and dedupes files across all search directories.
// A per-directory collect error propagates (cancel-during-collection is
// checked separately by the caller via ctx.Err).
func (a *App) collectReplaceFiles(ctx context.Context, req SearchRequest, pattern *regexp.Regexp) ([]fileMeta, error) {
	// collectFilesToProcess aborts a cancelled walk with SkipAll, which yields
	// partial candidates and a nil error (file_collection.go:91). Without the
	// ctx check in ReplaceInFiles a cancel during collection would fall
	// through and report "no matches" — indistinguishable from a genuinely
	// empty result.
	return collectAcrossDirs(ctx, req,
		func(c context.Context, singleReq SearchRequest) ([]fileMeta, error) {
			return a.collectFilesToProcess(c, singleReq, pattern)
		},
		func(_ string, err error) ([]fileMeta, error) {
			return nil, err
		})
}

// stageReplacements matches each file and stages line replacements. Files
// are staged with a bounded worker pool (NumCPU): the old code was a serial
// ReadFile+Split loop while the search path used workers — on a 500-file
// replace the staging phase was the whole wall time.
func (a *App) stageReplacements(
	ctx context.Context,
	job replaceJob,
	result *ReplaceResult,
) []stagedFile {
	if len(job.files) == 0 {
		return nil
	}
	numWorkers := runtime.NumCPU()
	if numWorkers < 2 {
		numWorkers = 2
	}
	if numWorkers > len(job.files) {
		numWorkers = len(job.files)
	}
	type stagedOut struct {
		idx    int
		staged stagedFile
		diffs  []FileReplacement
		ok     bool
	}
	workChan := make(chan int, min(len(job.files), 2*numWorkers))
	outChan := make(chan stagedOut, min(len(job.files), 2*numWorkers))
	var wg sync.WaitGroup
	repl := []byte(job.req.Replacement)
	for range numWorkers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			defer func() {
				if r := recover(); r != nil {
					a.logError("Replace stage worker panicked", nil, logrus.Fields{"panic": r})
				}
			}()
			for {
				select {
				case <-ctx.Done():
					return
				case idx, ok := <-workChan:
					if !ok {
						return
					}
					meta := job.files[idx]
					st, diffs, ok := a.stageOneFile(meta, job, repl)
					select {
					case outChan <- stagedOut{idx: idx, staged: st, diffs: diffs, ok: ok}:
					case <-ctx.Done():
						return
					}
				}
			}
		}()
	}
	go func() {
		defer close(workChan)
		for i := range job.files {
			select {
			case <-ctx.Done():
				return
			case workChan <- i:
			}
		}
	}()
	go func() {
		wg.Wait()
		close(outChan)
	}()
	// Collect in index order so staged output is deterministic regardless
	// of worker completion order.
	byIdx := make(map[int]stagedOut, len(job.files))
	next := 0
	staged := make([]stagedFile, 0, len(job.files))
	flush := func() {
		for {
			o, ok := byIdx[next]
			if !ok {
				return
			}
			delete(byIdx, next)
			job.progress("staging", next+1, len(job.files), job.files[next].absPath, next == len(job.files)-1)
			if o.ok {
				result.Files = append(result.Files, o.diffs...)
				result.LinesChanged += len(o.diffs)
				staged = append(staged, o.staged)
			}
			next++
			if ctx.Err() != nil {
				return
			}
		}
	}
	for o := range outChan {
		if ctx.Err() != nil {
			a.logInfo("Replace cancelled during staging", logrus.Fields{
				"filesScanned": next,
				"totalFiles":   len(job.files),
			})
			job.progress("cancelled", next, len(job.files), "", true)
			// Drain to let workers exit, then return what we have.
			for range outChan {
			}
			break
		}
		byIdx[o.idx] = o
		flush()
	}
	flush()
	return staged
}

// stageOneFile stages a single file's replacements. Split out so the worker
// pool above stays readable.
func (a *App) stageOneFile(meta fileMeta, job replaceJob, repl []byte) (stagedFile, []FileReplacement, bool) {
	cleanPath, err := a.sanitizePath(meta.absPath)
	if err != nil {
		a.logWarn("Skipping replace on unsafe path", logrus.Fields{"path": meta.absPath, "error": err.Error()})
		return stagedFile{}, nil, false
	}
	info, err := os.Lstat(cleanPath)
	if err != nil {
		return stagedFile{}, nil, false // file vanished since collection
	}
	if info.Mode()&os.ModeSymlink != 0 {
		a.logWarn("Skipping replace on symlink", logrus.Fields{"path": cleanPath})
		return stagedFile{}, nil, false
	}

	content, err := os.ReadFile(cleanPath)
	if err != nil {
		a.logWarn("Skipping replace on unreadable file", logrus.Fields{"path": cleanPath, "error": err.Error()})
		return stagedFile{}, nil, false
	}

	lines := bytes.Split(content, []byte("\n"))
	fileDiffs := make([]FileReplacement, 0, 4)
	for j, line := range lines {
		// Single Find (not Match+ReplaceAllString): bytes path avoids the
		// string(line) round-trip for non-matching lines; matched lines
		// still need strings for the result payload.
		if !job.pattern.Match(line) {
			continue
		}
		oldLine := string(line)
		// ReplaceAllLiteral on bytes avoids re-scanning via string; the
		// result is compared as bytes first to skip no-op allocations.
		newBytes := job.pattern.ReplaceAllLiteral(line, repl)
		if bytes.Equal(newBytes, line) {
			continue // no-op replacement — never write
		}
		newLine := string(newBytes)
		fileDiffs = append(fileDiffs, FileReplacement{
			FilePath: cleanPath,
			LineNum:  j + 1,
			OldLine:  oldLine,
			NewLine:  newLine,
		})
		lines[j] = newBytes
	}

	if len(fileDiffs) == 0 {
		return stagedFile{}, nil, false
	}
	return stagedFile{path: cleanPath, lines: lines, mode: info.Mode().Perm()}, fileDiffs, true
}

// applyReplacements writes each changed file atomically.
func (a *App) applyReplacements(
	ctx context.Context,
	staged []stagedFile,
	result *ReplaceResult,
	emitProgress progressFunc,
) error {
	for i, sf := range staged {
		if ctx.Err() != nil {
			a.logWarn("Replace cancelled during write", logrus.Fields{
				"filesWritten": i,
				"totalFiles":   len(staged),
			})
			emitProgress("cancelled", i, len(staged), "", true)
			return fmt.Errorf("%w during write (%d/%d files written, no rollback): %w", ErrReplaceCancelled, i, len(staged), ctx.Err())
		}
		newContent := bytes.Join(sf.lines, []byte("\n"))
		if err := writeFileAtomic(sf.path, newContent, sf.mode); err != nil {
			return fmt.Errorf("failed to write %q (%d/%d files written before failure): %w", sf.path, i, len(staged), err)
		}
		emitProgress("writing", i+1, len(staged), sf.path, i == len(staged)-1)
	}
	return nil
}

// writeFileAtomic writes content to path via a temp file in the same
// directory + os.Rename, preserving the original file mode. Same-dir temp
// guarantees Rename is on one filesystem (atomic). VCS is the undo path; no
// .bak is written. On any error before rename, the temp file is removed.
func writeFileAtomic(path string, content []byte, mode os.FileMode) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".cs-replace-*")
	if err != nil {
		return fmt.Errorf("create temp file in %s: %w", dir, err)
	}
	tmpName := tmp.Name()
	cleanup := func() {
		if tmpName != "" {
			_ = os.Remove(tmpName)
		}
	}
	defer cleanup()

	if _, err := tmp.Write(content); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("write temp file for %s: %w", path, err)
	}
	// fsync before rename: without it, a crash/power loss between Write and
	// Rename can leave a zero-length or partially-written file at the target
	// path (the rename is durable, the data may not be).
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("sync temp file for %s: %w", path, err)
	}
	if err := tmp.Chmod(mode); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("chmod temp file for %s: %w", path, err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close temp file for %s: %w", path, err)
	}
	if err := os.Rename(tmpName, path); err != nil {
		return fmt.Errorf("rename temp file to %s: %w", path, err)
	}
	tmpName = "" // success — nothing to clean up
	return nil
}
