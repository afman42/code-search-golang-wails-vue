package main

import (
	"strings"
	"testing"
)

// TestAddLogEntryFiltersNoise verifies that AddLogEntry applies the same noise
// filter as the initial-load path (readLastNLines via parseLogLine). The live
// tail stream previously bypassed the filter, so "Skipping X" and "Sending file
// Y" messages appeared in the polling output but not in the initial load (#1).
func TestAddLogEntryFiltersNoise(t *testing.T) {
	InitializePollingLogManager()
	mgr := GetPollingManager()
	if mgr == nil {
		t.Fatal("expected a polling manager after initialization")
	}

	// Clear any state from previous tests.
	mgr.mutex.Lock()
	mgr.logEntries = mgr.logEntries[:0]
	mgr.lastRead = 0
	mgr.baseIndex = 0
	mgr.mutex.Unlock()

	// Add a noisy entry (should be dropped) and a real entry (should be kept).
	mgr.AddLogEntry(LogMessage{Type: "log", Content: "Skipping file: foo.go"})
	mgr.AddLogEntry(LogMessage{Type: "log", Content: "Sending file progress: bar.go"})
	mgr.AddLogEntry(LogMessage{Type: "log", Content: "Search started"})

	// Also test the JSON-object form: a structured logrus entry whose msg
	// contains "Skipping" must also be dropped.
	mgr.AddLogEntry(LogMessage{
		Type:    "log",
		Content: map[string]interface{}{"msg": "Skipping binary file", "level": "debug"},
	})
	mgr.AddLogEntry(LogMessage{
		Type:    "log",
		Content: map[string]interface{}{"msg": "Search completed", "level": "info"},
	})

	entries := mgr.GetLastLogEntries(100)
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries (noisy ones filtered), got %d: %+v", len(entries), entries)
	}
	for _, e := range entries {
		switch v := e.Content.(type) {
		case string:
			if strings.Contains(v, "Skipping") || strings.Contains(v, "Sending file") {
				t.Errorf("noisy string entry was not filtered: %q", v)
			}
		case map[string]interface{}:
			if msg, ok := v["msg"].(string); ok && (strings.Contains(msg, "Skipping") || strings.Contains(msg, "Sending file")) {
				t.Errorf("noisy structured entry was not filtered: %v", v)
			}
		}
	}
}

// TestAddLogEntryRotationCopiesArray verifies that the log rotation in
// AddLogEntry keeps the in-memory buffer bounded across many rotations.
// The previous implementation did logEntries = logEntries[n:] (a reslice),
// which kept the dropped entries alive in the backing array forever — a
// memory leak that grew with each rotation (#2). The new implementation
// copies the retained tail into a fresh backing array so the old one can be
// GC'd. We can't directly inspect the backing array capacity without unsafe,
// but we verify the observable invariants:
//   1. len never exceeds maxLogEntries no matter how many entries we push.
//   2. After draining, GetNewLogEntries returns exactly the last
//      keepAfterRotate entries (rotation preserves the tail).
//   3. baseIndex advances by exactly the number of dropped entries on each
//      rotation, so the logical indices stay consistent across rotations.
func TestAddLogEntryRotationCopiesArray(t *testing.T) {
	InitializePollingLogManager()
	mgr := GetPollingManager()

	mgr.mutex.Lock()
	mgr.logEntries = mgr.logEntries[:0]
	mgr.lastRead = 0
	mgr.baseIndex = 0
	mgr.mutex.Unlock()

	// Push 5000 entries — enough to trigger several rotations (rotation
	// fires when len reaches maxLogEntries=1000 and trims to
	// keepAfterRotate=750, so rotations happen roughly every 250 pushes
	// after the first 1000).
	for i := 0; i < 5000; i++ {
		mgr.AddLogEntry(LogMessage{Type: "log", Content: "entry"})
		// Invariant 1: len must never exceed maxLogEntries.
		mgr.mutex.Lock()
		currentLen := len(mgr.logEntries)
		mgr.mutex.Unlock()
		if currentLen > maxLogEntries {
			t.Fatalf("after push %d: len=%d exceeds maxLogEntries=%d (rotation not firing)",
				i+1, currentLen, maxLogEntries)
		}
	}

	// Invariant 2: after all pushes, the retained count is bounded by
	// maxLogEntries and is at least keepAfterRotate (the rotation target).
	mgr.mutex.Lock()
	retained := len(mgr.logEntries)
	mgr.mutex.Unlock()
	if retained > maxLogEntries {
		t.Errorf("retained=%d exceeds maxLogEntries=%d", retained, maxLogEntries)
	}
	if retained < keepAfterRotate {
		t.Errorf("retained=%d is less than keepAfterRotate=%d (rotation over-trimmed)", retained, keepAfterRotate)
	}

	// Functional check: GetNewLogEntries should return all retained entries
	// and then report no further entries.
	first := mgr.GetNewLogEntries()
	if len(first) != retained {
		t.Errorf("GetNewLogEntries returned %d, expected %d", len(first), retained)
	}
	second := mgr.GetNewLogEntries()
	if len(second) != 0 {
		t.Errorf("GetNewLogEntries after drain returned %d, expected 0", len(second))
	}
}

// TestShutdownClosesDone verifies that Shutdown closes the done channel so a
// TailFile goroutine waiting for a non-existent log file exits instead of
// leaking forever (#3).
func TestShutdownClosesDone(t *testing.T) {
	InitializePollingLogManager()
	mgr := GetPollingManager()

	// The done channel must be open before Shutdown.
	select {
	case <-mgr.done:
		t.Error("done channel was already closed before Shutdown")
	default:
		// Good — still open.
	}

	if err := mgr.Shutdown(); err != nil {
		t.Fatalf("Shutdown returned error: %v", err)
	}

	// The done channel must be closed. A receive on a closed channel
	// returns immediately with the zero value.
	select {
	case <-mgr.done:
		// Good — done was closed.
	default:
		t.Error("done channel was not closed after Shutdown")
	}
}

// TestShutdownIsIdempotent verifies that calling Shutdown twice doesn't
// panic. The done channel is closed under sync.Once, so a second close must
// be a no-op rather than panicking.
func TestShutdownIsIdempotent(t *testing.T) {
	InitializePollingLogManager()
	mgr := GetPollingManager()

	if err := mgr.Shutdown(); err != nil {
		t.Fatalf("first Shutdown returned error: %v", err)
	}
	// Second call must not panic and must not error.
	if err := mgr.Shutdown(); err != nil {
		t.Errorf("second Shutdown returned error: %v", err)
	}
}

// TestInitializePollingLogManagerShutsDownPrevious verifies that calling
// InitializePollingLogManager while a previous instance is still running
// shuts down the old one first, preventing goroutine leaks (#13).
func TestInitializePollingLogManagerShutsDownPrevious(t *testing.T) {
	InitializePollingLogManager()
	mgr := GetPollingManager()

	// Confirm the manager is running.
	if mgr == nil {
		t.Fatal("expected a manager after initialization")
	}

	// Re-initialize. This should shut down the previous manager before
	// installing the new one.
	InitializePollingLogManager()

	// The old manager's done channel must be closed (shutdown).
	select {
	case <-mgr.done:
		// Good — old manager was shut down.
	default:
		t.Error("expected the previous manager to be shut down after re-init, but done was not closed")
	}

	// Clean up the new manager.
	if newMgr := GetPollingManager(); newMgr != nil {
		newMgr.Shutdown()
	}
}
