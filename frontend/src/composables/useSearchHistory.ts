import { loadRecentSearches, saveRecentSearches } from "../utils/localStorageUtils";

const MAX_RECENT_SEARCHES = 5;

export interface RecentSearch {
  query: string;
  extension: string;
}

export function useSearchHistory(initialSearches: RecentSearch[] = loadRecentSearches()) {
  let recentSearches = initialSearches;

  const addRecentSearch = (query: string, extension: string) => {
    const newSearch: RecentSearch = { query, extension };

    recentSearches = recentSearches.filter(
      (s) => !(s.query === newSearch.query && s.extension === newSearch.extension),
    );

    recentSearches.unshift(newSearch);

    if (recentSearches.length > MAX_RECENT_SEARCHES) {
      recentSearches = recentSearches.slice(0, MAX_RECENT_SEARCHES);
    }

    saveRecentSearches(recentSearches);
    return recentSearches;
  };

  return {
    recentSearches,
    addRecentSearch,
  };
}