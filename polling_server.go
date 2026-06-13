package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/nxadm/tail"
)

// LogMessage represents a message sent through the polling system
type LogMessage struct {
	Type    string      `json:"type"`
	Content interface{} `json:"content"`
}

// PollingLogManager manages log entries for polling
type PollingLogManager struct {
	logEntries   []LogMessage
	mutex        sync.RWMutex
	server       *http.Server
	tail         *tail.Tail
	lastRead     int // Index to track where we last read up to
	baseIndex    int // Base index to handle array rotation
}

var pollingManager *PollingLogManager

// InitializePollingLogManager creates and starts the polling log manager
func InitializePollingLogManager() {
	pollingManager = &PollingLogManager{
		logEntries: make([]LogMessage, 0),
		lastRead:   0,
		baseIndex:  0,
		mutex:      sync.RWMutex{},
	}
}

// AddLogEntry adds a new log entry to the manager
func (p *PollingLogManager) AddLogEntry(logMsg LogMessage) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	// Limit the size of the log entries to prevent memory bloat
	if len(p.logEntries) >= 1000 {
		// Calculate how many entries we're removing
		removedCount := len(p.logEntries) - 750
		p.logEntries = p.logEntries[removedCount:] // Keep last 750 entries
		// Adjust baseIndex and lastRead accordingly
		p.baseIndex += removedCount

		// Ensure lastRead doesn't go below baseIndex
		if p.lastRead < p.baseIndex {
			p.lastRead = p.baseIndex
		}
	}

	p.logEntries = append(p.logEntries, logMsg)
}

// GetNewLogEntries returns log entries that have been added since the last poll
func (p *PollingLogManager) GetNewLogEntries() []LogMessage {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	// Calculate the actual index in the current array
	actualLastReadIndex := p.lastRead - p.baseIndex

	// If the actual index is negative, it means the entries we were tracking have been rotated out
	if actualLastReadIndex < 0 {
		actualLastReadIndex = 0
	}

	// If the actual index is beyond the current array size, return empty
	if actualLastReadIndex >= len(p.logEntries) {
		// Update lastRead to the current end of the array
		p.lastRead = p.baseIndex + len(p.logEntries)
		return []LogMessage{}
	}

	newEntries := p.logEntries[actualLastReadIndex:]
	p.lastRead = p.baseIndex + len(p.logEntries)

	return newEntries
}

// GetLastLogEntries returns the last n log entries
func (p *PollingLogManager) GetLastLogEntries(n int) []LogMessage {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	startIndex := 0
	if len(p.logEntries) > n {
		startIndex = len(p.logEntries) - n
	}

	return p.logEntries[startIndex:]
}

// readLastNLines returns up to 20 of the most recent (non-empty, non-noisy) log
// lines from fileName using a ring-buffer approach so we only hold 20 entries in
// memory instead of reading the entire file and then trimming.
func readLastNLines(fileName string) ([]LogMessage, error) {
	file, err := os.Open(fileName)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	const maxLines = 20
	ring := make([]LogMessage, maxLines)
	idx := 0
	count := 0

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		// Parse the line — JSON structured or plain text — and filter noise.
		msg, skip := parseLogLine(line)
		if skip {
			continue
		}
		ring[idx] = msg
		idx = (idx + 1) % maxLines
		if count < maxLines {
			count++
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	// Flatten the ring buffer in chronological order.
	// For count < maxLines the entries are ring[0..count).
	// For count == maxLines the oldest entry is at ring[idx], so we read idx..end then 0..idx.
	out := make([]LogMessage, 0, count)
	if count < maxLines {
		out = append(out, ring[:count]...)
	} else {
		out = append(out, ring[idx:]...)
		out = append(out, ring[:idx]...)
	}
	return out, nil
}

// parseLogLine parses a single log line (JSON structured or plain text) and
// returns the LogMessage. The skip bool is true when the entry should be
// filtered out (noisy internal messages).
func parseLogLine(line string) (LogMessage, bool) {
	var logContent interface{}
	if err := json.Unmarshal([]byte(line), &logContent); err == nil {
		// Structured JSON log — check for noisy messages
		if obj, ok := logContent.(map[string]interface{}); ok {
			if msg, ok := obj["msg"].(string); ok {
				if strings.Contains(msg, "Skipping") || strings.Contains(msg, "Sending file") {
					return LogMessage{}, true
				}
			}
		}
		return LogMessage{Type: "log", Content: logContent}, false
	}

	// Plain text log — filter noise by substring
	if strings.Contains(line, "Skipping") || strings.Contains(line, "Sending file") {
		return LogMessage{}, true
	}
	return LogMessage{Type: "log", Content: line}, false
}

// setCORSHeaders sets common CORS and content-type headers for polling endpoints.
func setCORSHeaders(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
}

// HandleLogPolling handles HTTP requests for polling log entries
func (p *PollingLogManager) HandleLogPolling() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		setCORSHeaders(w)
		if r.Method == "OPTIONS" {
			return
		}

		newEntries := p.GetNewLogEntries()
		if err := json.NewEncoder(w).Encode(newEntries); err != nil {
			log.Printf("Error encoding log entries: %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
		}
	}
}

// HandleGetInitialLogs returns the initial set of logs (last 20 entries)
func (p *PollingLogManager) HandleGetInitialLogs(filePath string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		setCORSHeaders(w)
		if r.Method == "OPTIONS" {
			return
		}

		entries, err := readLastNLines(filePath)
		if err != nil {
			log.Printf("Error reading initial logs from file %s: %v", filePath, err)
			http.Error(w, "Could not read initial logs", http.StatusNotFound)
			return
		}
		if err := json.NewEncoder(w).Encode(entries); err != nil {
			log.Printf("Error encoding initial log entries: %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
		}
	}
}

// StartPollingServer starts an HTTP server for log polling
func (p *PollingLogManager) StartPollingServer(port int) {
	// Create a new ServeMux and register the polling handlers
	mux := http.NewServeMux()
	logFilePath := filepath.Join("logs", "app.log")
	
	mux.HandleFunc("/poll", p.HandleLogPolling())
	mux.HandleFunc("/initial", p.HandleGetInitialLogs(logFilePath))

	// Create an HTTP server instance
	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", port), // Bind to all interfaces
		Handler: mux,
	}
	p.mutex.Lock()
	p.server = server
	p.mutex.Unlock()

	// Start HTTP server on a separate goroutine
	go func() {
		log.Printf("Starting polling server on :%d\n", port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("Failed to start polling server: %v", err)
		}
	}()

	// Start tailing the log file in another goroutine
	go p.TailFile(logFilePath)
}

func (p *PollingLogManager) TailFile(filepath string) {
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

	// Store the tail handle before entering the (blocking) read loop so that
	// Shutdown can clean it up. Assigning after the loop would be unreachable
	// during normal operation since the range below blocks for the app lifetime.
	p.mutex.Lock()
	p.tail = t
	p.mutex.Unlock()

	for line := range t.Lines {
		if line.Text != "" {
			// Check if the line is a structured log (JSON format) or plain text
			var logContent interface{}
			if err := json.Unmarshal([]byte(line.Text), &logContent); err == nil {
				// It's valid JSON, use the parsed object
				logMessage := LogMessage{
					Type:    "log",
					Content: logContent,
				}
				p.AddLogEntry(logMessage)
			} else {
				// It's plain text, store as is
				logMessage := LogMessage{
					Type:    "log",
					Content: line.Text,
				}
				p.AddLogEntry(logMessage)
			}
		}
	}
}

// Shutdown gracefully shuts down the polling server
func (p *PollingLogManager) Shutdown() error {
	// Snapshot the server/tail handles under lock so we don't race with
	// StartPollingServer and TailFile, which set them from other goroutines.
	p.mutex.Lock()
	server := p.server
	t := p.tail
	p.mutex.Unlock()

	// Close the HTTP server to stop accepting new connections
	if server != nil {
		log.Println("Shutting down polling server...")
		if err := server.Close(); err != nil {
			log.Printf("Error closing polling server: %v", err)
			return err
		}
	}

	// Stop tailing if it's active
	if t != nil {
		log.Println("Stopping log tailing...")
		t.Cleanup()
	}

	log.Println("Polling manager shutdown completed")
	return nil
}

// GetPollingManager returns the singleton polling manager
func GetPollingManager() *PollingLogManager {
	return pollingManager
}