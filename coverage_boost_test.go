package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sirupsen/logrus"
)

func TestLogLevelFromEnvBranches(t *testing.T) {
	cases := []struct {
		env  string
		want logrus.Level
	}{
		{"trace", logrus.TraceLevel},
		{"TRACE", logrus.TraceLevel},
		{"debug", logrus.DebugLevel},
		{" warn ", logrus.WarnLevel},
		{"warning", logrus.WarnLevel},
		{"error", logrus.ErrorLevel},
		{"", logrus.InfoLevel},
		{"unknown", logrus.InfoLevel},
	}
	for _, c := range cases {
		t.Setenv("LOG_LEVEL", c.env)
		if got := logLevelFromEnv(); got != c.want {
			t.Errorf("LOG_LEVEL=%q: got %v want %v", c.env, got, c.want)
		}
	}
}

func TestNewLRUPatternCacheClamp(t *testing.T) {
	if c := NewLRUPatternCache(0); c.maxSize != 100 {
		t.Errorf("0 clamp: got %d want 100", c.maxSize)
	}
	if c := NewLRUPatternCache(-5); c.maxSize != 100 {
		t.Errorf("-5 clamp: got %d want 100", c.maxSize)
	}
	if c := NewLRUPatternCache(5); c.maxSize != 5 {
		t.Errorf("5: got %d want 5", c.maxSize)
	}
}

func TestLogFrontendBranches(t *testing.T) {
	app := NewApp()
	app.LogFrontend("info", "   ", nil)
	app.LogFrontend("info", "", map[string]interface{}{"k": "v"})
	app.LogFrontend("", "hello default level", nil)
	app.LogFrontend("  DEBUG ", "debug msg", map[string]interface{}{"a": 1})
	app.LogFrontend("info", "info msg", nil)
	app.LogFrontend("warn", "warn msg", nil)
	app.LogFrontend("warning", "warning alias", nil)
	app.LogFrontend("error", "error msg", map[string]interface{}{"x": 1})
	app.LogFrontend("unknown", "fallback to info", nil)
	app.LogFrontend("INFO", "case insensitive", nil)
}

func TestShutdownBranches(t *testing.T) {
	InitializePollingLogManager()
	app := NewApp()
	app.shutdown(nil)
	app.shutdown(nil)

	pollingMu.Lock()
	old := pollingManager
	pollingManager = nil
	pollingMu.Unlock()
	app2 := NewApp()
	app2.shutdown(nil)
	pollingMu.Lock()
	pollingManager = old
	pollingMu.Unlock()
}

func TestReadLastNLinesAndSeedFromFile(t *testing.T) {
	dir := t.TempDir()

	if _, err := readLastNLines(filepath.Join(dir, "nope.log"), 10); err == nil {
		t.Fatal("want error for missing file")
	}
	InitializePollingLogManager()
	mgr := GetPollingManager()
	mgr.mutex.Lock()
	mgr.logEntries = mgr.logEntries[:0]
	mgr.lastRead = 0
	mgr.baseIndex = 0
	mgr.mutex.Unlock()
	mgr.SeedFromFile(filepath.Join(dir, "nope.log"), 10)
	if n := len(mgr.GetLastLogEntries(100)); n != 0 {
		t.Fatalf("seed missing file: want 0 entries, got %d", n)
	}

	content := strings.Join([]string{
		`{"msg":"hello"}`,
		`plain line`,
		``,
		`   `,
		`{"num":123}`,
		`last line`,
		``,
	}, "\n")
	path := filepath.Join(dir, "app.log")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	entries, err := readLastNLines(path, 10)
	if err != nil {
		t.Fatalf("readLastNLines: %v", err)
	}
	if len(entries) != 4 {
		t.Fatalf("want 4 entries, got %d: %+v", len(entries), entries)
	}
	entries2, _ := readLastNLines(path, 2)
	if len(entries2) != 2 {
		t.Fatalf("n=2: want 2, got %d", len(entries2))
	}
	if entries2[1].Content != "last line" {
		t.Errorf("last entry want 'last line', got %v", entries2[1].Content)
	}

	mgr.SeedFromFile(path, 2)
	if got := len(mgr.GetLastLogEntries(10)); got < 2 {
		t.Fatalf("SeedFromFile: want >=2, got %d", got)
	}

	emptyPath := filepath.Join(dir, "empty.log")
	_ = os.WriteFile(emptyPath, []byte("  \n\n \n"), 0o644)
	emptyEntries, _ := readLastNLines(emptyPath, 10)
	if len(emptyEntries) != 0 {
		t.Errorf("empty file: want 0, got %d", len(emptyEntries))
	}
	mgr.SeedFromFile(emptyPath, 10)

	bigPath := filepath.Join(dir, "big.log")
	var bigLines []string
	for range 20 {
		bigLines = append(bigLines, `line `+strings.Repeat("x", 5))
	}
	_ = os.WriteFile(bigPath, []byte(strings.Join(bigLines, "\n")), 0o644)
	bigEntries, _ := readLastNLines(bigPath, 5)
	if len(bigEntries) != 5 {
		t.Errorf("big file n=5: want 5, got %d", len(bigEntries))
	}
}
