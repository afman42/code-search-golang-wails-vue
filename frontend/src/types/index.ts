// Barrel re-export of all domain type modules.
// Import types from '@/types' rather than the individual files.

export type {
  SearchResult,
  SearchRequest,
  SearchProgress,
  EditorAvailability,
  EditorDetectionStatus,
  SearchState,
  TreeItem,
  SymbolInfo,
  FileReplacement,
  ReplaceRequest,
  ReplaceResult,
} from "./search";

export type { RecentSearch } from "./recentSearch";

export type { SyntaxHighlightOptions } from "./syntax";

export type { LogEntry } from "./logs";

export type { KeyboardShortcutHandlers } from "./keyboard";

export type { ToastType, Toast, ToastOptions, ToastStore } from "./toast";
