package main

import (
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"
)

// Tests for the editor-launch fixes: shared-args aliasing (#appendPath),
// zombie reaping (startAndReap), and OpenInDefaultEditor path validation.

// TestAppendPathDoesNotAliasSharedArgs verifies the fix for the slice-aliasing
// bug: openInEditor used to call append(args, path) directly on the args
// slices owned by the package-level editorBindings map. Two concurrent
// launches sharing one backing array would overwrite each other's path entry.
// appendPath must return a fresh slice so earlier results stay stable.
func TestAppendPathDoesNotAliasSharedArgs(t *testing.T) {
	// Simulate a shared bindings slice with spare capacity, exactly like
	// editorBindings["VSCode"].args = []string{"--goto"} built once at init.
	shared := make([]string, 1, 8)
	shared[0] = "--goto"

	first := appendPath(shared, "/tmp/a.go")
	second := appendPath(shared, "/tmp/b.go")

	if got := strings.Join(first, " "); got != "--goto /tmp/a.go" {
		t.Errorf("first result corrupted: %q", got)
	}
	// With a plain append, second's write would land in the same backing
	// array slot and first would silently become "--goto /tmp/b.go".
	if got := strings.Join(first, " "); got != "--goto /tmp/a.go" {
		t.Errorf("first result mutated by second appendPath: %q", got)
	}
	if got := strings.Join(second, " "); got != "--goto /tmp/b.go" {
		t.Errorf("second result wrong: %q", got)
	}
	// The shared slice itself must be untouched.
	if len(shared) != 1 || shared[0] != "--goto" {
		t.Errorf("shared args slice mutated: %v", shared)
	}
}

// TestAppendPathEmptyArgs covers the nil-args editors (most bindings pass nil).
func TestAppendPathEmptyArgs(t *testing.T) {
	got := appendPath(nil, "/x/y.go")
	if len(got) != 1 || got[0] != "/x/y.go" {
		t.Errorf("appendPath(nil, path) = %v, want [/x/y.go]", got)
	}
}

// TestStartAndReapReapsChild verifies that a short-lived child started via
// startAndReap is waited on and does not linger as a zombie. Before the fix,
// every Start without Wait leaked one zombie for the app's lifetime.
func TestStartAndReapReapsChild(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("zombie check uses ps; Windows has no zombies (different process model)")
	}
	if _, err := exec.LookPath("ps"); err != nil {
		t.Skip("ps not available")
	}
	if _, err := exec.LookPath("sh"); err != nil {
		t.Skip("sh not available")
	}

	cmd := exec.Command("sh", "-c", "exit 0")
	if err := startAndReap(cmd); err != nil {
		t.Fatalf("startAndReap failed: %v", err)
	}
	pid := cmd.Process.Pid

	// Give the reaper goroutine time to Wait() the exited child.
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		out, err := exec.Command("ps", "-o", "stat=", "-p", strconv.Itoa(pid)).Output()
		if err != nil {
			return // process no longer in the table — fully reaped
		}
		if !strings.Contains(string(out), "Z") {
			return // alive or defunct-free; not a zombie
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Errorf("child pid %d still a zombie after 5s — startAndReap did not reap it", pid)
}

// TestOpenInDefaultEditorValidatesPath verifies the Linux binding no longer
// forwards raw frontend paths to xdg-open: traversal and nonexistent paths
// are rejected by validation before any process is spawned.
func TestOpenInDefaultEditorValidatesPath(t *testing.T) {
	app := NewApp()

	cases := []struct {
		name     string
		path     string
		contains string
	}{
		{"traversal_relative", "../../../etc/passwd", "directory traversal"},
		{"traversal_component", "/tmp/../etc/passwd", "directory traversal"},
		{"nonexistent", "/nonexistent/path/file.txt", "does not exist"},
		{"empty", "", "required"},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			err := app.OpenInDefaultEditor(tt.path)
			if err == nil {
				t.Fatalf("expected error for %q, got nil", tt.path)
			}
			if !strings.Contains(err.Error(), tt.contains) {
				t.Errorf("error %q does not mention %q", err.Error(), tt.contains)
			}
		})
	}
}