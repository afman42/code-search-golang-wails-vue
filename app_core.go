package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"

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
	// Properly shut down the polling server
	pollingManager := GetPollingManager()
	if pollingManager != nil {
		err := pollingManager.Shutdown()
		if err != nil {
			a.logError("Error shutting down polling server", err, nil)
		} else {
			a.logInfo("Polling server shut down successfully", nil)
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
