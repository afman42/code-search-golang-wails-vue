import { reactive } from "vue";
import {
  SelectDirectory as GoSelectDirectory,
  SearchWithProgress as GoSearchWithProgress,
  CancelSearch as GoCancelSearch,
  GetKnownTextExtensions as GoGetKnownTextExtensions,
} from "@wails/go/main/App";
import { EventsOn } from "@wails/runtime";
import type { RecentSearch, SearchProgress, SearchRequest, SearchResult, SearchState } from "@/types";
import {
  loadRecentSearches,
  saveRecentSearches,
  recentSearchKey,
} from "@/utils/localStorageUtils";
import {
  DEFAULT_MAX_FILE_SIZE,
  DEFAULT_MAX_RESULTS,
  DEFAULT_MIN_FILE_SIZE,
} from "@/constants/appConstants";
import { formatFilePath as formatFilePathUtil } from "@/utils/fileUtils";
import { highlightMatch as highlightMatchUtil } from "@/utils/searchUiUtils";
import {
  copyToClipboardWithToast,
  openFileLocationWithToast,
} from "@/utils/toastUtils";
import { toastManager } from "@/composables/useToast";
import {
  makeDefaultEditorAvailability,
  makeDefaultEditorDetectionStatus,
  startEditorDetection,
} from "@/composables/useEditorDetection";
import { findFuzzyMatches } from "@/utils/fuzzyMatch";
import { toErrorMessage } from "@/utils/errorUtils";

// Coerce an untyped Wails "search-progress" event payload into a SearchProgress.
// The payload crosses the JS bridge as `unknown`; read each field defensively
// so a missing/renamed field degrades to a default rather than throwing.
function coerceProgress(payload: unknown): SearchProgress {
  const p = (payload && typeof payload === "object" ? payload : {}) as Record<string, unknown>;
  const num = (v: unknown): number => (typeof v === "number" ? v : 0);
  const str = (v: unknown): string => (typeof v === "string" ? v : "");
  return {
    processedFiles: num(p.processedFiles),
    totalFiles: num(p.totalFiles),
    currentFile: str(p.currentFile),
    resultsCount: num(p.resultsCount),
    status: str(p.status),
  };
}

export function useSearch() {
  const data = reactive<SearchState>({
    directory: "",
    query: "",
    extension: "",
    caseSensitive: false,
    useRegex: false,
    includeBinary: false,
    maxFileSize: DEFAULT_MAX_FILE_SIZE,
    maxResults: DEFAULT_MAX_RESULTS,
    searchSubdirs: true,
    resultText: "Please enter search parameters below 👇",
    searchResults: [] as SearchResult[],
    truncatedResults: false,
    isSearching: false,
    searchProgress: {
      processedFiles: 0,
      totalFiles: 0,
      currentFile: "",
      resultsCount: 0,
      status: "",
    },
    showProgress: false,
    minFileSize: DEFAULT_MIN_FILE_SIZE,
    excludePatterns: [],
    allowedFileTypes: [],
    knownTextExtensions: [],
    recentSearches: loadRecentSearches() as RecentSearch[],
    error: null,
    availableEditors: makeDefaultEditorAvailability(),
    editorDetectionStatus: makeDefaultEditorDetectionStatus(),
    fuzzySearch: false,
    contextLines: 3,
    directories: [],
  });

  let currentProgressCleanup: (() => void) | null = null;
  // Set to true by cancelSearch so the in-flight searchCode continuation knows
  // a cancel happened while it was awaiting GoSearchWithProgress. Without this,
  // cancelSearch() clears the results but the awaited promise then repopulates
  // them (and overwrites the "cancelled" message) with partial matches.
  let wasCancelled = false;
  // editorDetectionCleanup releases the editor-detection event subscriptions
  // (start/progress/complete) that subscribeToEditorDetectionEvents
  // registers. Captured here so the composable's cleanup() can tear them
  // down on unmount instead of leaking the listeners for the app lifetime
  // (#17).
  let editorDetectionCleanup: (() => void) | null = null;

  const selectDirectory = async () => {
    try {
      const selectedDir = await GoSelectDirectory("Select Directory to Search");

      if (selectedDir && typeof selectedDir === "string") {
        data.directory = selectedDir;
        data.error = null;
        toastManager.success("Directory selection add success");
      } else if (selectedDir === "") {
        console.log("Directory selection was cancelled by user");
        toastManager.info(
          "Directory selection was cancelled by user",
          "Directory Selection Cancel",
        );
      }
    } catch (error: unknown) {
      console.error("Directory selection failed:", error);
      let errorMessage =
        "Directory selection failed. Please enter the directory path manually.";

      if (error && typeof error === "object" && "message" in error) {
        const errorStr = toErrorMessage(error);
        if (errorStr.includes("not implemented")) {
          errorMessage =
            "Directory selection is not available on this platform.\nPlease enter the directory path manually.";
        } else if (errorStr.includes("no suitable directory picker")) {
          errorMessage =
            "No directory picker found. Please install zenity (GNOME) or kdialog (KDE) to use the directory picker,\nor enter the directory path manually.";
        } else {
          errorMessage = `Directory selection failed: ${errorStr}\nPlease enter the directory path manually.`;
        }
      }

      data.error = errorMessage;
      toastManager.error(errorMessage, "Directory Selection Error");
    }
  };

  const addToRecentSearches = () => {
    // Persist the directory too so a history/suggestion entry can be re-run
    // against the same folder even after the user browses elsewhere.
    const newSearch = {
      query: data.query,
      extension: data.extension,
      directory: data.directory || "",
    };

    data.recentSearches = data.recentSearches.filter(
      (s) => recentSearchKey(s) !== recentSearchKey(newSearch),
    );

    data.recentSearches.unshift(newSearch);

    if (data.recentSearches.length > 5) {
      data.recentSearches = data.recentSearches.slice(0, 5);
    }

    saveRecentSearches(data.recentSearches);
  };

  const searchCode = async () => {
    data.error = null;
    wasCancelled = false;

    if (!data.directory) {
      toastManager.error(
        "Please specify a directory to search in",
        "Directory Required",
      );
      data.error = "Directory is required";
      return;
    }

    if (!data.query) {
      toastManager.error("Please enter a search query", "Query Required");
      data.error = "Query is required";
      return;
    }

    if (typeof data.maxFileSize !== "number" || data.maxFileSize < 0) {
      toastManager.error(
        "Please enter a valid maximum file size (non-negative number)",
        "Invalid File Size",
      );
      data.error = "Invalid max file size";
      return;
    }

    if (typeof data.minFileSize !== "number" || data.minFileSize < 0) {
      toastManager.error(
        "Please enter a valid minimum file size (non-negative number)",
        "Invalid File Size",
      );
      data.error = "Invalid min file size";
      return;
    }

    if (typeof data.maxResults !== "number" || data.maxResults <= 0) {
      toastManager.error(
        "Please enter a valid maximum number of results (positive number)",
        "Invalid Results Limit",
      );
      data.error = "Invalid max results";
      return;
    }

    data.isSearching = true;
    data.showProgress = true;
    data.searchResults = [];
    data.truncatedResults = false;
    data.resultText = "Searching...";
    data.error = null;
    data.searchProgress = {
      processedFiles: 0,
      totalFiles: 0,
      currentFile: "",
      resultsCount: 0,
      status: "started",
    };

    let query = data.query;
    if (data.useRegex) {
      try {
        new RegExp(query);
      } catch (e: unknown) {
        const msg = toErrorMessage(e);
        data.resultText = `Invalid regex pattern: ${msg}`;
        data.error = `Invalid regex: ${msg}`;
        data.isSearching = false;
        data.showProgress = false;
        return;
      }
    }

    const searchRequest: SearchRequest = {
      directory: data.directory,
      query: query,
      extension: data.extension,
      caseSensitive: data.caseSensitive,
      includeBinary: data.includeBinary,
      maxFileSize: Number(data.maxFileSize) || 10485760,
      minFileSize: Number(data.minFileSize) || 0,
      maxResults: Number(data.maxResults) || 1000,
      searchSubdirs: data.searchSubdirs,
      useRegex: data.useRegex,
      excludePatterns: Array.isArray(data.excludePatterns)
        ? data.excludePatterns.filter((s) => s.length > 0)
        : [],
      allowedFileTypes: Array.isArray(data.allowedFileTypes)
        ? data.allowedFileTypes.filter((s) => s.length > 0)
        : [],
      fuzzySearch: data.fuzzySearch,
      contextLines: data.contextLines,
      directories: Array.isArray(data.directories)
        ? data.directories.filter((s) => s.length > 0)
        : [],
    };

    try {
      currentProgressCleanup = EventsOn(
        "search-progress",
        (payload: unknown) => {
          const progress = coerceProgress(payload);
          data.searchProgress = progress;

          if (progress.status === "in-progress") {
            data.resultText = `Searching... Processed ${progress.processedFiles} of ${progress.totalFiles} files, found ${progress.resultsCount} matches`;
          } else if (progress.status === "completed") {
            data.resultText = `Search completed! Processed ${progress.processedFiles} files, found ${progress.resultsCount} matches`;
            if (progress.resultsCount > 0) {
              toastManager.success(
                `Search completed! Found ${progress.resultsCount} matches`,
                "Search Complete",
              );
            } else {
              toastManager.info(
                "Search completed! No matches found",
                "Search Complete",
              );
            }
            // The "completed" event is the terminal one — remove the listener
            // immediately instead of waiting on an arbitrary 500ms timer (#16).
            if (currentProgressCleanup) {
              currentProgressCleanup();
              currentProgressCleanup = null;
            }
          } else if (progress.status === "cancelled") {
            data.resultText = "Search was cancelled";
            data.isSearching = false;
            data.showProgress = false;
            toastManager.info("Search was cancelled", "Search Cancelled");
            if (currentProgressCleanup) {
              currentProgressCleanup();
              currentProgressCleanup = null;
            }
          }
        },
      );

      const results = await GoSearchWithProgress(searchRequest);

      // If the user cancelled while we were awaiting the backend, cancelSearch()
      // already reset the state — don't repopulate results or update the
      // status text from this partial/empty response.
      if (wasCancelled) {
        return;
      }

      // Bug #6: the previous fallback `Array.isArray(results) ? results :
      // results || []` was dead code — when results isn't an array but is
      // truthy (e.g. an object), it would pass the object through to
      // searchResults and crash downstream code that iterates with
      // .map/.forEach. The correct fallback for "not an array" is always [].
      const processedResults = Array.isArray(results) ? results : [];

      data.searchResults = processedResults;

      if (data.fuzzySearch && !data.useRegex) {
        const fuzzyResults = processedResults
          .map((r: SearchResult) => {
            const isFuzzy = !r.content.toLowerCase().includes(query.toLowerCase());
            if (!isFuzzy) return r;
            const matches = findFuzzyMatches(r.content, query);
            if (matches.length === 0) return null;
            return {
              ...r,
              fuzzyMatch: true,
              similarityScore: matches[0].matchedChars.length / query.length,
            } as SearchResult;
          })
          .filter((r: SearchResult | null): r is SearchResult => r !== null);
        data.searchResults = fuzzyResults;
      }

      data.truncatedResults =
        data.searchResults.length >= data.maxResults &&
        data.maxResults > 0;

      data.resultText =
        data.searchResults.length > 0
          ? `Found ${data.searchResults.length} matches` +
            (data.truncatedResults ? " (limited)" : "") +
            (data.fuzzySearch && !data.useRegex ? " (fuzzy)" : "")
          : "No matches found";

      addToRecentSearches();

      // Safety net: if the "completed" event handler above already cleaned
      // up the listener, currentProgressCleanup is null and this is a no-op.
      // If the Go call returned without emitting "completed" (e.g. an error
      // path that the catch block below also handles, or a race where the
      // event was lost), this ensures we don't leak the listener. The
      // previous 500ms setTimeout was an arbitrary delay that could drop
      // late events; removing the listener synchronously here is both
      // simpler and correct (#16).
      if (currentProgressCleanup) {
        currentProgressCleanup();
        currentProgressCleanup = null;
      }
    } catch (error: unknown) {
      data.searchResults = [];
      const errorMessage = toErrorMessage(error);
      data.error = errorMessage;
      toastManager.error(errorMessage, "Search Error");
      console.error("Search error:", error);
    } finally {
      data.isSearching = false;
      data.showProgress = false;
      // Final safety net for listener cleanup: if we got here through an
      // error path that didn't hit the "completed" handler, make sure the
      // search-progress listener is released (#16, #17).
      if (currentProgressCleanup) {
        currentProgressCleanup();
        currentProgressCleanup = null;
      }
    }
  };

  const cancelSearch = async () => {
    wasCancelled = true;
    try {
      await GoCancelSearch();
      data.isSearching = false;
      data.showProgress = false;
      data.searchProgress.status = "cancelled";
      data.searchResults = [];

      if (currentProgressCleanup) {
        currentProgressCleanup();
        currentProgressCleanup = null;
      }
    } catch (error: unknown) {
      console.error("Cancel search failed:", error);
      const errorMessage = toErrorMessage(error, "Unknown error");
      data.resultText = `Cancel failed: ${errorMessage}`;
      data.error = `Cancel error: ${errorMessage}`;
      toastManager.error(
        `Failed to cancel search: ${errorMessage}`,
        "Cancel Error",
      );
      data.isSearching = false;
      data.showProgress = false;
    }
  };

  const formatFilePath = (filePath: string): string => {
    return formatFilePathUtil(filePath);
  };

  const highlightMatch = (text: string, query: string): string => {
    return highlightMatchUtil(text, query, data);
  };

  const copyToClipboard = async (text: string) => {
    return await copyToClipboardWithToast(text);
  };

  const openFileLocation = async (filePath: string) => {
    return await openFileLocationWithToast(filePath);
  };

  // Editor detection is its own concern — startEditorDetection owns the event
  // subscriptions and the initial status pull. We keep the cleanup handle so
  // this composable's cleanup() can release the listeners on unmount (#17).
  editorDetectionCleanup = startEditorDetection(
    data.availableEditors,
    data.editorDetectionStatus,
  );

  // Populate the known-text extension list from the backend so the
  // "Allowed File Types" dropdown is driven by the same source of truth
  // that decides whether a file gets binary-probed. Failures are
  // non-fatal — the dropdown just stays empty and the custom-input field
  // still lets users type any extension manually.
  const fetchKnownTextExtensions = async () => {
    try {
      const exts = await GoGetKnownTextExtensions();
      if (Array.isArray(exts)) {
        data.knownTextExtensions = exts;
      }
    } catch (error: unknown) {
      console.error("Failed to load known text extensions:", error);
    }
  };
  void fetchKnownTextExtensions();

  // cleanup tears down every listener this composable registered so the
  // caller can release them on component unmount. Without this the
  // search-progress and editor-detection listeners would leak for the app
  // lifetime every time the host component unmounted (#17).
  const cleanup = () => {
    if (currentProgressCleanup) {
      currentProgressCleanup();
      currentProgressCleanup = null;
    }
    if (editorDetectionCleanup) {
      editorDetectionCleanup();
      editorDetectionCleanup = null;
    }
  };

  return {
    data,
    searchCode,
    cancelSearch,
    selectDirectory,
    formatFilePath,
    highlightMatch,
    copyToClipboard,
    openFileLocation,
    cleanup,
    focusSearch: () => {
      const input = document.getElementById('query') as HTMLInputElement;
      if (input) input.focus();
    },
    executeSearch: () => searchCode(),
    clearSearch: () => {
      data.query = '';
      data.searchResults = [];
      data.resultText = 'Please enter search parameters below 👇';
      data.error = null;
    }
  };
}