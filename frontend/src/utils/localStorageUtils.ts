// Utility functions for localStorage persistence with error handling

export const loadRecentSearches = () => {
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

export const saveRecentSearches = (searches: any[]) => {
  try {
    localStorage.setItem("codeSearchRecentSearches", JSON.stringify(searches));
  } catch (error) {
    console.error("Failed to save recent searches to localStorage:", error);
  }
};