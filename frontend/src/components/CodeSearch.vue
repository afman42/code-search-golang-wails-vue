<script lang="ts" setup>
import { reactive } from "vue";
import {
  SearchCode,
  ShowInFolder,
  SelectDirectory as GoSelectDirectory,
  SearchWithProgress as GoSearchWithProgress,
} from "../../wailsjs/go/main/App";
import { EventsOn } from "../../wailsjs/runtime";

// Define TypeScript interfaces for type safety
interface SearchResult {
  filePath: string;
  lineNum: number;
  content: string;
  matchedText: string;
  contextBefore: string[];
  contextAfter: string[];
}

interface SearchRequest {
  directory: string;
  query: string;
  extension: string;
  caseSensitive: boolean;
  includeBinary: boolean;
  maxFileSize: number;
  minFileSize: number;
  maxResults: number;
  searchSubdirs: boolean;
  useRegex?: boolean;    // Optional for backward compatibility
  excludePatterns: string[];
}

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

// Sanitize string for display to prevent XSS
function sanitizeString(str: string): string {
  if (!str) return '';
  return str.replace(/[<>]/g, (match) => {
    return match === '<' ? '&lt;' : '&gt;';
  });
}

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
  searchProgress: {
    processedFiles: 0,
    totalFiles: 0,
    currentFile: "",
    resultsCount: 0,
    status: "",
  }, // Progress information
  showProgress: false, // Whether to show progress bar
  minFileSize: 0, // Minimum file size filter (bytes)
  excludePatterns: "", // Comma-separated list of patterns to exclude (e.g., node_modules,*.log)
  recentSearches: loadRecentSearches() as Array<{
    query: string;
    extension: string;
  }>, // Recent searches history
  error: null as string | null, // Error message if any
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
    let errorMessage = "Directory selection failed. Please enter the directory path manually.";
    
    if (error && typeof error === "object" && "message" in error) {
      const errorStr = (error as Error).message || String(error);

      // Special handling for different error types
      if (errorStr.includes("not implemented")) {
        errorMessage = "Directory selection is not available on this platform.\nPlease enter the directory path manually.";
      } else if (errorStr.includes("no suitable directory picker")) {
        errorMessage = "No directory picker found. Please install zenity (GNOME) or kdialog (KDE) to use the directory picker,\nor enter the directory path manually.";
      } else {
        errorMessage = `Directory selection failed: ${errorStr}\nPlease enter the directory path manually.`;
      }
    }
    
    data.resultText = errorMessage;
    data.error = errorMessage;
  }
}

/**
 * Performs the code search operation with progress updates using the backend.
 * Handles validation, search execution, result processing, and error handling.
 * Also manages recent searches functionality.
 */
async function searchCode() {
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
  if (typeof data.maxFileSize !== 'number' || data.maxFileSize < 0) {
    data.resultText = "Please enter a valid maximum file size (non-negative number)";
    data.error = "Invalid max file size";
    return;
  }
  
  if (typeof data.minFileSize !== 'number' || data.minFileSize < 0) {
    data.resultText = "Please enter a valid minimum file size (non-negative number)";
    data.error = "Invalid min file size";
    return;
  }
  
  if (typeof data.maxResults !== 'number' || data.maxResults <= 0) {
    data.resultText = "Please enter a valid maximum number of results (positive number)";
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
      data.resultText = `Invalid regex pattern: ${e.message || 'Unknown error'}`;
      data.error = `Invalid regex: ${e.message || 'Unknown error'}`;
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
    excludePatterns: data.excludePatterns 
      ? data.excludePatterns
          .split(',')
          .map(s => s.trim())
          .filter(s => s.length > 0) // Remove empty patterns
      : [],
  };

  try {
    // Subscribe to progress events
    const progressCleanup = EventsOn("search-progress", (progressData: any) => {
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
    });

    // Execute the search using backend function with progress
    const results = await GoSearchWithProgress(searchRequest);
    
    // Ensure results is always an array, even if backend returns null/undefined
    const processedResults = Array.isArray(results) ? results : (results || []);

    data.searchResults = processedResults;

    // Check if results were truncated due to backend limit
    data.truncatedResults = processedResults.length === 1000; // backend limit

    // Update result text with final count
    data.resultText = processedResults.length > 0
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
}

/**
 * Formats file paths for display.
 * Shows just the filename with parent directory for context.
 * @param filePath The full file path to format
 * @returns A shortened version of the file path for display
 */
function formatFilePath(filePath: string): string {
  try {
    // Safety checks to prevent runtime errors
    if (!filePath || typeof filePath !== "string") return "";
    
    // Cross-platform path handling (support both / and \)
    const normalizedPath = filePath.replace(/\\/g, '/');
    const parts = normalizedPath.split('/');
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
}

/**
 * Highlights matches in text by wrapping them in HTML mark tags.
 * Handles both regular string matching and regex matching.
 * @param text The text to highlight matches in
 * @param query The search query to highlight
 * @returns The text with highlighted matches
 */
function highlightMatch(text: string, query: string): string {
  try {
    // Safety checks to prevent runtime errors
    if (!text || typeof text !== "string") return "";
    if (!query || typeof query !== "string") return text;

    // Use safe access to component data
    const useRegex = data && typeof data.useRegex === "boolean" ? data.useRegex : false;
    const caseSensitive = data && typeof data.caseSensitive === "boolean" ? data.caseSensitive : false;

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
}

/**
 * Copies text to the system clipboard.
 * @param text The text to copy to clipboard
 */
async function copyToClipboard(text: string) {
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
}

/**
 * Opens the file's containing folder in the system file manager.
 * Uses the backend function to handle cross-platform compatibility.
 * @param filePath The path to the file whose folder should be opened
 */
async function openFileLocation(filePath: string) {
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
    data.resultText = `Could not open file location: ${error.message || 'Operation failed'}`;
    data.error = `Open folder error: ${error.message || 'Operation failed'}`;
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
          <label for="min-filesize">Min File Size (bytes):</label>
          <input
            id="min-filesize"
            v-model.number="data.minFileSize"
            class="input"
            type="number"
            placeholder="0"
          />
        </div>

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
        <label for="exclude-patterns">Exclude Patterns (comma-separated):</label>
        <input
          id="exclude-patterns"
          v-model="data.excludePatterns"
          class="input"
          type="text"
          placeholder="e.g., node_modules,.git,*.log,build"
        />
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

    <div id="result" class="result" :class="{ 'error': data.error }">{{ data.resultText }}</div>
    
    <!-- Error display -->
    <div v-if="data.error" class="error-message" id="error-display">
      {{ data.error }}
    </div>

    <!-- Progress Bar -->
    <div v-if="data.showProgress" class="progress-container">
      <div class="progress-bar">
        <div 
          class="progress-fill" 
          :style="{ width: data.searchProgress.totalFiles > 0 ? 
            (data.searchProgress.processedFiles / data.searchProgress.totalFiles * 100) + '%' : '0%' }"
        ></div>
      </div>
      <div class="progress-info">
        <span>Processed: {{ data.searchProgress.processedFiles }} / {{ data.searchProgress.totalFiles }} files</span>
        <span>Results: {{ data.searchProgress.resultsCount }}</span>
      </div>
      <div v-if="data.searchProgress.currentFile" class="current-file">
        Processing: {{ formatFilePath(data.searchProgress.currentFile) }}
      </div>
    </div>

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
        
        <!-- Display context lines before match -->
        <div 
          v-for="(contextLine, ctxIndex) in result.contextBefore"
          :key="'before-' + index + '-' + ctxIndex"
          class="context-line context-before"
          v-html="highlightMatch(contextLine, data.query || '')"
        ></div>
        
        <!-- Display the matched line -->
        <div
          class="result-content"
          v-html="highlightMatch(result.content || '', data.query || '')"
        ></div>
        
        <!-- Display context lines after match -->
        <div 
          v-for="(contextLine, ctxIndex) in result.contextAfter"
          :key="'after-' + index + '-' + ctxIndex"
          class="context-line context-after"
          v-html="highlightMatch(contextLine, data.query || '')"
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

.result.error {
  color: #e74c3c;
}

.error-message {
  max-width: 600px;
  margin: 0.5rem auto;
  padding: 10px;
  background-color: #fadbd8;
  border: 1px solid #e74c3c;
  border-radius: 4px;
  color: #c0392b;
  text-align: center;
  font-size: 0.9em;
}

.progress-container {
  max-width: 600px;
  margin: 1.5rem auto;
  padding: 0 20px;
}

.progress-bar {
  width: 100%;
  height: 20px;
  background-color: #ecf0f1;
  border-radius: 10px;
  overflow: hidden;
  margin-bottom: 8px;
  box-shadow: inset 0 1px 3px rgba(0,0,0,0.2);
}

.progress-fill {
  height: 100%;
  background: linear-gradient(to right, #3498db, #2980b9);
  transition: width 0.3s ease;
  border-radius: 10px;
  position: relative;
  overflow: hidden;
}

.progress-fill::after {
  content: '';
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background-image: linear-gradient(
    -45deg,
    rgba(255, 255, 255, .2) 25%,
    transparent 25%,
    transparent 50%,
    rgba(255, 255, 255, .2) 50%,
    rgba(255, 255, 255, .2) 75%,
    transparent 75%
  );
  background-size: 30px 30px;
  animation: progress-shine 1.5s infinite linear;
  opacity: 0.3;
}

@keyframes progress-shine {
  0% {
    background-position: 0 0;
  }
  100% {
    background-position: 30px 30px;
  }
}

.progress-info {
  display: flex;
  justify-content: space-between;
  font-size: 0.9em;
  color: #7f8c8d;
  margin-bottom: 5px;
}

.current-file {
  font-size: 0.85em;
  color: #95a5a6;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.export-buttons {
  display: flex;
  gap: 8px;
  margin-left: 10px;
}

.export-btn {
  background-color: #3498db;
  color: white;
  border: none;
  padding: 4px 8px;
  border-radius: 3px;
  cursor: pointer;
  font-size: 0.8em;
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

.context-line {
  font-family: monospace;
  padding: 4px 8px;
  background-color: #f0f0f0;
  border-left: 2px solid #bdc3c7;
  white-space: pre-wrap;
  word-break: break-word;
  overflow-x: auto;
  font-size: 0.9em;
  color: #7f8c8d;
}

.context-before {
  border-left-color: #3498db;
}

.context-after {
  border-left-color: #9b59b6;
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
