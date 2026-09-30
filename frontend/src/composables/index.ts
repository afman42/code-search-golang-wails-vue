// Barrel re-export of all composables.
// Import from '@/composables' rather than individual files.

export { useCodeHighlighting } from "./useCodeHighlighting";
export {
  makeDefaultEditorAvailability,
  makeDefaultEditorDetectionStatus,
  subscribeToEditorDetectionEvents,
  startEditorDetection,
} from "./useEditorDetection";
export { useFilePreview } from "./useFilePreview";
export type { FilePreviewState } from "./useFilePreview";
export { useKeyboardShortcuts } from "./useKeyboardShortcuts";
export { parseLogEntry, useLogStreaming } from "./useLogStreaming";
export { useLogViewer } from "./useLogViewer";
export { useMatchNavigation } from "./useMatchNavigation";
export { useSelectionManager } from "./useSelectionManager";
export { useSearch } from "./useSearch";
export { useReplace } from "./useReplace";
export { coerceProgress, coerceResultBatch } from "./searchProgress";
export { useSymbolSearch } from "./useSymbolSearch";
export { useTheme } from "./useTheme";
export type { AppTheme } from "./useTheme";
export { THEME_STORAGE_KEY } from "./useTheme";
// NOTE: useToast / toastManager intentionally NOT re-exported here.
// Import from "@/composables/useToast" directly to avoid a
// composables -> services -> composables import cycle.
