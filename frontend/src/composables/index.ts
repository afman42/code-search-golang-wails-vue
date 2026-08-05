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
export { useMatchNavigation } from "./useMatchNavigation";
export { useSearch } from "./useSearch";
export { useTheme } from "./useTheme";
export type { AppTheme } from "./useTheme";
export { THEME_STORAGE_KEY } from "./useTheme";
export { useToast, toastManager } from "./useToast";
