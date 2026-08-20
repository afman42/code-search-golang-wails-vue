// Escape regex special characters in a string so it can be used as a literal
// pattern in a RegExp constructor. Covers all 14 special regex metacharacters.
export function escapeRegExp(s: string): string {
  return s.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");
}