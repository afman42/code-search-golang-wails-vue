package main

import (
	"context"
	"sync"

	"github.com/sirupsen/logrus"
)

// App struct holds the application context and provides methods for the frontend to call.
type App struct {
	ctxMu               sync.RWMutex // Guards ctx: startup's write races emit-path reads
	ctx                 context.Context
	logger              *logrus.Logger
	searchMu            sync.Mutex          // Guards access to searchCancel
	searchCancel        *searchCancelHandle // Cancel handle for active searches
	editorsMu           sync.RWMutex        // Guards access to availableEditors
	availableEditors    EditorAvailability  // Cache of available editors detected at startup
	ready               int32               // Set to 1 once startup() has run; read via IsAppReady
	editorDetectionDone int32               // Set to 1 once detectAvailableEditors completes; read via GetEditorDetectionStatus
	patternCache        *LRUPatternCache    // LRU cache for compiled regex patterns
	symbolIndex         *symbolIndexCache   // Cached symbol indices per directory
	collectionIndex     *collectionCache    // Cached file-collection results per directory+fingerprint+filter
}

// setCtx stores the Wails runtime context. Called once from startup; locked
// because emit paths read ctx concurrently from binding goroutines.
func (a *App) setCtx(ctx context.Context) {
	a.ctxMu.Lock()
	defer a.ctxMu.Unlock()
	a.ctx = ctx
}

// getCtx returns the Wails runtime context (nil in unit tests that build App
// directly). Returns a copy under RLock so callers never race startup's
// write; use the returned value for both nil-check and use.
func (a *App) getCtx() context.Context {
	a.ctxMu.RLock()
	defer a.ctxMu.RUnlock()
	return a.ctx
}

// searchCancelHandle wraps a cancel function so the stored cancel can be
// compared by pointer identity. context.CancelFunc values (closures) are only
// comparable to nil, so a pointer wrapper is the way to tell "this search's
// cancel" from a later search's.
type searchCancelHandle struct {
	cancel context.CancelFunc
}

// collectStats holds the counters gathered during the directory walk for
// logging at the end of collection. It's returned by walkDirectoryTree so
// the caller can log a single summary line without passing the App's logger
// deep into the walk.
type collectStats struct {
	filesCollected int
	filesSkipped   int
	dirsSkipped    int
}

// fileMeta carries the per-file metadata gathered during collection so the
// worker pool can process a file without repeating syscalls. The absolute path
// and size are computed once in collectFilesToProcess (file_collection.go);
// reusing them avoids a second os.Stat and filepath.Abs per file.
type fileMeta struct {
	absPath string
	size    int64
}
