import { reactive } from "vue";
import {
  ShowInFolder,
  SelectDirectory as GoSelectDirectory,
  SearchWithProgress as GoSearchWithProgress,
} from "../../wailsjs/go/main/App";
import { EventsOn } from "../../wailsjs/runtime";
import { SearchRequest, SearchResult, SearchState } from "../types/search";

// Utility functions for localStorage persistence of recent searches with error handling
const loadRecentSearches = () => {
  try {
    const saved = localStorage.getItem("codeSearchRecentSearches");
    if (saved) {
      return JSON.parse(saved);
    }
    return [];
  } catch (error) {
    console.error("Failed to load recent searches from localStorage:", error);
    return [];
  }
};

const saveRecentSearches = (searches: any[]) => {
  try {
    localStorage.setItem("codeSearchRecentSearches", JSON.stringify(searches));
  } catch (error) {
    console.error("Failed to save recent searches to localStorage:", error);
  }
};

export function useSearch() {
  // Reactive data object containing all state for the component
  const data = reactive<SearchState>({
    directory: "", // Directory path to search in
    query: "", // Search query string
    extension: "", // File extension filter (optional)
    caseSensitive: false, // Whether search should be case sensitive
    useRegex: false, // Whether to treat query as regex
    includeBinary: false, // Whether to include binary files in search
    maxFileSize: 10485760, // Max file size in bytes (10MB default)
    maxResults: 1000, // Max number of results (1000 default)
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
    minFileSize: 0, // Minimum file size filter (bytes)
    excludePatterns: [], // Array of patterns to exclude (e.g., ["node_modules","*.log"])
    recentSearches: loadRecentSearches() as Array<{
      query: string;
      extension: string;
    }>, // Recent searches history
    error: null, // Error message if any
  });

  // Sanitize string for display to prevent XSS
  const sanitizeString = (str: string): string => {
    if (!str) return "";
    return str.replace(/[<>]/g, (match) => {
      return match === "<" ? "&lt;" : "&gt;";
    });
  };

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
      data.resultText = "Please specify a directory to search in";
      data.error = "Directory is required";
      return;
    }

    if (!data.query) {
      data.resultText = "Please enter a search query";
      data.error = "Query is required";
      return;
    }

    // Validate numeric inputs
    if (typeof data.maxFileSize !== "number" || data.maxFileSize < 0) {
      data.resultText =
        "Please enter a valid maximum file size (non-negative number)";
      data.error = "Invalid max file size";
      return;
    }

    if (typeof data.minFileSize !== "number" || data.minFileSize < 0) {
      data.resultText =
        "Please enter a valid minimum file size (non-negative number)";
      data.error = "Invalid min file size";
      return;
    }

    if (typeof data.maxResults !== "number" || data.maxResults <= 0) {
      data.resultText =
        "Please enter a valid maximum number of results (positive number)";
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
    };

    try {
      // Subscribe to progress events
      const progressCleanup = EventsOn(
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

      // Clean up the progress listener after a delay
      setTimeout(() => {
        progressCleanup();
      }, 500);
    } catch (error: any) {
      // Handle any errors that occurred during search
      data.searchResults = [];
      data.resultText = `Error: ${error.message || "Unknown error occurred"}`;
      data.error = error.message || "Unknown error occurred";
      console.error("Search error:", error);
    } finally {
      // Always reset loading state
      data.isSearching = false;
      data.showProgress = false;
    }
  };

  /**
   * Formats file paths for display.
   * Shows just the filename with parent directory for context.
   * @param filePath The full file path to format
   * @returns A shortened version of the file path for display
   */
  const formatFilePath = (filePath: string): string => {
    try {
      // Safety checks to prevent runtime errors
      if (!filePath || typeof filePath !== "string") return "";

      // Cross-platform path handling (support both / and \)
      const normalizedPath = filePath.replace(/\\/g, "/");
      const parts = normalizedPath.split("/");
      const fileName = parts[parts.length - 1];

      // Check if we have at least a file name
      if (!fileName) return filePath; // Return original if we can't parse

      const parentDir = parts.length > 1 ? parts[parts.length - 2] : "";

      // Return "parent/filename" format, or just filename if no parent
      if (parentDir) {
        return `${parentDir}/${fileName}`;
      }
      return fileName;
    } catch (error) {
      console.error("Error in formatFilePath:", error);
      // Return the original path if formatting fails
      return filePath;
    }
  };

  /**
   * Highlights matches in text by wrapping them in HTML mark tags.
   * Handles both regular string matching and regex matching.
   * @param text The text to highlight matches in
   * @param query The search query to highlight
   * @returns The text with highlighted matches
   */
  const highlightMatch = (text: string, query: string): string => {
    try {
      // Safety checks to prevent runtime errors
      if (!text || typeof text !== "string") return "";
      if (!query || typeof query !== "string") return text;

      // Use safe access to component data
      const useRegex =
        data && typeof data.useRegex === "boolean" ? data.useRegex : false;
      const caseSensitive =
        data && typeof data.caseSensitive === "boolean"
          ? data.caseSensitive
          : false;

      // Escape special regex characters if not using regex search
      // This prevents regex special characters in regular search from being interpreted as regex
      let escapedQuery = query;
      if (!useRegex) {
        escapedQuery = query.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");
      }

      // Ensure the query is not empty after escaping to prevent catastrophic backtracking
      if (!escapedQuery) return text;

      // Create a regex with appropriate flags (g for global, i for case insensitive if needed)
      const flags = caseSensitive ? "g" : "gi";
      const regex = new RegExp(`(${escapedQuery})`, flags);

      // Perform the replacement and return the result
      return text.replace(regex, '<mark class="highlight">$1</mark>');
    } catch (error) {
      console.error("Error in highlightMatch:", error);
      // If highlighting fails, return the original text to avoid breaking the UI
      return text;
    }
  };

  /**
   * Copies text to the system clipboard.
   * @param text The text to copy to clipboard
   */
  const copyToClipboard = async (text: string) => {
    try {
      // Validate input
      if (!text || typeof text !== "string") {
        console.warn("Attempted to copy empty or invalid text to clipboard");
        return;
      }

      // Try modern clipboard API first
      if (navigator.clipboard && window.isSecureContext) {
        await navigator.clipboard.writeText(text);
        // Optional: provide user feedback
        console.log("Text copied to clipboard");
      } else {
        // Fallback for older browsers or insecure contexts
        const textArea = document.createElement("textarea");
        textArea.value = text;
        textArea.style.position = "fixed";
        textArea.style.opacity = "0";
        document.body.appendChild(textArea);
        textArea.select();
        document.execCommand("copy");
        document.body.removeChild(textArea);
        console.log("Text copied to clipboard using fallback method");
      }
    } catch (err) {
      console.error("Failed to copy text to clipboard: ", err);
      // Show user-friendly error message
      data.resultText = "Failed to copy text to clipboard";
      data.error = "Clipboard error";
    }
  };

  /**
   * Opens the file's containing folder in the system file manager.
   * Uses the backend function to handle cross-platform compatibility.
   * @param filePath The path to the file whose folder should be opened
   */
  const openFileLocation = async (filePath: string) => {
    try {
      // Validate input
      if (!filePath || typeof filePath !== "string") {
        console.warn("Invalid file path provided to openFileLocation");
        data.resultText = "Invalid file path";
        return;
      }

      await ShowInFolder(filePath);
      console.log("Successfully opened file location:", filePath);
    } catch (error: any) {
      console.error("Failed to open file location:", error);
      // Provide user feedback
      data.resultText = `Could not open file location: ${error.message || "Operation failed"}`;
      data.error = `Open folder error: ${error.message || "Operation failed"}`;
    }
  };

  return {
    data,
    searchCode,
    selectDirectory,
    formatFilePath,
    highlightMatch,
    copyToClipboard,
    openFileLocation,
    sanitizeString,
  };
}
