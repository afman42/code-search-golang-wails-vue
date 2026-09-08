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

// AddLogEntry adds a new log entry to the manager. All entries are stored —
// noise filtering is done only in the UI layer so "add all logs" is honored.
func (p *PollingLogManager) AddLogEntry(logMsg LogMessage) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	// Limit the size of the log entries to prevent memory bloat. Copy the
	// retained tail into a fresh backing array so the dropped entries (which
	// were previously kept alive by the resliced header) can be GC'd.
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
	out := make([]LogMessage, len(p.logEntries)-startIndex)
	copy(out, p.logEntries[startIndex:])
	return out
}

// SeedFromFile reads the last n lines from filePath and populates the
// observes new writes. Used at startup to backfill logs/app.log.
func (p *PollingLogManager) SeedFromFile(filePath string, n int) {
	entries, err := readLastNLines(filePath, n)
	if err != nil || len(entries) == 0 {
		return
	}
	for _, e := range entries {
		p.AddLogEntry(e)
	}
}

func readLastNLines(filePath string, n int) ([]LogMessage, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}
	lines := strings.Split(string(data), "\n")
	// Keep only last n non-empty lines.
	var relevant []string
	for i := len(lines) - 1; i >= 0 && len(relevant) < n; i-- {
		if strings.TrimSpace(lines[i]) == "" {
			continue
		}
		relevant = append(relevant, lines[i])
	}
	// Reverse to preserve chronological order.
	for i, j := 0, len(relevant)-1; i < j; i, j = i+1, j-1 {
		relevant[i], relevant[j] = relevant[j], relevant[i]
	}
	var out []LogMessage
	for _, line := range relevant {
		msg, skip := parseLogLine(line)
		if skip {
			continue
		}
		out = append(out, msg)
	}
	return out, nil
}

// parseLogLine parses a single raw log line (as read from the log file) into a
// LogMessage. All lines are kept — filtering is UI-only.
func parseLogLine(line string) (LogMessage, bool) {
	if strings.TrimSpace(line) == "" {
		return LogMessage{}, true
	}
	var logContent interface{}
	if err := json.Unmarshal([]byte(line), &logContent); err == nil {
		return LogMessage{Type: "log", Content: logContent}, false
	}
	// Plain text log
	return LogMessage{Type: "log", Content: line}, false
}

// parseLogEntryMessage returns content unchanged — kept for callers that
// previously routed through the shared noise filter. Filtering is now UI-only.
func parseLogEntryMessage(raw interface{}) (interface{}, bool) {
	return raw, false
}

// isNoisyMessage is deprecated: all logs are shown. Kept for existing tests
// that assert the historical pattern; always returns false so nothing is filtered.
func isNoisyMessage(msg string) bool {
	_ = msg
	return false
}

// StartLogTailing seeds the buffer from the existing log file and starts
// tailing new writes. Seeding fixes "cannot see at all": the tail was
// opened with Whence:2 (EOF) so GetInitialLogs returned empty until a new
// write arrived; seeding backfills the last 200 lines before the tail starts.
func (p *PollingLogManager) StartLogTailing() {
	logFilePath := filepath.Join("logs", "app.log")
	p.SeedFromFile(logFilePath, 200)
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
		// ReOpen re-opens the file after a rename/rotation: setupLogger
		// rotates app.log -> app.log.1 past 10MB, and without ReOpen the
		// tailer keeps following the renamed inode, silently freezing the
		// live log view after the first rotation.
		tail.Config{Location: &tail.SeekInfo{Offset: 0, Whence: 2}, Follow: true, ReOpen: true},
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
	pollingMu.Lock()
	defer pollingMu.Unlock()
	return pollingManager
}
