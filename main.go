package main

import (
	"embed"
	"os"
	"log"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	// Create an instance of the app structure
	app := NewApp()

	// Ensure the logs directory exists
	logDir := "logs"
	if err := os.MkdirAll(logDir, 0755); err != nil {
		log.Printf("Error creating logs directory: %v", err)
	}

	// Initialize the polling log manager and start tailing the log file.
	// The manager's in-memory buffer is consumed by the frontend via the
	// GetInitialLogs() and GetNewLogs() Wails bindings — no HTTP server needed.
	InitializePollingLogManager()

	pollingManager := GetPollingManager()
	if pollingManager != nil {
		pollingManager.StartLogTailing()
	}

	// Create application with options
	err := wails.Run(&options.App{
		Title:  "code-search-golang",
		Width:  1024,
		Height: 768,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 255, G: 255, B: 255, A: 1},
		OnStartup:        app.startup,
		OnShutdown:       app.shutdown,
		Bind: []interface{}{
			app,
		},
	})
	if err != nil {
		log.Printf("Error running application: %v", err)
	}
}
