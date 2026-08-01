package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sync"
	"sync/atomic"

	"github.com/sirupsen/logrus"
)

// App struct holds the application context and provides methods for the frontend to call.
type App struct {
	ctx              context.Context
	logger           *logrus.Logger
	searchMu         sync.Mutex         // Guards access to searchCancel
	searchCancel     context.CancelFunc // Cancel function for active searches
	editorsMu        sync.RWMutex       // Guards access to availableEditors
	availableEditors EditorAvailability // Cache of available editors detected at startup
	ready            int32              // Set to 1 once startup() has run; read via IsAppReady
}

// IsAppReady reports whether backend startup has completed. The frontend calls
// this on mount to avoid a race with the one-shot "app-ready" event: if the
// backend emitted the event before the frontend registered its listener, the
// event is missed and this pull-based check lets the UI proceed immediately
// instead of waiting for the fallback timeout.
func (a *App) IsAppReady() bool {
	return atomic.LoadInt32(&a.ready) == 1
}

// markReady records that startup has completed. Safe to call from the startup
// goroutine while IsAppReady is read from bound-method goroutines.
func (a *App) markReady() {
	atomic.StoreInt32(&a.ready, 1)
}

// setSearchCancel stores the cancel function for the active search under lock.
func (a *App) setSearchCancel(cancel context.CancelFunc) {
	a.searchMu.Lock()
	defer a.searchMu.Unlock()
	a.searchCancel = cancel
}

// clearSearchCancel clears the stored cancel function under lock.
func (a *App) clearSearchCancel() {
	a.searchMu.Lock()
	defer a.searchMu.Unlock()
	a.searchCancel = nil
}

// cancelActiveSearch cancels the active search (if any) under lock and reports
// whether a search was actually cancelled.
func (a *App) cancelActiveSearch() bool {
	a.searchMu.Lock()
	defer a.searchMu.Unlock()
	if a.searchCancel != nil {
		a.searchCancel()
		return true
	}
	return false
}

// NewApp creates a new App application struct.
// This function is called during application initialization.
func NewApp() *App {
	app := &App{}
	app.setupLogger()
	return app
}

// shutdown is called when the app is shutting down. This is a Wails lifecycle method.
func (a *App) shutdown(ctx context.Context) {
	// Shut down the polling manager so its log-tail goroutine and file
	// handles are released. The in-memory buffer is discarded — the
	// frontend will fetch fresh entries on next launch.
	pollingManager := GetPollingManager()
	if pollingManager != nil {
		err := pollingManager.Shutdown()
		if err != nil {
			a.logError("Error shutting down log manager", err, nil)
		} else {
			a.logInfo("Log manager shut down successfully", nil)
		}
	}
}

// ReadFileLog resolves a log file name to its absolute path under the logs/ directory.
// Despite its name, it does not read the file — it returns the full path so the frontend
// can fetch the content via the polling server. The name is kept for Wails binding compatibility.
func (a *App) ReadFileLog(filePath string) (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		a.logError("Error Current Directory Not Found", err, nil)
		return "", fmt.Errorf("failed to get current working directory: %w", err)
	}
	return filepath.Join(dir, "logs", filePath), nil
}

// GetInitialLogs returns the last 20 log entries from the polling manager's
// in-memory buffer. The frontend LogViewer calls this on mount to populate
// the preview section without an HTTP round-trip. Using a Wails binding
// (IPC) instead of the HTTP polling server avoids CORS and mixed-content
// issues in production builds, where the webview serves the frontend over a
// secure/custom scheme and blocks plain-HTTP fetches.
func (a *App) GetInitialLogs() []LogMessage {
	pm := GetPollingManager()
	if pm == nil {
		return []LogMessage{}
	}
	return pm.GetLastLogEntries(20)
}

// GetNewLogs returns log entries that have been added since the last call.
// The frontend LogViewer polls this on an interval (the same pattern the
// HTTP /poll endpoint served). Each call advances the per-manager read
// cursor so the next call returns only entries added since this one.
func (a *App) GetNewLogs() []LogMessage {
	pm := GetPollingManager()
	if pm == nil {
		return []LogMessage{}
	}
	return pm.GetNewLogEntries()
}

// WorkerPool caches goroutines and buffers for reuse between searches
type searchWorker struct {
	pattern   *regexp.Regexp
	buffer    []byte
	workingOn bool
}

var globalSearchWorkerPool = &sync.Pool{
	New: func() interface{} {
		return &searchWorker{
			buffer: make([]byte, 1024),
		}
	},
}

func getSearchWorker() *searchWorker {
	w := globalSearchWorkerPool.Get().(*searchWorker)
	return w
}

func putSearchWorker(w *searchWorker) {
	w.workingOn = false
	if cap(w.buffer) < 1024 {
		w.buffer = nil // Don't cache very small buffers
	}
	globalSearchWorkerPool.Put(w)
}

// reusePattern avoids recompiling regex patterns when same query is searched repeatedly
var patternCache = sync.Map{}
var patternMu sync.RWMutex
var cacheSize int64

func cachedCompileRegex(query string, caseSensitive bool) (*regexp.Regexp, error) {
	key := fmt.Sprintf("%d:%s", mapBoolToInt(caseSensitive), query)
	
	patternMu.RLock()
	if cached, ok := patternCache.Load(key); ok {
		patternMu.RUnlock()
		return cached.(*regexp.Regexp), nil
	}
	patternMu.RUnlock()
	
	patternMu.Lock()
	defer patternMu.Unlock()
	
	// Double-check after acquiring write lock
	if cached, ok := patternCache.Load(key); ok {
		return cached.(*regexp.Regexp), nil
	}
	
	pattern, err := compileRegex(query, caseSensitive)
	if err != nil {
		return nil, err
	}
	
	patternCache.Store(key, pattern)
	atomic.AddInt64(&cacheSize, 1)
	
	// Limit cache size to prevent memory leaks
	size := atomic.LoadInt64(&cacheSize)
	if size > 100 {
		// Remove old entries periodically
		var keysToRemove []string
		patternCache.Range(func(k, v interface{}) bool {
			if len(keysToRemove) < 10 {
				keysToRemove = append(keysToRemove, k.(string))
			}
			return true
		})
		for _, k := range keysToRemove {
			patternCache.Delete(k)
			atomic.AddInt64(&cacheSize, -1)
		}
	}
	
	return pattern, nil
}

func mapBoolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

// Compile helper that matches internal naming. Uses Go's inline flag syntax
// (?flags) rather than prepending raw flag chars, which would be treated as
// literal characters in the pattern.
func compileRegex(query string, caseSensitive bool) (*regexp.Regexp, error) {
	if caseSensitive {
		return regexp.Compile(query)
	}
	return regexp.Compile("(?i)" + query)
}
