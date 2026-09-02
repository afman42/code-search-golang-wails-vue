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

import type { SearchRequest, SearchResult, SymbolInfo, ReplaceRequest, ReplaceResult, FileReplacement } from "@/types";
import { EDITOR_CATALOG } from "@/constants/editors";

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
  // A large file (60 lines) under a SEPARATE root so /mock/project counts are
  // unaffected. Previewing it exceeds CodeModal's >50-line threshold, which is
  // what mounts the match-navigation controls. It also carries 12 "needle"
  // lines so a search produces >10 results (exercises pagination at 10/page).
  "/mock/big/huge.go": {
    content: (() => {
      const lines: string[] = ["package big", ""];
      for (let i = 1; i <= 58; i++) {
        // Every 5th line contains the searchable token "needle" (12 total).
        lines.push(i % 5 === 0 ? `\tprocess("needle ${i}")` : `\tstep${i}()`);
      }
      return lines.join("\n");
    })(),
  },
  // A second project root, used to exercise multi-directory search: adding
  // /mock/lib as an extra directory broadens a /mock/project "hello" search.
  "/mock/lib/extra.go": {
    content: [
      "package lib",
      "",
      "func extra() string {",
      '\treturn "hello from lib"',
      "}",
    ].join("\n"),
  },
};

const MOCK_SYMBOLS: SymbolInfo[] = [
  { name: "main", type: "function", line: 5, signature: "func main()", file: "/mock/project/main.go" },
  { name: "greet", type: "function", line: 10, signature: "func greet(name string)", file: "/mock/project/main.go" },
  { name: "helper", type: "function", line: 4, signature: "func helper() string", file: "/mock/project/util.go" },
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
// Mirror the backend's directory scoping (search_engine.go): search req.directory
// plus any req.directories, deduplicated, and only files under those roots. A
// file is "under" a root if its path starts with root + "/". Also honor
// req.excludePatterns by dropping any file whose path contains an excluded
// path component (matches the backend's matchesPattern component semantics
// closely enough for the mock FS).

// fuzzyWindowMatch mirrors fuzzyBestWindow in search_fuzzy.go — sliding a
// len(query)-wide window over textLower and returning the best
// positional-match count with its start, or null when no window reaches the
// threshold. Both inputs must already be lowercased. Mirrors the frontend's
// findFuzzyMatches scoring.
function fuzzyWindowMatch(textLower: string, queryLower: string, threshold: number): { count: number; pos: number } | null {
	if (queryLower.length === 0 || textLower.length < queryLower.length || textLower.length > 50000) {
		return null;
	}
	let bestCount = -1, bestPos = 0;
	const last = textLower.length - queryLower.length;
	for (let pos = 0; pos <= last; pos++) {
		let count = 0;
		for (let i = 0; i < queryLower.length; i++) {
			if (textLower.charCodeAt(pos + i) === queryLower.charCodeAt(i)) {
				count++;
			}
		}
		if (count >= threshold && count > bestCount) {
			bestCount = count;
			bestPos = pos;
			if (count === queryLower.length) {
				break; // perfect window cannot be beaten
			}
		}
	}
	return bestCount >= 0 ? { count: bestCount, pos: bestPos } : null;
}

// Mirrors backend searchContextLines (search_context.go): 0/unset means
// "default 2", values above 10 are capped at 10.
function resolveContextLines(n: number): number {
	if (!n || n <= 0) return 2;
	return n > 10 ? 10 : n;
}

// Mirrors backend matchExtension loosely: case-insensitive last-segment
// extension match, leading dot optional. Empty allow-list permits everything.
function extensionAllowed(filePath: string, allowedTypes: string[]): boolean {
	if (allowedTypes.length === 0) return true;
	const ext = filePath.split(".").pop()?.toLowerCase() ?? "";
	return allowedTypes.some(
		(t) => t.replace(/^\./, "").toLowerCase() === ext,
	);
}

function buildResults(req: SearchRequest): SearchResult[] {
	const query = req?.query ?? "";
	const caseSensitive = !!req?.caseSensitive;
	const useRegex = !!req?.useRegex;
	const fuzzySearch = !!req?.fuzzySearch;
	const results: SearchResult[] = [];
	if (!query) return results;

	const roots = [req?.directory, ...(req?.directories ?? [])]
		.filter((d): d is string => !!d)
		.map((d) => d.replace(/\/+$/, ""));
	const uniqueRoots = Array.from(new Set(roots));
	const excludes = (req?.excludePatterns ?? []).filter((p) => p.length > 0);
	const allowedTypes = (req?.allowedFileTypes ?? []).filter((t) => t.length > 0);
	const ctxLines = resolveContextLines(Number(req?.contextLines) || 0);

	let matcher: (line: string) => boolean;
	if (useRegex) {
		const re = new RegExp(query, caseSensitive ? "" : "i");
		matcher = (line) => re.test(line);
	} else {
		const needle = caseSensitive ? query : query.toLowerCase();
		matcher = (line) => (caseSensitive ? line : line.toLowerCase()).includes(needle);
	}

	// Phase 1: exact matches (identical to original behavior).
	for (const [filePath, file] of Object.entries(MOCK_FS)) {
		const underRoot =
			uniqueRoots.length === 0 ||
			uniqueRoots.some(
				(root) => filePath === root || filePath.startsWith(root + "/"),
			);
		const excluded = excludes.some((pat) => filePath.split("/").includes(pat));
		if (!underRoot || excluded || !extensionAllowed(filePath, allowedTypes)) continue;
		const lines = file.content.split("\n");
		lines.forEach((line, idx) => {
			if (matcher(line)) {
				results.push({
					filePath,
					lineNum: idx + 1,
					content: line,
					matchedText: query,
					contextBefore: lines.slice(Math.max(0, idx - ctxLines), idx),
					contextAfter: lines.slice(idx + 1, idx + 1 + ctxLines),
				});
			}
		});
	}

	// Phase 2: fuzzy near-miss candidates fill any remaining quota.
	const cap = Number(req?.maxResults) || 1000;
	if (fuzzySearch && !useRegex && results.length < cap) {
		const qLower = query.toLowerCase();
		const threshold = Math.max(1, Math.floor(qLower.length * 0.6));
		for (const [filePath, file] of Object.entries(MOCK_FS)) {
			if (results.length >= cap) break;
			const underRoot =
				uniqueRoots.length === 0 ||
				uniqueRoots.some(
					(root) => filePath === root || filePath.startsWith(root + "/"),
				);
			const excluded = excludes.some((pat) => filePath.split("/").includes(pat));
			if (!underRoot || excluded || !extensionAllowed(filePath, allowedTypes)) continue;
			const lines = file.content.split("\n");
			lines.forEach((line, idx) => {
				if (results.length >= cap) return;
				if (matcher(line)) return; // already an exact hit — skip
				const trimmed = line.trim();
				if (!trimmed || trimmed.length < qLower.length || trimmed.length > 50000) return;
				const lower = trimmed.toLowerCase();
				const hit = fuzzyWindowMatch(lower, qLower, threshold);
				if (hit) {
					const matchedText = trimmed.slice(hit.pos, hit.pos + qLower.length);
					results.push({
						filePath,
						lineNum: idx + 1,
						content: line,
						matchedText,
						contextBefore: lines.slice(Math.max(0, idx - ctxLines), idx),
						contextAfter: lines.slice(idx + 1, idx + 1 + ctxLines),
					});
				}
			});
		}
	}

	// Deterministic order regardless of file iteration order.
	results.sort((a, b) => {
		if (a.filePath !== b.filePath) {
			return a.filePath.localeCompare(b.filePath);
		}
		return a.lineNum - b.lineNum;
	});

	return results.slice(0, cap);
}

let cancelled = false;

interface MockApp {
  IsAppReady(): Promise<boolean>;
  SearchWithProgress(req: SearchRequest): Promise<SearchResult[]>;
  CancelSearch(): Promise<null>;
  ExportSearchResults(results: SearchResult[], format: string): Promise<string>;
  ReplaceInFiles(req: ReplaceRequest): Promise<ReplaceResult>;
  ClearSymbolCache(): Promise<null>;
  ReadFile(filePath: string): Promise<string>;
  ReadFileLog(filePath: string): Promise<string>;
  ShowInFolder(filePath: string): Promise<null>;
  GetAllSymbols(directory: string, maxResults: number): Promise<SymbolInfo[]>;
  SearchSymbols(name: string, directory: string, maxResults: number): Promise<SymbolInfo[]>;
  SelectDirectory(): Promise<string>;
  ValidateDirectory(directory: string): Promise<boolean>;
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
    // frontend's search-progress listener path is exercised. The backend uses
    // "in-progress" for the running state (search_progress.go); "started" is
    // only used in the initial client-side SearchState.
    emit("search-progress", {
      processedFiles: 0,
      totalFiles,
      currentFile: "",
      resultsCount: 0,
      status: "in-progress",
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
    // Mirror the backend's incremental "search-results" batches (see
    // resultBatcher in search_workers.go) so the frontend's streaming append
    // path is exercised, not just the resolved-value path. Two batches with
    // monotonic seq starting at 1 is enough to cover append + ordering; the
    // resolved value below still replaces them, exactly as in production.
    const split = Math.ceil(results.length / 2);
    [results.slice(0, split), results.slice(split)].forEach((batch, i) => {
      if (batch.length > 0) emit("search-results", { seq: i + 1, results: batch });
    });
    emit("search-progress", {
      processedFiles: totalFiles,
      totalFiles,
      currentFile: "",
      resultsCount: results.length,
      failedFiles: 0,
      failedPaths: [],
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

  // Mirrors backend replace.go: literal-only, dry-run by default, atomic
  // apply. Mutates MOCK_FS on apply so a re-search reflects the change.
  ReplaceInFiles: async (req: ReplaceRequest) => {
    if (!req || !req.search) {
      throw new Error("replace: missing search request");
    }
    if (req.search.useRegex) {
      throw new Error("replace is literal-only; disable regex search to replace");
    }
    if (!req.search.query) {
      throw new Error("query is required");
    }
    const query = req.search.query;
    const caseSensitive = !!req.search.caseSensitive;
    // Escape the query so it is treated literally; flags g + optional i.
    const flags = caseSensitive ? "g" : "gi";
    const needle = new RegExp(
      query.replace(/[.*+?^${}()|[\]\\]/g, "\\$&"),
      flags,
    );

    const files: FileReplacement[] = [];
    const filesChanged: Record<string, string[]> = {}; // filePath -> new lines

    for (const [filePath, file] of Object.entries(MOCK_FS)) {
      const lines = file.content.split("\n");
      const changed: string[] = [];
      lines.forEach((line, idx) => {
        if (!needle.test(line)) return;
        needle.lastIndex = 0;
        const newLine = line.replace(needle, req.replacement ?? "");
        if (newLine === line) return; // no-op — never report/write
        files.push({
          filePath,
          lineNum: idx + 1,
          oldLine: line,
          newLine,
        });
        changed[idx] = newLine;
      });
      if (changed.length > 0) {
        filesChanged[filePath] = changed;
      }
    }

    if (req.apply) {
      for (const [filePath, changed] of Object.entries(filesChanged)) {
        const file = MOCK_FS[filePath];
        if (!file) continue;
        const lines = file.content.split("\n");
        changed.forEach((newLine, idx) => {
          if (newLine !== undefined) lines[idx] = newLine;
        });
        file.content = lines.join("\n");
      }
    }

    return {
      files,
      filesChanged: Object.keys(filesChanged).length,
      linesChanged: files.length,
    };
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

  // Mirrors the backend contract (system_integration.go ValidateDirectory):
  // true only for a path that exists and is a readable directory. The mock FS
  // has no directory entries, so a path counts as a directory when any mock
  // file lives under it — that is what makes "/mock/project" valid and a typo
  // like "/mock/projekt" invalid, which is the case useSearch guards against.
  ValidateDirectory: async (directory: string) => {
    if (!directory) return false;
    const root = directory.endsWith("/") ? directory : `${directory}/`;
    return Object.keys(MOCK_FS).some((p) => p.startsWith(root));
  },

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

  // Mirrors the backend contract: a full EditorAvailability record with every
  // known editor key boolean (all false in the browser mock — no editors).
  // Keys derived from EDITOR_CATALOG so the mock can never drift from the
  // real option list; systemdefault stays literal (not cataloged).
  GetAvailableEditors: async () => ({
    ...Object.fromEntries(EDITOR_CATALOG.map(({ key }) => [key, false])),
    systemdefault: false,
  }),
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
  Warning(m: string): void;
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
    Warning: (m) => console.warn("[wails:warn]", m),
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
