package main

import (
	"context"
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