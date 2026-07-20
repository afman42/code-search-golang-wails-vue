import { vi } from "vitest";
import { useSearch } from "../../../src/composables/useSearch";

// Mock the localStorage
const localStorageMock = (() => {
  let store: Record<string, string> = {};
  return {
    getItem: (key: string) => store[key] || null,
    setItem: (key: string, value: string) => {
      store[key] = value.toString();
    },
    removeItem: (key: string) => {
      delete store[key];
    },
    clear: () => {
      store = {};
    },
  };
})();

Object.defineProperty(window, "localStorage", {
  value: localStorageMock,
});

// Import the Wails modules for access to their mocked functions
import * as AppModule from "../../../wailsjs/go/main/App";
import * as RuntimeModule from "../../../wailsjs/runtime";

describe("useSearch composable - fixes (#5, #6, #15, #16, #17)", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    localStorage.clear();

    (
      AppModule.SearchWithProgress as any
    ).mockResolvedValue([]);
    (AppModule.SelectDirectory as any).mockResolvedValue(
      "/selected/directory",
    );
    (AppModule.CancelSearch as any).mockResolvedValue(undefined);
    (AppModule.GetEditorDetectionStatus as any).mockResolvedValue({
      availableEditors: {},
      totalAvailable: 0,
      detectionComplete: true,
    });

    // Mock EventsOn to return a cleanup function. Track which event names
    // were registered so the FE 15/16/17 tests can assert on them.
    (RuntimeModule.EventsOn as any).mockReturnValue(vi.fn());
  });

  // Bug #5: truncatedResults was hardcoded to compare against 1000.
  // With maxResults set to 500, a 500-result search should report truncated.
  test("#5 truncatedResults respects data.maxResults, not hardcoded 1000", async () => {
    const { data, searchCode } = useSearch();

    // Set maxResults to 500 (not the default 1000).
    data.directory = "/test";
    data.query = "test";
    data.maxResults = 500;

    // Backend returns exactly 500 results — that fills the limit, so the
    // search WAS truncated (more results may have been dropped server-side).
    const fiveHundredResults = Array.from({ length: 500 }, (_, i) => ({
      filePath: `/test/file${i}.go`,
      lineNum: i + 1,
      content: "test",
      matchedText: "test",
      contextBefore: [],
      contextAfter: [],
    }));
    (AppModule.SearchWithProgress as any).mockResolvedValue(
      fiveHundredResults,
    );

    await searchCode();

    expect(data.searchResults).toHaveLength(500);
    expect(data.truncatedResults).toBe(true);
    expect(data.resultText).toContain("(limited)");
  });

  // Bug #5 (complement): a search that returns FEWER than maxResults must
  // NOT report truncated, regardless of the absolute count.
  test("#5 truncatedResults is false when results < maxResults (even at 1000)", async () => {
    const { data, searchCode } = useSearch();

    data.directory = "/test";
    data.query = "test";
    data.maxResults = 2000; // higher than the 1000 we'll return

    const oneThousandResults = Array.from({ length: 1000 }, (_, i) => ({
      filePath: `/test/file${i}.go`,
      lineNum: i + 1,
      content: "test",
      matchedText: "test",
      contextBefore: [],
      contextAfter: [],
    }));
    (AppModule.SearchWithProgress as any).mockResolvedValue(
      oneThousandResults,
    );

    await searchCode();

    expect(data.searchResults).toHaveLength(1000);
    // 1000 < 2000, so NOT truncated — the old code would have falsely
    // reported truncated because it compared against the hardcoded 1000.
    expect(data.truncatedResults).toBe(false);
    expect(data.resultText).not.toContain("(limited)");
  });

  // Bug #6: the old fallback `Array.isArray(results) ? results : results || []`
  // would pass a truthy non-array (e.g. an object) through to searchResults,
  // crashing downstream iteration. The fix always returns [] for non-arrays.
  test("#6 non-array results fall back to [] (not the truthy object)", async () => {
    const { data, searchCode } = useSearch();

    data.directory = "/test";
    data.query = "test";

    // Backend returns a truthy non-array (e.g. an error object that
    // somehow survived the binding). The old fallback would pass this
    // through; the new one must coerce to [].
    (AppModule.SearchWithProgress as any).mockResolvedValue({
      error: "not an array",
    });

    await searchCode();

    // searchResults MUST be an array, not the object. Downstream code
    // (.map, .forEach, v-for) would crash otherwise.
    expect(Array.isArray(data.searchResults)).toBe(true);
    expect(data.searchResults).toEqual([]);
  });

  test("#6 null results fall back to []", async () => {
    const { data, searchCode } = useSearch();

    data.directory = "/test";
    data.query = "test";

    (AppModule.SearchWithProgress as any).mockResolvedValue(null);

    await searchCode();

    expect(Array.isArray(data.searchResults)).toBe(true);
    expect(data.searchResults).toEqual([]);
  });

  // FE #15: the old code called fetchEditorDetectionStatus after a 1s
  // setTimeout. The fix calls it immediately on composable setup. Verify
  // GetEditorDetectionStatus is called (without fake timers) and the
  // status is reflected.
  test("#15 fetchEditorDetectionStatus is called on setup, not after setTimeout", async () => {
    // We can't easily assert "no setTimeout was used" without fake timers,
    // but we CAN assert that GetEditorDetectionStatus was called and the
    // status was applied. If the composable still used setTimeout(1000),
    // the call wouldn't have happened by the time this test body runs.
    (AppModule.GetEditorDetectionStatus as any).mockResolvedValue({
      availableEditors: { vscode: true, subl: true },
      totalAvailable: 2,
      detectionComplete: true,
    });

    const { data } = useSearch();

    // The composable's fetchEditorDetectionStatus is async and uses a
    // dynamic import() internally, so we need to flush several microtasks
    // for the import + mock resolution to complete. vi.waitFor handles
    // this robustly without coupling to the exact number of ticks.
    await vi.waitFor(() => {
      expect(AppModule.GetEditorDetectionStatus).toHaveBeenCalled();
    });
    await vi.waitFor(() => {
      expect(data.editorDetectionStatus.totalAvailable).toBe(2);
      expect(data.editorDetectionStatus.detectionComplete).toBe(true);
    });
  });

  // FE #16: the old code delayed listener cleanup by 500ms after the
  // search completed. The fix removes the listener immediately in the
  // "completed" handler (and in finally as a safety net). Verify the
  // EventsOn cleanup function is called after searchCode completes.
  test("#16 search-progress listener is cleaned up immediately after search completes", async () => {
    // Track the cleanup function returned by EventsOn for the
    // "search-progress" subscription.
    const progressCleanup = vi.fn();
    (RuntimeModule.EventsOn as any).mockImplementation(
      (eventName: string) => {
        if (eventName === "search-progress") {
          return progressCleanup;
        }
        return vi.fn();
      },
    );

    const { data, searchCode } = useSearch();

    data.directory = "/test";
    data.query = "test";
    (AppModule.SearchWithProgress as any).mockResolvedValue([
      {
        filePath: "/x",
        lineNum: 1,
        content: "test",
        matchedText: "test",
        contextBefore: [],
        contextAfter: [],
      },
    ]);

    await searchCode();

    // The cleanup function for the search-progress listener must have
    // been called by the time searchCode resolves. The old 500ms
    // setTimeout would NOT have called it yet.
    expect(progressCleanup).toHaveBeenCalled();
  });

  // FE #16 (safety net): if the Go call rejects, the listener must still
  // be cleaned up (the old code only cleaned up in the success path's
  // setTimeout, so an error path leaked the listener).
  test("#16 search-progress listener is cleaned up on error", async () => {
    const progressCleanup = vi.fn();
    (RuntimeModule.EventsOn as any).mockImplementation(
      (eventName: string) => {
        if (eventName === "search-progress") {
          return progressCleanup;
        }
        return vi.fn();
      },
    );

    const { data, searchCode } = useSearch();

    data.directory = "/test";
    data.query = "test";
    (AppModule.SearchWithProgress as any).mockRejectedValue(
      new Error("backend exploded"),
    );

    await searchCode();

    // Even on error, the listener must be cleaned up (the finally block
    // is the safety net).
    expect(progressCleanup).toHaveBeenCalled();
  });

  // FE #17: the composable now returns a cleanup() function. Calling it
  // must tear down the search-progress listener (if active) AND the
  // editor-detection subscriptions so the host component can release them
  // on unmount.
  //
  // Note: the search-progress listener is only registered when searchCode
  // is called, so this test triggers a search first to populate
  // currentProgressCleanup, then asserts cleanup() tears it down.
  test("#17 cleanup() tears down all event listeners", async () => {
    const cleanups = {
      searchProgress: vi.fn(),
      editorStart: vi.fn(),
      editorProgress: vi.fn(),
      editorComplete: vi.fn(),
    };
    (RuntimeModule.EventsOn as any).mockImplementation(
      (eventName: string) => {
        switch (eventName) {
          case "search-progress":
            return cleanups.searchProgress;
          case "editor-detection-start":
            return cleanups.editorStart;
          case "editor-detection-progress":
            return cleanups.editorProgress;
          case "editor-detection-complete":
            return cleanups.editorComplete;
          default:
            return vi.fn();
        }
      },
    );

    const { data, searchCode, cleanup } = useSearch();

    // Trigger a search so the search-progress listener is registered.
    data.directory = "/test";
    data.query = "test";
    (AppModule.SearchWithProgress as any).mockResolvedValue([]);
    await searchCode();

    // The search has completed, so the "completed" handler or the finally
    // block has already cleaned up the search-progress listener. That's
    // the FE #16 fix working as expected. Now call cleanup() to tear down
    // the editor-detection subscriptions (which are still active).
    expect(cleanups.editorStart).not.toHaveBeenCalled();
    expect(cleanups.editorProgress).not.toHaveBeenCalled();
    expect(cleanups.editorComplete).not.toHaveBeenCalled();

    cleanup();

    // After cleanup, the editor-detection listeners must be torn down.
    // Without this, they would leak for the app lifetime every time the
    // host component unmounted (#17).
    expect(cleanups.editorStart).toHaveBeenCalled();
    expect(cleanups.editorProgress).toHaveBeenCalled();
    expect(cleanups.editorComplete).toHaveBeenCalled();
  });

  // FE #17 (active listener case): if a search is in progress when
  // cleanup() is called, the search-progress listener must still be torn
  // down. This simulates a user navigating away mid-search.
  test("#17 cleanup() tears down an in-progress search-progress listener", async () => {
    const cleanups = {
      searchProgress: vi.fn(),
      editorStart: vi.fn(),
      editorProgress: vi.fn(),
      editorComplete: vi.fn(),
    };
    (RuntimeModule.EventsOn as any).mockImplementation(
      (eventName: string) => {
        switch (eventName) {
          case "search-progress":
            return cleanups.searchProgress;
          case "editor-detection-start":
            return cleanups.editorStart;
          case "editor-detection-progress":
            return cleanups.editorProgress;
          case "editor-detection-complete":
            return cleanups.editorComplete;
          default:
            return vi.fn();
        }
      },
    );

    // Make SearchWithProgress hang so the listener is still registered
    // when we call cleanup() (simulating unmount mid-search).
    let resolveSearch: (value: any) => void;
    (AppModule.SearchWithProgress as any).mockReturnValue(
      new Promise((resolve) => {
        resolveSearch = resolve;
      }),
    );

    const { data, searchCode, cleanup } = useSearch();
    data.directory = "/test";
    data.query = "test";

    // Start the search but don't await it — it's still in progress.
    void searchCode();
    // Let the EventsOn call execute (it happens before the await resolves).
    await Promise.resolve();

    // The search-progress listener is now registered but NOT yet cleaned
    // up (the search hasn't completed). Calling cleanup() must release it.
    expect(cleanups.searchProgress).not.toHaveBeenCalled();
    cleanup();
    expect(cleanups.searchProgress).toHaveBeenCalled();

    // Let the hanging search resolve so the test doesn't leave a dangling
    // promise (the finally block will try to clean up the already-nulled
    // listener, which is a safe no-op).
    resolveSearch!([]);
  });

  // FE #17 (idempotency): calling cleanup() twice must not throw. The
  // internal cleanup handles are nulled after the first call, so the
  // second call is a no-op.
  test("#17 cleanup() is idempotent", () => {
    const { cleanup } = useSearch();

    expect(() => {
      cleanup();
      cleanup();
    }).not.toThrow();
  });
});
