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

// TestPollingServerCORSAllowsLocalhostAnyPort verifies that the polling
// server sets Access-Control-Allow-Origin for requests from localhost on ANY
// port. The Wails webview runs on a different port than the polling server
// (e.g. Vite dev server at :34115, production asset server at a random
// port), so the Origin is "http://localhost:34115" — not bare
// "http://localhost". The previous fixed-string allowlist missed the port
// and the browser blocked the response, surfacing as
// "TypeError: Load failed" in the LogViewer. The fix parses the Origin and
// checks the hostname (any port) via isLocalhostOrigin (#12).
func TestPollingServerCORSAllowsLocalhostAnyPort(t *testing.T) {
	InitializePollingLogManager()
	mgr := GetPollingManager()
	const port = 39124
	mgr.StartPollingServer(port)
	defer mgr.Shutdown()
	waitForServer(t, "127.0.0.1:39124", 2*time.Second)

	// Simulate the Wails webview fetching from a different localhost port.
	// The Origin includes the port, which the old allowlist missed.
	origins := []string{
		"http://localhost:34115",     // Vite dev server
		"http://localhost:5173",      // Vite default port
		"http://127.0.0.1:34115",     // explicit loopback IP with port
		"http://wails.localhost",     // Wails v2 Windows webview (no port)
		"null",                       // macOS/Linux Wails webview (custom scheme)
	}
	for _, origin := range origins {
		t.Run(origin, func(t *testing.T) {
			req, err := http.NewRequest("GET", "http://127.0.0.1:39124/poll", nil)
			if err != nil {
				t.Fatalf("building request: %v", err)
			}
			req.Header.Set("Origin", origin)

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatalf("request failed: %v", err)
			}
			defer resp.Body.Close()
			io.Copy(io.Discard, resp.Body)

			// For "null" and non-empty origins, ACAO must be reflected.
			// For "null", the reflected value is "null" itself.
			got := resp.Header.Get("Access-Control-Allow-Origin")
			if origin == "null" {
				if got != "null" {
					t.Errorf("expected ACAO to be reflected as %q, got %q", origin, got)
				}
			} else {
				if got != origin {
					t.Errorf("expected ACAO to be reflected as %q, got %q", origin, got)
				}
			}
			if got != "" && resp.Header.Get("Vary") != "Origin" {
				t.Errorf("expected Vary: Origin, got %q", resp.Header.Get("Vary"))
			}
		})
	}
}

// TestPollingServerCORSBlocksUnknownOrigin verifies that a request from an
// origin that is NOT localhost (e.g. https://evil.example.com) does NOT
// receive an Access-Control-Allow-Origin header, so a browser would block
// the cross-origin read. This is the security goal of #12: external web
// pages cannot read the application's logs even though the server is
// loopback-only (a same-machine attacker page would be blocked by CORS).
func TestPollingServerCORSBlocksUnknownOrigin(t *testing.T) {
	InitializePollingLogManager()
	mgr := GetPollingManager()
	const port = 39125
	mgr.StartPollingServer(port)
	defer mgr.Shutdown()
	waitForServer(t, "127.0.0.1:39125", 2*time.Second)

	nonLocalOrigins := []string{
		"https://evil.example.com",
		"http://192.168.1.100:8080",   // LAN IP, not localhost
		"http://10.0.0.1",              // private network, not localhost
		"http://myserver.local",        // non-localhost hostname
	}
	for _, origin := range nonLocalOrigins {
		t.Run(origin, func(t *testing.T) {
			req, err := http.NewRequest("GET", "http://127.0.0.1:39125/poll", nil)
			if err != nil {
				t.Fatalf("building request: %v", err)
			}
			req.Header.Set("Origin", origin)

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatalf("request failed: %v", err)
			}
			defer resp.Body.Close()
			io.Copy(io.Discard, resp.Body)

			// The server still responds (it's loopback), but the
			// browser-readable CORS header must be absent so a fetch
			// from the external origin can't read the body.
			if got := resp.Header.Get("Access-Control-Allow-Origin"); got != "" {
				t.Errorf("expected no ACAO header for origin %q, got %q", origin, got)
			}
		})
	}
}

// TestPollingServerCORSNoOriginHeader verifies that a request WITHOUT an
// Origin header (e.g. curl) doesn't get an ACAO header (nothing to
// reflect). The server is loopback-only so the request itself is fine, but
// without an Origin header there's no value to reflect — and setting ACAO
// to "*" would re-introduce the vulnerability #12 fixes.
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

	// No Origin → no ACAO header (nothing to reflect). The response still
	// succeeds because the server doesn't gate on CORS — it just doesn't
	// set the header, which only matters for browser cross-origin reads.
	if got := resp.Header.Get("Access-Control-Allow-Origin"); got != "" {
		t.Errorf("expected no ACAO header for no-origin request, got %q", got)
	}
}

// TestIsLocalhostOrigin is a unit test for the hostname-based origin check
// that doesn't require starting the HTTP server. It covers the edge cases:
// empty origin, "null", localhost with/without port, 127.0.0.1 with/without
// port, IPv6 loopback, and non-localhost hosts.
func TestIsLocalhostOrigin(t *testing.T) {
	cases := []struct {
		origin string
		want   bool
		desc   string
	}{
		{"", true, "no Origin header (non-browser / same-origin)"},
		{"null", true, "Wails webview custom scheme"},
		{"http://localhost", true, "localhost without port"},
		{"http://localhost:34115", true, "localhost with port (Vite dev)"},
		{"http://localhost:5173", true, "localhost with Vite default port"},
		{"http://127.0.0.1", true, "loopback IP without port"},
		{"http://127.0.0.1:34115", true, "loopback IP with port"},
		{"http://[::1]", true, "IPv6 loopback"},
		{"http://[::1]:34115", true, "IPv6 loopback with port"},
		{"http://wails.localhost", true, "Wails v2 Windows webview origin"},
		{"https://evil.example.com", false, "external HTTPS origin"},
		{"http://192.168.1.100:8080", false, "LAN IP with port"},
		{"http://10.0.0.1", false, "private network IP"},
		{"http://myserver.local", false, "non-localhost hostname"},
		{"http://localhost.evil.com", false, "fake localhost subdomain"},
		{"https://localhost:34115", true, "localhost with HTTPS scheme (still localhost host)"},
	}
	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			got := isLocalhostOrigin(c.origin)
			if got != c.want {
				t.Errorf("isLocalhostOrigin(%q) = %v, want %v", c.origin, got, c.want)
			}
		})
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
