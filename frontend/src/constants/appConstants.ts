// Constants for the application

// Default search settings
export const DEFAULT_MAX_FILE_SIZE = 10 * 1024 * 1024; // 10MB in bytes
export const DEFAULT_MAX_RESULTS = 1000;
export const DEFAULT_MIN_FILE_SIZE = 0;
export const APP_READY_TIMEOUT = 3000; // 3 seconds
export const MAX_PATH_DISPLAY_LENGTH = 80;
export const TRUNCATED_PATH_MAX_LENGTH = 50;

// Storage keys
export const RECENT_SEARCHES_STORAGE_KEY = 'codeSearchRecentSearches';
export const STORAGE_KEYS = {
  RECENT_SEARCHES: 'code_search_recent_searches',
  SYMBOL_INDEX_STATS: 'symbol_index_stats',
} as const;

// CSS classes
export const HIGHLIGHT_CLASS = 'highlight';

// Symbol search configuration constants
export const MAX_SEARCH_RESULTS = 50;
export const MAX_DEFINITION_RESULTS = 10;
export const MAX_USAGES_RESULTS = 20;
export const SEARCH_DEBOUNCE_MS = 300;
export const INDEX_MAX_RETRIES = 3;
export const INDEX_RETRY_DELAY_MS = 1000;
