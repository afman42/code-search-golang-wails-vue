package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
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

// maxLogEntries caps the in-memory log buffer. When this limit is hit the
// oldest entries are dropped. The value is intentionally larger than the
// rotation target (keepAfterRotate) so a single rotation doesn't immediately
// trigger the next one.
const maxLogEntries = 1000

// keepAfterRotate is the number of entries retained after a rotation. The
// previous implementation resliced without copying, which kept the first
// dropped entries alive in the backing array forever (memory leak). The
// rotation now copies into a fresh slice so the old backing array can be
// garbage-collected.
const keepAfterRotate = 750

// PollingLogManager manages log entries for polling
type PollingLogManager struct {
	logEntries []LogMessage
	mutex      sync.RWMutex
	server     *http.Server
	tail       *tail.Tail
	lastRead   int           // Index to track where we last read up to
	baseIndex  int           // Base index to handle array rotation
	done       chan struct{} // Closed by Shutdown to signal TailFile's wait-loop to exit
	doneOnce   sync.Once     // Guards close(done) against double-close panic
}

var (
	pollingManager *PollingLogManager
	pollingMu      sync.Mutex
)

// InitializePollingLogManager creates and starts the polling log manager.
// Calling it twice without shutting down the previous manager would previously
// overwrite the global and leak the first HTTP server (#13). Now it shuts down
// any still-running previous instance before installing the new one. Tests
// that call this between cases still get a fresh manager because Shutdown
// nils out the server handle.
func InitializePollingLogManager() {
	pollingMu.Lock()
	defer pollingMu.Unlock()
	if pollingManager != nil && pollingManager.server != nil {
		// Previous instance is still running — shut it down first so its
		// goroutines and TCP listener are released before we replace it.
		_ = pollingManager.Shutdown()
	}
	pollingManager = &PollingLogManager{
		logEntries: make([]LogMessage, 0, maxLogEntries),
		lastRead:   0,
		baseIndex:  0,
		done:       make(chan struct{}),
	}
}

// AddLogEntry adds a new log entry to the manager. Noisy entries (those that
// parseLogLine flags) are dropped here as well so the live tail stream and the
// initial-load path apply the same filter (#1).
func (p *PollingLogManager) AddLogEntry(logMsg LogMessage) {
	// Re-parse through the shared filter so the live tail stream doesn't
	// admit entries the initial-load path would have dropped (#1).
	if _, skip := parseLogEntryMessage(logMsg.Content); skip {
		return
	}

	p.mutex.Lock()
	defer p.mutex.Unlock()

	// Limit the size of the log entries to prevent memory bloat. Copy the
	// retained tail into a fresh backing array so the dropped entries (which
	// were previously kept alive by the resliced header) can be GC'd (#2).
	if len(p.logEntries) >= maxLogEntries {
		removedCount := len(p.logEntries) - keepAfterRotate
		kept := make([]LogMessage, keepAfterRotate)
		copy(kept, p.logEntries[removedCount:])
		p.logEntries = kept
		p.baseIndex += removedCount

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

	actualLastReadIndex := p.lastRead - p.baseIndex

	if actualLastReadIndex < 0 {
		actualLastReadIndex = 0
	}

	if actualLastReadIndex >= len(p.logEntries) {
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
	out := make([]LogMessage, 0, count)
	if count < maxLines {
		out = append(out, ring[:count]...)
	} else {
		out = append(out, ring[idx:]...)
		out = append(out, ring[:idx]...)
	}
	return out, nil
}

// parseLogLine parses a single raw log line (as read from the log file) into a
// LogMessage. The skip bool is true when the entry should be filtered out
// (noisy internal messages). This is the file-reading counterpart to
// parseLogEntryMessage and shares the same noise rules.
func parseLogLine(line string) (LogMessage, bool) {
	var logContent interface{}
	if err := json.Unmarshal([]byte(line), &logContent); err == nil {
		// Structured JSON log — reuse the shared noise check.
		if _, skip := parseLogEntryMessage(logContent); skip {
			return LogMessage{}, true
		}
		return LogMessage{Type: "log", Content: logContent}, false
	}

	// Plain text log
	if _, skip := parseLogEntryMessage(line); skip {
		return LogMessage{}, true
	}
	return LogMessage{Type: "log", Content: line}, false
}

// parseLogEntryMessage is the single source of truth for noise filtering. It
// accepts either a raw string (plain-text log line) or a parsed JSON object
// (structured logrus entry) and returns (content, skip). Both the initial-load
// path (readLastNLines via parseLogLine) and the live tail path (AddLogEntry)
// route through here so they apply identical rules (#1).
//
// The returned content is the value that should be stored on LogMessage.Content
// (for a string input, the same string; for an object, the same object). When
// skip is true the caller must drop the entry.
func parseLogEntryMessage(raw interface{}) (interface{}, bool) {
	switch v := raw.(type) {
	case string:
		if isNoisyMessage(v) {
			return nil, true
		}
		return v, false
	case map[string]interface{}:
		if msg, ok := v["msg"].(string); ok && isNoisyMessage(msg) {
			return nil, true
		}
		return v, false
	default:
		return raw, false
	}
}

// isNoisyMessage reports whether a log message should be filtered out of the
// viewer. The current rule is "contains 'Skipping' or 'Sending file'" — the
// per-file progress lines that flood the log during a search and add no value
// in the UI.
func isNoisyMessage(msg string) bool {
	return strings.Contains(msg, "Skipping") || strings.Contains(msg, "Sending file")
}

// isLocalhostOrigin reports whether the given Origin header value identifies
// a localhost origin (any port). The Wails webview runs on a different port
// than the polling server (e.g. the Vite dev server at :34115, or the
// production asset server at a random port), so the Origin is
// "http://localhost:NNNN" — not bare "http://localhost". A fixed string
// allowlist would miss the port and the browser would block the response,
// surfacing as "TypeError: Load failed" in the LogViewer.
//
// Parsing the Origin and checking the hostname (not the port) fixes this
// while still blocking non-localhost origins — the security goal of the
// CORS fix (#12) is preserved: a web page served from an external domain
// cannot read the logs, because its Origin hostname won't be localhost.
//
// "null" is the Origin sent by webviews with a custom URL scheme (e.g. the
// Wails v2 macOS/Linux webview), so it's allowed too.
func isLocalhostOrigin(origin string) bool {
	if origin == "" {
		// No Origin header — not a cross-origin request from a browser, or
		// a same-origin request. The server is loopback-only, so the
		// network layer already restricts who can reach it. Allow it.
		return true
	}
	if origin == "null" {
		// Wails v2 macOS/Linux webview uses a custom scheme that reports
		// Origin: null. This is the app's own webview, not an attacker.
		return true
	}
	u, err := url.Parse(origin)
	if err != nil {
		return false
	}
	host := u.Hostname()
	// "localhost" and "127.0.0.1" cover the standard loopback origins on
	// any port. "::1" is IPv6 loopback. "wails.localhost" is the Wails v2
	// Windows webview origin (a special TLD, not a subdomain of localhost).
	return host == "localhost" || host == "127.0.0.1" || host == "::1" || host == "wails.localhost"
}

// setCORSHeaders sets common CORS and content-type headers for polling
// endpoints. The Access-Control-Allow-Origin is reflected from the request's
// Origin header only when it's a localhost origin (any port); otherwise the
// header is omitted entirely so the browser blocks the cross-origin read
// (#12). This prevents external web pages from reading the logs while
// allowing the Wails webview (which runs on a different localhost port) to
// fetch them.
func setCORSHeaders(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	origin := r.Header.Get("Origin")
	if isLocalhostOrigin(origin) {
		if origin != "" {
			// Reflect the exact origin back so the browser's CORS check
			// passes. We can't use "*" because the request includes
			// credentials-adjacent headers and some browsers reject
			// wildcard in that case.
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Vary", "Origin")
		}
		w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	}
}

// HandleLogPolling handles HTTP requests for polling log entries
func (p *PollingLogManager) HandleLogPolling() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		setCORSHeaders(w, r)
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
		setCORSHeaders(w, r)
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
	mux := http.NewServeMux()
	logFilePath := filepath.Join("logs", "app.log")

	mux.HandleFunc("/poll", p.HandleLogPolling())
	mux.HandleFunc("/initial", p.HandleGetInitialLogs(logFilePath))

	// Bind to the loopback interface only (127.0.0.1) rather than all interfaces.
	// The log stream is consumed solely by the local frontend, so there's no
	// reason to expose it on the LAN. Binding to localhost also avoids the
	// Windows Defender Firewall prompt that appears on first launch when a
	// process listens on 0.0.0.0.
	server := &http.Server{
		Addr:    fmt.Sprintf("127.0.0.1:%d", port),
		Handler: mux,
	}
	p.mutex.Lock()
	p.server = server
	p.mutex.Unlock()

	// Start HTTP server on a separate goroutine
	go func() {
		log.Printf("Starting polling server on %s\n", server.Addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("Failed to start polling server: %v", err)
		}
	}()

	// Start tailing the log file in another goroutine
	go p.TailFile(logFilePath)
}

// TailFile tails the given log file and appends each new line to the polling
// manager's in-memory buffer. The wait-for-file-exists loop now selects on
// p.done so Shutdown can unblock it instead of leaking the goroutine forever
// when the log file is never created (#3).
func (p *PollingLogManager) TailFile(filePath string) {
	// Wait for the file to be created if it doesn't exist yet. The select
	// below also watches p.done so a shutdown before the file appears
	// unblocks the goroutine instead of leaking it (#3).
	for {
		if _, err := os.Stat(filePath); err == nil {
			break
		} else if !os.IsNotExist(err) {
			log.Printf("Error checking log file: %v", err)
		}
		select {
		case <-p.done:
			// Shutdown was called before the log file existed — give up
			// cleanly instead of looping forever.
			return
		case <-time.After(500 * time.Millisecond):
		}
	}

	t, err := tail.TailFile(
		filePath,
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
		if line.Text == "" {
			continue
		}
		// Route through the shared parser/filter so the live tail stream
		// applies the same noise filter as the initial-load path (#1).
		msg, skip := parseLogLine(line.Text)
		if skip {
			continue
		}
		p.AddLogEntry(msg)
	}
}

// Shutdown gracefully shuts down the polling server. It also closes p.done to
// unblock TailFile's wait-for-file loop in the case where the log file was
// never created (#3). Safe to call multiple times: the done channel is closed
// under a sync.Once so repeated calls don't panic.
func (p *PollingLogManager) Shutdown() error {
	// Snapshot the server/tail handles under lock so we don't race with
	// StartPollingServer and TailFile, which set them from other goroutines.
	p.mutex.Lock()
	server := p.server
	t := p.tail
	p.mutex.Unlock()

	// Signal any waiting TailFile goroutines to exit. doneOnce protects
	// against the double-close panic when Shutdown is called twice (e.g. by
	// a test's defer plus a subsequent InitializePollingLogManager).
	p.doneOnce.Do(func() { close(p.done) })

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

	// Clear the server handle so a subsequent InitializePollingLogManager
	// sees this instance as "not running" and doesn't try to shut it down
	// again.
	p.mutex.Lock()
	p.server = nil
	p.mutex.Unlock()

	log.Println("Polling manager shutdown completed")
	return nil
}

// GetPollingManager returns the singleton polling manager
func GetPollingManager() *PollingLogManager {
	return pollingManager
}
