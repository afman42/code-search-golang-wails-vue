package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/nxadm/tail"
	"github.com/sirupsen/logrus"
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
	// Snapshot under lock, shut down outside the lock (Shutdown blocks on
	// tail Cleanup), then re-lock to install the replacement.
	pollingMu.Lock()
	old := pollingManager
	pollingMu.Unlock()
	if old != nil {
		// Previous instance is still running — shut it down first so its
		// goroutines are released before we replace it.
		_ = old.Shutdown()
	}
	pollingMu.Lock()
	defer pollingMu.Unlock()
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
	// RLock fast path for the common empty-poll: no clone, no write lock.
	// The old code took Lock on every call, serializing readers against
	// every AddLogEntry from the search workers.
	p.mutex.RLock()
	actualLastReadIndex := p.lastRead - p.baseIndex
	if actualLastReadIndex < 0 {
		actualLastReadIndex = 0
	}
	empty := actualLastReadIndex >= len(p.logEntries)
	p.mutex.RUnlock()
	if empty {
		return []LogMessage{}
	}
	// Non-empty: clone under RLock, then publish the cursor. A concurrent
	// rotation between clone and cursor-publish only moves lastRead forward
	// past entries already cloned — next poll re-reads from the newer
	// cursor, so nothing is lost or duplicated.
	p.mutex.RLock()
	// Re-check under the lock: a rotation may have trimmed the slice
	// between the fast-path check and here.
	idx := p.lastRead - p.baseIndex
	if idx < 0 {
		idx = 0
	}
	if idx >= len(p.logEntries) {
		p.mutex.RUnlock()
		return []LogMessage{}
	}
	out := slices.Clone(p.logEntries[idx:])
	newLast := p.baseIndex + len(p.logEntries)
	p.mutex.RUnlock()
	p.mutex.Lock()
	if newLast > p.lastRead {
		p.lastRead = newLast
	}
	p.mutex.Unlock()
	return out
}

// GetLastLogEntries returns the last n log entries
func (p *PollingLogManager) GetLastLogEntries(n int) []LogMessage {
	p.mutex.RLock()
	defer p.mutex.RUnlock()
	var startIndex int
	if len(p.logEntries) > n {
		startIndex = len(p.logEntries) - n
	}
	return slices.Clone(p.logEntries[startIndex:])
}

// SeedFromFile reads the last n lines from filePath and populates the
// observes new writes. Used at startup to backfill logs/app.log.
func (p *PollingLogManager) SeedFromFile(filePath string, n int) {
	entries, err := readLastNLines(filePath, n)
	if err != nil || len(entries) == 0 {
		if err != nil {
			logrus.WithFields(logrus.Fields{"filePath": filePath}).WithError(err).Warn("SeedFromFile: failed to read log file")
		}
		return
	}
	for _, e := range entries {
		p.AddLogEntry(e)
	}
}

func readLastNLines(filePath string, n int) ([]LogMessage, error) {
	// Tail-seek instead of ReadFile+Split: the old code read the whole
	// (up to 10MB) log into RAM + string-copied it just to keep the last
	// n lines. Seek from the end and read backwards in 32KB chunks.
	if n <= 0 {
		return []LogMessage{}, nil
	}
	f, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("read log file %s: %w", filePath, err)
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		return nil, fmt.Errorf("read log file %s: %w", filePath, err)
	}
	const chunkSize = 32 * 1024
	// Cap the backwards scan: lines longer than 1MB are pathological for a
	// log tail; stop scanning rather than walking a 10MB single line.
	const maxScanBack = 1024 * 1024
	var buf []byte
	offset := info.Size()
	scanned := int64(0)
	lines := 0
	for offset > 0 && lines <= n && scanned < maxScanBack {
		size := int64(chunkSize)
		if size > offset {
			size = offset
		}
		if scanned+size > maxScanBack {
			size = maxScanBack - scanned
		}
		offset -= size
		scanned += size
		chunk := make([]byte, size)
		if _, err := f.ReadAt(chunk, offset); err != nil {
			return nil, fmt.Errorf("read log file %s: %w", filePath, err)
		}
		buf = append(chunk, buf...)
		for _, b := range chunk {
			if b == '\n' {
				lines++
			}
		}
	}
	// Split only the tail we kept (at most ~n+1 lines plus one partial).
	raw := strings.Split(string(buf), "\n")
	// Keep only last n non-empty lines.
	relevant := make([]string, 0, n)
	for i := len(raw) - 1; i >= 0 && len(relevant) < n; i-- {
		if strings.TrimSpace(raw[i]) == "" {
			continue
		}
		relevant = append(relevant, raw[i])
	}
	// Reverse to preserve chronological order.
	for i, j := 0, len(relevant)-1; i < j; i, j = i+1, j-1 {
		relevant[i], relevant[j] = relevant[j], relevant[i]
	}
	out := make([]LogMessage, 0, len(relevant))
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
	var logContent any
	if err := json.Unmarshal([]byte(line), &logContent); err == nil {
		return LogMessage{Type: "log", Content: logContent}, false
	}
	// Plain text log
	return LogMessage{Type: "log", Content: line}, false
}

// parseLogEntryMessage returns content unchanged — kept for callers that
// previously routed through the shared noise filter. Filtering is now UI-only.
func parseLogEntryMessage(raw any) (any, bool) {
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
	// Guard double-start: stop any existing tail before spawning a new one
	// so two TailFile goroutines never tail (and append) concurrently.
	p.mutex.Lock()
	old := p.tail
	p.tail = nil
	p.running = false
	p.mutex.Unlock()
	if old != nil {
		old.Cleanup()
	}
	p.mutex.Lock()
	p.running = true
	p.mutex.Unlock()
	go p.TailFile(logFilePath)
}

// TailFile tails the given log file and appends each new line to the polling
// manager's in-memory buffer. The wait-for-file-exists loop now selects on
// p.done so Shutdown can unblock it instead of leaking the goroutine forever
// when the log file is never created (#3).
func (p *PollingLogManager) TailFile(filePath string) {
	// A TailFile panic (e.g. a bad path tripping the tail library) must not
	// take down the whole backend: recover, record, and return.
	defer func() {
		if r := recover(); r != nil {
			logrus.WithFields(logrus.Fields{"filePath": filePath, "panic": r}).Warn("TailFile recovered from panic")
		}
	}()
	// Wait for the file to be created if it doesn't exist yet. The select
	// below also watches p.done so a shutdown before the file appears
	// unblocks the goroutine instead of leaking it (#3).
	timer := time.NewTimer(500 * time.Millisecond)
	defer timer.Stop()
	for {
		_, statErr := os.Stat(filePath)
		if statErr == nil {
			break
		}
		if !errors.Is(statErr, fs.ErrNotExist) {
			logrus.WithFields(logrus.Fields{"filePath": filePath}).WithError(statErr).Warn("Error checking log file")
		}
		timer.Reset(500 * time.Millisecond)
		select {
		case <-p.done:
			// Shutdown was called before the log file existed — give up
			// cleanly instead of looping forever.
			p.mutex.Lock()
			p.running = false
			p.mutex.Unlock()
			return
		case <-timer.C:
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
		p.mutex.Lock()
		p.running = false
		p.mutex.Unlock()
		logrus.WithFields(logrus.Fields{"filePath": filePath}).WithError(err).Warn("tail file error")
		return
	}

	// Store the tail handle before entering the (blocking) read loop so that
	// Shutdown can clean it up. Assigning after the loop would be unreachable
	// during normal operation since the range below blocks for the app lifetime.
	p.mutex.Lock()
	p.tail = t
	p.mutex.Unlock()

	defer func() {
		p.mutex.Lock()
		if p.tail == t {
			p.tail = nil
		}
		p.running = false
		p.mutex.Unlock()
	}()
	for {
		select {
		case <-p.done:
			t.Cleanup()
			return
		case line, ok := <-t.Lines:
			if !ok {
				return
			}
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
}

// Shutdown gracefully shuts down the polling manager. It closes p.done to
// unblock TailFile's wait-for-file loop in the case where the log file was
// never created (#3). Safe to call multiple times: the done channel is closed
// under a sync.Once so repeated calls don't panic.
func (p *PollingLogManager) Shutdown() error {
	// Signal any waiting TailFile goroutines to exit. doneOnce protects
	// against the double-close panic when Shutdown is called twice.
	p.doneOnce.Do(func() { close(p.done) })

	// Snapshot and nil-out under the mutex so a concurrent Shutdown or
	// StartLogTailing can't double-Cleanup the same handle or store a new
	// tail after we read it (store-after-read leak). Cleanup itself runs
	// outside the lock so we don't block other mutex users.
	p.mutex.Lock()
	t := p.tail
	p.tail = nil
	p.running = false
	p.mutex.Unlock()

	// Stop tailing if it's active
	if t != nil {
		t.Cleanup()
	}

	return nil
}

// GetPollingManager returns the singleton polling manager
func GetPollingManager() *PollingLogManager {
	pollingMu.Lock()
	defer pollingMu.Unlock()
	return pollingManager
}
