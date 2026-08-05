// Diff utilities for the InlineDiffView component.
//
// The previous InlineDiffView only wrapped the FIRST query occurrence in a
// <mark> via String.replace (which replaces one match in modern engines). This
// module computes ALL match ranges, builds segmented HTML with git-style diff
// markers, and truncates very long lines around the matches so the result
// panel stays readable for minified/generated code.

import DOMPurify from "dompurify";

export interface MatchRange {
  start: number;
  end: number;
}

/**
 * Find all match ranges of `query` in `text`. Honors case-sensitivity.
 * For regex queries the caller should pass the already-compiled pattern; this
 * helper is for the common literal / escaped-query path.
 */
export function findMatchRanges(
  text: string,
  query: string,
  caseSensitive: boolean,
): MatchRange[] {
  if (!query || !text) return [];
  const flags = caseSensitive ? "g" : "gi";
  const escaped = query.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");
  const re = new RegExp(escaped, flags);
  const ranges: MatchRange[] = [];
  let m: RegExpExecArray | null;
  while ((m = re.exec(text)) !== null) {
    if (m[0].length === 0) {
      re.lastIndex++;
      continue;
    }
    ranges.push({ start: m.index, end: m.index + m[0].length });
  }
  return ranges;
}

/** Maximum line length before truncation kicks in. */
const MAX_LINE_DISPLAY = 200;
/** Characters of context shown on each side of a match when truncating. */
const TRUNCATION_CONTEXT = 40;

export interface DiffSegment {
  text: string;
  type: "normal" | "match" | "truncation";
}

/**
 * Split a line into segments: non-match text, match spans, and truncation
 * markers. If the line exceeds MAX_LINE_DISPLAY and has matches, it is sliced
 * around the first match range with ellipses on either side.
 */
export function buildDiffSegments(
  line: string,
  ranges: MatchRange[],
): DiffSegment[] {
  if (!line) return [];
  if (ranges.length === 0) return [{ text: line, type: "normal" }];

  // Truncate long lines around the first match.
  let workingText = line;
  let offset = 0;
  const firstMatch = ranges[0];
  if (line.length > MAX_LINE_DISPLAY) {
    const sliceStart = Math.max(0, firstMatch.start - TRUNCATION_CONTEXT);
    const sliceEnd = Math.min(
      line.length,
      firstMatch.end + TRUNCATION_CONTEXT,
    );
    workingText = line.slice(sliceStart, sliceEnd);
    offset = sliceStart;
    ranges = ranges
      .filter((r) => r.start >= sliceStart && r.end <= sliceEnd)
      .map((r) => ({ start: r.start - offset, end: r.end - offset }));
  }

  const segments: DiffSegment[] = [];
  let cursor = 0;
  const hadPrefix = offset > 0;
  const hadSuffix = offset + workingText.length < line.length;

  if (hadPrefix) {
    segments.push({ text: "…", type: "truncation" });
  }

  for (const range of ranges) {
    if (range.start > cursor) {
      segments.push({
        text: workingText.slice(cursor, range.start),
        type: "normal",
      });
    }
    segments.push({
      text: workingText.slice(range.start, range.end),
      type: "match",
    });
    cursor = range.end;
  }
  if (cursor < workingText.length) {
    segments.push({
      text: workingText.slice(cursor),
      type: "normal",
    });
  }

  if (hadSuffix) {
    segments.push({ text: "…", type: "truncation" });
  }

  return segments;
}

/**
 * Render diff segments as sanitized HTML. Match segments get a
 * <mark class="diff-match"> wrapper; truncation markers get a span.
 */
export function renderDiffHtml(segments: DiffSegment[]): string {
  let html = "";
  for (const seg of segments) {
    const escaped = escapeHtml(seg.text);
    if (seg.type === "match") {
      html += `<mark class="diff-match">${escaped}</mark>`;
    } else if (seg.type === "truncation") {
      html += `<span class="diff-truncation">${escaped}</span>`;
    } else {
      html += escaped;
    }
  }
  return DOMPurify.sanitize(html, {
    ALLOWED_TAGS: ["mark", "span"],
    ALLOWED_ATTR: ["class"],
  });
}

function escapeHtml(s: string): string {
  return s
    .replace(/&/g, "&amp;")
    .replace(/</g, "&lt;")
    .replace(/>/g, "&gt;")
    .replace(/"/g, "&quot;")
    .replace(/'/g, "&#039;");
}
