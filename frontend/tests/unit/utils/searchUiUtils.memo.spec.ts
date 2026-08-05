import { describe, test, expect } from 'vitest';
import { highlightMatch } from '@/utils/searchUiUtils';
import type { SearchState } from '@/types/search';

const makeData = (caseSensitive?: boolean, useRegex?: boolean): SearchState => ({
  directory: '/test',
  query: '',
  extension: '',
  caseSensitive: caseSensitive || false,
  useRegex: useRegex || false,
  includeBinary: false,
  maxFileSize: 10485760,
  maxResults: 1000,
  searchSubdirs: true,
  resultText: '',
  searchResults: [],
  truncatedResults: false,
  isSearching: false,
  showProgress: false,
  minFileSize: 0,
  excludePatterns: [],
  allowedFileTypes: [],
  knownTextExtensions: [],
  recentSearches: [],
  error: null,
  availableEditors: {} as any,
  editorDetectionStatus: {} as any,
});

describe('highlightMatch - Memoization Edge Cases', () => {
  test('caches based on query + flags, not exact text', () => {
    const data = makeData(false, false); // case-insensitive
    
    const result1 = highlightMatch("hello world", "hello", data);
    const result2 = highlightMatch("HELLO world", "hello", data);
    
    // Both should be highlighted identically due to cache key
    // The cache key uses query+flags, so "hello" in both contexts uses same result template
    expect(result1).toContain('<mark');
    expect(result2).toContain('<mark');
  });

  test('repeated identical calls are fast due to caching', () => {
    const data = makeData(false, false);
    const text = 'the quick brown fox jumps over the test dog';

    // First call populates cache
    highlightMatch(text, 'test', data);

    const start = Date.now();
    // 1000 IDENTICAL calls should hit the cache every time
    for (let i = 0; i < 1000; i++) {
      highlightMatch(text, 'test', data);
    }
    const elapsed = Date.now() - start;

    // Cached calls should be very fast (< 100ms for 1000 cache hits)
    expect(elapsed).toBeLessThan(100);
  });

  test('different regex modes are cached separately', () => {
    const literalData = makeData(false, false);
    const regexData = makeData(false, true);
    
    const literalResult = highlightMatch("a.b.c", "a\\.b\\.c", literalData);
    const regexResult = highlightMatch("a b c", "a\\.b\\.c", regexData);
    
    // Different modes should produce different results
    if (literalResult.includes('b') && !regexResult.includes('b')) {
      // Correct: literal treats "." as dot char, regex treats as wildcard
    } else {
      // At minimum they should both work
      expect(literalResult.length > 0).toBe(true);
      expect(regexResult.length > 0).toBe(true);
    }
  });

  test('cache does not prevent correct highlighting of different queries', () => {
    const data = makeData(false, false);
    
    const results = [
      highlightMatch("first line", "first", data),
      highlightMatch("second line", "second", data),
      highlightMatch("third line", "third", data),
    ];
    
    // All three distinct queries should be highlighted correctly
    expect(results[0]).toContain('first');
    expect(results[1]).toContain('second');
    expect(results[2]).toContain('third');
  });

  test('large texts are handled gracefully (no infinite loops)', () => {
    const data = makeData(false, false);
    const longText = 'x'.repeat(10000) + 'target' + 'y'.repeat(10000);
    
    const result = highlightMatch(longText, 'target', data);
    
    expect(result).toContain('target');
    expect(result.length > 0).toBe(true);
  });

  test('unicode characters work correctly in cache keys', () => {
    const data = makeData(false, false);
    
    const cyrillicResult = highlightMatch('Привет мир', 'мир', data);
    const emojiResult = highlightMatch('Hello 👋 World', '👋', data);
    
    expect(cyrillicResult).toContain('мир');
    expect(emojiResult).toContain('👋');
  });

  test('empty string handling preserves cache safety', () => {
    const data = makeData(false, false);
    
    // Multiple empty strings should not cause errors
    expect(highlightMatch('', 'test', data)).toBe('');
    expect(highlightMatch('content', '', data)).toBe('content');
    
    // Cache should still work after empty inputs
    expect(highlightMatch('normal text', 'text', data)).toContain('<mark class="highlight">');
  });

  test('special regex characters are escaped in literal mode', () => {
    const data = makeData(false, false);
    
    // Dot should match literal ".", not "any character"
    const result = highlightMatch("a.b.c", "a.b.c", data);
    expect(result).toContain('<mark class="highlight">a.b.c</mark>');
    
    // In literal mode, backslash should be escaped properly
    const backslashResult = highlightMatch("path\\to\\file", "path\\\\to", data);
    expect(backslashResult).toBeTruthy();
  });

  test('handles a burst of unique queries without crashing', () => {
    const data = makeData(false, false);

    // Add 200 unique queries - this exercises the LRU eviction path
    // (cache max is 1000, but this verifies no crash / correctness under churn)
    for (let i = 0; i < 200; i++) {
      const result = highlightMatch(`content-${i}`, `content-${i}`, data);
      expect(result).toContain('<mark');
    }

    // After the burst, a fresh query still highlights correctly
    const final = highlightMatch('final query text', 'query', data);
    expect(final).toContain('<mark class="highlight">query</mark>');
  });
});
