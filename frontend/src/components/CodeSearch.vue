<script lang="ts" setup>
import { reactive } from "vue";
import {
  SearchCode,
  ShowInFolder,
  SelectDirectory as GoSelectDirectory,
} from "../../wailsjs/go/main/App";

// Define TypeScript interfaces for type safety
interface SearchResult {
  filePath: string;
  lineNum: number;
  content: string;
  matchedText: string;
}

interface SearchRequest {
  directory: string;
  query: string;
  extension: string;
  caseSensitive: boolean;
  includeBinary: boolean;
  maxFileSize: number;
  maxResults: number;
  searchSubdirs: boolean;
}

// Utility functions for localStorage persistence of recent searches
const loadRecentSearches = () => {
  const saved = localStorage.getItem("codeSearchRecentSearches");
  return saved ? JSON.parse(saved) : [];
};

const saveRecentSearches = (searches: any[]) => {
  localStorage.setItem("codeSearchRecentSearches", JSON.stringify(searches));
};

// Reactive data object containing all state for the component
const data = reactive({
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
  recentSearches: loadRecentSearches() as Array<{
    query: string;
    extension: string;
  }>, // Recent searches history
});

/**
 * Handles directory selection by opening a native system directory picker.
 * Uses the Go backend function to show a cross-platform directory selection dialog.
 */
async function selectDirectory() {
  try {
    // Call the backend Go function to open the system directory picker
    const selectedDir = await GoSelectDirectory("Select Directory to Search");

    // If a directory was selected, update the input field
    if (selectedDir) {
      data.directory = selectedDir;
    }
  } catch (error) {
    console.error("Directory selection failed:", error);

    // Provide user-friendly error message based on the error
    if (error && typeof error === "object" && "message" in error) {
      const errorMessage = (error as Error).message;

      // Special handling for different error types
      if (errorMessage.includes("not implemented")) {
        alert(
          "Directory selection is not available on this platform.\nPlease enter the directory path manually.",
        );
      } else if (errorMessage.includes("no suitable directory picker")) {
        alert(
          "No directory picker found. Please install zenity (GNOME) or kdialog (KDE) to use the directory picker,\nor enter the directory path manually.",
        );
      } else {
        alert(
          `Directory selection failed: ${errorMessage}\nPlease enter the directory path manually.`,
        );
      }
    } else {
      alert(
        "Directory selection failed. Please enter the directory path manually.",
      );
    }
  }
}

/**
 * Performs the code search operation using the backend.
 * Handles validation, search execution, result processing, and error handling.
 * Also manages recent searches functionality.
 */
async function searchCode() {
  // Validate required inputs before starting search
  if (!data.directory || !data.query) {
    data.resultText = "Please specify both directory and search query";
    return;
  }

  // Set loading state
  data.isSearching = true;
  data.searchResults = [];
  data.truncatedResults = false;
  data.resultText = "Searching...";

  // Prepare the query based on whether we're using regex
  let query = data.query;
  if (data.useRegex) {
    // If using regex, validate the pattern first to prevent errors
    try {
      new RegExp(query);
    } catch (e) {
      data.resultText = "Invalid regex pattern";
      data.isSearching = false;
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
    maxFileSize: data.maxFileSize,
    maxResults: data.maxResults,
    searchSubdirs: data.searchSubdirs,
  };

  try {
    // Execute the search using backend function
    const results = await SearchCode(searchRequest);
    console.log(results);
    // Ensure results is always an array, even if backend returns null/undefined
    const processedResults = Array.isArray(results) ? results : results || [];

    data.searchResults = processedResults;

    // Check if results were truncated due to backend limit
    data.truncatedResults =
      processedResults && processedResults.length === 1000; // backend limit

    // Update result text with count
    data.resultText =
      processedResults && processedResults.length > 0
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
    console.log(data);
  } catch (error: any) {
    // Handle any errors that occurred during search
    data.searchResults = [];
    data.resultText = `Error: ${error.message || "Unknown error occurred"}`;
    console.error("Search error:", error);
  } finally {
    // Always reset loading state
    data.isSearching = false;
  }
}

/**
 * Formats file paths for display.
 * Shows just the filename with parent directory for context.
 * @param filePath The full file path to format
 * @returns A shortened version of the file path for display
 */
function formatFilePath(filePath: string): string {
  // Split path into components and extract filename and parent directory
  const parts = filePath.split("/");
  const fileName = parts[parts.length - 1];
  const parentDir = parts.length > 1 ? parts[parts.length - 2] : "";

  // Return "parent/filename" format, or just filename if no parent
  if (parentDir) {
    return `${parentDir}/${fileName}`;
  }
  return fileName;
}

/**
 * Highlights matches in text by wrapping them in HTML mark tags.
 * Handles both regular string matching and regex matching.
 * @param text The text to highlight matches in
 * @param query The search query to highlight
 * @returns The text with highlighted matches
 */
function highlightMatch(text: string, query: string): string {
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

  // Create a regex with appropriate flags (g for global, i for case insensitive if needed)
  const flags = caseSensitive ? "g" : "gi";
  const regex = new RegExp(`(${escapedQuery})`, flags);

  return text.replace(regex, '<mark class="highlight">$1</mark>');
}

/**
 * Copies text to the system clipboard.
 * @param text The text to copy to clipboard
 */
async function copyToClipboard(text: string) {
  try {
    await navigator.clipboard.writeText(text);
    // Could add visual feedback here (e.g., show "Copied!" message)
  } catch (err) {
    console.error("Failed to copy text: ", err);
  }
}

/**
 * Opens the file's containing folder in the system file manager.
 * Uses the backend function to handle cross-platform compatibility.
 * @param filePath The path to the file whose folder should be opened
 */
async function openFileLocation(filePath: string) {
  try {
    await ShowInFolder(filePath);
  } catch (error) {
    console.error("Failed to open file location:", error);
    // Fallback: just log the path if the function fails
    console.log(`File location: ${filePath}`);
  }
}
</script>

<template>
  <main>
    <div id="search-controls" class="search-controls">
      <div class="control-group">
        <label for="directory">Directory:</label>
        <div class="directory-input">
          <input
            id="directory"
            v-model="data.directory"
            class="input directory"
            type="text"
            placeholder="Enter directory to search"
          />
          <button class="btn select-dir" @click="selectDirectory">
            Browse
          </button>
        </div>
      </div>

      <div class="control-group">
        <label for="query">Search Query:</label>
        <input
          id="query"
          v-model="data.query"
          class="input"
          type="text"
          placeholder="Enter search term"
          @keyup.enter="searchCode"
        />
      </div>

      <div class="control-group">
        <label for="extension">File Extension (optional):</label>
        <input
          id="extension"
          v-model="data.extension"
          class="input"
          type="text"
          placeholder="e.g., go, js, ts"
        />
      </div>

      <div class="options-group">
        <div class="control-group checkbox-group">
          <input
            id="case-sensitive"
            v-model="data.caseSensitive"
            type="checkbox"
          />
          <label for="case-sensitive">Case Sensitive</label>
        </div>

        <div class="control-group checkbox-group">
          <input id="regex-search" v-model="data.useRegex" type="checkbox" />
          <label for="regex-search">Regex Search</label>
        </div>

        <div class="control-group checkbox-group">
          <input id="include-binary" v-model="data.includeBinary" type="checkbox" />
          <label for="include-binary">Include Binary</label>
        </div>
        
        <div class="control-group checkbox-group">
          <input id="search-subdirs" v-model="data.searchSubdirs" type="checkbox" />
          <label for="search-subdirs">Search Subdirs</label>
        </div>
      </div>

      <div class="options-group">
        <div class="control-group">
          <label for="max-filesize">Max File Size (bytes):</label>
          <input
            id="max-filesize"
            v-model.number="data.maxFileSize"
            class="input"
            type="number"
            placeholder="10485760 (10MB)"
          />
        </div>

        <div class="control-group">
          <label for="max-results">Max Results:</label>
          <input
            id="max-results"
            v-model.number="data.maxResults"
            class="input"
            type="number"
            placeholder="1000"
          />
        </div>
      </div>

      <div class="control-group">
        <button
          class="btn search-btn"
          @click="searchCode"
          :disabled="data.isSearching"
        >
          <span v-if="data.isSearching" class="spinner"></span>
          {{ data.isSearching ? "Searching..." : "Search Code" }}
        </button>
      </div>
    </div>

    <div id="result" class="result">{{ data.resultText }}</div>

    <!-- Search Results -->
    <div
      v-if="
        data.searchResults &&
        Array.isArray(data.searchResults) &&
        data.searchResults.length > 0
      "
      class="results-container"
    >
      <div class="results-header">
        <h3>Search Results:</h3>
        <div class="results-summary">
          Found
          {{
            data.searchResults && Array.isArray(data.searchResults)
              ? data.searchResults.length
              : 0
          }}
          matches
          <span v-if="data.truncatedResults">(truncated)</span>
        </div>
      </div>
      <div
        v-for="(result, index) in data.searchResults &&
        Array.isArray(data.searchResults)
          ? data.searchResults
          : []"
        :key="index"
        class="result-item"
      >
        <div class="result-header">
          <div class="file-info">
            <span
              class="file-path"
              @click="openFileLocation(result.filePath)"
              title="Click to show in folder"
            >
              {{ formatFilePath(result.filePath) }}
            </span>
            <span class="line-num">Line {{ result.lineNum }}</span>
            <span class="matched-text" v-if="result.matchedText && result.matchedText !== data.query">
              (Matched: "{{ result.matchedText }}")
            </span>
          </div>
          <button
            class="copy-btn"
            @click="copyToClipboard(result.content)"
            title="Copy line"
          >
            Copy
          </button>
        </div>
        <div
          class="result-content"
          v-html="highlightMatch(result.content || '', data.query || '')"
        ></div>
      </div>
    </div>
  </main>
</template>

<style scoped>
.result {
  height: 20px;
  line-height: 20px;
  margin: 1.5rem auto;
  text-align: center;
}

.search-controls {
  max-width: 600px;
  margin: 0 auto;
  padding: 20px;
}

.search-controls .control-group {
  margin-bottom: 15px;
}

.search-controls .options-group {
  display: flex;
  gap: 15px;
  margin-bottom: 15px;
  flex-wrap: wrap;
}

.search-controls label {
  display: block;
  margin-bottom: 5px;
  font-weight: bold;
}

.directory-input {
  display: flex;
  gap: 10px;
}

.directory-input .input {
  flex: 1;
}

.select-dir {
  width: auto;
  padding: 0 15px;
}

.checkbox-group {
  display: flex;
  align-items: center;
  gap: 8px;
}

.checkbox-group label {
  margin-bottom: 0;
  font-weight: normal;
}

.search-btn {
  width: 100%;
  padding: 10px;
  background-color: #27ae60;
  color: white;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
}

.search-btn:hover {
  background-color: #219653;
}

.search-btn:disabled {
  background-color: #bdc3c7;
  cursor: not-allowed;
}

.spinner {
  display: inline-block;
  width: 12px;
  height: 12px;
  border: 2px solid rgba(255, 255, 255, 0.3);
  border-radius: 50%;
  border-top-color: #fff;
  animation: spin 1s ease-in-out infinite;
}

@keyframes spin {
  to {
    transform: rotate(360deg);
  }
}

.results-container {
  max-width: 800px;
  margin: 20px auto;
  padding: 0 20px;
}

.results-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 10px;
}

.results-summary {
  color: #7f8c8d;
  font-size: 0.9em;
}

.result-item {
  border: 1px solid #ddd;
  border-radius: 4px;
  margin-bottom: 10px;
  padding: 10px;
  background-color: #fafafa;
  transition: box-shadow 0.2s;
}

.result-item:hover {
  box-shadow: 0 2px 5px rgba(0, 0, 0, 0.1);
}

.result-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 5px;
  flex-wrap: wrap;
  gap: 5px;
}

.file-info {
  display: flex;
  align-items: center;
  gap: 10px;
  flex: 1;
}

.file-path {
  font-weight: bold;
  color: #2980b9;
  cursor: pointer;
  text-decoration: underline;
}

.file-path:hover {
  color: #3498db;
}

.line-num {
  color: #7f8c8d;
  font-size: 0.9em;
  background-color: #ecf0f1;
  padding: 2px 6px;
  border-radius: 3px;
}

.matched-text {
  color: #27ae60;
  font-size: 0.85em;
  font-style: italic;
  margin-left: 10px;
}

.copy-btn {
  background-color: #95a5a6;
  color: white;
  border: none;
  padding: 4px 8px;
  border-radius: 3px;
  cursor: pointer;
  font-size: 0.8em;
}

.copy-btn:hover {
  background-color: #7f8c8d;
}

.result-content {
  font-family: monospace;
  padding: 8px;
  background-color: #f8f9fa;
  border-left: 3px solid #3498db;
  white-space: pre-wrap;
  word-break: break-word;
  overflow-x: auto;
}

.highlight {
  background-color: #f1c40f;
  padding: 1px 2px;
  border-radius: 2px;
  font-weight: bold;
}

.select-dir:hover {
  background-image: linear-gradient(to top, #cfd9df 0%, #e2ebf0 100%);
  color: #333333;
}

.directory-input .input {
  border: none;
  border-radius: 3px;
  outline: none;
  height: 30px;
  line-height: 30px;
  padding: 0 10px;
  background-color: rgba(240, 240, 240, 1);
  -webkit-font-smoothing: antialiased;
}

.directory-input .input:hover {
  border: none;
  background-color: rgba(255, 255, 255, 1);
}

.directory-input .input:focus {
  border: none;
  background-color: rgba(255, 255, 255, 1);
}
</style>
