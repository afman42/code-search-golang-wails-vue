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

// editorBindingName maps the frontend editor keys (emitted by EditorSelect.vue)
// to the binding names expected by the backend's OpenInEditorByName dispatcher.
// The backend's editorBindings map (system_integration.go) uses these exact
// names as keys, so adding a new editor only requires one entry here + one
// entry in the backend map — no new Wails binding method per editor.
//
// The "default" key is intentionally absent: the backend's OpenInDefaultEditor
// is a separate method (not part of editorBindings) because it dispatches to
// the OS default (xdg-open / explorer) rather than a specific editor command.
// openInEditor handles "default" as a special case below.
const editorBindingName: Record<string, string> = {
  vscode: "VSCode",
  vscodium: "VSCodium",
  sublime: "Sublime",
  jetbrains: "JetBrains", // Note: OpenInJetBrains routes by file extension internally
  geany: "Geany",
  goland: "GoLand",
  pycharm: "PyCharm",
  intellij: "IntelliJ",
  webstorm: "WebStorm",
  phpstorm: "PhpStorm",
  clion: "CLion",
  rider: "Rider",
  androidstudio: "AndroidStudio",
  emacs: "Emacs",
  neovide: "Neovide",
  codeblocks: "CodeBlocks",
  devcpp: "DevCpp",
  notepadplusplus: "NotepadPlusPlus",
  visualstudio: "VisualStudio",
  eclipse: "Eclipse",
  netbeans: "NetBeans",
  neovim: "Neovim",
  vim: "Vim",
};

const editorDisplayName: Record<string, string> = {
  vscode: "VSCode",
  vscodium: "VSCodium",
  sublime: "Sublime Text",
  jetbrains: "JetBrains IDE",
  geany: "Geany",
  goland: "GoLand",
  pycharm: "PyCharm",
  intellij: "IntelliJ IDEA",
  webstorm: "WebStorm",
  phpstorm: "PhpStorm",
  clion: "CLion",
  rider: "Rider",
  androidstudio: "Android Studio",
  emacs: "Emacs",
  neovide: "Neovide",
  codeblocks: "Code::Blocks",
  devcpp: "Dev-C++",
  notepadplusplus: "Notepad++",
  visualstudio: "Visual Studio",
  eclipse: "Eclipse",
  netbeans: "NetBeans",
  neovim: "Neovim",
  vim: "Vim",
  default: "Default Editor",
};

/**
 * Opens a file in the specified editor via the Wails backend binding.
 *
 * Uses the generic OpenInEditorByName dispatcher for named editors (VSCode,
 * Sublime, etc.) and falls back to OpenInDefaultEditor for the "default" key.
 * This replaces the previous per-editor dynamic dispatch (calling OpenInVSCode,
 * OpenInSublime, etc. by name) with a single Wails call — keeping the frontend
 * in sync with the backend's table-driven editorBindings map.
 *
 * @param editorKey The editor identifier (e.g. "vscode", "sublime", "default")
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

    const displayName = editorDisplayName[editorKey] || editorKey;

    // The "default" editor key is a special case: it calls OpenInDefaultEditor
    // (which dispatches to xdg-open / explorer) rather than OpenInEditorByName
    // (which looks up a named editor in editorBindings). This mirrors the
    // backend where OpenInDefaultEditor is a separate method, not part of the
    // editorBindings map.
    if (editorKey === "default") {
      if (typeof OpenInDefaultEditor !== "function") {
        setError("OpenInDefaultEditor function not found");
        setResultText("OpenInDefaultEditor function not found");
        return;
      }
      await OpenInDefaultEditor(filePath);
      setResultText(`File opened in ${displayName}: ${filePath}`);
      return;
    }

    // Named editor: use the generic OpenInEditorByName dispatcher with the
    // binding name from the editorBindingName map. This is the compatibility
    // point with the backend's table-driven editorBindings.
    const bindingName = editorBindingName[editorKey];
    if (!bindingName) {
      setError(`Unknown editor: ${editorKey}`);
      setResultText(`Unknown editor: ${editorKey}`);
      return;
    }

    if (typeof OpenInEditorByName !== "function") {
      setError("OpenInEditorByName function not found");
      setResultText("OpenInEditorByName function not found");
      return;
    }

    await OpenInEditorByName(bindingName, filePath);
    setResultText(`File opened in ${displayName}: ${filePath}`);
  } catch (error: unknown) {
    const displayName = editorDisplayName[editorKey] || editorKey;
    console.error(`Failed to open file in ${displayName}:`, error);
    const msg = toErrorMessage(error, "Operation failed");
    setResultText(`Could not open file in ${displayName}: ${msg}`);
    setError(`${displayName} open error: ${msg}`);
  }
};
