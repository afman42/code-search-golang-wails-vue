package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"runtime/debug"
	"strings"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

// maxLogFileSize caps the on-disk log file size. When logs/app.log exceeds
// this, setupLogger rotates it to logs/app.log.1 (overwriting any previous
// rotation) before opening a fresh app.log. This bounds disk usage to
// ~2×maxLogFileSize (current + rotated) instead of growing unbounded — the
// previous implementation appended forever and a long-running install could
// fill disk (the file was already 17 MB at review time).
const maxLogFileSize = 10 * 1024 * 1024 // 10 MB

// maxAllowedResults caps MaxResults to bound memory: each result carries
// Content + ContextBefore/After strings, so 1M results would OOM the desktop
// app. 10k matches the frontend's pagination (10/page) and is well above
// realistic interactive use.
const maxAllowedResults = 10000

// maxQueryLength caps query length to bound regex compilation cost and
// result payload. Go's RE2 is linear-time (no catastrophic backtracking),
// but a 100k-char regex still wastes CPU/memory. 2000 chars is well above
// realistic interactive use (typical query <100 chars).
const maxQueryLength = 2000

// nullByte is a single-element slice used by bytes.Contains for the binary
// detection check. Declared once to avoid per-call allocation.
var nullByte = []byte{0}

// logLevelFromEnv resolves the log level from LOG_LEVEL, defaulting to Info.
// Debug was the hardcoded default and shipped every per-file debug line to
// disk; an env override keeps debug available for development without
// forcing production logs to that verbosity.
func logLevelFromEnv() logrus.Level {
	switch strings.ToLower(strings.TrimSpace(os.Getenv("LOG_LEVEL"))) {
	case "trace":
		return logrus.TraceLevel
	case "debug":
		return logrus.DebugLevel
	case "warn", "warning":
		return logrus.WarnLevel
	case "error":
		return logrus.ErrorLevel
	default:
		return logrus.InfoLevel
	}
}

// rotateLogFileIfNeeded checks whether logs/app.log exceeds maxLogFileSize and,
// if so, renames it to logs/app.log.1 (replacing any prior rotation). Errors
// are non-fatal: if rotation fails (e.g. permission), we fall through and
// append to the existing file rather than blocking logging entirely.
func rotateLogFileIfNeeded(logPath string) {
	info, err := os.Stat(logPath)
	if err != nil {
		return // file doesn't exist yet — nothing to rotate
	}
	if info.Size() < maxLogFileSize {
		return
	}
	// Rename to .1, overwriting a previous rotation. os.Rename on the same
	// filesystem is atomic; on Windows, a pre-existing target blocks rename,
	// so remove it first.
	rotated := logPath + ".1"
	if err := os.Remove(rotated); err != nil && !errors.Is(err, fs.ErrNotExist) {
		fmt.Fprintf(os.Stderr, "log rotation: failed to remove %s: %v\n", rotated, err)
	}
	if err := os.Rename(logPath, rotated); err != nil {
		fmt.Fprintf(os.Stderr, "log rotation: failed to rename %s: %v\n", logPath, err)
	}
}

// rotatingFileWriter is an io.Writer that rotates logs/app.log at write time
// once it crosses maxLogFileSize, so a long-running session stays bounded
// instead of only rotating at the next startup. It reopens the file after
// rotating. All writes are serialized by mu.
type rotatingFileWriter struct {
	mu      sync.Mutex
	path    string
	file    *os.File
	written int64
}

func newRotatingFileWriter(path string) (*rotatingFileWriter, error) {
	rotateLogFileIfNeeded(path)
	// 0600: logs carry file paths and query text; no reason to be
	// world-readable (gosec G306).
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return nil, err
	}
	var size int64
	if info, statErr := f.Stat(); statErr == nil {
		size = info.Size()
	}
	return &rotatingFileWriter{path: path, file: f, written: size}, nil
}

func (w *rotatingFileWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.written+int64(len(p)) > maxLogFileSize {
		// Close, rotate to .1, reopen a fresh app.log. On any failure keep
		// writing to the current file rather than dropping logs. Close error
		// intentionally ignored: the handle is being replaced regardless.
		_ = w.file.Close()
		rotateLogFileIfNeeded(w.path)
		if f, err := os.OpenFile(w.path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600); err == nil {
			w.file = f
			w.written = 0
		} else {
			// Reopen the original in append mode so logging survives.
			// If even that fails, keep the stale handle (best effort).
			if f2, err2 := os.OpenFile(w.path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600); err2 == nil {
				w.file = f2
				// Reset accounting from the reopened file's real size so a
				// later rotation is still detected (previously drifted).
				if info, statErr := f2.Stat(); statErr == nil {
					w.written = info.Size()
				} else {
					w.written = 0
				}
			}
		}
	}
	n, err := w.file.Write(p)
	w.written += int64(n)
	return n, err
}

// setupLogger initializes the logger with file output and console output
func (a *App) setupLogger() {
	// Create logger instance
	logger := logrus.New()

	// Log level from LOG_LEVEL env (debug|info|warn|error|trace); default
	// Info so production doesn't ship debug noise (per-file progress lines,
	// every skipped file) to disk. Dev/test sets LOG_LEVEL=debug.
	logger.SetLevel(logLevelFromEnv())

	// Create logs directory if it doesn't exist
	err := os.MkdirAll("logs", 0o755)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create logs directory: %v\n", err)
		logger.SetOutput(os.Stdout) // fallback to stdout
		a.logger = logger
		return
	}

	// Use a write-time rotating writer so a long-running session stays
	// bounded to ~2×maxLogFileSize instead of only rotating at next startup.
	logPath := "logs/app.log"
	rw, err := newRotatingFileWriter(logPath)
	if err == nil {
		logger.SetOutput(io.MultiWriter(rw, os.Stdout))
	} else {
		logger.SetOutput(os.Stdout) // fallback to stdout
		logger.WithError(err).Warn("Failed to open log file, using stdout only")
	}

	// Set JSON formatter for structured logs
	logger.SetFormatter(&logrus.JSONFormatter{
		TimestampFormat: time.RFC3339,
	})

	a.logger = logger
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods.
func (a *App) startup(ctx context.Context) {
	a.setCtx(ctx)

	// Log application startup. The JSON formatter already stamps time, so no
	// explicit timestamp field is needed here.
	a.logInfo("Application starting", nil)

	// Emit app-ready immediately so the frontend can show the main UI without
	// waiting for editor detection. Editor detection probes the system PATH for
	// ~21 editors, which can take noticeable time and must not block first paint.
	if ctx != nil {
		a.safeEmitEvent("app-ready", map[string]any{
			"status":    "ready",
			"timestamp": time.Now().Unix(),
		})
	}

	// Mark ready so the frontend's IsAppReady() poll succeeds even if it missed
	// the one-shot app-ready event above (the event and the frontend listener
	// registration are not ordered).
	a.markReady()

	// Detect available editors in the background (this will emit its own
	// progress/completion events as results come in).
	go a.detectAvailableEditors()
}

// emitToManager pushes a structured log entry directly to the polling buffer so
// the UI viewer sees it instantly without waiting for file tail latency.
func (a *App) emitToManager(level, message string, fields logrus.Fields, err error) {
	pm := GetPollingManager()
	if pm == nil {
		return
	}
	entry := make(map[string]any, len(fields)+4)
	for k, v := range fields {
		entry[k] = v
	}
	entry["level"] = level
	entry["msg"] = message
	entry["time"] = time.Now().Format(time.RFC3339)
	if err != nil {
		entry["error"] = err.Error()
	}
	pm.AddLogEntry(LogMessage{Type: "log", Content: entry})
}

// logInfo logs an informational message with optional fields
func (a *App) logInfo(message string, fields logrus.Fields) {
	if a.logger != nil {
		a.logger.WithFields(fields).Info(message)
	}
	a.emitToManager("info", message, fields, nil)
	// Also send to Wails runtime for console output
	if ctx := a.getCtx(); ctx != nil {
		wailsRuntime.LogInfo(ctx, message)
	}
}

// logWarn logs a warning message with optional fields
func (a *App) logWarn(message string, fields logrus.Fields) {
	if a.logger != nil {
		a.logger.WithFields(fields).Warn(message)
	}
	a.emitToManager("warning", message, fields, nil)
	// Also send to Wails runtime for console output
	if ctx := a.getCtx(); ctx != nil {
		wailsRuntime.LogWarning(ctx, message)
	}
}

// warnErr logs a warning with a structured error cause. Unlike logError it
// does not imply the caller failed noisily: Wails bindings use it for the
// single boundary log while returning the wrapped error to the frontend.
// Nil-logger safe (unit tests build &App{} without setupLogger).
func (a *App) warnErr(message string, err error, fields logrus.Fields) {
	if a.logger != nil {
		a.logger.WithError(err).WithFields(fields).Warn(message)
	}
	a.emitToManager("warning", message, fields, err)
}

// logError logs an error message with optional fields
func (a *App) logError(message string, err error, fields logrus.Fields) {
	if a.logger != nil {
		a.logger.WithFields(fields).WithError(err).Error(message)
	}
	a.emitToManager("error", message, fields, err)
	// Also send to Wails runtime for console output
	if ctx := a.getCtx(); ctx != nil {
		if err != nil {
			wailsRuntime.LogError(ctx, message+": "+err.Error())
		} else {
			wailsRuntime.LogError(ctx, message)
		}
	}
}

// logDebug logs a debug message with optional fields
func (a *App) logDebug(message string, fields logrus.Fields) {
	// Fast path: skip the emitToManager map alloc + time.Format + buffer
	// append when debug is off (production default is Info). The old code
	// built the entry unconditionally, so hot-loop logDebug calls allocated
	// even with debug off.
	//
	// Soundness: emitToManager is the UI live-log feed. When debug is off,
	// debug entries are dropped from BOTH disk and UI — consistent (no
	// disk/UI divergence), and polling-buffer tests use no-logger Apps
	// which still emit (nil-logger path below).
	if a.logger != nil {
		if a.logger.IsLevelEnabled(logrus.DebugLevel) {
			a.logger.WithFields(fields).Debug(message)
			a.emitToManager("debug", message, fields, nil)
		}
		return
	}
	// No logger (unit tests): still emit so polling-buffer tests observe.
	a.emitToManager("debug", message, fields, nil)
}

// safeEmitEvent safely emits a Wails event, ignoring errors when not in proper context.
// In test environments or when the Wails runtime is unavailable, EventsEmit panics;
// we catch that panic here so callers don't need to worry about the runtime state.
//
// The recover is scoped to EventsEmit only: a panic anywhere else (nil deref,
// malformed payload) must NOT be swallowed — those are real bugs that should
// surface, not be hidden behind a blanket recover.
func (a *App) safeEmitEvent(eventName string, data any) {
	// Copy ctx under lock and use the copy for both checks: startup's write
	// races these reads from binding goroutines.
	ctx := a.getCtx()
	if ctx == nil {
		return
	}

	select {
	case <-ctx.Done():
		return
	default:
	}

	emitViaWails := func() {
		// Swallow only the "runtime not ready / not in Wails context"
		// panic that EventsEmit raises in tests and dev mocks — but record
		// it so a silently-dropped event is diagnosable.
		defer func() {
			if r := recover(); r != nil {
				fields := logrus.Fields{
					"event": eventName,
					"panic": r,
					"stack": string(debug.Stack()),
				}
				if a.logger != nil {
					a.logger.WithFields(fields).Warn("safeEmitEvent recovered from EventsEmit panic")
				}
				a.emitToManager("warning", "safeEmitEvent recovered from EventsEmit panic", logrus.Fields{
					"event": eventName,
					"panic": r,
				}, nil)
			}
		}()
		wailsRuntime.EventsEmit(ctx, eventName, data)
	}
	emitViaWails()
}
