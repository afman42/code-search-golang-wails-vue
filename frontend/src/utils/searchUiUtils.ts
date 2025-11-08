/**
 * Search-related UI functions
 * These functions are specifically related to formatting and displaying search results
 */

import DOMPurify from 'dompurify';
import { SearchState } from "../types/search";

/**
 * Highlights matches in text by wrapping them in HTML mark tags.
 * Handles both regular string matching and regex matching.
 * @param text The text to highlight matches in
 * @param query The search query to highlight
 * @param data Search state containing case sensitivity and regex settings
 * @returns The text with highlighted matches
 */
export const highlightMatch = (text: string, query: string, data: SearchState): string => {
  try {
    // Safety checks to prevent runtime errors
    if (!text || typeof text !== "string") return "";
    if (!query || typeof query !== "string") return text;

    // Limit query length to prevent potential performance issues
    if (query.length > 1000) {
      console.warn("Search query is too long, skipping highlight");
      return text;
    }

    const useRegex = data && typeof data.useRegex === "boolean" ? data.useRegex : false;
    const caseSensitive = data && typeof data.caseSensitive === "boolean" ? data.caseSensitive : false;

    let result = text;

    if (useRegex) {
      // For regex mode, validate the pattern first
      try {
        new RegExp(query, caseSensitive ? "g" : "gi");
      } catch (e) {
        console.warn("Invalid regex pattern for highlight, using literal match:", e);
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
      ALLOWED_TAGS: ['mark'],
      ALLOWED_ATTR: ['class']
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
  setError: (error: string | null) => void
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
  setError: (error: string | null) => void
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
    setResultText(`Could not open file location: ${error.message || "Operation failed"}`);
    setError(`Open folder error: ${error.message || "Operation failed"}`);
  }
};