/**
 * Search-related UI functions
 * These functions are specifically related to formatting and displaying search results
 */

import type { SearchRequest, SearchState } from "@/types";
import { toErrorMessage } from "./errorUtils";
import { OpenInDefaultEditor, OpenInEditorByName } from "@wails/go/main/App";
import {
  DEFAULT_MAX_FILE_SIZE,
  DEFAULT_MAX_RESULTS,
} from "@/constants/appConstants";
import { EDITOR_CATALOG } from "@/constants/editors";

/**
 * Builds the SearchRequest sent to the backend from the current SearchState.
 * Shared by useSearch and useReplace so both flows send identical parameters.
 * maxFileSize/maxResults use nullish coalescing so an explicit 0 (unlimited /
 * no cap) is preserved instead of being coerced away by `||`.
 */
export const buildSearchRequest = (data: SearchState): SearchRequest => {
  return {
    directory: data.directory,
    query: data.query,
    extension: data.extension,
    caseSensitive: data.caseSensitive,
    includeBinary: data.includeBinary,
    maxFileSize: data.maxFileSize ?? DEFAULT_MAX_FILE_SIZE,
    minFileSize: data.minFileSize ?? 0,
    maxResults: data.maxResults ?? DEFAULT_MAX_RESULTS,
    useRegex: data.useRegex,
    excludePatterns: Array.isArray(data.excludePatterns)
      ? data.excludePatterns.filter((s) => s.length > 0)
      : [],
    allowedFileTypes: Array.isArray(data.allowedFileTypes)
      ? data.allowedFileTypes.filter((s) => s.length > 0)
      : [],
    fuzzySearch: data.fuzzySearch,
    contextLines: data.contextLines,
    directories: Array.isArray(data.directories)
      ? data.directories.filter((s) => s.length > 0)
      : [],
    respectGitignore: data.respectGitignore,
  };
};

// editorBindingName and editorDisplayName are derived from the shared
// EDITOR_CATALOG (constants/editors.ts), which mirrors the backend's
// editorCatalog (system_integration.go): one catalog row per editor drives
// the <option> list, the binding name passed to OpenInEditorByName, and the
// availability flag.
//
// The "default" key is intentionally absent from the catalog: the backend's
// OpenInDefaultEditor is a separate method because it dispatches to the OS
// default (xdg-open / explorer) rather than a specific editor command.
// openInEditor handles "default" as a special case below.
const editorBindingName: Record<string, string> = Object.fromEntries(
  EDITOR_CATALOG.map(({ key, binding }) => [key, binding]),
);

const editorDisplayName: Record<string, string> = {
  ...Object.fromEntries(EDITOR_CATALOG.map(({ key, label }) => [key, label])),
  default: "Default Editor",
};

/**
 * Opens a file in the specified editor via the Wails backend binding.
 *
 * Uses the generic OpenInEditorByName dispatcher for named editors (VSCode,
 * Sublime, etc.) and falls back to OpenInDefaultEditor for the "default" key.
 *
 * @param editorKey The editor identifier from EDITOR_CATALOG keys or
 *   "default" (e.g. "vscode", "sublime", "default")
 * @param filePath The path to the file to open
 * @param setResultText Function to update result text in the UI
 * @param setError Function to update error in the UI
 */
export const openInEditor = async (
  editorKey: string,
  filePath: string,
  setResultText: (text: string) => void,
  setError: (error: string | null) => void,
) => {
  try {
    if (!filePath || typeof filePath !== "string") {
      console.warn(`Invalid file path provided to openInEditor (${editorKey})`);
      setResultText("Invalid file path");
      return;
    }

    const displayName = editorDisplayName[editorKey];

    // The "default" editor key is a special case: it calls OpenInDefaultEditor
    // (which dispatches to xdg-open / explorer) rather than OpenInEditorByName
    // (which looks up a named editor in the backend's editorCatalog). This
    // mirrors the backend where OpenInDefaultEditor is a separate method.
    if (editorKey === "default") {
      await OpenInDefaultEditor(filePath);
      setResultText(`File opened in ${displayName}: ${filePath}`);
      return;
    }

    // Named editor: use the generic OpenInEditorByName dispatcher with the
    // binding name from EDITOR_CATALOG. Unknown keys are rejected here, so
    // every key reaching this point is present in editorDisplayName above.
    const bindingName = editorBindingName[editorKey];
    if (!bindingName) {
      setError(`Unknown editor: ${editorKey}`);
      setResultText(`Unknown editor: ${editorKey}`);
      return;
    }

    await OpenInEditorByName(bindingName, filePath);
    setResultText(`File opened in ${displayName}: ${filePath}`);
  } catch (error: unknown) {
    const displayName = editorDisplayName[editorKey];
    console.error(`Failed to open file in ${displayName}:`, error);
    const msg = toErrorMessage(error, "Operation failed");
    setResultText(`Could not open file in ${displayName}: ${msg}`);
    setError(`${displayName} open error: ${msg}`);
  }
};
