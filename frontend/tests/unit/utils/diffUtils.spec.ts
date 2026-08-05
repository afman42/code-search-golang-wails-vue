import { describe, test, expect } from 'vitest';
import {
  findMatchRanges,
  buildDiffSegments,
  renderDiffHtml,
} from '../../../src/utils/diffUtils';

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

  test('escapes HTML in content', () => {
    const html = renderDiffHtml([
      { text: '<script>alert(1)</script>', type: 'normal' },
    ]);
    expect(html).not.toContain('<script>');
    expect(html).toContain('&lt;script&gt;');
  });

  test('returns empty string for no segments', () => {
    expect(renderDiffHtml([])).toBe('');
  });
});
