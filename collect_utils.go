package main

import (
	"context"
	"path/filepath"
	"sync"
)

// ---------------------------------------------------------------------------
// Shared multi-directory collection
//
// collectSearchFiles (search_engine.go) and collectReplaceFiles (replace.go)
// were two copies of the same flow: expand Directory+Directories into a
// cleaned dir list, collect per-dir with Directories=nil to avoid recursion,
// then dedupe by absolute path. The only behavioral differences were error
// handling (search logs + aborts the whole collect, replace propagates the
// error) and terminal logging — parameterized via onDirError.
// ---------------------------------------------------------------------------

// expandSearchDirs merges Directory + Directories into a cleaned,
// de-duplicated directory list. The raw spelling is preserved for collection
// (matching prior behavior: search appends `d`, replace appends `cleaned` —
// both spellings resolve to the same absPath and dedupe downstream covers it).
func expandSearchDirs(req SearchRequest) []string {
	searchDirs := []string{req.Directory}
	seen := map[string]bool{filepath.Clean(req.Directory): true}
	for _, d := range req.Directories {
		if d == "" {
			continue
		}
		cleaned := filepath.Clean(d)
		if !seen[cleaned] {
			seen[cleaned] = true
			searchDirs = append(searchDirs, d)
		}
	}
	return searchDirs
}

// dedupeFilesByAbsPath drops files already seen by absolute path. It reuses
// the input backing array (files[:0] filter), as both callsites did.
func dedupeFilesByAbsPath(files []fileMeta) []fileMeta {
	seen := make(map[string]bool, len(files))
	deduped := files[:0]
	for _, f := range files {
		if seen[f.absPath] {
			continue
		}
		seen[f.absPath] = true
		deduped = append(deduped, f)
	}
	return deduped
}

// collectAcrossDirs collects files for every directory in the request.
// Directories are collected in PARALLEL (one goroutine per dir, results
// merged in request order): multi-root searches previously serialized the
// full walk + probe per root. Single-dir (the common case) takes the direct
// path with zero goroutine overhead.
//
// collectOne runs the per-directory collection for a request scoped to a
// single directory (Directories=nil). onDirError decides the failure mode
// and its return value is passed straight through: search aborts the whole
// collect with (nil, 0)-equivalent (nil, nil), replace propagates the error.
func collectAcrossDirs(
	ctx context.Context,
	req SearchRequest,
	collectOne func(context.Context, SearchRequest) ([]fileMeta, error),
	onDirError func(dir string, err error) ([]fileMeta, error),
) ([]fileMeta, error) {
	dirs := expandSearchDirs(req)
	if len(dirs) <= 1 {
		filesToProcess := []fileMeta{}
		for _, dir := range dirs {
			singleReq := req
			singleReq.Directory = dir
			singleReq.Directories = nil // avoid recursion
			dirFiles, err := collectOne(ctx, singleReq)
			if err != nil {
				return onDirError(dir, err)
			}
			filesToProcess = append(filesToProcess, dirFiles...)
		}
		return dedupeFilesByAbsPath(filesToProcess), nil
	}
	type dirResult struct {
		idx   int
		files []fileMeta
		err   error
		dir   string
	}
	results := make([]dirResult, len(dirs))
	var wg sync.WaitGroup
	for i, dir := range dirs {
		wg.Add(1)
		go func() {
			defer wg.Done()
			singleReq := req
			singleReq.Directory = dir
			singleReq.Directories = nil
			files, err := collectOne(ctx, singleReq)
			results[i] = dirResult{idx: i, files: files, err: err, dir: dir}
		}()
	}
	wg.Wait()
	// Merge in request order so multi-root output is deterministic
	// regardless of which walk finished first.
	filesToProcess := []fileMeta{}
	for _, r := range results {
		if r.err != nil {
			return onDirError(r.dir, r.err)
		}
		filesToProcess = append(filesToProcess, r.files...)
	}
	return dedupeFilesByAbsPath(filesToProcess), nil
}
