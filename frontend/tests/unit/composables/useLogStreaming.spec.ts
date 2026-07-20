import { vi, describe, test, expect, beforeEach, afterEach } from "vitest";

// Mock the Wails binding modules before any imports that use them
vi.mock("../../../wailsjs/go/main/App", () => ({
  GetInitialLogs: vi.fn(),
  GetNewLogs: vi.fn(),
}));

import { GetInitialLogs, GetNewLogs } from "../../../wailsjs/go/main/App";
import { parseLogEntry } from "../../../src/composables/useLogStreaming";

// We test parseLogEntry and composable logic via a factory helper that
// avoids the real onMounted/onUnmounted hooks (which run in setup context).
// For the composable itself, we test it via the LogViewer component tests.

describe("parseLogEntry", () => {
  test("handles structured JSON log", () => {
    const result = parseLogEntry({
      type: "log",
      content: {
        msg: "Test message",
        level: "info",
        time: "2024-01-01T00:00:00Z",
      },
    });

    expect(result).not.toBeNull();
    expect(result!.message).toBe("Test message");
    expect(result!.level).toBe("INFO");
    expect(result!.timestamp).toBeDefined();
  });

  test("skips entries with 'Skipping' in message", () => {
    const result = parseLogEntry({
      type: "log",
      content: { msg: "Skipping hidden directory" },
    });

    expect(result).toBeNull();
  });

  test("skips entries with 'Sending file' in message", () => {
    const result = parseLogEntry({
      type: "log",
      content: "Sending file progress: foo.go",
    });

    expect(result).toBeNull();
  });

  test("handles plain text content", () => {
    const result = parseLogEntry({
      type: "log",
      content: "Plain text log line",
    });

    expect(result).not.toBeNull();
    expect(result!.message).toBe("Plain text log line");
    expect(result!.level).toBe("INFO");
  });

  test("handles missing content gracefully", () => {
    const result = parseLogEntry({
      type: "log",
      content: null,
    });

    expect(result).not.toBeNull();
    expect(result!.message).toContain("Received log event without content");
  });

  test("handles undefined content gracefully", () => {
    const result = parseLogEntry({
      type: "log",
    });

    expect(result).not.toBeNull();
    expect(result!.message).toContain("Received log event without content");
  });

  test("extracts level from various field names", () => {
    const result1 = parseLogEntry({
      type: "log",
      content: { msg: "test", level: "error" },
    });
    expect(result1!.level).toBe("ERROR");

    const result2 = parseLogEntry({
      type: "log",
      content: { msg: "test", Level: "WARN" },
    });
    expect(result2!.level).toBe("WARN");

    const result3 = parseLogEntry({
      type: "log",
      content: { msg: "test", lvl: "debug" },
    });
    expect(result3!.level).toBe("DEBUG");
  });

  test("falls back to INFO when no level is present", () => {
    const result = parseLogEntry({
      type: "log",
      content: { msg: "test" },
    });
    expect(result!.level).toBe("INFO");
  });

  test("formats timestamp from time field", () => {
    const result = parseLogEntry({
      type: "log",
      content: {
        msg: "test",
        time: "2024-06-15T10:30:00Z",
      },
    });
    expect(result).not.toBeNull();
    // Should produce a locale-specific time string, not empty
    expect(result!.timestamp).toBeTruthy();
    expect(result!.timestamp).not.toContain("Invalid");
  });
});

describe("Wails binding mocks", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  test("GetInitialLogs resolves", async () => {
    vi.mocked(GetInitialLogs).mockResolvedValue([]);
    const result = await GetInitialLogs();
    expect(Array.isArray(result)).toBe(true);
  });

  test("GetNewLogs resolves", async () => {
    vi.mocked(GetNewLogs).mockResolvedValue([]);
    const result = await GetNewLogs();
    expect(Array.isArray(result)).toBe(true);
  });

  test("GetInitialLogs returns entries", async () => {
    vi.mocked(GetInitialLogs).mockResolvedValue([
      { type: "log", content: { msg: "test", level: "info" } },
    ]);
    const result = await GetInitialLogs();
    expect(result.length).toBe(1);
    expect(result[0].content.msg).toBe("test");
  });

  test("GetNewLogs returns new entries after first call", async () => {
    vi.mocked(GetNewLogs).mockResolvedValueOnce([{ type: "log", content: "first" }]);
    vi.mocked(GetNewLogs).mockResolvedValueOnce([{ type: "log", content: "second" }]);

    const first = await GetNewLogs();
    expect(first.length).toBe(1);
    expect(first[0].content).toBe("first");

    const second = await GetNewLogs();
    expect(second.length).toBe(1);
    expect(second[0].content).toBe("second");
  });
});
