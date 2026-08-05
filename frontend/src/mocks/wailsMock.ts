// Browser-mode mock for the Wails Go backend.
//
// The generated bindings in wailsjs/ call window.go.main.App.* and
// window.runtime.*, which only exist inside the Wails WebView. When the Vue
// frontend is served over plain HTTP (vite dev / Playwright E2E), those globals
// are absent, so IsAppReady() rejects, file reads fail, and search never
// returns — the app renders a black/empty screen.
//
// This module installs a faithful in-browser stand-in so the full
// search -> results -> preview and symbol-search UX flows can be driven end to
// end without the Go process. It is only loaded when VITE_WAILS_MOCK is set
// (see main.ts); production Wails builds never import it.

import type { SearchRequest, SearchResult, SymbolInfo } from "@/types";

// Mirrors backend LogMessage (models.go): { type: "log", content: ... }.
interface LogMessage {
  type: string;
  content: unknown;
}

interface MockFile {
  content: string;
}

// A tiny synthetic project the mocked backend "searches". Deterministic so
// Playwright assertions are stable.
const MOCK_FS: Record<string, MockFile> = {
  "/mock/project/main.go": {
    content: [
      "package main",
      "",
      'import "fmt"',
      "",
      "func main() {",
      '\tfmt.Println("hello world")',
      '\tgreet("kiro")',
      "}",
      "",
      "func greet(name string) {",
      '\tfmt.Printf("hello %s\\n", name)',
      "}",
    ].join("\n"),
  },
  "/mock/project/util.go": {
    content: [
      "package main",
      "",
      "// helper returns a greeting prefix.",
      "func helper() string {",
      '\treturn "hello"',
      "}",
    ].join("\n"),
  },
  "/mock/project/README.md": {
    content: ["# Mock Project", "", "Says hello to the world.", ""].join("\n"),
  },
};

const MOCK_SYMBOLS: SymbolInfo[] = [
  { name: "main", type: "function", line: 5, endLine: 8, signature: "func main()", file: "/mock/project/main.go" },
  { name: "greet", type: "function", line: 10, endLine: 12, signature: "func greet(name string)", file: "/mock/project/main.go" },
  { name: "helper", type: "function", line: 4, endLine: 6, signature: "func helper() string", file: "/mock/project/util.go" },
];

// ---- minimal event bus mirroring wails runtime EventsOn/EventsEmit --------
type EventCallback = (...data: unknown[]) => void;
const listeners: Record<string, EventCallback[]> = {};

function emit(eventName: string, ...data: unknown[]): void {
  (listeners[eventName] || []).forEach((cb) => {
    try {
      cb(...data);
    } catch (e) {
      // Match runtime behaviour: a throwing listener must not break the emit.
      console.error("[wailsMock] listener error for", eventName, e);
    }
  });
}

function on(eventName: string, cb: EventCallback): () => void {
  (listeners[eventName] ||= []).push(cb);
  return () => off(eventName, cb);
}

function off(eventName: string, cb?: EventCallback): void {
  if (!listeners[eventName]) return;
  if (!cb) {
    delete listeners[eventName];
    return;
  }
  listeners[eventName] = listeners[eventName].filter((l) => l !== cb);
}

function delay(ms: number): Promise<void> {
  // NOTE: Promise.withResolvers is unavailable under this project's pinned
  // TypeScript (4.9.5) lib, so the executor form is required here.
  return new Promise<void>((resolve) => setTimeout(resolve, ms));
}

// ---- search ----------------------------------------------------------------
function buildResults(req: SearchRequest): SearchResult[] {
  const query = req?.query ?? "";
  const caseSensitive = !!req?.caseSensitive;
  const useRegex = !!req?.useRegex;
  const results: SearchResult[] = [];
  if (!query) return results;

  let matcher: (line: string) => boolean;
  if (useRegex) {
    const re = new RegExp(query, caseSensitive ? "" : "i");
    matcher = (line) => re.test(line);
  } else {
    const needle = caseSensitive ? query : query.toLowerCase();
    matcher = (line) => (caseSensitive ? line : line.toLowerCase()).includes(needle);
  }

  for (const [filePath, file] of Object.entries(MOCK_FS)) {
    const lines = file.content.split("\n");
    lines.forEach((line, idx) => {
      if (matcher(line)) {
        results.push({
          filePath,
          lineNum: idx + 1,
          content: line,
          matchedText: query,
          contextBefore: lines.slice(Math.max(0, idx - 1), idx),
          contextAfter: lines.slice(idx + 1, idx + 2),
        });
      }
    });
  }
  const cap = Number(req?.maxResults) || 1000;
  return results.slice(0, cap);
}

let cancelled = false;

interface MockApp {
  IsAppReady(): Promise<boolean>;
  SearchWithProgress(req: SearchRequest): Promise<SearchResult[]>;
  CancelSearch(): Promise<null>;
  ExportSearchResults(results: SearchResult[], format: string): Promise<string>;
  ClearSymbolCache(): Promise<null>;
  ReadFile(filePath: string): Promise<string>;
  ReadFileLog(filePath: string): Promise<string>;
  ShowInFolder(filePath: string): Promise<null>;
  GetAllSymbols(directory: string, maxResults: number): Promise<SymbolInfo[]>;
  SearchSymbols(name: string, directory: string, maxResults: number): Promise<SymbolInfo[]>;
  SelectDirectory(): Promise<string>;
  GetDirectoryContents(directory: string): Promise<string[]>;
  GetKnownTextExtensions(): Promise<string[]>;
  GetInitialLogs(): Promise<LogMessage[]>;
  GetNewLogs(): Promise<LogMessage[]>;
  GetAvailableEditors(): Promise<Record<string, boolean>>;
  GetEditorDetectionStatus(): Promise<Record<string, unknown>>;
  [key: string]: (...args: never[]) => Promise<unknown>;
}

const App: MockApp = {
  IsAppReady: async () => true,

  SearchWithProgress: async (req: SearchRequest) => {
    cancelled = false;
    const results = buildResults(req);
    const totalFiles = Object.keys(MOCK_FS).length;
    // Emit progress asynchronously to mirror the streaming backend so the
    // frontend's search-progress listener path is exercised.
    emit("search-progress", {
      processedFiles: 0,
      totalFiles,
      currentFile: "",
      resultsCount: 0,
      status: "started",
    });
    await delay(10);
    if (cancelled) {
      emit("search-progress", {
        processedFiles: totalFiles,
        totalFiles,
        currentFile: "",
        resultsCount: 0,
        status: "cancelled",
      });
      return [];
    }
    emit("search-progress", {
      processedFiles: totalFiles,
      totalFiles,
      currentFile: "",
      resultsCount: results.length,
      status: "completed",
    });
    return results;
  },

  CancelSearch: async () => {
    cancelled = true;
    return null;
  },

  ExportSearchResults: async (_results: SearchResult[], _format: string) => {
    return "/mock/export/search-results.csv";
  },

  ClearSymbolCache: async () => null,

  ReadFile: async (filePath: string) => {
    const f = MOCK_FS[filePath];
    if (!f) throw new Error(`mock: file not found: ${filePath}`);
    return f.content;
  },

  ReadFileLog: async (filePath: string) => {
    const f = MOCK_FS[filePath];
    return f ? f.content : "";
  },

  ShowInFolder: async () => null,

  // Signatures mirror the real Go bindings: (directory, maxResults) and
  // (name, directory, maxResults). The frontend MUST pass a directory.
  GetAllSymbols: async (directory: string) => {
    if (!directory) return [];
    // Mirror the backend's symbol-progress stream so the panel's real progress
    // bar is exercised end to end.
    const total = MOCK_SYMBOLS.length;
    MOCK_SYMBOLS.forEach((_, i) => {
      emit("symbol-progress", { processed: i + 1, total, currentFile: MOCK_SYMBOLS[i].file });
    });
    return MOCK_SYMBOLS;
  },

  SearchSymbols: async (name: string, directory: string, maxResults: number) => {
    if (!directory) return [];
    if (!name) return MOCK_SYMBOLS.slice(0, maxResults || 50);
    const q = name.toLowerCase();
    return MOCK_SYMBOLS.filter(
      (s) => s.name.toLowerCase().includes(q) || (s.signature || "").toLowerCase().includes(q),
    ).slice(0, maxResults || 50);
  },

  SelectDirectory: async () => "/mock/project",

  GetDirectoryContents: async () => Object.keys(MOCK_FS),

  // Mirrors the real backend binding: builds the known-text list WITHOUT the
  // leading dot (e.g. "go", not ".go") and sorted. Keeping the mock in parity
  // with text_extensions.go means dev:mock exercises the exact values the UI
  // dropdown drives in a real (Wails) run.
  GetKnownTextExtensions: async () =>
    ["c", "cpp", "css", "go", "html", "java", "js", "json", "jsx", "md",
     "py", "rb", "rs", "sh", "sql", "toml", "ts", "tsx", "txt", "vue",
     "xml", "yaml", "yml"].sort(),

  // Match the backend contract: these bindings return ARRAYS (possibly empty).
  // Returning a string here made useLogStreaming treat it as a failure, retry
  // 5 times, and inject a fake error log entry on every app mount.
  GetInitialLogs: async () => [],
  GetNewLogs: async () => [],

  GetAvailableEditors: async () => ({}),
  GetEditorDetectionStatus: async () => ({
    detectionComplete: true,
    totalAvailable: 0,
    message: "No editors detected (mock)",
    detectionProgress: 100,
    detectingEditors: false,
    detectedEditors: [],
    availableEditors: {},
  }),
};

// Editor "Open in X" bindings — all no-ops in the browser. Only the generic
// dispatcher and the default-editor binding remain on the backend; the
// per-editor OpenInX wrappers were removed in favor of OpenInEditorByName.
const editorOpeners = ["OpenInDefaultEditor", "OpenInEditorByName"];

interface WailsRuntime {
  EventsOn(name: string, cb: EventCallback): () => void;
  EventsOnMultiple(name: string, cb: EventCallback, max: number): () => void;
  EventsOnce(name: string, cb: EventCallback): () => void;
  EventsOff(name: string, ...more: string[]): void;
  EventsEmit(name: string, ...data: unknown[]): void;
  LogPrint(m: string): void;
  LogTrace(m: string): void;
  LogDebug(m: string): void;
  LogInfo(m: string): void;
  LogWarning(m: string): void;
  LogError(m: string): void;
  LogFatal(m: string): void;
  WindowSetTitle(title: string): void;
  Environment(): Promise<{ buildType: string; platform: string; arch: string }>;
}

interface WailsWindow extends Window {
  go?: { main: { App: MockApp } };
  runtime?: WailsRuntime;
}

export function installWailsMock(): void {
  const w = window as WailsWindow;
  if (w.runtime && w.go) return; // real Wails present — never override.

  const app: MockApp = { ...App };
  for (const name of editorOpeners) {
    if (!(name in app)) app[name] = async () => null;
  }

  w.go = { main: { App: app } };
  w.runtime = {
    EventsOn: on,
    EventsOnMultiple: (name, cb) => on(name, cb),
    EventsOnce: (name, cb) => {
      const dispose = on(name, (...d: unknown[]) => {
        dispose();
        cb(...d);
      });
      return dispose;
    },
    EventsOff: (name, ...more) => {
      off(name);
      more.forEach((n) => off(n));
    },
    EventsEmit: emit,
    LogPrint: (m) => console.log("[wails]", m),
    LogTrace: (m) => console.log("[wails:trace]", m),
    LogDebug: (m) => console.debug("[wails:debug]", m),
    LogInfo: (m) => console.info("[wails:info]", m),
    LogWarning: (m) => console.warn("[wails:warn]", m),
    LogError: (m) => console.error("[wails:error]", m),
    LogFatal: (m) => console.error("[wails:fatal]", m),
    WindowSetTitle: () => {},
    Environment: async () => ({ buildType: "mock", platform: "web", arch: "wasm" }),
  };

  // Fire app-ready shortly after install so App.vue's event path is exercised
  // (IsAppReady() also resolves true, so the UI shows regardless).
  setTimeout(() => emit("app-ready", { status: "ready", timestamp: Date.now() }), 0);
  console.info("[wailsMock] installed browser mock backend");
}
