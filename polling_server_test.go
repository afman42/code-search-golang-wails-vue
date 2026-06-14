package main

import (
	"io"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"
)

// waitForServer polls addr until it accepts a TCP connection or the deadline
// passes, so the tests don't race the server's goroutine startup.
func waitForServer(t *testing.T, addr string, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		conn, err := net.DialTimeout("tcp", addr, 100*time.Millisecond)
		if err == nil {
			conn.Close()
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("polling server did not start listening on %s within %s", addr, timeout)
}

// TestPollingServerBindsToLoopback verifies the polling server listens on the
// loopback interface only. Binding to 0.0.0.0 would expose the log stream on the
// LAN and trigger the Windows Defender Firewall prompt on first launch.
func TestPollingServerBindsToLoopback(t *testing.T) {
	InitializePollingLogManager()
	mgr := GetPollingManager()
	if mgr == nil {
		t.Fatal("expected a polling manager after initialization")
	}

	// Use an uncommon high port to avoid clashing with a real instance.
	const port = 39117
	mgr.StartPollingServer(port)
	defer func() {
		if err := mgr.Shutdown(); err != nil {
			t.Errorf("Shutdown returned error: %v", err)
		}
	}()

	loopbackAddr := "127.0.0.1:39117"
	waitForServer(t, loopbackAddr, 2*time.Second)

	// The configured address must be loopback-scoped, not all interfaces.
	mgr.mutex.RLock()
	addr := ""
	if mgr.server != nil {
		addr = mgr.server.Addr
	}
	mgr.mutex.RUnlock()

	if !strings.HasPrefix(addr, "127.0.0.1:") {
		t.Errorf("expected server to bind to 127.0.0.1, got %q", addr)
	}

	// The endpoint should be reachable over loopback.
	resp, err := http.Get("http://" + loopbackAddr + "/poll")
	if err != nil {
		t.Fatalf("failed to reach polling server on loopback: %v", err)
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200 from /poll, got %d", resp.StatusCode)
	}
}

// TestPollingServerNotReachableOnNonLoopback confirms the server does not answer
// on a non-loopback local address, proving it is not bound to all interfaces.
func TestPollingServerNotReachableOnNonLoopback(t *testing.T) {
	// Find a non-loopback IPv4 address for this host. If there isn't one
	// (e.g. an isolated CI container), skip — there's nothing to assert against.
	hostIP := firstNonLoopbackIPv4(t)
	if hostIP == "" {
		t.Skip("no non-loopback IPv4 address available; skipping")
	}

	InitializePollingLogManager()
	mgr := GetPollingManager()
	const port = 39118
	mgr.StartPollingServer(port)
	defer mgr.Shutdown()

	waitForServer(t, "127.0.0.1:39118", 2*time.Second)

	// A request to the host's external IP must NOT succeed, since the server is
	// bound to loopback only.
	client := http.Client{Timeout: 500 * time.Millisecond}
	resp, err := client.Get("http://" + net.JoinHostPort(hostIP, "39118") + "/poll")
	if err == nil {
		resp.Body.Close()
		t.Errorf("polling server unexpectedly reachable on non-loopback address %s", hostIP)
	}
}

func firstNonLoopbackIPv4(t *testing.T) string {
	t.Helper()
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return ""
	}
	for _, a := range addrs {
		ipNet, ok := a.(*net.IPNet)
		if !ok || ipNet.IP.IsLoopback() {
			continue
		}
		if ip4 := ipNet.IP.To4(); ip4 != nil {
			return ip4.String()
		}
	}
	return ""
}
