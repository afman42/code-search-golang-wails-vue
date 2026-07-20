package main

import (
	"encoding/json"
	"log"
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

// PollingLogManager manages log entries for the Wails GetInitialLogs and
// GetNewLogs bindings. It tails the log file and maintains a bounded
// in-memory buffer. No HTTP server is involved — the frontend consumes
// entries via IPC (Wails bindings), not HTTP polling.
type PollingLogManager struct {
	logEntries []LogMessage
	mutex      sync.RWMutex
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

// InitializePollingLogManager creates the polling log manager. Calling it
// twice shuts down any still-running previous instance before installing the
// new one.
func InitializePollingLogManager() {
	pollingMu.Lock()
	defer pollingMu.Unlock()
	if pollingManager != nil {
		// Previous instance is still running — shut it down first so its
		// goroutines are released before we replace it.
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

// StartLogTailing starts tailing the log file in a goroutine. The tailed
// entries are added to the in-memory buffer and consumed by the frontend via
// the GetInitialLogs() and GetNewLogs() Wails bindings.
func (p *PollingLogManager) StartLogTailing() {
	logFilePath := filepath.Join("logs", "app.log")
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

// Shutdown gracefully shuts down the polling manager. It closes p.done to
// unblock TailFile's wait-for-file loop in the case where the log file was
// never created (#3). Safe to call multiple times: the done channel is closed
// under a sync.Once so repeated calls don't panic.
func (p *PollingLogManager) Shutdown() error {
	p.mutex.Lock()
	t := p.tail
	p.mutex.Unlock()

	// Signal any waiting TailFile goroutines to exit. doneOnce protects
	// against the double-close panic when Shutdown is called twice.
	p.doneOnce.Do(func() { close(p.done) })

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
