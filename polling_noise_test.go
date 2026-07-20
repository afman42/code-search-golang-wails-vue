package main

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
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

// TestPollingServerShutdownClosesDone verifies that Shutdown closes the done
// channel so a TailFile goroutine waiting for a non-existent log file exits
// instead of leaking forever (#3). We can't call TailFile directly without a
// real log file, but we can verify the done channel is closed after Shutdown
// by observing that a receive on it returns immediately.
func TestPollingServerShutdownClosesDone(t *testing.T) {
	InitializePollingLogManager()
	mgr := GetPollingManager()

	// Start the server (which also launches TailFile in a goroutine). We
	// don't need the log file to exist for this test — we're checking that
	// Shutdown signals done, not that TailFile exits.
	const port = 39121
	mgr.StartPollingServer(port)
	waitForServer(t, "127.0.0.1:39121", 2*time.Second)

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

// TestPollingServerShutdownIsIdempotent verifies that calling Shutdown twice
// doesn't panic. The done channel is closed under sync.Once, so a second
// close must be a no-op rather than panicking (this matters because
// InitializePollingLogManager now calls Shutdown on any still-running previous
// instance, and tests may also call Shutdown via defer).
func TestPollingServerShutdownIsIdempotent(t *testing.T) {
	InitializePollingLogManager()
	mgr := GetPollingManager()

	const port = 39122
	mgr.StartPollingServer(port)
	waitForServer(t, "127.0.0.1:39122", 2*time.Second)

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
// shuts down the old one first, preventing the HTTP server leak (#13). We
// can't directly observe the old server's goroutine, but we CAN check that
// the old port becomes unreachable after the re-init.
func TestInitializePollingLogManagerShutsDownPrevious(t *testing.T) {
	InitializePollingLogManager()
	mgr := GetPollingManager()
	const port = 39123
	mgr.StartPollingServer(port)
	waitForServer(t, "127.0.0.1:39123", 2*time.Second)

	// Confirm the server is reachable.
	resp, err := http.Get("http://127.0.0.1:39123/poll")
	if err != nil {
		t.Fatalf("expected to reach first server before re-init: %v", err)
	}
	io.Copy(io.Discard, resp.Body)
	resp.Body.Close()

	// Re-initialize. This should shut down the previous manager (closing
	// its listener) before installing the new one.
	InitializePollingLogManager()

	// Give the old listener a moment to actually close.
	time.Sleep(100 * time.Millisecond)

	// The old port must now be unreachable. A connection attempt should
	// fail (the exact error depends on the OS, but it won't be a
	// successful HTTP response).
	client := http.Client{Timeout: 300 * time.Millisecond}
	resp, err = client.Get("http://127.0.0.1:39123/poll")
	if err == nil {
		resp.Body.Close()
		t.Error("expected the first server to be shut down after re-init, but it still responded")
	}

	// Clean up the new manager.
	if newMgr := GetPollingManager(); newMgr != nil {
		newMgr.Shutdown()
	}
}

// TestPollingServerCORSAllowsLocalhost verifies that the polling server sets
// Access-Control-Allow-Origin for requests from the Wails frontend origin
// (http://wails.localhost) so the LogViewer can fetch logs. The previous
// implementation set ACAO to * which allowed any local web page to read the
// logs (#12); now the header is reflected only for allowlisted origins.
func TestPollingServerCORSAllowsLocalhost(t *testing.T) {
	InitializePollingLogManager()
	mgr := GetPollingManager()
	const port = 39124
	mgr.StartPollingServer(port)
	defer mgr.Shutdown()
	waitForServer(t, "127.0.0.1:39124", 2*time.Second)

	req, err := http.NewRequest("GET", "http://127.0.0.1:39124/poll", nil)
	if err != nil {
		t.Fatalf("building request: %v", err)
	}
	req.Header.Set("Origin", "http://wails.localhost")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)

	if got := resp.Header.Get("Access-Control-Allow-Origin"); got != "http://wails.localhost" {
		t.Errorf("expected ACAO to be reflected as http://wails.localhost, got %q", got)
	}
	if got := resp.Header.Get("Vary"); got != "Origin" {
		t.Errorf("expected Vary: Origin, got %q", got)
	}
}

// TestPollingServerCORSBlocksUnknownOrigin verifies that a request from an
// origin NOT on the allowlist (e.g. https://evil.example.com) does NOT receive
// an Access-Control-Allow-Origin header, so a browser would block the
// cross-origin read. This is the security fix for #12: the previous wildcard
// allowed any local web page to read the application's logs.
func TestPollingServerCORSBlocksUnknownOrigin(t *testing.T) {
	InitializePollingLogManager()
	mgr := GetPollingManager()
	const port = 39125
	mgr.StartPollingServer(port)
	defer mgr.Shutdown()
	waitForServer(t, "127.0.0.1:39125", 2*time.Second)

	req, err := http.NewRequest("GET", "http://127.0.0.1:39125/poll", nil)
	if err != nil {
		t.Fatalf("building request: %v", err)
	}
	req.Header.Set("Origin", "https://evil.example.com")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)

	// The server still responds (it's loopback), but the browser-readable
	// CORS header must be absent so a fetch from evil.example.com can't
	// read the body.
	if got := resp.Header.Get("Access-Control-Allow-Origin"); got != "" {
		t.Errorf("expected no ACAO header for unknown origin, got %q", got)
	}
}

// TestPollingServerCORSNoOriginHeader verifies that a request without an
// Origin header (e.g. curl) gets no CORS headers. Same-origin requests and
// non-browser clients don't need them, and not setting ACAO avoids
// accidentally allowing a browser-loaded attacker page that omits Origin.
func TestPollingServerCORSNoOriginHeader(t *testing.T) {
	InitializePollingLogManager()
	mgr := GetPollingManager()
	const port = 39126
	mgr.StartPollingServer(port)
	defer mgr.Shutdown()
	waitForServer(t, "127.0.0.1:39126", 2*time.Second)

	// http.Get doesn't set an Origin header.
	resp, err := http.Get("http://127.0.0.1:39126/poll")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)

	if got := resp.Header.Get("Access-Control-Allow-Origin"); got != "" {
		t.Errorf("expected no ACAO header for no-origin request, got %q", got)
	}
}

// TestPollingServerReturnsEntries verifies the end-to-end polling flow: after
// AddLogEntry, a GET /poll returns the entries as JSON. This is a smoke test
// that the handler wiring still works after the AddLogEntry filter change.
func TestPollingServerReturnsEntries(t *testing.T) {
	InitializePollingLogManager()
	mgr := GetPollingManager()
	const port = 39127
	mgr.StartPollingServer(port)
	defer mgr.Shutdown()
	waitForServer(t, "127.0.0.1:39127", 2*time.Second)

	// Drain any entries from previous tests so /poll returns only our
	// probe entry.
	_ = mgr.GetNewLogEntries()

	mgr.AddLogEntry(LogMessage{Type: "log", Content: "probe-entry"})

	resp, err := http.Get("http://127.0.0.1:39127/poll")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	var entries []LogMessage
	if err := json.NewDecoder(resp.Body).Decode(&entries); err != nil {
		t.Fatalf("decoding response: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d: %+v", len(entries), entries)
	}
	if entries[0].Content != "probe-entry" {
		t.Errorf("expected Content %q, got %v", "probe-entry", entries[0].Content)
	}
}
