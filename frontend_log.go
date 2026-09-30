package main

import (
	"fmt"
	"strings"

	"github.com/sirupsen/logrus"
)

// maxFrontendLogMessage caps a single frontend-supplied log message (~1KB).
// Longer values are truncated with a marker so one chatty caller cannot bloat
// the in-memory log buffer or the on-disk log file.
const maxFrontendLogMessage = 1024

// maxFrontendLogFields caps how many fields a single LogFrontend call may
// carry, and maxFrontendLogFieldValue caps each value's rendered length.
const maxFrontendLogFields = 20

const maxFrontendLogFieldValue = 512

// frontendLogFieldAllowlist is the set of field keys LogFrontend accepts. The
// frontend is untrusted input: anything outside this list is dropped so a
// compromised or buggy caller cannot inject misleading structured keys.
var frontendLogFieldAllowlist = map[string]bool{
	"component": true, "action": true, "directory": true, "file": true,
	"queryLength": true, "resultsCount": true, "durationMs": true,
	"status": true, "code": true, "detail": true,
}

// scrubFrontendFields allow-lists, counts, sizes, and redacts frontend fields.
func scrubFrontendFields(fields map[string]any) logrus.Fields {
	if len(fields) == 0 {
		return nil
	}
	out := make(logrus.Fields, len(fields))
	var count int
	for k, v := range fields {
		if count >= maxFrontendLogFields {
			break
		}
		if !frontendLogFieldAllowlist[k] {
			continue
		}
		s := scrubFrontendValue(k, v)
		if len(s) > maxFrontendLogFieldValue {
			s = s[:maxFrontendLogFieldValue] + "…[truncated]"
		}
		out[k] = s
		count++
	}
	return out
}

// scrubFrontendValue renders a field value as a string, redacting anything
// whose key looks secret-bearing. Values (not keys) are scrubbed too: the
// key "detail" may still carry "token=abc", so scan the rendered value.
func scrubFrontendValue(key string, v any) string {
	rendered := strings.TrimSpace(fmt.Sprint(v))
	lr := strings.ToLower(key + " " + rendered)
	for _, secret := range []string{"password", "token", "secret", "api-key", "apikey", "bearer", "auth"} {
		if strings.Contains(lr, secret) {
			return "[redacted]"
		}
	}
	return rendered
}

// LogFrontend funnels frontend log traffic (search errors, replace failures, UI actions) into the same backend buffer
// so they appear in the unified LogViewer alongside backend logs.
//
// The frontend is untrusted input: keys are allow-listed, message and field
// values are size-capped, and secret-looking values are redacted. Unknown
// levels default to warn so a miscategorized error is not hidden as info.
func (a *App) LogFrontend(level, message string, fields map[string]any) {
	if len(message) > maxFrontendLogMessage {
		message = message[:maxFrontendLogMessage] + "…[truncated]"
	}
	if strings.TrimSpace(message) == "" {
		return
	}
	level = strings.ToLower(strings.TrimSpace(level))
	if level == "" {
		level = "warn"
	}
	switch level {
	case "debug", "info", "warn", "warning", "error":
		if level == "warning" {
			level = "warn"
		}
	default:
		level = "warn"
	}
	lf := scrubFrontendFields(fields)
	switch level {
	case "debug":
		a.logDebug("[frontend] "+message, lf)
	case "info":
		a.logInfo("[frontend] "+message, lf)
	case "warn":
		a.logWarn("[frontend] "+message, lf)
	case "error":
		a.logError("[frontend] "+message, nil, lf)
	default: // unreachable: switch above normalizes to the four cases
		a.logWarn("[frontend] "+message, lf)
	}
}
