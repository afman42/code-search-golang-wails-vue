// Utility functions for localStorage persistence with error handling.

import type { RecentSearch } from "@/types";

export const RECENT_SEARCHES_KEY = "codeSearchRecentSearches";

export function loadRecentSearches(): RecentSearch[] {
  try {
    const saved = localStorage.getItem(RECENT_SEARCHES_KEY);
    if (saved) {
      const parsed = JSON.parse(saved);
      if (!Array.isArray(parsed)) return [];
      // Filter to well-formed entries: anything else (nulls, numbers, partial
      // objects from older versions) is dropped rather than crashing callers.
      return parsed.filter((entry): entry is RecentSearch => {
        if (typeof entry !== "object" || entry === null) return false;
        if (!("query" in entry) || !("extension" in entry)) return false;
        return (
          typeof entry.query === "string" &&
          typeof entry.extension === "string"
        );
      });
    }
    return [];
  } catch (error) {
    console.error("Failed to load recent searches from localStorage:", error);
    return [];
  }
}

export function saveRecentSearches(searches: RecentSearch[]): void {
  try {
    localStorage.setItem(RECENT_SEARCHES_KEY, JSON.stringify(searches));
  } catch (error) {
    // Storage quota exceeded or storage disabled — non-fatal, the app keeps
    // working without persisted history.
    console.error("Failed to save recent searches to localStorage:", error);
  }
}

/**
 * Build a stable identity key for a recent search so deduplication compares
 * the full entry (query + extension + directory) instead of the query alone.
 */
export function recentSearchKey(search: {
  query: string;
  extension: string;
  directory?: string;
}): string {
  return `${search.query}\u0000${search.extension}\u0000${search.directory ?? ""}`;
}

/**
 * Remove a recent search. Only the provided identity fields filter the list:
 * pass directory to scope removal to one exact entry, or omit it to remove
 * every entry matching query (and extension, when given).
 */
export function removeRecentSearch(search: {
  query: string;
  extension?: string;
  directory?: string;
}): void {
  try {
    const searches = loadRecentSearches();
    const filtered = searches.filter(
      (s) =>
        !(
          s.query === search.query &&
          (search.extension === undefined || s.extension === search.extension) &&
          (search.directory === undefined || s.directory === search.directory)
        ),
    );
    saveRecentSearches(filtered);
  } catch (error) {
    console.error("Failed to remove search from recent searches:", error);
  }
}
