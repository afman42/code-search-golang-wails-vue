package main

import (
	"container/list"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sync/atomic"
)

// NewLRUPatternCache creates a new LRU cache with the specified max size
func NewLRUPatternCache(maxSize int64) *LRUPatternCache {
	return &LRUPatternCache{
		cache:   make(map[string]*list.Element),
		list:    list.New(),
		maxSize: maxSize,
	}
}

// Get retrieves a value from the cache, moving it to the front if found
func (c *LRUPatternCache) Get(key string) (*regexp.Regexp, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if elem, ok := c.cache[key]; ok {
		c.list.MoveToFront(elem)
		return elem.Value.(*lruEntry).value, true
	}
	return nil, false
}

// Set adds or updates a value in the cache, evicting old entries if necessary.
// Eviction is O(1) per entry: the key is stored inside the list element, so no
// map scan is needed to find which key the back element belongs to.
func (c *LRUPatternCache) Set(key string, value *regexp.Regexp) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Check if key exists and move to front
	if elem, ok := c.cache[key]; ok {
		c.list.MoveToFront(elem)
		elem.Value.(*lruEntry).value = value
		return
	}

	// Add new entry with the key stored alongside the value
	elem := c.list.PushFront(&lruEntry{key: key, value: value})
	c.cache[key] = elem

	// Evict the back (least-recently-used) element while over capacity
	for c.maxSize > 0 && int64(len(c.cache)) > c.maxSize {
		backElem := c.list.Back()
		if backElem == nil {
			break
		}
		c.list.Remove(backElem)
		delete(c.cache, backElem.Value.(*lruEntry).key)
	}
}

func getPatternCacheKey(useRegex bool, caseSensitive bool, query string) string {
	return fmt.Sprintf("%d:%t:%s", mapBoolToInt(useRegex), caseSensitive, query)
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
	app := &App{
		patternCache: NewLRUPatternCache(100), // Max 100 patterns in cache
		symbolIndex:  newSymbolIndexCache(),
	}
	// Activate the persistent symbol index once at construction. Assigning it
	// per binding call raced the standalone scan goroutine reading it.
	globalSymbolIndex = app.symbolIndex
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

func mapBoolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
