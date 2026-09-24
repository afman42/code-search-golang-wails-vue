// Barrel re-export of all reusable UI components.
// Import from '@/components/ui' rather than individual .vue files,
// EXCEPT CodeModal and LogViewer: those are lazy-loaded via defineAsyncComponent
// + direct paths (see CodeSearch.vue, SearchResults.vue). Keeping them out of
// the barrel is what lets Vite split them into separate chunks.

export { default as ActionButtons } from "./ActionButtons.vue";
// (CodeModal intentionally excluded — lazy chunk, import "./CodeModal.vue" directly)
export { default as DirectoryPicker } from "./DirectoryPicker.vue";
export { default as EditorSelect } from "./EditorSelect.vue";
export { default as EditorStatusDisplay } from "./EditorStatusDisplay.vue";
export { default as EnhancedTreeItem } from "./EnhancedTreeItem.vue";
export { default as InlineDiffView } from "./InlineDiffView.vue";
// (LogViewer intentionally excluded — lazy chunk, import "./LogViewer.vue" directly)
export { default as PatternSelector } from "./PatternSelector.vue";
export { default as ProgressIndicator } from "./ProgressIndicator.vue";
export { default as QueryInput } from "./QueryInput.vue";
export { default as SearchForm } from "./SearchForm.vue";
export { default as SearchHistorySidebar } from "./SearchHistorySidebar.vue";
export { default as SearchOptions } from "./SearchOptions.vue";
export { default as SearchResults } from "./SearchResults.vue";
export { default as SearchSuggestions } from "./SearchSuggestions.vue";
export { default as SizeLimitOptions } from "./SizeLimitOptions.vue";
export { default as SymbolSearch } from "./SymbolSearch.vue";
export { default as ToastNotification } from "./ToastNotification.vue";
export { default as TreeViewPanel } from "./TreeViewPanel.vue";
export { default as MatchNavigationControls } from "./MatchNavigationControls.vue";
export { default as ModalFooter } from "./ModalFooter.vue";
export { default as PaginationControls } from "./PaginationControls.vue";
export { default as ExportActions } from "./ExportActions.vue";
export { default as ReplacePreview } from "./ReplacePreview.vue";
export { default as ReplaceProgress } from "./ReplaceProgress.vue";
