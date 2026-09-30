package main

import (
	"sync"

	"github.com/nxadm/tail"
)

// LogMessage represents a message sent through the polling system.
type LogMessage struct {
	Type    string `json:"type"`
	Content any    `json:"content"`
}

// PollingLogManager manages log entries for the Wails GetInitialLogs and
// GetNewLogs bindings. It tails the log file and maintains a bounded
// in-memory buffer. No HTTP server is involved — the frontend consumes
// entries via IPC (Wails bindings), not HTTP polling.
//
// Thread safety: every method on *PollingLogManager takes p.mutex (Lock for
// writers, RLock for readers). Fields must only be touched with the mutex
// held, except p.done / p.doneOnce which are close-once broadcast channels
// safe for concurrent select/recv. p.tail is installed by TailFile and
// cleared by Shutdown under p.mutex; Cleanup itself runs outside the lock.
type PollingLogManager struct {
	logEntries []LogMessage
	mutex      sync.RWMutex
	tail       *tail.Tail
	running    bool          // True while a TailFile goroutine is active
	lastRead   int           // Index to track where we last read up to
	baseIndex  int           // Base index to handle array rotation
	done       chan struct{} // Closed by Shutdown to signal TailFile's wait-loop to exit
	doneOnce   sync.Once     // Guards close(done) against double-close panic
}
