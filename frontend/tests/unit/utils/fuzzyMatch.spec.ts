import { describe, test, expect } from 'vitest';
import { fuzzyMatch, findFuzzyMatches, normalizeQuery } from '../../../src/utils/fuzzyMatch';

describe('fuzzyMatch', () => {
  test('matches identical text', () => {
    expect(fuzzyMatch('test message', 'test message')).toBe(true);
  });

  test('is case-insensitive', () => {
    expect(fuzzyMatch('Test Message', 'test message')).toBe(true);
    expect(fuzzyMatch('TEST MESSAGE', 'TEST message')).toBe(true);
  });

  test('ignores whitespace in the query', () => {
    expect(fuzzyMatch('test message', 'test   message')).toBe(true);
    expect(fuzzyMatch('testmessage', 'test message')).toBe(true);
  });

  test('matches ordered subsequences above the similarity threshold', () => {
    expect(fuzzyMatch('the quick brown fox', 'quick fox')).toBe(true);
    expect(fuzzyMatch('test message', 'test messgae')).toBe(true);
  });

  test('rejects queries that are too dissimilar (below threshold)', () => {
    // Fewer than 80% of the query characters appear in order.
    expect(fuzzyMatch('hello world', 'zzzzzzzzzzzzzzzzzzzz')).toBe(false);
    expect(fuzzyMatch('alpha beta gamma', 'delta epsilon')).toBe(false);
  });

  test('rejects matches with too few ordered characters', () => {
    // Only 3 of 5 query chars match in order (< 80% = 4).
    expect(fuzzyMatch('abc def', 'abcxy')).toBe(false);
  });

  test('returns false when query is longer than text', () => {
    expect(fuzzyMatch('short', 'a much longer query than the text')).toBe(false);
  });

  test('handles empty inputs', () => {
    expect(fuzzyMatch('', 'query')).toBe(false);
    expect(fuzzyMatch('text', '')).toBe(false);
    expect(fuzzyMatch('', '')).toBe(false);
  });

  test('matches punctuation and numbers in order', () => {
    expect(fuzzyMatch('const x = 42;', 'x=42')).toBe(true);
  });
});

describe('findFuzzyMatches', () => {
  test('finds sliding-window matches above the similarity threshold', () => {
    const matches = findFuzzyMatches('test message', 'test');
    expect(matches.length).toBeGreaterThan(0);
    expect(matches[0].start).toBe(0);
    expect(matches[0].matchedChars.length).toBe(4);
  });

  test('is case-insensitive', () => {
    const matches = findFuzzyMatches('TEST MESSAGE', 'test');
    expect(matches.length).toBeGreaterThan(0);
  });

  test('returns matches across the whole text, not just the start', () => {
    const matches = findFuzzyMatches('aaa test aaa test', 'test');
    expect(matches.length).toBe(2);
  });

  test('returns empty array for too-dissimilar windows', () => {
    const matches = findFuzzyMatches('abcdefghij', 'zxy');
    expect(matches).toEqual([]);
  });

  test('handles empty inputs', () => {
    expect(findFuzzyMatches('', 'test')).toEqual([]);
    expect(findFuzzyMatches('test', '')).toEqual([]);
  });

  test('bails out on very long texts to avoid O(n*m) blowups', () => {
    const longText = 'x'.repeat(50001) + 'test';
    const matches = findFuzzyMatches(longText, 'test');
    expect(matches).toEqual([]);
  });
});

describe('normalizeQuery', () => {
  test('lowercases and trims', () => {
    expect(normalizeQuery('  Hello  ')).toBe('hello');
  });

  test('collapses internal whitespace runs', () => {
    expect(normalizeQuery('a   b\t\tc')).toBe('a b c');
  });

  test('strips punctuation not relevant to code queries', () => {
    expect(normalizeQuery('func(abc){}')).toBe('funcabc');
  });

  test('preserves tokens that appear in code (dots, dashes, slashes)', () => {
    expect(normalizeQuery('./src/utils - test')).toBe('./src/utils - test');
  });

  test('handles empty input', () => {
    expect(normalizeQuery('')).toBe('');
  });
});
