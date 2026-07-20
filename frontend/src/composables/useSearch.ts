import { reactive } from "vue";
import {
  SelectDirectory as GoSelectDirectory,
  SearchWithProgress as GoSearchWithProgress,
  CancelSearch as GoCancelSearch,
  GetKnownTextExtensions as GoGetKnownTextExtensions,
} from "../../wailsjs/go/main/App";
import { EventsOn } from "../../wailsjs/runtime";
import { SearchRequest, SearchResult, SearchState } from "../types/search";
import {
  loadRecentSearches,
  saveRecentSearches,
} from "../utils/localStorageUtils";
import {
  DEFAULT_MAX_FILE_SIZE,
  DEFAULT_MAX_RESULTS,
  DEFAULT_MIN_FILE_SIZE,
} from "../constants/appConstants";
import { formatFilePath as formatFilePathUtil } from "../utils/fileUtils";
import { highlightMatch as highlightMatchUtil } from "../utils/searchUiUtils";
import {
  copyToClipboardWithToast,
  openFileLocationWithToast,
} from "../utils/toastUtils";
import { toastManager } from "./useToast";
import {
  makeDefaultEditorAvailability,
  makeDefaultEditorDetectionStatus,
  subscribeToEditorDetectionEvents,
} from "./useEditorDetection";

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
    // Sorted list of file extensions the backend treats as universally
    // text (no leading dot). Populated from the backend's
    // GetKnownTextExtensions() binding and consumed by the SearchForm
    // dropdown. Empty on first paint until the call resolves.
    knownTextExtensions: [],
    recentSearches: loadRecentSearches() as Array<{
      query: string;
      extension: string;
    }>,
    error: null,
    availableEditors: makeDefaultEditorAvailability(),
    editorDetectionStatus: makeDefaultEditorDetectionStatus(),
  });

  let currentProgressCleanup: (() => void) | null = null;
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
    } catch (error: any) {
      console.error("Directory selection failed:", error);
      let errorMessage =
        "Directory selection failed. Please enter the directory path manually.";

      if (error && typeof error === "object" && "message" in error) {
        const errorStr = (error as Error).message || String(error);
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
    const newSearch = { query: data.query, extension: data.extension };

    data.recentSearches = data.recentSearches.filter(
      (s: any) =>
        !(s.query === newSearch.query && s.extension === newSearch.extension),
    );

    data.recentSearches.unshift(newSearch);

    if (data.recentSearches.length > 5) {
      data.recentSearches = data.recentSearches.slice(0, 5);
    }

    saveRecentSearches(data.recentSearches);
  };

  const searchCode = async () => {
    data.error = null;

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
      } catch (e: any) {
        data.resultText = `Invalid regex pattern: ${e.message || "Unknown error"}`;
        data.error = `Invalid regex: ${e.message || "Unknown error"}`;
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
    };

    try {
      currentProgressCleanup = EventsOn(
        "search-progress",
        (progressData: any) => {
          if (progressData) {
            data.searchProgress = {
              processedFiles: progressData.processedFiles || 0,
              totalFiles: progressData.totalFiles || 0,
              currentFile: progressData.currentFile || "",
              resultsCount: progressData.resultsCount || 0,
              status: progressData.status || "",
            };

            if (progressData.status === "in-progress") {
              data.resultText = `Searching... Processed ${progressData.processedFiles || 0} of ${progressData.totalFiles || 0} files, found ${progressData.resultsCount || 0} matches`;
            } else if (progressData.status === "completed") {
              data.resultText = `Search completed! Processed ${progressData.processedFiles || 0} files, found ${progressData.resultsCount || 0} matches`;
              if (progressData.resultsCount > 0) {
                toastManager.success(
                  `Search completed! Found ${progressData.resultsCount} matches`,
                  "Search Complete",
                );
              } else {
                toastManager.info(
                  "Search completed! No matches found",
                  "Search Complete",
                );
              }
              // The "completed" event is the terminal one — remove the
              // listener immediately instead of waiting on an arbitrary
              // 500ms timer. The previous setTimeout(500) could either
              // drop a late-arriving "completed" event (if it took >500ms)
              // or silently swallow events that arrived in the 500ms
              // window after the await resolved (#16). Cleaning up here
              // mirrors the "cancelled" handler below.
              if (currentProgressCleanup) {
                currentProgressCleanup();
                currentProgressCleanup = null;
              }
            } else if (progressData.status === "cancelled") {
              data.resultText = "Search was cancelled";
              data.isSearching = false;
              data.showProgress = false;
              toastManager.info("Search was cancelled", "Search Cancelled");
              if (currentProgressCleanup) {
                currentProgressCleanup();
                currentProgressCleanup = null;
              }
            }
          }
        },
      );

      const results = await GoSearchWithProgress(searchRequest);

      // Bug #6: the previous fallback `Array.isArray(results) ? results :
      // results || []` was dead code — when results isn't an array but is
      // truthy (e.g. an object), it would pass the object through to
      // searchResults and crash downstream code that iterates with
      // .map/.forEach. The correct fallback for "not an array" is always [].
      const processedResults = Array.isArray(results) ? results : [];

      data.searchResults = processedResults;
      // Bug #5: the previous check `processedResults.length === 1000`
      // hardcoded the default max results. If the user set maxResults to
      // 500, a 500-result search would wrongly report "not truncated" and
      // a 501-result search (impossible) would never flag. Use the actual
      // configured limit so the truncatedResults flag reflects reality.
      data.truncatedResults =
        processedResults.length >= data.maxResults &&
        data.maxResults > 0;

      data.resultText =
        processedResults.length > 0
          ? `Found ${processedResults.length} matches` +
            (data.truncatedResults ? " (limited)" : "")
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
    } catch (error: any) {
      data.searchResults = [];
      const errorMessage = error.message || "Unknown error occurred";
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
    } catch (error: any) {
      console.error("Cancel search failed:", error);
      const errorMessage = error.message || "Unknown error";
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

  const fetchEditorDetectionStatus = async () => {
    try {
      const { GetEditorDetectionStatus } = await import(
        "../../wailsjs/go/main/App"
      );
      const status = await GetEditorDetectionStatus();
      if (status) {
        data.availableEditors =
          status.availableEditors || data.availableEditors;
        data.editorDetectionStatus.availableEditors =
          status.availableEditors || data.availableEditors;
        data.editorDetectionStatus.totalAvailable = status.totalAvailable || 0;
        if (status.totalAvailable !== undefined) {
          data.editorDetectionStatus.message = `Detection complete! Found ${status.totalAvailable} editor(s).`;
        }
        data.editorDetectionStatus.detectionComplete = true;
        data.editorDetectionStatus.detectingEditors = false;
      }
    } catch (error: any) {
      console.error("Failed to fetch editor detection status:", error);
    }
  };

  // Subscribe to the live editor-detection events (start/progress/complete).
  // Capture the cleanup function so the composable's cleanup() can tear these
  // listeners down on unmount instead of leaking them for the app lifetime
  // (#17).
  editorDetectionCleanup = subscribeToEditorDetectionEvents(
    data.availableEditors,
    data.editorDetectionStatus,
  );

  // Fetch the editor-detection status immediately instead of after a 1s
  // setTimeout (#15). The previous timer was a race-condition workaround for
  // the case where the backend's "editor-detection-complete" event had
  // already fired before subscribeToEditorDetectionEvents registered its
  // listener. Calling GetEditorDetectionStatus immediately handles that
  // case directly: if detection is already complete, the status is reflected
  // at first paint; if it's still running, the subscriptions above will
  // catch the live progress. No timer, no race.
  void fetchEditorDetectionStatus();

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
    } catch (error: any) {
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
  };
}