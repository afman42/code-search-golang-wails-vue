/**
 * Search-related UI functions
 * These functions are specifically related to formatting and displaying search results
 */

import DOMPurify from "dompurify";
import { SearchState } from "../types/search";

/**
 * Highlights matches in text by wrapping them in HTML mark tags.
 * Handles both regular string matching and regex matching.
 * @param text The text to highlight matches in
 * @param query The search query to highlight
 * @param data Search state containing case sensitivity and regex settings
 * @returns The text with highlighted matches
 */
export const highlightMatch = (
  text: string,
  query: string,
  data: SearchState,
): string => {
  try {
    if (!text || typeof text !== "string") return "";
    if (!query || typeof query !== "string") return text;

    if (query.length > 1000) {
      console.warn("Search query is too long, skipping highlight");
      return text;
    }

    const useRegex =
      data && typeof data.useRegex === "boolean" ? data.useRegex : false;
    const caseSensitive =
      data && typeof data.caseSensitive === "boolean"
        ? data.caseSensitive
        : false;

    let result = text;

    if (useRegex) {
      try {
        new RegExp(query, caseSensitive ? "g" : "gi");
      } catch (e) {
        console.warn(
          "Invalid regex pattern for highlight, using literal match:",
          e,
        );
        const escapedQuery = query.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");
        const flags = caseSensitive ? "g" : "gi";
        const regex = new RegExp(`(${escapedQuery})`, flags);
        return text.replace(regex, '<mark class="highlight">$1</mark>');
      }

      const flags = caseSensitive ? "g" : "gi";
      const regex = new RegExp(`(${query})`, flags);

      if (text.length > 10000) {
        return text;
      }

      try {
        result = text.replace(regex, '<mark class="highlight">$1</mark>');
      } catch (e) {
        console.error("Regex replace failed, returning original text:", e);
        return text;
      }
    } else {
      const escapedQuery = query.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");

      if (!escapedQuery) return text;

      const flags = caseSensitive ? "g" : "gi";
      const regex = new RegExp(`(${escapedQuery})`, flags);

      try {
        result = text.replace(regex, '<mark class="highlight">$1</mark>');
      } catch (e) {
        console.error("Literal replace failed, returning original text:", e);
        return text;
      }
    }

    if (result.length > 100000) {
      console.warn("Highlighted result is too long, consider truncating");
    }

    return DOMPurify.sanitize(result, {
      ALLOWED_TAGS: ["mark"],
      ALLOWED_ATTR: ["class"],
    });
  } catch (error) {
    console.error("Error in highlightMatch:", error);
    return text;
  }
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
  atom: "Atom",
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
  atom: "Atom",
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
    const wailsModule = await import("../../wailsjs/go/main/App");

    // The "default" editor key is a special case: it calls OpenInDefaultEditor
    // (which dispatches to xdg-open / explorer) rather than OpenInEditorByName
    // (which looks up a named editor in editorBindings). This mirrors the
    // backend where OpenInDefaultEditor is a separate method, not part of the
    // editorBindings map.
    if (editorKey === "default") {
      const fn = wailsModule.OpenInDefaultEditor;
      if (typeof fn !== "function") {
        setError("OpenInDefaultEditor function not found");
        setResultText("OpenInDefaultEditor function not found");
        return;
      }
      await fn(filePath);
      console.log(`Successfully opened file in ${displayName}:`, filePath);
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

    const fn = wailsModule.OpenInEditorByName;
    if (typeof fn !== "function") {
      setError("OpenInEditorByName function not found");
      setResultText("OpenInEditorByName function not found");
      return;
    }

    await fn(bindingName, filePath);
    console.log(`Successfully opened file in ${displayName}:`, filePath);
    setResultText(`File opened in ${displayName}: ${filePath}`);
  } catch (error: any) {
    const displayName = editorDisplayName[editorKey] || editorKey;
    console.error(`Failed to open file in ${displayName}:`, error);
    setResultText(
      `Could not open file in ${displayName}: ${error.message || "Operation failed"}`,
    );
    setError(
      `${displayName} open error: ${error.message || "Operation failed"}`,
    );
  }
};
