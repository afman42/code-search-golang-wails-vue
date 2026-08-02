// Utility functions for localStorage persistence with error handling

export function loadRecentSearches<T = RecentSearch>(): T[] {
  try {
    const saved = localStorage.getItem("codeSearchRecentSearches");
    if (saved) {
      return JSON.parse(saved) as T[];
    }
    return [];
  } catch (error) {
    console.error("Failed to load recent searches from localStorage:", error);
    return [];
  }
}

export function saveRecentSearches<T>(searches: T[]): void {
  try {
    localStorage.setItem("codeSearchRecentSearches", JSON.stringify(searches));
  } catch (error) {
    console.error("Failed to save recent searches to localStorage:", error);
  }
}

// Search suggestion related types
export interface RecentSearch {
  query: string;
  timestamp: number;
  frequency: number;
}

/**
 * Add a search query to recent searches
 * @param query - The search query text
 * @param timestamp - Unix timestamp (defaults to now)
 * @param frequency - How many times this query has been searched (defaults to 1)
 */
export const addToRecentSearch = (
  query: string,
  timestamp?: number,
  frequency: number = 1
) => {
  try {
    const searches = loadRecentSearches();
    const now = timestamp ?? Date.now();
    
    // Check if this query already exists
    const existingIndex = searches.findIndex(
      (s: RecentSearch) => s.query === query
    );
    
    if (existingIndex >= 0) {
      // Update existing entry: increment frequency and update timestamp
      searches[existingIndex].frequency += frequency;
      searches[existingIndex].timestamp = now;
    } else {
      // Add new entry
      searches.push({ query, timestamp: now, frequency });
    }
    
    // Sort by timestamp descending (most recent first)
    searches.sort((a: RecentSearch, b: RecentSearch) => b.timestamp - a.timestamp);
    
    saveRecentSearches(searches);
  } catch (error) {
    console.error("Failed to add search to recent searches:", error);
  }
};

/**
 * Get recent search suggestions for autocomplete
 * @param limit - Maximum number of suggestions to return (default: 5)
 * @returns Array of recent searches sorted by most recent
 */
export const getRecentSuggestions = (limit: number = 5) => {
  try {
    const searches = loadRecentSearches();
    // Return up to limit items, sorted by timestamp (most recent first)
    return searches.slice(0, limit);
  } catch (error) {
    console.error("Failed to get recent suggestions from localStorage:", error);
    return [];
  }
};

/**
 * Remove a specific search query from recent searches
 * @param query - The search query to remove
 */
export const removeRecentSearch = (query: string) => {
  try {
    const searches = loadRecentSearches();
    const filtered = searches.filter(
      (s: RecentSearch) => s.query !== query
    );
    saveRecentSearches(filtered);
  } catch (error) {
    console.error("Failed to remove search from recent searches:", error);
  }
};