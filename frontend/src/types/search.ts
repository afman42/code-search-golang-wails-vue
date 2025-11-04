// Define TypeScript interfaces for type safety

export interface SearchResult {
  filePath: string;
  lineNum: number;
  content: string;
  matchedText: string;
  contextBefore: string[];
  contextAfter: string[];
}

export interface SearchRequest {
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
  allowedFileTypes: string[]; // List of file extensions that are allowed to be searched (if empty, all types allowed)
}

export interface SearchProgress {
  processedFiles: number;
  totalFiles: number;
  currentFile: string;
  resultsCount: number;
  status: string;
}

export interface SearchState {
  directory: string;
  query: string;
  extension: string;
  caseSensitive: boolean;
  useRegex: boolean;
  includeBinary: boolean;
  maxFileSize: number;
  maxResults: number;
  searchSubdirs: boolean;
  resultText: string;
  searchResults: SearchResult[];
  truncatedResults: boolean;
  isSearching: boolean;
  searchProgress: SearchProgress;
  showProgress: boolean;
  minFileSize: number;
  excludePatterns: string[];
  allowedFileTypes: string[];
  recentSearches: Array<{
    query: string;
    extension: string;
  }>;
  error: string | null;
}