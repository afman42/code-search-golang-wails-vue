import { reactive } from "vue";
import {
  SelectDirectory as GoSelectDirectory,
  SearchWithProgress as GoSearchWithProgress,
  CancelSearch as GoCancelSearch,
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
    recentSearches: loadRecentSearches() as Array<{
      query: string;
      extension: string;
    }>,
    error: null,
    availableEditors: makeDefaultEditorAvailability(),
    editorDetectionStatus: makeDefaultEditorDetectionStatus(),
  });

  let currentProgressCleanup: (() => void) | null = null;

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

      const processedResults = Array.isArray(results) ? results : results || [];

      data.searchResults = processedResults;
      data.truncatedResults = processedResults.length === 1000;

      data.resultText =
        processedResults.length > 0
          ? `Found ${processedResults.length} matches` +
            (data.truncatedResults ? " (limited)" : "")
          : "No matches found";

      addToRecentSearches();

      if (currentProgressCleanup) {
        setTimeout(() => {
          if (currentProgressCleanup) {
            currentProgressCleanup();
            currentProgressCleanup = null;
          }
        }, 500);
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

  subscribeToEditorDetectionEvents(
    data.availableEditors,
    data.editorDetectionStatus,
  );

  setTimeout(() => {
    fetchEditorDetectionStatus();
  }, 1000);

  return {
    data,
    searchCode,
    cancelSearch,
    selectDirectory,
    formatFilePath,
    highlightMatch,
    copyToClipboard,
    openFileLocation,
  };
}