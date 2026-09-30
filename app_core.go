package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
)

// initialLogEntries is how many recent entries GetInitialLogs returns for the
// LogViewer's initial population.
const initialLogEntries = 20

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

// setSearchCancel stores the caller's handle for the active search under lock.
// It must store the SAME handle the caller keeps, because clearSearchCancel
// retires the slot by pointer identity; allocating a second handle here made
// every clear a no-op and left a dead cancel registered forever.
func (a *App) setSearchCancel(handle *searchCancelHandle) {
	a.searchMu.Lock()
	defer a.searchMu.Unlock()
	a.searchCancel = handle
}

// clearSearchCancel clears the stored cancel handle under lock, but ONLY if
// it still belongs to the search that is finishing. With overlapping searches,
// the later search's setSearchCancel overwrote the stored one; clearing
// unconditionally would nil the later search's cancel and make its Cancel
// button dead.
func (a *App) clearSearchCancel(handle *searchCancelHandle) {
	a.searchMu.Lock()
	defer a.searchMu.Unlock()
	if a.searchCancel == handle {
		a.searchCancel = nil
	}
}

// cancelActiveSearch cancels the active search (if any) and reports
// whether a search was actually cancelled. The handle is copied under lock
// and cancelled after unlock: cancel() runs arbitrary context-callback code
// that must never execute while holding searchMu.
func (a *App) cancelActiveSearch() bool {
	a.searchMu.Lock()
	handle := a.searchCancel
	a.searchMu.Unlock()
	if handle != nil {
		handle.cancel()
		return true
	}
	return false
}

// NewApp creates a new App application struct.
// This function is called during application initialization.
func NewApp() *App {
	app := &App{
		patternCache:    NewLRUPatternCache(100), // Max 100 patterns in cache
		symbolIndex:     newSymbolIndexCache(),
		collectionIndex: newCollectionCache(),
	}
	// Activate the persistent symbol cache once at construction. Assigning it
	// per binding call raced the standalone scan goroutine reading it.
	globalSymbolIndex.Store(app.symbolIndex)
	app.setupLogger()
	return app
}

// shutdown is called when the app is shutting down. This is a Wails lifecycle method.
func (a *App) shutdown(ctx context.Context) {
	// Cancel any in-flight search so its workers stop promptly instead of
	// running to completion during teardown.
	a.cancelActiveSearch()

	// Shut down the polling manager so its log-tail goroutine and file
	// handles are released. The in-memory buffer is discarded — the
	// frontend will fetch fresh entries on next launch.
	pollingManager := GetPollingManager()
	if pollingManager != nil {
		if err := pollingManager.Shutdown(); err != nil {
			a.warnErr("Log manager shutdown reported an error", err, nil)
		}
	}
}

// ReadFileLog resolves a log file name to its absolute path under the logs/ directory.
// Despite its name, it does not read the file — it returns the full path so the frontend
// can fetch the content via the polling server. The name is kept for Wails binding compatibility.
func (a *App) ReadFileLog(filePath string) (string, error) {
	// The frontend passes a bare log file name; joining a ".."-containing or
	// absolute path here would escape the logs/ directory (../../etc/passwd).
	// Reject any path that is not a plain file name.
	if filePath == "" || containsDotDotComponent(filePath) || filepath.IsAbs(filePath) {
		return "", fmt.Errorf("%w: %s", ErrInvalidLogFileName, filePath)
	}
	dir, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get current working directory: %w", err)
	}
	logsDir := filepath.Join(dir, "logs")
	full := filepath.Join(logsDir, filePath)
	// Defense in depth: the join must stay inside logs/.
	if full != logsDir && !strings.HasPrefix(full, logsDir+string(filepath.Separator)) {
		return "", fmt.Errorf("%w: %s", ErrInvalidLogFileName, filePath)
	}
	return full, nil
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
	return pm.GetLastLogEntries(initialLogEntries)
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
