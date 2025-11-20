package main

import (
	"context"
	"os"
	"path/filepath"

	"github.com/sirupsen/logrus"
)

// App struct holds the application context and provides methods for the frontend to call.
type App struct {
	ctx              context.Context
	logger           *logrus.Logger
	searchCancel     context.CancelFunc // Cancel function for active searches
	availableEditors EditorAvailability // Cache of available editors detected at startup
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

func (a *App) ReadFileLog(filePath string) (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		a.logError("Error Current Directory Not Found", err, nil)
		return "", nil
	}
	return filepath.Join(dir, "logs", filePath), nil
}
