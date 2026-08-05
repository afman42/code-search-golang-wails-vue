import { toastManager } from '@/composables/useToast';
import { toErrorMessage } from '@/utils/errorUtils';
import { ShowInFolder } from '@wails/go/main/App';

/**
 * Wrapper for copyToClipboard that shows toast notifications
 * @param text The text to copy to clipboard
 */
export const copyToClipboardWithToast = async (text: string) => {
  try {
    // Import the original function but we need to handle the old signature
    // Since we can't directly modify the original function, we'll create a wrapper
    // that handles the toast notifications without changing the original function
    
    // For now, we'll implement the copy functionality directly
    if (!text || typeof text !== "string") {
      toastManager.error('Cannot copy empty or invalid text', 'Copy Error');
      return false;
    }

    if (navigator.clipboard && window.isSecureContext) {
      await navigator.clipboard.writeText(text);
      toastManager.success('Copied to clipboard successfully', 'Copy Success');
      return true;
    } else {
      // Fallback for older browsers or insecure contexts
      const textArea = document.createElement("textarea");
      textArea.value = text;
      textArea.style.position = "fixed";
      textArea.style.opacity = "0";
      document.body.appendChild(textArea);
      textArea.select();
      document.execCommand("copy");
      document.body.removeChild(textArea);
      toastManager.success('Copied to clipboard successfully', 'Copy Success');
      return true;
    }
  } catch (err) {
    console.error("Failed to copy text to clipboard: ", err);
    toastManager.error('Failed to copy to clipboard', 'Copy Error');
    return false;
  }
};

/**
 * Wrapper for openFileLocation that shows toast notifications
 * @param filePath The path to the file whose folder should be opened
 */
export const openFileLocationWithToast = async (filePath: string) => {
  try {
    // Validate input
    if (!filePath || typeof filePath !== "string") {
      toastManager.error('Invalid file path provided', 'Open Folder Error');
      throw new Error("Invalid file path");
    }

    // Import ShowInFolder statically — the dynamic import was a lint
    // violation (ts-no-dynamic-import) and unnecessary: the module path
    // is a literal known at author time.
    await ShowInFolder(filePath);

    // Extract the filename handling both Unix (/) and Windows (\)
    // separators. The previous code used split('/').pop() || split('\\').pop()
    // but the || short-circuited for backslash-only paths (split('/') returns
    // the whole string as one element, which is truthy, so the backslash
    // split never ran). Splitting on both separators at once fixes this.
    const fileName = filePath.split(/[/\\]/).pop() || filePath;
    toastManager.info(`Opened containing folder for: ${fileName}`, 'Folder Opened');
  } catch (error: unknown) {
    console.error("Failed to open file location:", error);
    const errorMessage = toErrorMessage(error, "Operation failed");
    toastManager.error(`Could not open file location: ${errorMessage}`, 'Open Folder Error');
    throw error;
  }
};

/**
 * Wrapper for editor opening functions that shows toast notifications
 * @param editorFn The editor opening function to call
 * @param filePath The file path to open
 * @param editorName The name of the editor for the toast message
 */
export const openInEditorWithToast = async (
  editorFn: (filePath: string) => Promise<void>,
  filePath: string,
  editorName: string
) => {
  try {
    if (!filePath || typeof filePath !== "string") {
      toastManager.error('Invalid file path provided', `${editorName} Open Error`);
      return;
    }

    await editorFn(filePath);
    toastManager.success(`File opened in ${editorName}`, `${editorName} Success`);
  } catch (error: unknown) {
    console.error(`Failed to open file in ${editorName}:`, error);
    const errorMessage = toErrorMessage(error, "Operation failed");
    toastManager.error(`Could not open file in ${editorName}: ${errorMessage}`, `${editorName} Error`);
  }
};