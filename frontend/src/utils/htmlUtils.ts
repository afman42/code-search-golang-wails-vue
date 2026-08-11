/**
 * HTML utility functions for escaping and sanitization.
 */

/**
 * Escapes HTML special characters to prevent XSS attacks.
 * Converts &, <, >, ", ' to their HTML entities.
 */
export const escapeHtml = (unsafe: string): string => {
  if (!unsafe) return "";
  return unsafe
    .replace(/&/g, "&amp;")
    .replace(/</g, "&lt;")
    .replace(/>/g, "&gt;")
    .replace(/"/g, "&quot;")
    .replace(/'/g, "&#039;");
};
