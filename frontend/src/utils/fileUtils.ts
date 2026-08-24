import { toastManager } from "@/composables/useToast";
import { toErrorMessage } from "./errorUtils";
import { ReadFileLog } from "@wails/go/main/App";
import { openInEditor } from "./searchUiUtils";
import { EDITOR_CATALOG, type EditorKey } from "@/constants/editors";
// Utility functions for file operations and path formatting

/**
 * Formats a file path for display, truncating long paths
 * @param filePath - The full file path to format
 * @returns A formatted path string suitable for display
 */
export const formatFilePath = (filePath: string): string => {
  if (!filePath) return "";
  // Truncate long paths for better display
  if (filePath.length > 80) {
    const pathParts = filePath.split("/");
    if (pathParts.length > 5) {
      return "..." + pathParts.slice(-3).join("/");
    }
  }
  return filePath;
};

/**
 * Truncates a file path to show only the end portion
 * @param filePath - The full file path to truncate
 * @param maxLength - Maximum length of the truncated path (default 50)
 * @returns A truncated path string
 */
export const truncatePath = (
  filePath: string,
  maxLength: number = 50,
): string => {
  if (!filePath) return "";
  if (filePath.length <= maxLength) {
    return filePath;
  }
  return "..." + filePath.slice(-maxLength + 3); // +3 for the '...' prefix
};

/**
 * Shortens a directory path for display in compact UI rows by keeping only
 * the last path segment, e.g. /home/user/projects/foo -> foo.
 * @param path - The full directory path to shorten
 * @returns A path suitable for display in compact UI rows
 */
export const shortDirectory = (path: string): string => {
  if (!path) return "";
  const parts = path.split(/[\\/]/);
  return parts[parts.length - 1] || path;
};


/** True when `value` is an EDITOR_CATALOG key or the "default" pseudo-entry. */
const isEditorKey = (value: string): value is EditorKey =>
  value === "default" || EDITOR_CATALOG.some(({ key }) => key === value);
// Handle editor selection and open file in selected editor
export const handleEditorSelect = async (event: Event, filePath: string) => {
  const target = event.target as HTMLSelectElement;
  const editor = target.value;

  target.selectedIndex = 0;

  if (!editor) return;

  try {
    let logPath = filePath;
    if (logPath.endsWith(".log")) {
      logPath = await ReadFileLog(logPath);
    }

    // Narrow the raw <select> value before dispatching. Unknown values keep
    // today's fallback exactly: they still reach openInEditor, which reports
    // them ("Unknown editor: …") — nothing is swallowed here.
    const onSuccess = (text: string) => {
      toastManager.success(text, `${editor} Success`);
    };
    const onError = (err: string | null) => {
      toastManager.error(err ?? "Unknown error", `${editor} Error`);
    };
    if (!isEditorKey(editor)) {
      await openInEditor(editor, logPath, onSuccess, onError);
      return;
    }
    await openInEditor(editor, logPath, onSuccess, onError);
  } catch (error: unknown) {
    console.error(`Failed to open file in ${editor}:`, error);
    const errorMessage = toErrorMessage(error, "Unknown error");
    toastManager.error(
      `Could not open file in ${editor}: ${errorMessage}`,
      `${editor} Error`,
    );
  }
};
