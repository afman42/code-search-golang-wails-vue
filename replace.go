package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"

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

	// Apply the same defaults/validation as SearchWithProgress so a
	// minimal request (e.g. MaxFileSize 0) behaves identically to search.
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

	// Compile the exact pattern search would use, so case-sensitivity and
	// escaping behave identically to the search the user just ran.
	pattern, err := a.compileSearchPattern(req.Search)
	if err != nil {
		return ReplaceResult{}, err
	}

	// Collect files across all search directories, same dedup as
	// SearchWithProgress. Reuses the collection cache + gitignore filter.
	searchDirs := []string{req.Search.Directory}
	seen := map[string]bool{filepath.Clean(req.Search.Directory): true}
	for _, d := range req.Search.Directories {
		if d == "" {
			continue
		}
		cleaned := filepath.Clean(d)
		if !seen[cleaned] {
			seen[cleaned] = true
			// Append the cleaned path, matching the seen-map key (a raw
			// append would drift for paths differing only in separators).
			searchDirs = append(searchDirs, cleaned)
		}
	}

	var filesToProcess []fileMeta
	for _, dir := range searchDirs {
		singleReq := req.Search
		singleReq.Directory = dir
		singleReq.Directories = nil // avoid recursion
		dirFiles, err := a.collectFilesToProcess(context.Background(), singleReq, pattern)
		if err != nil {
			return ReplaceResult{}, err
		}
		filesToProcess = append(filesToProcess, dirFiles...)
	}

	// Dedupe by absolute path (nested dirs would double-collect).
	seenFiles := make(map[string]bool, len(filesToProcess))
	deduped := filesToProcess[:0]
	for _, f := range filesToProcess {
		if seenFiles[f.absPath] {
			continue
		}
		seenFiles[f.absPath] = true
		deduped = append(deduped, f)
	}
	filesToProcess = deduped

	// Match each file and stage line replacements.
	result := ReplaceResult{Files: []FileReplacement{}}
	type stagedFile struct {
		path  string
		lines [][]byte // full file, split on "\n"; replaced lines swapped in
		mode  os.FileMode
	}
	var staged []stagedFile

	for _, meta := range filesToProcess {
		cleanPath, err := a.sanitizePath(meta.absPath)
		if err != nil {
			// Defense in depth: collection already traversal-checked, but
			// refuse to write anything that doesn't re-pass.
			a.logWarn("Skipping replace on unsafe path", logrus.Fields{"path": meta.absPath, "error": err.Error()})
			continue
		}
		// Lstat (not Stat): collection already skips symlinks, but defense in
		// depth — a symlink swapped in since collection must not be written
		// through (os.Rename would replace the link itself, destroying it).
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
		for i, line := range lines {
			if pattern.Match(line) {
				oldLine := string(line)
				newLine := pattern.ReplaceAllLiteralString(oldLine, req.Replacement)
				if newLine == oldLine {
					continue // no-op replacement — never write
				}
				fileDiffs = append(fileDiffs, FileReplacement{
					FilePath: cleanPath,
					LineNum:  i + 1,
					OldLine:  oldLine,
					NewLine:  newLine,
				})
				lines[i] = []byte(newLine)
			}
		}

		if len(fileDiffs) == 0 {
			continue
		}
		result.Files = append(result.Files, fileDiffs...)
		result.LinesChanged += len(fileDiffs)
		result.FilesChanged++
		staged = append(staged, stagedFile{path: cleanPath, lines: lines, mode: info.Mode().Perm()})
	}

	// Apply: write each changed file atomically. A write failure aborts the
	// whole replace with an error; already-written files stay written (the
	// user re-runs after fixing the cause). The error reports exactly how
	// many files were written before the failure so the user knows what
	// already changed.
	if req.Apply {
		for i, sf := range staged {
			newContent := bytes.Join(sf.lines, []byte("\n"))
			if err := writeFileAtomic(sf.path, newContent, sf.mode); err != nil {
				return ReplaceResult{}, fmt.Errorf("failed to write %s: %w (%d/%d files written before failure)", sf.path, err, i, len(staged))
			}
		}
		a.logInfo("Replace applied", logrus.Fields{
			"filesChanged": result.FilesChanged,
			"linesChanged": result.LinesChanged,
		})
	}

	// Deterministic order regardless of walk order.
	sort.Slice(result.Files, func(i, j int) bool {
		if result.Files[i].FilePath != result.Files[j].FilePath {
			return result.Files[i].FilePath < result.Files[j].FilePath
		}
		return result.Files[i].LineNum < result.Files[j].LineNum
	})

	return result, nil
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
