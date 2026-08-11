import { describe, test, expect } from 'vitest';
import { findFuzzyMatches } from '@/utils';

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
