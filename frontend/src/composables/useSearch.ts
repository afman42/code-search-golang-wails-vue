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

export function useSearch() {
  // Reactive data object containing all state for the component
  const data = reactive<SearchState>({
    directory: "", // Directory path to search in
    query: "", // Search query string
    extension: "", // File extension filter (optional)
    caseSensitive: false, // Whether search should be case sensitive
    useRegex: false, // Whether to treat query as regex
    includeBinary: false, // Whether to include binary files in search
    maxFileSize: DEFAULT_MAX_FILE_SIZE, // Max file size in bytes (10MB default)
    maxResults: DEFAULT_MAX_RESULTS, // Max number of results (1000 default)
    searchSubdirs: true, // Whether to search subdirectories
    resultText: "Please enter search parameters below 👇", // Status text
    searchResults: [] as SearchResult[], // Search results array
    truncatedResults: false, // Whether results were truncated (due to limit)
    isSearching: false, // Whether a search is currently in progress
    searchProgress: {
      processedFiles: 0,
      totalFiles: 0,
      currentFile: "",
      resultsCount: 0,
      status: "",
    }, // Progress information
    showProgress: false, // Whether to show progress bar
    minFileSize: DEFAULT_MIN_FILE_SIZE, // Minimum file size filter (bytes)
    excludePatterns: [], // Array of patterns to exclude (e.g., ["node_modules","*.log"])
    allowedFileTypes: [], // Array of file extensions that are allowed (empty means all allowed)
    recentSearches: loadRecentSearches() as Array<{
      query: string;
      extension: string;
    }>, // Recent searches history
    error: null, // Error message if any
    availableEditors: {
      vscode: false,
      vscodium: false,
      sublime: false,
      atom: false,
      jetbrains: false,
      geany: false,
      goland: false,
      pycharm: false,
      intellij: false,
      webstorm: false,
      phpstorm: false,
      clion: false,
      rider: false,
      androidstudio: false,
      systemdefault: true,
      emacs: false,
      neovide: false,
      codeblocks: false,
      devcpp: false,
      notepadplusplus: false,
      visualstudio: false,
      eclipse: false,
      netbeans: false,
    }, // Editor availability initialized as empty
    editorDetectionStatus: {
      detectionComplete: false,
      totalAvailable: 0,
      message: "Initializing editor detection...",
      detectionProgress: 0,
      detectingEditors: true,
      detectedEditors: [],
      availableEditors: {
        vscode: false,
        vscodium: false,
        sublime: false,
        atom: false,
        jetbrains: false,
        geany: false,
        goland: false,
        pycharm: false,
        intellij: false,
        webstorm: false,
        phpstorm: false,
        clion: false,
        rider: false,
        androidstudio: false,
        systemdefault: true,
        emacs: false,
        neovide: false,
        codeblocks: false,
        devcpp: false,
        notepadplusplus: false,
        visualstudio: false,
        eclipse: false,
        netbeans: false,
      },
    },
  });

  // Store the progress listener cleanup function
  let currentProgressCleanup: (() => void) | null = null;

  /**
   * Handles directory selection by opening a native system directory picker.
   * Uses the Go backend function to show a cross-platform directory selection dialog.
   */
  const selectDirectory = async () => {
    try {
      // Call the backend Go function to open the system directory picker
      const selectedDir = await GoSelectDirectory("Select Directory to Search");

      // If a directory was selected, update the input field
      if (selectedDir && typeof selectedDir === "string") {
        data.directory = selectedDir;
        data.error = null; // Clear any previous errors
      } else if (selectedDir === "") {
        // User cancelled the dialog
        console.log("Directory selection was cancelled by user");
        toastManager.info(
          "Directory selection was cancelled by user",
          "Directory Selection Cancel",
        );
      }
    } catch (error: any) {
      console.error("Directory selection failed:", error);

      // Provide user-friendly error message based on the error
      let errorMessage =
        "Directory selection failed. Please enter the directory path manually.";

      if (error && typeof error === "object" && "message" in error) {
        const errorStr = (error as Error).message || String(error);

        // Special handling for different error types
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

      data.resultText = errorMessage;
      data.error = errorMessage;
      toastManager.error(errorMessage, "Directory Selection Error");
    }
  };

  /**
   * Performs the code search operation with progress updates using the backend.
   * Handles validation, search execution, result processing, and error handling.
   * Also manages recent searches functionality.
   */
  const searchCode = async () => {
    // Clear previous errors
    data.error = null;

    // Validate required inputs before starting search
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

    // Validate numeric inputs
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

    // Set loading state
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

    // Prepare the query based on whether we're using regex
    let query = data.query;
    if (data.useRegex) {
      // If using regex, validate the pattern first to prevent errors
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

    // Prepare search request with current parameters
    const searchRequest: SearchRequest = {
      directory: data.directory,
      query: query,
      extension: data.extension,
      caseSensitive: data.caseSensitive,
      includeBinary: data.includeBinary,
      maxFileSize: Number(data.maxFileSize) || 10485760, // Ensure numeric value
      minFileSize: Number(data.minFileSize) || 0, // Ensure numeric value
      maxResults: Number(data.maxResults) || 1000, // Ensure numeric value
      searchSubdirs: data.searchSubdirs,
      useRegex: data.useRegex,
      excludePatterns: Array.isArray(data.excludePatterns)
        ? data.excludePatterns.filter((s) => s.length > 0) // Remove empty patterns
        : [],
      allowedFileTypes: Array.isArray(data.allowedFileTypes)
        ? data.allowedFileTypes.filter((s) => s.length > 0) // Remove empty extensions
        : [],
    };

    try {
      // Subscribe to progress events
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

            // Update the result status to show progress
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
              // Clean up the progress listener immediately on cancellation
              if (currentProgressCleanup) {
                currentProgressCleanup();
                currentProgressCleanup = null;
              }
            }
          }
        },
      );

      // Execute the search using backend function with progress
      const results = await GoSearchWithProgress(searchRequest);

      // Ensure results is always an array, even if backend returns null/undefined
      const processedResults = Array.isArray(results) ? results : results || [];

      data.searchResults = processedResults;

      // Check if results were truncated due to backend limit
      data.truncatedResults = processedResults.length === 1000; // backend limit

      // Update result text with final count
      data.resultText =
        processedResults.length > 0
          ? `Found ${processedResults.length} matches` +
            (data.truncatedResults ? " (limited)" : "")
          : "No matches found";

      // Add this search to recent searches history
      const newSearch = {
        query: data.query,
        extension: data.extension,
      };

      // Remove any duplicate of this search to avoid duplicates in history
      data.recentSearches = data.recentSearches.filter(
        (s: any) =>
          !(s.query === newSearch.query && s.extension === newSearch.extension),
      );

      // Add to front of list (most recent first)
      data.recentSearches.unshift(newSearch);

      // Keep only the last 5 searches to prevent localStorage bloat
      if (data.recentSearches.length > 5) {
        data.recentSearches = data.recentSearches.slice(0, 5);
      }

      // Persist recent searches to localStorage
      saveRecentSearches(data.recentSearches);

      // Clean up the progress listener after a delay if not already cleaned up
      if (currentProgressCleanup) {
        setTimeout(() => {
          if (currentProgressCleanup) {
            currentProgressCleanup();
            currentProgressCleanup = null;
          }
        }, 500);
      }
    } catch (error: any) {
      // Handle any errors that occurred during search
      data.searchResults = [];
      const errorMessage = error.message || "Unknown error occurred";
      data.resultText = `Error: ${errorMessage}`;
      data.error = errorMessage;
      toastManager.error(errorMessage, "Search Error");
      console.error("Search error:", error);
    } finally {
      // Always reset loading state
      data.isSearching = false;
      data.showProgress = false;
    }
  };

  /**
   * Cancels the active search operation.
   * Calls the backend CancelSearch function to terminate the running search.
   */
  const cancelSearch = async () => {
    try {
      // Call the backend function to cancel the search
      await GoCancelSearch();

      // Reset search state after cancellation
      data.isSearching = false;
      data.showProgress = false;

      // Update progress status to show cancellation
      data.searchProgress.status = "cancelled";

      // Update UI to reflect cancelled search
      data.searchResults = []; // Clear any partial results

      // Clean up the progress listener if it exists
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

      // Still reset UI state even if the cancel call failed
      data.isSearching = false;
      data.showProgress = false;
      data.showProgress = false;
    }
  };

  /**
   * Wrapper function for formatFilePath utility to maintain backward compatibility
   */
  const formatFilePath = (filePath: string): string => {
    return formatFilePathUtil(filePath);
  };

  /**
   * Wrapper function for highlightMatch utility to maintain backward compatibility
   */
  const highlightMatch = (text: string, query: string): string => {
    return highlightMatchUtil(text, query, data);
  };

  /**
   * Wrapper function for copyToClipboard utility with toast notifications
   */
  const copyToClipboard = async (text: string) => {
    return await copyToClipboardWithToast(text);
  };

  /**
   * Wrapper function for openFileLocation utility with toast notifications
   */
  const openFileLocation = async (filePath: string) => {
    return await openFileLocationWithToast(filePath);
  };

  // Subscribe to editor detection events
  const subscribeToEditorDetectionEvents = () => {
    // Import EventsOn from Wails runtime
    import("../../wailsjs/runtime")
      .then((runtime) => {
        // Listen for editor detection start
        const cleanupStart = runtime.EventsOn(
          "editor-detection-start",
          (eventData: any) => {
            data.editorDetectionStatus = {
              detectionComplete: false,
              totalAvailable: 0,
              message: eventData?.message || "Starting editor detection...",
              detectionProgress: 0,
              detectingEditors: true,
              detectedEditors: [],
              availableEditors: data.availableEditors,
            };
          },
        );

        // Listen for editor detection progress
        const cleanupProgress = runtime.EventsOn(
          "editor-detection-progress",
          (eventData: any) => {
            if (eventData) {
              data.editorDetectionStatus.message =
                eventData.message || "Detecting editors...";
              data.editorDetectionStatus.detectionProgress =
                Math.round(eventData.progress) || 0;

              // Add detected editor to the list if it's available
              if (eventData.available && eventData.editor) {
                if (
                  !data.editorDetectionStatus.detectedEditors.includes(
                    eventData.editor,
                  )
                ) {
                  data.editorDetectionStatus.detectedEditors.push(
                    eventData.editor,
                  );
                }
              }
            }
          },
        );

        // Listen for editor detection completion
        const cleanupComplete = runtime.EventsOn(
          "editor-detection-complete",
          (eventData: any) => {
            data.editorDetectionStatus.detectionComplete = true;
            data.editorDetectionStatus.totalAvailable =
              eventData?.totalFound || 0;
            data.editorDetectionStatus.message = `Detection complete! Found ${eventData?.totalFound || 0} editor(s).`;
            data.editorDetectionStatus.detectionProgress = 100;
            data.editorDetectionStatus.detectingEditors = false;

            // Get final editor availability status
            fetchEditorDetectionStatus();
          },
        );

        // Cleanup function to remove event listeners when needed
        return () => {
          if (cleanupStart) cleanupStart();
          if (cleanupProgress) cleanupProgress();
          if (cleanupComplete) cleanupComplete();
        };
      })
      .catch((err) => {
        console.error(
          "Error importing runtime for editor detection events:",
          err,
        );
      });
  };

  // Function to fetch complete editor detection status
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

  // Load available editors during initialization
  subscribeToEditorDetectionEvents();

  // Initially fetch editor detection status
  setTimeout(() => {
    fetchEditorDetectionStatus();
  }, 1000); // Slight delay to allow detection to start

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
