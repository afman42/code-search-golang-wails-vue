package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
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
		return ReplaceResult{}, err
	}
	req.Search = validated

	a.logDebug("Starting replace operation", logrus.Fields{
		"queryLength":       len(req.Search.Query),
		"apply":             req.Apply,
		"replacementLength": len(req.Replacement),
	})

	pattern, err := a.compileSearchPattern(req.Search)
	if err != nil {
		return ReplaceResult{}, err
	}

	ctx, cancel, cancelHandle := a.createSearchContext()
	defer func() {
		a.clearSearchCancel(cancelHandle)
		cancel()
	}()

	filesToProcess, err := a.collectReplaceFiles(ctx, req.Search, pattern)
	if err != nil {
		return ReplaceResult{}, err
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
			return ReplaceResult{}, err
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

// stageReplacements matches each file and stages line replacements.
func (a *App) stageReplacements(
	ctx context.Context,
	job replaceJob,
	result *ReplaceResult,
) []stagedFile {
	staged := make([]stagedFile, 0, len(job.files))
	for i, meta := range job.files {
		if ctx.Err() != nil {
			a.logInfo("Replace cancelled during staging", logrus.Fields{
				"filesScanned": i,
				"totalFiles":   len(job.files),
			})
			job.progress("cancelled", i, len(job.files), "", true)
			return staged
		}
		job.progress("staging", i+1, len(job.files), meta.absPath, i == len(job.files)-1)

		cleanPath, err := a.sanitizePath(meta.absPath)
		if err != nil {
			a.logWarn("Skipping replace on unsafe path", logrus.Fields{"path": meta.absPath, "error": err.Error()})
			continue
		}
		info, err := os.Lstat(cleanPath)
		if err != nil {
			continue // file vanished since collection
		}
		if info.Mode()&os.ModeSymlink != 0 {
			a.logWarn("Skipping replace on symlink", logrus.Fields{"path": cleanPath})
			continue
		}

		content, err := os.ReadFile(cleanPath)
		if err != nil {
			a.logWarn("Skipping replace on unreadable file", logrus.Fields{"path": cleanPath, "error": err.Error()})
			continue
		}

		lines := bytes.Split(content, []byte("\n"))
		fileDiffs := []FileReplacement{}
		for j, line := range lines {
			if job.pattern.Match(line) {
				oldLine := string(line)
				newLine := job.pattern.ReplaceAllLiteralString(oldLine, job.req.Replacement)
				if newLine == oldLine {
					continue // no-op replacement — never write
				}
				fileDiffs = append(fileDiffs, FileReplacement{
					FilePath: cleanPath,
					LineNum:  j + 1,
					OldLine:  oldLine,
					NewLine:  newLine,
				})
				lines[j] = []byte(newLine)
			}
		}

		if len(fileDiffs) == 0 {
			continue
		}
		result.Files = append(result.Files, fileDiffs...)
		result.LinesChanged += len(fileDiffs)
		staged = append(staged, stagedFile{path: cleanPath, lines: lines, mode: info.Mode().Perm()})
	}
	return staged
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
