import { describe, test, expect, beforeEach, vi } from 'vitest';
import {
  RECENT_SEARCHES_KEY,
  loadRecentSearches,
  saveRecentSearches,
  recentSearchKey,
  removeRecentSearch,
} from '../../../src/utils/localStorageUtils';
import type { RecentSearch } from '../../../src/types/recentSearch';

beforeEach(() => {
  localStorage.clear();
  vi.restoreAllMocks();
});

describe('localStorageUtils', () => {
  test('saveRecentSearches then loadRecentSearches round-trips', () => {
    const searches: RecentSearch[] = [
      { query: 'foo', extension: 'go' },
      { query: 'bar', extension: '', directory: '/tmp' },
    ];

    saveRecentSearches(searches);
    expect(loadRecentSearches()).toEqual(searches);
  });

  test('loadRecentSearches returns [] when nothing is stored', () => {
    expect(loadRecentSearches()).toEqual([]);
  });

  test('loadRecentSearches returns [] when the stored value is not an array', () => {
    localStorage.setItem(RECENT_SEARCHES_KEY, JSON.stringify({ not: 'an array' }));
    expect(loadRecentSearches()).toEqual([]);
  });

  test('loadRecentSearches returns [] when the stored value is invalid JSON', () => {
    localStorage.setItem(RECENT_SEARCHES_KEY, '{broken json!!');
    expect(loadRecentSearches()).toEqual([]);
  });

  test('saveRecentSearches tolerates storage quota exhaustion without throwing', () => {
    // Simulate a full / disabled localStorage: setItem throws.
    vi.spyOn(localStorage.__proto__, 'setItem').mockImplementation(() => {
      throw new Error('QuotaExceededError');
    });

    expect(() => {
      saveRecentSearches([{ query: 'x', extension: '' }]);
    }).not.toThrow();
  });

  test('loadRecentSearches tolerates getItem throwing (storage disabled)', () => {
    vi.spyOn(localStorage.__proto__, 'getItem').mockImplementation(() => {
      throw new Error('SecurityError: access is denied');
    });

    expect(loadRecentSearches()).toEqual([]);
  });

  test('recentSearchKey is stable and distinguishes full entries', () => {
    const a = { query: 'foo', extension: 'go', directory: '/a' };
    const same = { query: 'foo', extension: 'go', directory: '/a' };
    const differentDir = { query: 'foo', extension: 'go', directory: '/b' };
    const sameKey = { query: 'foo', extension: 'go', directory: undefined };

    expect(recentSearchKey(a)).toBe(recentSearchKey(same));
    expect(recentSearchKey(a)).not.toBe(recentSearchKey(differentDir));
    expect(recentSearchKey(a)).not.toBe(recentSearchKey(sameKey));
  });

  test('removeRecentSearch removes by exact query + extension + directory', () => {
    const searches: RecentSearch[] = [
      { query: 'foo', extension: 'go', directory: '/a' },
      { query: 'foo', extension: 'go', directory: '/b' },
      { query: 'foo', extension: 'ts' },
    ];
    saveRecentSearches(searches);

    removeRecentSearch({ query: 'foo', extension: 'go', directory: '/a' });

    const remaining = loadRecentSearches();
    expect(remaining).toHaveLength(2);
    expect(remaining.some((s) => s.directory === '/a')).toBe(false);
    expect(remaining.some((s) => s.directory === '/b')).toBe(true);
  });

  test('removeRecentSearch without directory removes every matching query', () => {
    saveRecentSearches([
      { query: 'foo', extension: 'go', directory: '/a' },
      { query: 'foo', extension: 'go', directory: '/b' },
      { query: 'bar', extension: 'go' },
    ]);

    removeRecentSearch({ query: 'foo', extension: 'go' });

    expect(loadRecentSearches()).toEqual([{ query: 'bar', extension: 'go' }]);
  });

  test('removeRecentSearch leaves unrelated entries untouched', () => {
    saveRecentSearches([{ query: 'foo', extension: 'go' }]);

    removeRecentSearch({ query: 'foo', extension: 'ts' });

    expect(loadRecentSearches()).toEqual([{ query: 'foo', extension: 'go' }]);
  });

  test('removeRecentSearch is a no-op on an empty store', () => {
    expect(() => {
      removeRecentSearch({ query: 'foo' });
    }).not.toThrow();
    expect(loadRecentSearches()).toEqual([]);
  });
});
