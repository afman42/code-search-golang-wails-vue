package main

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/sirupsen/logrus"
)

// maxDirectoryListing bounds how many directories a single GetDirectoryContents
// call returns. Deliberately far below maxCachedFiles / maxSymbolScanFiles
// (200k, collection_index.go / symbol_scan.go): this list crosses the IPC
// bridge to render a UI tree, and no tree widget is usable past a few tens of
// thousands of nodes, so the useful ceiling is the renderer's, not memory's.
const maxDirectoryListing = 50_000

// maxDirectoryDepth bounds how deep the walk descends below the requested
// root. A generated or deeply nested tree can blow up the listing without ever
// tripping maxDirectoryListing at any single level; depth is the other bound.
const maxDirectoryDepth = 32

// GetDirectoryContents returns a list of all directory paths in the specified path.
// This function recursively walks the directory tree and collects all directories.
// Hidden directories (dot-prefixed, e.g. .git, .vscode) are skipped, matching the
// search collection walk.
//
// Bounded by maxDirectoryListing and maxDirectoryDepth. Hitting the listing cap
// is an ERROR, not a truncated success: the signature has no room for a
// "truncated" flag (it is a generated Wails binding contract), and a tree view
// silently missing most of the tree is worse than an actionable message telling
// the user to pick a narrower root. Depth pruning is reported by log only —
// pruning one pathological subtree still leaves a coherent listing.
func (a *App) GetDirectoryContents(path string) ([]string, error) {
	var items []string
	pruned := 0
	truncated := false
	// Clean the root once so the depth arithmetic below is exact: a root
	// passed with a trailing separator ("/a/b/") would not prefix-match the
	// cleaned paths WalkDir produces ("/a/b/c") and would over-count depth.
	cleanRoot := filepath.Clean(path)

	// Cancellation follows this file's convention: a.ctx is tolerated as nil
	// (as in safeEmitEvent) rather than required as in SelectDirectory, since
	// a walk has no dialog to attach to. When present, app shutdown aborts the
	// walk instead of holding the IPC call open over a huge tree.
	err := filepath.WalkDir(path, func(itemPath string, d fs.DirEntry, err error) error {
		if a.ctx != nil {
			select {
			case <-a.ctx.Done():
				return a.ctx.Err()
			default:
			}
		}
		if err != nil {
			// A directory that vanished mid-walk is not a problem; any other
			// error (permission, I/O) truncates the listing silently and
			// must surface instead of returning a partial tree as if
			// complete.
			if os.IsNotExist(err) {
				return nil
			}
			return err
		}
		if d.IsDir() {
			// Skip hidden directories that start with a dot (e.g., .git, .vscode)
			if strings.HasPrefix(d.Name(), ".") && itemPath != path {
				return filepath.SkipDir
			}
			// Depth relative to the requested root. Children come from
			// filepath.Join(root, name), so they always carry cleanRoot as a
			// prefix; counting separators in the remaining suffix is
			// equivalent to filepath.Rel without the allocation.
			if strings.Count(strings.TrimPrefix(itemPath, cleanRoot), string(filepath.Separator)) > maxDirectoryDepth {
				pruned++
				return filepath.SkipDir
			}
			if len(items) >= maxDirectoryListing {
				truncated = true
				return filepath.SkipAll
			}
			items = append(items, itemPath) // Only add directories, not files
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	if pruned > 0 {
		a.logWarn("Pruned directory subtrees deeper than the depth bound", logrus.Fields{
			"directory":   path,
			"prunedDirs":  pruned,
			"maxDepth":    maxDirectoryDepth,
			"listedCount": len(items),
		})
	}
	if truncated {
		a.logWarn("Directory listing hit the result cap", logrus.Fields{
			"directory": path,
			"limit":     maxDirectoryListing,
		})
		return nil, fmt.Errorf("directory %s has more than %d subdirectories; choose a narrower directory", path, maxDirectoryListing)
	}

	return items, nil
}
