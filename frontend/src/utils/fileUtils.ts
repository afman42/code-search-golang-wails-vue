import { toastManager } from "../composables/useToast";
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

// Handle editor selection and open file in selected editor
export const handleEditorSelect = async (event: Event, filePath: string) => {
  const target = event.target as HTMLSelectElement;
  const editor = target.value;

  target.selectedIndex = 0;

  if (!editor) return;

  try {
    if (filePath.endsWith(".log")) {
      const { ReadFileLog } = await import("../../wailsjs/go/main/App");
      filePath = await ReadFileLog(filePath);
    }

    const { openInEditor } = await import("./searchUiUtils");
    await openInEditor(
      editor,
      filePath,
      (text) => {
        toastManager.success(text, `${editor} Success`);
      },
      (err) => {
        toastManager.error(err!, `${editor} Error`);
      },
    );
  } catch (error: any) {
    console.error(`Failed to open file in ${editor}:`, error);
    const errorMessage = error.message || "Unknown error";
    toastManager.error(
      `Could not open file in ${editor}: ${errorMessage}`,
      `${editor} Error`,
    );
  }
};
