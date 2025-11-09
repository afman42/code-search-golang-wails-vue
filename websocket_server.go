package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/gorilla/websocket"
	"github.com/nxadm/tail"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
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

type Client struct {
	socket *websocket.Conn
	send   chan LogMessage  // Changed from []LogMessage to LogMessage
}

// WebSocketManager manages WebSocket connections
type WebSocketManager struct {
	clients    map[*Client]bool
	broadcast  chan string
	register   chan *Client
	unregister chan *Client
	server     *http.Server
	tail       *tail.Tail
}

var wsManager *WebSocketManager

// InitializeWebSocketManager creates and starts the WebSocket manager
func InitializeWebSocketManager() {
	wsManager = &WebSocketManager{
		broadcast:  make(chan string),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		clients:    make(map[*Client]bool),
	}
}

func (b *WebSocketManager) Run() {
	for {
		select {
		case client := <-b.register:
			b.clients[client] = true
		case client := <-b.unregister:
			if _, ok := b.clients[client]; ok {
				delete(b.clients, client)
				close(client.send)  // Close the channel when unregistering client
			}
		case message := <-b.broadcast:
			var logMessage LogMessage  // Changed to single LogMessage
			err := json.Unmarshal([]byte(message), &logMessage)
			if err != nil {
				log.Printf("Error unmarshaling broadcast message: %v", err)
				continue
			}
			
			// Send the message to all connected clients
			for client := range b.clients {
				select {
				case client.send <- logMessage:  // Send single message, no closing
				default:
					// If client send channel is full, unregister the client
					b.unregister <- client
				}
			}
		}
	}
}

// Broadcast sends a message to all connected clients
func (manager *WebSocketManager) Broadcast(message LogMessage) {
	messageBytes, err := json.Marshal(message)
	if err != nil {
		log.Printf("Error marshaling log message: %v", err)
		return
	}
	manager.broadcast <- string(messageBytes)
}

func readLastNLines(fileName string) ([]LogMessage, error) {
	file, err := os.Open(fileName)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	lines := make([]LogMessage, 0)

	for scanner.Scan() {
		lines = append(lines, LogMessage{Type: "log", Content: scanner.Text()})
		if len(lines) > 20 {
			lines = lines[1:]
		}
	}

	if scanner.Err() != nil {
		return nil, scanner.Err()
	}

	return lines, nil
}

func (b *WebSocketManager) initialRead(client *Client, filePath string) {
	// Send last n lines from file to the client one by one
	lines, err := readLastNLines(filePath)
	if err != nil {
		log.Println(err)
		return
	}
	
	// Send each log message individually instead of as an array
	for _, line := range lines {
		select {
		case client.send <- line:
		default:
			// If client send channel is full, exit the loop
			return
		}
	}
}

// GetWebSocketManager returns the singleton WebSocket manager
func GetWebSocketManager() *WebSocketManager {
	return wsManager
}

// HandleWebSocket handles WebSocket connections
func (manager *WebSocketManager) HandleWebSocket(b *WebSocketManager, filePath string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ws, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Printf("WebSocket upgrade error: %v", err)
			return
		}
		client := &Client{socket: ws, send: make(chan LogMessage, 256)} // Buffered channel to prevent blocking
		b.register <- client

		go b.initialRead(client, filePath)

		go func() {
			defer func() {
				b.unregister <- client
				ws.Close()
			}()

			// Just read messages to keep connection alive, but don't process them
			for {
				_, _, err := ws.ReadMessage()
				if err != nil {
					log.Printf("WebSocket read error: %v", err)
					b.unregister <- client
					break
				}
			}
		}()

		go func() {
			defer ws.Close()
			for {
				message, ok := <-client.send
				if !ok {
					// Channel was closed, exit the goroutine
					return
				}
				if err := ws.WriteJSON(message); err != nil {
					log.Printf("WebSocket write error: %v", err)
					break
				}
			}
		}()
	}
}

// StartWebSocketServer starts a separate HTTP server for WebSocket connections
func (manager *WebSocketManager) StartWebSocketServer(port int) {
	// Create a new ServeMux and register the WebSocket handler
	mux := http.NewServeMux()
	mux.HandleFunc("/ws", manager.HandleWebSocket(wsManager, filepath.Join("logs", "app.log")))

	// Create an HTTP server instance
	manager.server = &http.Server{
		Addr:    fmt.Sprintf(":%d", port), // Bind to all interfaces
		Handler: mux,
	}

	// Start the WebSocket manager's run loop in a separate goroutine
	go wsManager.Run()

	// Start HTTP server on a separate goroutine
	go func() {
		log.Printf("Starting WebSocket server on :%d\n", port)
		if err := manager.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("Failed to start WebSocket server: %v", err)
		}
	}()

	// Start tailing the log file in another goroutine
	go manager.TailFile(filepath.Join("logs", "app.log"))
}

func (b *WebSocketManager) TailFile(filepath string) {
	// Wait for the file to be created if it doesn't exist yet
	for {
		if _, err := os.Stat(filepath); err == nil {
			// File exists, break the loop
			break
		} else if !os.IsNotExist(err) {
			// There's an error other than "not exists", log it but continue
			log.Printf("Error checking log file: %v", err)
		}
		
		// File doesn't exist yet, wait a bit before checking again
		time.Sleep(500 * time.Millisecond)
	}

	t, err := tail.TailFile(
		filepath,
		tail.Config{Location: &tail.SeekInfo{Offset: 0, Whence: 2}, Follow: true},
	)
	if err != nil {
		log.Printf("tail file err: %v", err)
		return
	}

	for line := range t.Lines {
		if line.Text != "" {
			logMessage := LogMessage{
				Type:    "log",
				Content: line.Text,
			}
			messageBytes, err := json.Marshal(logMessage)
			if err != nil {
				log.Printf("Error marshaling log message: %v", err)
				continue
			}
			b.broadcast <- string(messageBytes)
		}
	}
	b.tail = t
}

// Shutdown gracefully shuts down the WebSocket server
func (manager *WebSocketManager) Shutdown() error {
	// Close the HTTP server to stop accepting new connections
	if manager.server != nil {
		log.Println("Shutting down WebSocket server...")
		if err := manager.server.Close(); err != nil {
			log.Printf("Error closing WebSocket server: %v", err)
			return err
		}
	}

	// Stop tailing if it's active
	if manager.tail != nil {
		log.Println("Stopping log tailing...")
		manager.tail.Cleanup()
	}

	log.Println("WebSocket manager shutdown completed")
	return nil
}
