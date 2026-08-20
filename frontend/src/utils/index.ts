// Barrel re-export of pure utility helpers.
// Import from '@/utils' rather than individual files.

export { toErrorMessage, asRecord } from "./errorUtils";
export { escapeRegExp } from "./regexUtils";
export {
  formatFilePath,
  truncatePath,
  shortDirectory,
  handleEditorSelect,
} from "./fileUtils";
export { findFuzzyMatches } from "./fuzzyMatch";
export { buildSearchRequest, openInEditor } from "./searchUiUtils";
export {
  copyToClipboardWithToast,
  openFileLocationWithToast,
} from "./toastUtils";
export {
  RECENT_SEARCHES_KEY,
  loadRecentSearches,
  saveRecentSearches,
  recentSearchKey,
  removeRecentSearch,
} from "./localStorageUtils";
export type { MatchRange, DiffSegment } from "./diffUtils";
export {
  findMatchRanges,
  buildDiffSegments,
  renderDiffHtml,
} from "./diffUtils";
