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
    // Safety checks to prevent runtime errors
    if (!text || typeof text !== "string") return "";
    if (!query || typeof query !== "string") return text;

    // Limit query length to prevent potential performance issues
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
      // For regex mode, validate the pattern first
      try {
        new RegExp(query, caseSensitive ? "g" : "gi");
      } catch (e) {
        console.warn(
          "Invalid regex pattern for highlight, using literal match:",
          e,
        );
        // Fallback to literal matching if regex is invalid
        const escapedQuery = query.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");
        const flags = caseSensitive ? "g" : "gi";
        const regex = new RegExp(`(${escapedQuery})`, flags);
        return text.replace(regex, '<mark class="highlight">$1</mark>');
      }

      // Use a timeout-based approach to prevent catastrophic backtracking
      // Create the regex and use it safely
      const flags = caseSensitive ? "g" : "gi";
      const regex = new RegExp(`(${query})`, flags);

      // Limit the number of replacements to prevent performance issues
      // Use split and join method for basic highlighting as an alternative
      try {
        result = text.replace(regex, '<mark class="highlight">$1</mark>');
      } catch (e) {
        console.error("Regex replace failed, returning original text:", e);
        return text;
      }
    } else {
      // For literal match mode, escape special regex characters
      const escapedQuery = query.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");

      // Ensure the escaped query is not empty after escaping
      if (!escapedQuery) return text;

      // Create a regex with appropriate flags (g for global, i for case insensitive if needed)
      const flags = caseSensitive ? "g" : "gi";
      const regex = new RegExp(`(${escapedQuery})`, flags);

      try {
        result = text.replace(regex, '<mark class="highlight">$1</mark>');
      } catch (e) {
        console.error("Literal replace failed, returning original text:", e);
        return text;
      }
    }

    // Limit the result length to prevent DOM performance issues
    if (result.length > 100000) {
      console.warn("Highlighted result is too long, consider truncating");
    }

    // Sanitize the result HTML to prevent XSS vulnerabilities
    return DOMPurify.sanitize(result, {
      ALLOWED_TAGS: ["mark"],
      ALLOWED_ATTR: ["class"],
    });
  } catch (error) {
    console.error("Error in highlightMatch:", error);
    // If highlighting fails, return the original text to avoid breaking the UI
    return text;
  }
};

/**
 * Copies text to the system clipboard.
 * @param text The text to copy to clipboard
 * @param setResultText Function to update result text in the UI
 * @param setError Function to update error in the UI
 */
export const copyToClipboard = async (
  text: string,
  setResultText: (text: string) => void,
  setError: (error: string | null) => void,
) => {
  try {
    // Validate input
    if (!text || typeof text !== "string") {
      console.warn("Attempted to copy empty or invalid text to clipboard");
      return;
    }

    // Try modern clipboard API first
    if (navigator.clipboard && window.isSecureContext) {
      await navigator.clipboard.writeText(text);
      // Optional: provide user feedback
      console.log("Text copied to clipboard");
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
      console.log("Text copied to clipboard using fallback method");
    }
  } catch (err) {
    console.error("Failed to copy text to clipboard: ", err);
    // Show user-friendly error message
    setResultText("Failed to copy text to clipboard");
    setError("Clipboard error");
  }
};

/**
 * Opens the file's containing folder in the system file manager.
 * Uses the backend function to handle cross-platform compatibility.
 * @param filePath The path to the file whose folder should be opened
 * @param setResultText Function to update result text in the UI
 * @param setError Function to update error in the UI
 */
export const openFileLocation = async (
  filePath: string,
  setResultText: (text: string) => void,
  setError: (error: string | null) => void,
) => {
  try {
    // Validate input
    if (!filePath || typeof filePath !== "string") {
      console.warn("Invalid file path provided to openFileLocation");
      setResultText("Invalid file path");
      return;
    }

    // Import ShowInFolder dynamically to avoid circular dependencies
    const { ShowInFolder } = await import("../../wailsjs/go/main/App");
    await ShowInFolder(filePath);
    console.log("Successfully opened file location:", filePath);
  } catch (error: any) {
    console.error("Failed to open file location:", error);
    // Provide user feedback
    setResultText(
      `Could not open file location: ${error.message || "Operation failed"}`,
    );
    setError(`Open folder error: ${error.message || "Operation failed"}`);
  }
};

/**
 * Opens a file in VSCode editor
 * @param filePath The path to the file to open in VSCode
 * @param setResultText Function to update result text in the UI
 * @param setError Function to update error in the UI
 */
export const openInVSCode = async (
  filePath: string,
  setResultText: (text: string) => void,
  setError: (error: string | null) => void,
) => {
  try {
    // Validate input
    if (!filePath || typeof filePath !== "string") {
      console.warn("Invalid file path provided to openInVSCode");
      setResultText("Invalid file path");
      return;
    }

    // Import OpenInVSCode from Wails bindings
    const { OpenInVSCode } = await import("../../wailsjs/go/main/App");
    await OpenInVSCode(filePath);
    console.log("Successfully opened file in VSCode:", filePath);
    setResultText(`File opened in VSCode: ${filePath}`);
  } catch (error: any) {
    console.error("Failed to open file in VSCode:", error);
    // Provide user feedback
    setResultText(
      `Could not open file in VSCode: ${error.message || "Operation failed"}`,
    );
    setError(`VSCode open error: ${error.message || "Operation failed"}`);
  }
};

/**
 * Opens a file in VSCodium editor
 * @param filePath The path to the file to open in VSCodium
 * @param setResultText Function to update result text in the UI
 * @param setError Function to update error in the UI
 */
export const openInVSCodium = async (
  filePath: string,
  setResultText: (text: string) => void,
  setError: (error: string | null) => void,
) => {
  try {
    // Validate input
    if (!filePath || typeof filePath !== "string") {
      console.warn("Invalid file path provided to openInVSCodium");
      setResultText("Invalid file path");
      return;
    }

    // Import OpenInVSCodium from Wails bindings
    const { OpenInVSCodium } = await import("../../wailsjs/go/main/App");
    await OpenInVSCodium(filePath);
    console.log("Successfully opened file in VSCodium:", filePath);
    setResultText(`File opened in VSCodium: ${filePath}`);
  } catch (error: any) {
    console.error("Failed to open file in VSCodium:", error);
    // Provide user feedback
    setResultText(
      `Could not open file in VSCodium: ${error.message || "Operation failed"}`,
    );
    setError(`VSCodium open error: ${error.message || "Operation failed"}`);
  }
};

/**
 * Opens a file in Sublime Text editor
 * @param filePath The path to the file to open in Sublime Text
 * @param setResultText Function to update result text in the UI
 * @param setError Function to update error in the UI
 */
export const openInSublime = async (
  filePath: string,
  setResultText: (text: string) => void,
  setError: (error: string | null) => void,
) => {
  try {
    // Validate input
    if (!filePath || typeof filePath !== "string") {
      console.warn("Invalid file path provided to openInSublime");
      setResultText("Invalid file path");
      return;
    }

    // Import OpenInSublime from Wails bindings
    const { OpenInSublime } = await import("../../wailsjs/go/main/App");
    await OpenInSublime(filePath);
    console.log("Successfully opened file in Sublime Text:", filePath);
    setResultText(`File opened in Sublime Text: ${filePath}`);
  } catch (error: any) {
    console.error("Failed to open file in Sublime Text:", error);
    // Provide user feedback
    setResultText(
      `Could not open file in Sublime Text: ${error.message || "Operation failed"}`,
    );
    setError(`Sublime Text open error: ${error.message || "Operation failed"}`);
  }
};

/**
 * Opens a file in Atom editor
 * @param filePath The path to the file to open in Atom
 * @param setResultText Function to update result text in the UI
 * @param setError Function to update error in the UI
 */
export const openInAtom = async (
  filePath: string,
  setResultText: (text: string) => void,
  setError: (error: string | null) => void,
) => {
  try {
    // Validate input
    if (!filePath || typeof filePath !== "string") {
      console.warn("Invalid file path provided to openInAtom");
      setResultText("Invalid file path");
      return;
    }

    // Import OpenInAtom from Wails bindings
    const { OpenInAtom } = await import("../../wailsjs/go/main/App");
    await OpenInAtom(filePath);
    console.log("Successfully opened file in Atom:", filePath);
    setResultText(`File opened in Atom: ${filePath}`);
  } catch (error: any) {
    console.error("Failed to open file in Atom:", error);
    // Provide user feedback
    setResultText(
      `Could not open file in Atom: ${error.message || "Operation failed"}`,
    );
    setError(`Atom open error: ${error.message || "Operation failed"}`);
  }
};

/**
 * Opens a file in JetBrains IDE
 * @param filePath The path to the file to open in JetBrains IDE
 * @param setResultText Function to update result text in the UI
 * @param setError Function to update error in the UI
 */
export const openInJetBrains = async (
  filePath: string,
  setResultText: (text: string) => void,
  setError: (error: string | null) => void,
) => {
  try {
    // Validate input
    if (!filePath || typeof filePath !== "string") {
      console.warn("Invalid file path provided to openInJetBrains");
      setResultText("Invalid file path");
      return;
    }

    // Import OpenInJetBrains from Wails bindings
    const { OpenInJetBrains } = await import("../../wailsjs/go/main/App");
    await OpenInJetBrains(filePath);
    console.log("Successfully opened file in JetBrains IDE:", filePath);
    setResultText(`File opened in JetBrains IDE: ${filePath}`);
  } catch (error: any) {
    console.error("Failed to open file in JetBrains IDE:", error);
    // Provide user feedback
    setResultText(
      `Could not open file in JetBrains IDE: ${error.message || "Operation failed"}`,
    );
    setError(
      `JetBrains IDE open error: ${error.message || "Operation failed"}`,
    );
  }
};

/**
 * Opens a file in Geany editor
 * @param filePath The path to the file to open in Geany
 * @param setResultText Function to update result text in the UI
 * @param setError Function to update error in the UI
 */
export const openInGeany = async (
  filePath: string,
  setResultText: (text: string) => void,
  setError: (error: string | null) => void,
) => {
  try {
    // Validate input
    if (!filePath || typeof filePath !== "string") {
      console.warn("Invalid file path provided to openInGeany");
      setResultText("Invalid file path");
      return;
    }

    // Import OpenInGeany from Wails bindings
    const { OpenInGeany } = await import("../../wailsjs/go/main/App");
    await OpenInGeany(filePath);
    console.log("Successfully opened file in Geany:", filePath);
    setResultText(`File opened in Geany: ${filePath}`);
  } catch (error: any) {
    console.error("Failed to open file in Geany:", error);
    // Provide user feedback
    setResultText(
      `Could not open file in Geany: ${error.message || "Operation failed"}`,
    );
    setError(`Geany open error: ${error.message || "Operation failed"}`);
  }
};

/**
 * Opens a file in GoLand editor
 * @param filePath The path to the file to open in GoLand
 * @param setResultText Function to update result text in the UI
 * @param setError Function to update error in the UI
 */
export const openInGoland = async (
  filePath: string,
  setResultText: (text: string) => void,
  setError: (error: string | null) => void,
) => {
  try {
    // Validate input
    if (!filePath || typeof filePath !== "string") {
      console.warn("Invalid file path provided to openInGoland");
      setResultText("Invalid file path");
      return;
    }

    // Import OpenInGoland from Wails bindings
    const { OpenInGoland } = await import("../../wailsjs/go/main/App");
    await OpenInGoland(filePath);
    console.log("Successfully opened file in GoLand:", filePath);
    setResultText(`File opened in GoLand: ${filePath}`);
  } catch (error: any) {
    console.error("Failed to open file in GoLand:", error);
    // Provide user feedback
    setResultText(
      `Could not open file in GoLand: ${error.message || "Operation failed"}`,
    );
    setError(`GoLand open error: ${error.message || "Operation failed"}`);
  }
};

/**
 * Opens a file in PyCharm editor
 * @param filePath The path to the file to open in PyCharm
 * @param setResultText Function to update result text in the UI
 * @param setError Function to update error in the UI
 */
export const openInPyCharm = async (
  filePath: string,
  setResultText: (text: string) => void,
  setError: (error: string | null) => void,
) => {
  try {
    // Validate input
    if (!filePath || typeof filePath !== "string") {
      console.warn("Invalid file path provided to openInPyCharm");
      setResultText("Invalid file path");
      return;
    }

    // Import OpenInPyCharm from Wails bindings
    const { OpenInPyCharm } = await import("../../wailsjs/go/main/App");
    await OpenInPyCharm(filePath);
    console.log("Successfully opened file in PyCharm:", filePath);
    setResultText(`File opened in PyCharm: ${filePath}`);
  } catch (error: any) {
    console.error("Failed to open file in PyCharm:", error);
    // Provide user feedback
    setResultText(
      `Could not open file in PyCharm: ${error.message || "Operation failed"}`,
    );
    setError(`PyCharm open error: ${error.message || "Operation failed"}`);
  }
};

/**
 * Opens a file in IntelliJ IDEA editor
 * @param filePath The path to the file to open in IntelliJ IDEA
 * @param setResultText Function to update result text in the UI
 * @param setError Function to update error in the UI
 */
export const openInIntelliJ = async (
  filePath: string,
  setResultText: (text: string) => void,
  setError: (error: string | null) => void,
) => {
  try {
    // Validate input
    if (!filePath || typeof filePath !== "string") {
      console.warn("Invalid file path provided to openInIntelliJ");
      setResultText("Invalid file path");
      return;
    }

    // Import OpenInIntelliJ from Wails bindings
    const { OpenInIntelliJ } = await import("../../wailsjs/go/main/App");
    await OpenInIntelliJ(filePath);
    console.log("Successfully opened file in IntelliJ IDEA:", filePath);
    setResultText(`File opened in IntelliJ IDEA: ${filePath}`);
  } catch (error: any) {
    console.error("Failed to open file in IntelliJ IDEA:", error);
    // Provide user feedback
    setResultText(
      `Could not open file in IntelliJ IDEA: ${error.message || "Operation failed"}`,
    );
    setError(
      `IntelliJ IDEA open error: ${error.message || "Operation failed"}`,
    );
  }
};

/**
 * Opens a file in WebStorm editor
 * @param filePath The path to the file to open in WebStorm
 * @param setResultText Function to update result text in the UI
 * @param setError Function to update error in the UI
 */
export const openInWebStorm = async (
  filePath: string,
  setResultText: (text: string) => void,
  setError: (error: string | null) => void,
) => {
  try {
    // Validate input
    if (!filePath || typeof filePath !== "string") {
      console.warn("Invalid file path provided to openInWebStorm");
      setResultText("Invalid file path");
      return;
    }

    // Import OpenInWebStorm from Wails bindings
    const { OpenInWebStorm } = await import("../../wailsjs/go/main/App");
    await OpenInWebStorm(filePath);
    console.log("Successfully opened file in WebStorm:", filePath);
    setResultText(`File opened in WebStorm: ${filePath}`);
  } catch (error: any) {
    console.error("Failed to open file in WebStorm:", error);
    // Provide user feedback
    setResultText(
      `Could not open file in WebStorm: ${error.message || "Operation failed"}`,
    );
    setError(`WebStorm open error: ${error.message || "Operation failed"}`);
  }
};

/**
 * Opens a file in PhpStorm editor
 * @param filePath The path to the file to open in PhpStorm
 * @param setResultText Function to update result text in the UI
 * @param setError Function to update error in the UI
 */
export const openInPhpStorm = async (
  filePath: string,
  setResultText: (text: string) => void,
  setError: (error: string | null) => void,
) => {
  try {
    // Validate input
    if (!filePath || typeof filePath !== "string") {
      console.warn("Invalid file path provided to openInPhpStorm");
      setResultText("Invalid file path");
      return;
    }

    // Import OpenInPhpStorm from Wails bindings
    const { OpenInPhpStorm } = await import("../../wailsjs/go/main/App");
    await OpenInPhpStorm(filePath);
    console.log("Successfully opened file in PhpStorm:", filePath);
    setResultText(`File opened in PhpStorm: ${filePath}`);
  } catch (error: any) {
    console.error("Failed to open file in PhpStorm:", error);
    // Provide user feedback
    setResultText(
      `Could not open file in PhpStorm: ${error.message || "Operation failed"}`,
    );
    setError(`PhpStorm open error: ${error.message || "Operation failed"}`);
  }
};

/**
 * Opens a file in CLion editor
 * @param filePath The path to the file to open in CLion
 * @param setResultText Function to update result text in the UI
 * @param setError Function to update error in the UI
 */
export const openInCLion = async (
  filePath: string,
  setResultText: (text: string) => void,
  setError: (error: string | null) => void,
) => {
  try {
    // Validate input
    if (!filePath || typeof filePath !== "string") {
      console.warn("Invalid file path provided to openInCLion");
      setResultText("Invalid file path");
      return;
    }

    // Import OpenInCLion from Wails bindings
    const { OpenInCLion } = await import("../../wailsjs/go/main/App");
    await OpenInCLion(filePath);
    console.log("Successfully opened file in CLion:", filePath);
    setResultText(`File opened in CLion: ${filePath}`);
  } catch (error: any) {
    console.error("Failed to open file in CLion:", error);
    // Provide user feedback
    setResultText(
      `Could not open file in CLion: ${error.message || "Operation failed"}`,
    );
    setError(`CLion open error: ${error.message || "Operation failed"}`);
  }
};

/**
 * Opens a file in Rider editor
 * @param filePath The path to the file to open in Rider
 * @param setResultText Function to update result text in the UI
 * @param setError Function to update error in the UI
 */
export const openInRider = async (
  filePath: string,
  setResultText: (text: string) => void,
  setError: (error: string | null) => void,
) => {
  try {
    // Validate input
    if (!filePath || typeof filePath !== "string") {
      console.warn("Invalid file path provided to openInRider");
      setResultText("Invalid file path");
      return;
    }

    // Import OpenInRider from Wails bindings
    const { OpenInRider } = await import("../../wailsjs/go/main/App");
    await OpenInRider(filePath);
    console.log("Successfully opened file in Rider:", filePath);
    setResultText(`File opened in Rider: ${filePath}`);
  } catch (error: any) {
    console.error("Failed to open file in Rider:", error);
    // Provide user feedback
    setResultText(
      `Could not open file in Rider: ${error.message || "Operation failed"}`,
    );
    setError(`Rider open error: ${error.message || "Operation failed"}`);
  }
};

/**
 * Opens a file in Android Studio editor
 * @param filePath The path to the file to open in Android Studio
 * @param setResultText Function to update result text in the UI
 * @param setError Function to update error in the UI
 */
export const openInAndroidStudio = async (
  filePath: string,
  setResultText: (text: string) => void,
  setError: (error: string | null) => void,
) => {
  try {
    // Validate input
    if (!filePath || typeof filePath !== "string") {
      console.warn("Invalid file path provided to openInAndroidStudio");
      setResultText("Invalid file path");
      return;
    }

    // Import OpenInAndroidStudio from Wails bindings
    const { OpenInAndroidStudio } = await import("../../wailsjs/go/main/App");
    await OpenInAndroidStudio(filePath);
    console.log("Successfully opened file in Android Studio:", filePath);
    setResultText(`File opened in Android Studio: ${filePath}`);
  } catch (error: any) {
    console.error("Failed to open file in Android Studio:", error);
    // Provide user feedback
    setResultText(
      `Could not open file in Android Studio: ${error.message || "Operation failed"}`,
    );
    setError(
      `Android Studio open error: ${error.message || "Operation failed"}`,
    );
  }
};

/**
 * Opens a file in the system's default editor
 * @param filePath The path to the file to open in the default editor
 * @param setResultText Function to update result text in the UI
 * @param setError Function to update error in the UI
 */
export const openInDefaultEditor = async (
  filePath: string,
  setResultText: (text: string) => void,
  setError: (error: string | null) => void,
) => {
  try {
    // Validate input
    if (!filePath || typeof filePath !== "string") {
      console.warn("Invalid file path provided to openInDefaultEditor");
      setResultText("Invalid file path");
      return;
    }

    // Import OpenInDefaultEditor from Wails bindings
    const { OpenInDefaultEditor } = await import("../../wailsjs/go/main/App");
    await OpenInDefaultEditor(filePath);
    console.log("Successfully opened file in default editor:", filePath);
    setResultText(`File opened in default editor: ${filePath}`);
  } catch (error: any) {
    console.error("Failed to open file in default editor:", error);
    // Provide user feedback
    setResultText(
      `Could not open file in default editor: ${error.message || "Operation failed"}`,
    );
    setError(
      `Default editor open error: ${error.message || "Operation failed"}`,
    );
  }
};

/**
 * Opens a file in Emacs editor
 * @param filePath The path to the file to open in Emacs
 * @param setResultText Function to update result text in the UI
 * @param setError Function to update error in the UI
 */
export const openInEmacs = async (
  filePath: string,
  setResultText: (text: string) => void,
  setError: (error: string | null) => void,
) => {
  try {
    // Validate input
    if (!filePath || typeof filePath !== "string") {
      console.warn("Invalid file path provided to openInEmacs");
      setResultText("Invalid file path");
      return;
    }

    // Import OpenInEmacs from Wails bindings
    const { OpenInEmacs } = await import("../../wailsjs/go/main/App");
    await OpenInEmacs(filePath);
    console.log("Successfully opened file in Emacs:", filePath);
    setResultText(`File opened in Emacs: ${filePath}`);
  } catch (error: any) {
    console.error("Failed to open file in Emacs:", error);
    // Provide user feedback
    setResultText(
      `Could not open file in Emacs: ${error.message || "Operation failed"}`,
    );
    setError(`Emacs open error: ${error.message || "Operation failed"}`);
  }
};

/**
 * Opens a file in Neovide editor
 * @param filePath The path to the file to open in Neovide
 * @param setResultText Function to update result text in the UI
 * @param setError Function to update error in the UI
 */
export const openInNeovide = async (
  filePath: string,
  setResultText: (text: string) => void,
  setError: (error: string | null) => void,
) => {
  try {
    // Validate input
    if (!filePath || typeof filePath !== "string") {
      console.warn("Invalid file path provided to openInNeovide");
      setResultText("Invalid file path");
      return;
    }

    // Import OpenInNeovide from Wails bindings
    const { OpenInNeovide } = await import("../../wailsjs/go/main/App");
    await OpenInNeovide(filePath);
    console.log("Successfully opened file in Neovide:", filePath);
    setResultText(`File opened in Neovide: ${filePath}`);
  } catch (error: any) {
    console.error("Failed to open file in Neovide:", error);
    // Provide user feedback
    setResultText(
      `Could not open file in Neovide: ${error.message || "Operation failed"}`,
    );
    setError(`Neovide open error: ${error.message || "Operation failed"}`);
  }
};

/**
 * Opens a file in Code::Blocks editor
 * @param filePath The path to the file to open in Code::Blocks
 * @param setResultText Function to update result text in the UI
 * @param setError Function to update error in the UI
 */
export const openInCodeBlocks = async (
  filePath: string,
  setResultText: (text: string) => void,
  setError: (error: string | null) => void,
) => {
  try {
    // Validate input
    if (!filePath || typeof filePath !== "string") {
      console.warn("Invalid file path provided to openInCodeBlocks");
      setResultText("Invalid file path");
      return;
    }

    // Import OpenInCodeBlocks from Wails bindings
    const { OpenInCodeBlocks } = await import("../../wailsjs/go/main/App");
    await OpenInCodeBlocks(filePath);
    console.log("Successfully opened file in Code::Blocks:", filePath);
    setResultText(`File opened in Code::Blocks: ${filePath}`);
  } catch (error: any) {
    console.error("Failed to open file in Code::Blocks:", error);
    // Provide user feedback
    setResultText(
      `Could not open file in Code::Blocks: ${error.message || "Operation failed"}`,
    );
    setError(`Code::Blocks open error: ${error.message || "Operation failed"}`);
  }
};

/**
 * Opens a file in Dev-C++ editor
 * @param filePath The path to the file to open in Dev-C++
 * @param setResultText Function to update result text in the UI
 * @param setError Function to update error in the UI
 */
export const openInDevCpp = async (
  filePath: string,
  setResultText: (text: string) => void,
  setError: (error: string | null) => void,
) => {
  try {
    // Validate input
    if (!filePath || typeof filePath !== "string") {
      console.warn("Invalid file path provided to openInDevCpp");
      setResultText("Invalid file path");
      return;
    }

    // Import OpenInDevCpp from Wails bindings
    const { OpenInDevCpp } = await import("../../wailsjs/go/main/App");
    await OpenInDevCpp(filePath);
    console.log("Successfully opened file in Dev-C++:", filePath);
    setResultText(`File opened in Dev-C++: ${filePath}`);
  } catch (error: any) {
    console.error("Failed to open file in Dev-C++:", error);
    // Provide user feedback
    setResultText(
      `Could not open file in Dev-C++: ${error.message || "Operation failed"}`,
    );
    setError(`Dev-C++ open error: ${error.message || "Operation failed"}`);
  }
};

/**
 * Opens a file in Notepad++ editor
 * @param filePath The path to the file to open in Notepad++
 * @param setResultText Function to update result text in the UI
 * @param setError Function to update error in the UI
 */
export const openInNotepadPlusPlus = async (
  filePath: string,
  setResultText: (text: string) => void,
  setError: (error: string | null) => void,
) => {
  try {
    // Validate input
    if (!filePath || typeof filePath !== "string") {
      console.warn("Invalid file path provided to openInNotepadPlusPlus");
      setResultText("Invalid file path");
      return;
    }

    // Import OpenInNotepadPlusPlus from Wails bindings
    const { OpenInNotepadPlusPlus } = await import("../../wailsjs/go/main/App");
    await OpenInNotepadPlusPlus(filePath);
    console.log("Successfully opened file in Notepad++:", filePath);
    setResultText(`File opened in Notepad++: ${filePath}`);
  } catch (error: any) {
    console.error("Failed to open file in Notepad++:", error);
    // Provide user feedback
    setResultText(
      `Could not open file in Notepad++: ${error.message || "Operation failed"}`,
    );
    setError(`Notepad++ open error: ${error.message || "Operation failed"}`);
  }
};

/**
 * Opens a file in Visual Studio editor
 * @param filePath The path to the file to open in Visual Studio
 * @param setResultText Function to update result text in the UI
 * @param setError Function to update error in the UI
 */
export const openInVisualStudio = async (
  filePath: string,
  setResultText: (text: string) => void,
  setError: (error: string | null) => void,
) => {
  try {
    // Validate input
    if (!filePath || typeof filePath !== "string") {
      console.warn("Invalid file path provided to openInVisualStudio");
      setResultText("Invalid file path");
      return;
    }

    // Import OpenInVisualStudio from Wails bindings
    const { OpenInVisualStudio } = await import("../../wailsjs/go/main/App");
    await OpenInVisualStudio(filePath);
    console.log("Successfully opened file in Visual Studio:", filePath);
    setResultText(`File opened in Visual Studio: ${filePath}`);
  } catch (error: any) {
    console.error("Failed to open file in Visual Studio:", error);
    // Provide user feedback
    setResultText(
      `Could not open file in Visual Studio: ${error.message || "Operation failed"}`,
    );
    setError(
      `Visual Studio open error: ${error.message || "Operation failed"}`,
    );
  }
};

/**
 * Opens a file in Eclipse IDE
 * @param filePath The path to the file to open in Eclipse
 * @param setResultText Function to update result text in the UI
 * @param setError Function to update error in the UI
 */
export const openInEclipse = async (
  filePath: string,
  setResultText: (text: string) => void,
  setError: (error: string | null) => void,
) => {
  try {
    // Validate input
    if (!filePath || typeof filePath !== "string") {
      console.warn("Invalid file path provided to openInEclipse");
      setResultText("Invalid file path");
      return;
    }

    // Import OpenInEclipse from Wails bindings
    const { OpenInEclipse } = await import("../../wailsjs/go/main/App");
    await OpenInEclipse(filePath);
    console.log("Successfully opened file in Eclipse:", filePath);
    setResultText(`File opened in Eclipse: ${filePath}`);
  } catch (error: any) {
    console.error("Failed to open file in Eclipse:", error);
    // Provide user feedback
    setResultText(
      `Could not open file in Eclipse: ${error.message || "Operation failed"}`,
    );
    setError(`Eclipse open error: ${error.message || "Operation failed"}`);
  }
};

/**
 * Opens a file in NetBeans IDE
 * @param filePath The path to the file to open in NetBeans
 * @param setResultText Function to update result text in the UI
 * @param setError Function to update error in the UI
 */
export const openInNetBeans = async (
  filePath: string,
  setResultText: (text: string) => void,
  setError: (error: string | null) => void,
) => {
  try {
    // Validate input
    if (!filePath || typeof filePath !== "string") {
      console.warn("Invalid file path provided to openInNetBeans");
      setResultText("Invalid file path");
      return;
    }

    // Import OpenInNetBeans from Wails bindings
    const { OpenInNetBeans } = await import("../../wailsjs/go/main/App");
    await OpenInNetBeans(filePath);
    console.log("Successfully opened file in NetBeans:", filePath);
    setResultText(`File opened in NetBeans: ${filePath}`);
  } catch (error: any) {
    console.error("Failed to open file in NetBeans:", error);
    // Provide user feedback
    setResultText(
      `Could not open file in NetBeans: ${error.message || "Operation failed"}`,
    );
    setError(`NetBeans open error: ${error.message || "Operation failed"}`);
  }
};
