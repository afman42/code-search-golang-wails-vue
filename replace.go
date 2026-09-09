package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"time"

	"github.com/sirupsen/logrus"
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
// ReplaceInFiles replaces the query match on each searched line with a literal
// replacement string. Dry-run by default; Apply=true commits the changes.
func (a *App) ReplaceInFiles(req ReplaceRequest) (ReplaceResult, error) {
	if req.Search.UseRegex {
		return ReplaceResult{}, errors.New("replace is literal-only; disable regex search to replace")
	}
	if req.Search.FuzzySearch {
		return ReplaceResult{}, errors.New("replace is literal-only; disable fuzzy search to replace")
	}
	if req.Search.Query == "" {
		return ReplaceResult{}, errors.New("query is required")
	}

	validated, err := a.validateAndSetDefaults(req.Search)
	if err != nil {
		return ReplaceResult{}, err
	}
	req.Search = validated

	a.logInfo("Starting replace operation", logrus.Fields{
		"directory":   req.Search.Directory,
		"query":       req.Search.Query,
		"apply":       req.Apply,
		"replacement": req.Replacement,
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
	emitProgress := func(phase string, processed, total int, currentFile string, force bool) {
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
	}

	if ctx.Err() != nil {
		a.logInfo("Replace cancelled during file collection", logrus.Fields{
			"directory": req.Search.Directory,
			"query":     req.Search.Query,
		})
		emitProgress("cancelled", 0, 0, "", true)
		return ReplaceResult{}, errors.New("replace cancelled during file collection: no files written")
	}

	staged := a.stageReplacements(ctx, filesToProcess, req, pattern, &result, emitProgress)
	result.FilesChanged = len(staged)

	if req.Apply {
		if err := a.applyReplacements(staged, ctx, &result, emitProgress); err != nil {
			return ReplaceResult{}, err
		}
		a.logInfo("Replace applied", logrus.Fields{
			"filesChanged": result.FilesChanged,
			"linesChanged": result.LinesChanged,
		})
	}

	emitProgress("complete", result.FilesChanged, result.FilesChanged, "", true)
	sort.Slice(result.Files, func(i, j int) bool {
		if result.Files[i].FilePath != result.Files[j].FilePath {
			return result.Files[i].FilePath < result.Files[j].FilePath
		}
		return result.Files[i].LineNum < result.Files[j].LineNum
	})
	return result, nil
}

// collectReplaceFiles gathers and dedupes files across all search directories.
func (a *App) collectReplaceFiles(ctx context.Context, req SearchRequest, pattern *regexp.Regexp) ([]fileMeta, error) {
	searchDirs := []string{req.Directory}
	seen := map[string]bool{filepath.Clean(req.Directory): true}
	for _, d := range req.Directories {
		if d == "" {
			continue
		}
		cleaned := filepath.Clean(d)
		if !seen[cleaned] {
			seen[cleaned] = true
			searchDirs = append(searchDirs, cleaned)
		}
	}

	var filesToProcess []fileMeta
	for _, dir := range searchDirs {
		singleReq := req
		singleReq.Directory = dir
		singleReq.Directories = nil // avoid recursion
		dirFiles, err := a.collectFilesToProcess(ctx, singleReq, pattern)
		if err != nil {
			return nil, err
		}
		filesToProcess = append(filesToProcess, dirFiles...)
	}

	// collectFilesToProcess aborts a cancelled walk with SkipAll, which yields
	// partial candidates and a nil error (file_collection.go:91). Without this
	// check a cancel during collection would fall through and report "no
	// matches" — indistinguishable from a genuinely empty result.
	seenFiles := make(map[string]bool, len(filesToProcess))
	deduped := filesToProcess[:0]
	for _, f := range filesToProcess {
		if seenFiles[f.absPath] {
			continue
		}
		seenFiles[f.absPath] = true
		deduped = append(deduped, f)
	}
	return deduped, nil
}
// stageReplacements matches each file and stages line replacements.
func (a *App) stageReplacements(ctx context.Context, filesToProcess []fileMeta, req ReplaceRequest, pattern *regexp.Regexp, result *ReplaceResult, emitProgress func(phase string, processed, total int, currentFile string, force bool)) []stagedFile {
	var staged []stagedFile
	for i, meta := range filesToProcess {
		if ctx.Err() != nil {
			a.logInfo("Replace cancelled during staging", logrus.Fields{
				"filesScanned": i,
				"totalFiles":   len(filesToProcess),
			})
			emitProgress("cancelled", i, len(filesToProcess), "", true)
			return staged
		}
		emitProgress("staging", i+1, len(filesToProcess), meta.absPath, i == len(filesToProcess)-1)

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
		var fileDiffs []FileReplacement
		for j, line := range lines {
			if pattern.Match(line) {
				oldLine := string(line)
				newLine := pattern.ReplaceAllLiteralString(oldLine, req.Replacement)
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
func (a *App) applyReplacements(staged []stagedFile, ctx context.Context, result *ReplaceResult, emitProgress func(phase string, processed, total int, currentFile string, force bool)) error {
	for i, sf := range staged {
		if ctx.Err() != nil {
			a.logWarn("Replace cancelled during write", logrus.Fields{
				"filesWritten": i,
				"totalFiles":   len(staged),
			})
			emitProgress("cancelled", i, len(staged), "", true)
			return fmt.Errorf("replace cancelled: %d/%d files written before cancellation, no rollback", i, len(staged))
		}
		newContent := bytes.Join(sf.lines, []byte("\n"))
		if err := writeFileAtomic(sf.path, newContent, sf.mode); err != nil {
			return fmt.Errorf("failed to write %s: %w (%d/%d files written before failure)", sf.path, err, i, len(staged))
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
		return err
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
		return err
	}
	// fsync before rename: without it, a crash/power loss between Write and
	// Rename can leave a zero-length or partially-written file at the target
	// path (the rename is durable, the data may not be).
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Chmod(mode); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmpName, path); err != nil {
		return err
	}
	tmpName = "" // success — nothing to clean up
	return nil
}
