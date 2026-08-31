import { describe, test, expect } from 'vitest';
import {
  findMatchRanges,
  buildDiffSegments,
  renderDiffHtml,
} from '@/utils';

describe('findMatchRanges', () => {
  test('finds a single match', () => {
    expect(findMatchRanges('hello world', 'world', false)).toEqual([
      { start: 6, end: 11 },
    ]);
  });

  test('finds multiple matches on one line', () => {
    expect(findMatchRanges('test test test', 'test', false)).toEqual([
      { start: 0, end: 4 },
      { start: 5, end: 9 },
      { start: 10, end: 14 },
    ]);
  });

  test('is case-insensitive by default', () => {
    expect(findMatchRanges('Hello HELLO hello', 'hello', false)).toHaveLength(3);
  });

  test('is case-sensitive when requested', () => {
    expect(findMatchRanges('Hello HELLO hello', 'hello', true)).toHaveLength(1);
  });

  test('escapes regex special characters in query', () => {
    expect(findMatchRanges('a.b.c', 'a.b', false)).toEqual([
      { start: 0, end: 3 },
    ]);
  });

  test('returns empty for empty inputs', () => {
    expect(findMatchRanges('', 'test', false)).toEqual([]);
    expect(findMatchRanges('test', '', false)).toEqual([]);
  });
});

describe('buildDiffSegments', () => {
  test('returns single normal segment when no matches', () => {
    const segs = buildDiffSegments('hello world', []);
    expect(segs).toEqual([{ text: 'hello world', type: 'normal' }]);
  });

  test('splits into normal + match + normal', () => {
    const segs = buildDiffSegments('hello world', [
      { start: 6, end: 11 },
    ]);
    expect(segs).toEqual([
      { text: 'hello ', type: 'normal' },
      { text: 'world', type: 'match' },
    ]);
  });

  test('handles match at start of line', () => {
    const segs = buildDiffSegments('test content', [
      { start: 0, end: 4 },
    ]);
    expect(segs[0]).toEqual({ text: 'test', type: 'match' });
    expect(segs[1]).toEqual({ text: ' content', type: 'normal' });
  });

  test('handles multiple matches', () => {
    const segs = buildDiffSegments('aXbXc', [
      { start: 1, end: 2 },
      { start: 3, end: 4 },
    ]);
    expect(segs).toEqual([
      { text: 'a', type: 'normal' },
      { text: 'X', type: 'match' },
      { text: 'b', type: 'normal' },
      { text: 'X', type: 'match' },
      { text: 'c', type: 'normal' },
    ]);
  });

  test('truncates long lines around the first match', () => {
    const longLine = 'x'.repeat(300) + 'NEEDLE' + 'y'.repeat(300);
    const segs = buildDiffSegments(longLine, [
      { start: 300, end: 306 },
    ]);
    // Should start with a truncation marker
    expect(segs[0]).toEqual({ text: '…', type: 'truncation' });
    // Should end with a truncation marker
    expect(segs[segs.length - 1]).toEqual({ text: '…', type: 'truncation' });
    // Should contain the match
    expect(segs.some((s) => s.type === 'match' && s.text === 'NEEDLE')).toBe(true);
  });

  test('does not truncate short lines', () => {
    const segs = buildDiffSegments('short line with NEEDLE', [
      { start: 16, end: 22 },
    ]);
    expect(segs.some((s) => s.type === 'truncation')).toBe(false);
  });

  test('handles empty line', () => {
    expect(buildDiffSegments('', [])).toEqual([]);
  });
});

describe('renderDiffHtml', () => {
  test('wraps match segments in mark tags', () => {
    const html = renderDiffHtml([
      { text: 'hello ', type: 'normal' },
      { text: 'world', type: 'match' },
    ]);
    expect(html).toContain('<mark class="diff-match">world</mark>');
    expect(html).toContain('hello ');
  });

  test('wraps truncation in span', () => {
    const html = renderDiffHtml([
      { text: '…', type: 'truncation' },
    ]);
    expect(html).toContain('<span class="diff-truncation">…</span>');
  });

  test('escapes script tags to inert text', () => {
    const html = renderDiffHtml([
      { text: '<script>alert(1)</script>', type: 'normal' },
    ]);
    // escapeHtml runs before DOMPurify, so the tag survives as visible text
    // instead of executable markup.
    expect(html).not.toContain('<script>');
    expect(html).toBe('&lt;script&gt;alert(1)&lt;/script&gt;');
  });

  test('returns empty string for no segments', () => {
    expect(renderDiffHtml([])).toBe('');
  });
});

describe('findMatchRanges — edge cases', () => {
  test('handles regex special chars in query', () => {
    expect(findMatchRanges('a(b)c', '(b)', false)).toEqual([
      { start: 1, end: 4 },
    ]);
  });

  test('handles overlapping-like patterns (greedy)', () => {
    // Regex is greedy: 'aa' consumes positions 0-1, leaving 'a' at 2 which
    // doesn't match 'aa'. So only one match.
    expect(findMatchRanges('aaa', 'aa', false)).toEqual([
      { start: 0, end: 2 },
    ]);
  });

  test('empty query returns empty', () => {
    expect(findMatchRanges('text', '', false)).toEqual([]);
  });

  test('query longer than text returns empty', () => {
    expect(findMatchRanges('hi', 'hello', false)).toEqual([]);
  });

  test('case-sensitive: only exact case matches', () => {
    const ranges = findMatchRanges('Foo foo FOO', 'Foo', true);
    expect(ranges).toEqual([{ start: 0, end: 3 }]);
  });

  test('case-insensitive: all case variants match', () => {
    const ranges = findMatchRanges('Foo foo FOO', 'foo', false);
    expect(ranges).toHaveLength(3);
  });
});

describe('buildDiffSegments — truncation edge cases', () => {
  test('truncates with match at very end of long line', () => {
    const longLine = 'x'.repeat(250) + 'NEEDLE';
    const segs = buildDiffSegments(longLine, [
      { start: 250, end: 256 },
    ]);
    expect(segs[0].type).toBe('truncation');
    expect(segs.some((s) => s.type === 'match' && s.text === 'NEEDLE')).toBe(true);
  });

  test('truncates with match at very start of long line', () => {
    const longLine = 'NEEDLE' + 'x'.repeat(250);
    const segs = buildDiffSegments(longLine, [
      { start: 0, end: 6 },
    ]);
    expect(segs.some((s) => s.type === 'match' && s.text === 'NEEDLE')).toBe(true);
    expect(segs[segs.length - 1].type).toBe('truncation');
  });

  test('multiple matches in long line: only visible ones kept', () => {
    const longLine = 'NEEDLE' + 'x'.repeat(250) + 'NEEDLE';
    const segs = buildDiffSegments(longLine, [
      { start: 0, end: 6 },
      { start: 256, end: 262 },
    ]);
    const matches = segs.filter((s) => s.type === 'match');
    // First match should be in the visible window
    expect(matches.some((s) => s.text === 'NEEDLE')).toBe(true);
  });

  test('short line with match at end: no truncation', () => {
    const segs = buildDiffSegments('hello NEEDLE', [
      { start: 6, end: 12 },
    ]);
    expect(segs.some((s) => s.type === 'truncation')).toBe(false);
    expect(segs[segs.length - 1]).toEqual({ text: 'NEEDLE', type: 'match' });
  });

  test('range straddling the slice boundary is clamped, not dropped', () => {
    // The old filter dropped any range not fully inside the window. A match
    // straddling sliceEnd must be clamped to the window instead.
    const longLine = 'x'.repeat(300) + 'NEEDLE' + 'y'.repeat(300);
    const segs = buildDiffSegments(longLine, [
      { start: 300, end: 306 }, // first match drives the window [260, 346)
      { start: 340, end: 350 }, // straddles sliceEnd (346)
    ]);
    const matches = segs.filter((s) => s.type === 'match');
    expect(matches).toHaveLength(2);
    expect(matches[0].text).toBe('NEEDLE');
    // Clamped to the visible window: chars 340..345 = 'yyyyyy'
    expect(matches[1].text).toBe('yyyyyy');
  });

  test('ranges fully outside the visible window are dropped', () => {
    const longLine = 'x'.repeat(300) + 'NEEDLE' + 'y'.repeat(300);
    const segs = buildDiffSegments(longLine, [
      { start: 300, end: 306 },
      { start: 500, end: 510 }, // far outside [260, 346)
    ]);
    const matches = segs.filter((s) => s.type === 'match');
    expect(matches).toHaveLength(1);
    expect(matches[0].text).toBe('NEEDLE');
  });
});
