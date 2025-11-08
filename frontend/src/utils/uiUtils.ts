import DOMPurify from 'dompurify';

/**
 * Highlights matched text in search results
 * @param text - The original text to highlight in
 * @param matchedText - The text that was matched
 * @param caseSensitive - Whether the search was case sensitive
 * @returns HTML string with highlighted matches
 */
export const highlightMatch = (text: string, matchedText: string, caseSensitive: boolean = false): string => {
  if (!matchedText) return DOMPurify.sanitize(text);
  
  // Escape special regex characters in the matched text
  const escapedMatchedText = matchedText.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
  
  // Create a regex pattern for the matched text
  const flags = caseSensitive ? 'g' : 'gi';
  const pattern = new RegExp(`(${escapedMatchedText})`, flags);
  
  // Replace matches with highlighted spans
  const highlightedText = text.replace(pattern, '<span class="highlight">$1</span>');
  
  // Sanitize the result to prevent XSS
  return DOMPurify.sanitize(highlightedText);
};

/**
 * Copies text to clipboard
 * @param text - The text to copy
 * @returns Promise that resolves when text is copied
 */
export const copyToClipboard = async (text: string): Promise<void> => {
  try {
    await navigator.clipboard.writeText(text);
    console.log('Text copied to clipboard');
  } catch (err) {
    console.error('Failed to copy text: ', err);
    // Fallback for older browsers
    const textArea = document.createElement('textarea');
    textArea.value = text;
    document.body.appendChild(textArea);
    textArea.select();
    document.execCommand('copy');
    document.body.removeChild(textArea);
  }
};