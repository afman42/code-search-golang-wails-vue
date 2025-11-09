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

	// Initialize the WebSocket manager
	InitializeWebSocketManager()

	// Start the WebSocket server on a separate port
	wsManager := GetWebSocketManager()
	if wsManager != nil {
		// Use port 34116 which is next to Wails default port 34115
		wsManager.StartWebSocketServer(34116)
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
		println("Error:", err.Error())
	}
}
