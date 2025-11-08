package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/nxadm/tail"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		// Allow connections from any origin during development
		// In production, you should be more restrictive
		return true
	},
}

// LogMessage represents a message sent through the WebSocket
type LogMessage struct {
	Type    string      `json:"type"`
	Content interface{} `json:"content"`
}

// WebSocketManager manages WebSocket connections
type WebSocketManager struct {
	connections map[*websocket.Conn]bool
	broadcast   chan []byte
	mu          sync.Mutex
	tail        *tail.Tail
}

var wsManager *WebSocketManager

// InitializeWebSocketManager creates and starts the WebSocket manager
func InitializeWebSocketManager() {
	wsManager = &WebSocketManager{
		connections: make(map[*websocket.Conn]bool),
		broadcast:   make(chan []byte),
	}
	
	// Start the broadcast handler
	go wsManager.handleBroadcasts()
}

// GetWebSocketManager returns the singleton WebSocket manager
func GetWebSocketManager() *WebSocketManager {
	return wsManager
}

// HandleWebSocket handles WebSocket connections
func (manager *WebSocketManager) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}
	defer conn.Close()

	manager.mu.Lock()
	manager.connections[conn] = true
	manager.mu.Unlock()

	// Send connection confirmation
	confirmation := LogMessage{
		Type:    "connected",
		Content: "Successfully connected to log stream",
	}
	confirmationBytes, _ := json.Marshal(confirmation)
	conn.WriteMessage(websocket.TextMessage, confirmationBytes)

	// Handle incoming messages (if any)
	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			log.Printf("WebSocket read error: %v", err)
			manager.mu.Lock()
			delete(manager.connections, conn)
			manager.mu.Unlock()
			break
		}
		log.Printf("Received WebSocket message: %s", message)
	}
}

// StartWebSocketServer starts a separate HTTP server for WebSocket connections
func (manager *WebSocketManager) StartWebSocketServer(port int) {
	// Register the WebSocket handler
	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		manager.HandleWebSocket(w, r)
	})

	// Start HTTP server on a separate goroutine
	go func() {
		log.Printf("Starting WebSocket server on :%d\n", port)
		if err := http.ListenAndServe(fmt.Sprintf(":%d", port), nil); err != nil {
			log.Printf("Failed to start WebSocket server: %v", err)
		}
	}()
}

// handleBroadcasts handles broadcasting messages to all connected clients
func (manager *WebSocketManager) handleBroadcasts() {
	for {
		message := <-manager.broadcast
		manager.mu.Lock()
		for conn := range manager.connections {
			err := conn.WriteMessage(websocket.TextMessage, message)
			if err != nil {
				log.Printf("WebSocket write error: %v", err)
				conn.Close()
				delete(manager.connections, conn)
			}
		}
		manager.mu.Unlock()
	}
}

// Broadcast sends a message to all connected clients
func (manager *WebSocketManager) Broadcast(message LogMessage) {
	messageBytes, err := json.Marshal(message)
	if err != nil {
		log.Printf("Error marshaling log message: %v", err)
		return
	}
	manager.broadcast <- messageBytes
}

// StartLogTailing starts tailing the log file and streaming new entries via WebSocket
func (manager *WebSocketManager) StartLogTailing(ctx context.Context) {
	go func() {
		// Ensure logs directory exists
		err := os.MkdirAll("logs", 0755)
		if err != nil {
			log.Printf("Error creating logs directory: %v", err)
			return
		}

		// Get absolute path for the log file to ensure consistency
		logFilePath := "logs/app.log"
		absLogPath, err := filepath.Abs(logFilePath)
		if err != nil {
			log.Printf("Error getting absolute path for log file: %v", err)
			absLogPath = logFilePath
		}

		log.Printf("Starting to tail log file: %s", absLogPath)
		
		// Tail the log file
		t, err := tail.TailFile(absLogPath, tail.Config{
			Follow:    true,
			MustExist: false, // Don't require the file to exist initially
			Poll:      true,  // Use polling instead of inotify for better compatibility
			Logger:    tail.DiscardingLogger, // Reduce internal logging
		})
		if err != nil {
			log.Printf("Error starting log tail on path %s: %v", absLogPath, err)
			return
		}
		defer t.Cleanup()

		manager.tail = t

		for {
			select {
			case <-ctx.Done():
				log.Println("Stopping log tailing")
				t.Cleanup()
				return
			case line := <-t.Lines:
				if line != nil {
					// Create log message with content
					logMessage := LogMessage{
						Type:    "log",
						Content: line.Text,
					}
					
					manager.Broadcast(logMessage)
				}
			}
		}
	}()
}

// SendSearchProgress broadcasts search progress updates
func (manager *WebSocketManager) SendSearchProgress(progress map[string]interface{}) {
	message := LogMessage{
		Type:    "search-progress",
		Content: progress,
	}
	manager.Broadcast(message)
}

// SendSearchResult broadcasts a search result
func (manager *WebSocketManager) SendSearchResult(result SearchResult) {
	message := LogMessage{
		Type:    "search-result",
		Content: result,
	}
	manager.Broadcast(message)
}