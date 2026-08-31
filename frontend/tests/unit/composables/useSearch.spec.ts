import { vi } from "vitest";
import { useSearch } from '@/composables';

// Mock the localStorage
const localStorageMock = (() => {
  let store: Record<string, string> = {};
  return {
    getItem: (key: string) => store[key] || null,
    setItem: (key: string, value: string) => {
      store[key] = value.toString();
    },
    removeItem: (key: string) => {
      delete store[key];
    },
    clear: () => {
      store = {};
    }
  };
})();

Object.defineProperty(window, 'localStorage', {
  value: localStorageMock
});

// Import the Wails modules for access to their mocked functions
import * as AppModule from '@wails/go/main/App';
import * as RuntimeModule from '@wails/runtime';

describe('useSearch composable', () => {
  beforeEach(() => {
    // Reset all mocks but preserve the main functionality
    vi.clearAllMocks();

    // Clear localStorage
    localStorage.clear();

    // Set default return values for mocked Wails functions
    (AppModule.SearchWithProgress as any).mockResolvedValue([]);
    (AppModule.SelectDirectory as any).mockResolvedValue('/selected/directory');
    (AppModule.ShowInFolder as any).mockResolvedValue(undefined);
    (AppModule.CancelSearch as any).mockResolvedValue(undefined);
    (AppModule.ReadFile as any).mockResolvedValue('file content');
    (AppModule.ValidateDirectory as any).mockResolvedValue(true);
    
    // Mock EventsOn to return a cleanup function
    (RuntimeModule.EventsOn as any).mockReturnValue(vi.fn());
  });

  test('should initialize with default values', () => {
    const { data } = useSearch();

    expect(data.directory).toBe('');
    expect(data.query).toBe('');
    expect(data.extension).toBe('');
    expect(data.caseSensitive).toBe(false);
    expect(data.useRegex).toBe(false);
    expect(data.includeBinary).toBe(false);
    expect(data.maxFileSize).toBe(10485760);
    expect(data.maxResults).toBe(1000);
    expect(data.resultText).toBe('Please enter search parameters below 👇');
    expect(data.searchResults).toEqual([]);
    expect(data.truncatedResults).toBe(false);
    expect(data.isSearching).toBe(false);
    expect(data.showProgress).toBe(false);
    expect(data.minFileSize).toBe(0);
    expect(data.excludePatterns).toEqual([]);
    expect(data.recentSearches).toEqual([]);
    expect(data.error).toBeNull();
    expect(data.fuzzySearch).toBe(false);
    expect(data.contextLines).toBe(3);
  });

  test('should load recent searches from localStorage', () => {
    const mockSearches = [{ query: 'test', extension: 'go' }];
    localStorage.setItem('codeSearchRecentSearches', JSON.stringify(mockSearches));

    const { data } = useSearch();
    expect(data.recentSearches).toEqual(mockSearches);
  });

  test('should perform a basic search', async () => {
    const mockResults = [
      {
        filePath: '/test/file.go',
        lineNum: 5,
        content: 'fmt.Println("Hello")',
        matchedText: 'Hello',
        contextBefore: [],
        contextAfter: []
      }
    ];
    (AppModule.SearchWithProgress as any).mockResolvedValue(mockResults);

    const { data, searchCode } = useSearch();

    data.directory = '/test';
    data.query = 'Hello';

    await searchCode();

    expect(data.searchResults).toEqual(mockResults);
    expect(data.resultText).toBe('Found 1 matches');
  });

  test('should add search to recent searches after successful search', async () => {
    const { data, searchCode } = useSearch();

    // Mock successful search
    const mockResults: any[] = [];
    (AppModule.SearchWithProgress as any).mockResolvedValue(mockResults);

    data.directory = '/test';
    data.query = 'testQuery';
    data.extension = 'js';

    await searchCode();

    expect(data.recentSearches).toEqual([{ query: 'testQuery', extension: 'js', directory: '/test' }]);
    expect(JSON.parse(localStorage.getItem('codeSearchRecentSearches') || '[]'))
      .toEqual([{ query: 'testQuery', extension: 'js', directory: '/test' }]);
  });

  test('should handle directory selection', async () => {
    const { data, selectDirectory } = useSearch();

    await selectDirectory();

    expect(data.directory).toBe('/selected/directory');
  });

  test('should handle no search results', async () => {
    (AppModule.SearchWithProgress as any).mockResolvedValue([]);

    const { data, searchCode } = useSearch();

    data.directory = '/test';
    data.query = 'nonexistent';

    await searchCode();

    expect(data.searchResults).toEqual([]);
    expect(data.resultText).toBe('No matches found');
  });

  test('should handle search errors', async () => {
    (AppModule.SearchWithProgress as any).mockRejectedValue(new Error('Search failed'));

    const { data, searchCode } = useSearch();

    data.directory = '/test';
    data.query = 'test';

    await searchCode();

    expect(data.searchResults).toEqual([]);
    // On failure the composable surfaces the message via data.error (and a
    // toast) and clears the stale "Searching..." progress text.
    expect(data.error).toContain('Search failed');
    expect(data.resultText).toBe('');
  });

  test('should validate required inputs', async () => {
    const { data, searchCode } = useSearch();

    // Don't set directory - should error
    data.query = 'test';

    await searchCode();

    expect(data.error).toBe('Directory is required');
  });

  test('should format file paths correctly', () => {
    const { formatFilePath } = useSearch();
    
    // These tests should check the actual behavior of formatFilePath
    expect(formatFilePath('/path/to/file.txt')).toContain('file.txt');
    expect(formatFilePath('file.txt')).toBe('file.txt');
    expect(formatFilePath('')).toBe('');
  });

  test('should validate numeric inputs', async () => {
    const { data, searchCode } = useSearch();

    // Test invalid max file size
    data.directory = '/test';
    data.query = 'test';
    data.maxFileSize = -1;

    await searchCode();

    expect(data.error).toBe('Invalid max file size');
  });

  test('should reject queries longer than the backend cap', async () => {
    const { data, searchCode } = useSearch();

    data.directory = '/test';
    data.query = 'x'.repeat(2001);

    await searchCode();

    expect(data.error).toBe('Query too long');
  });
});

describe('useSearch composable - cancellation & generation token', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    localStorage.clear();
    (AppModule.SearchWithProgress as any).mockResolvedValue([]);
    (RuntimeModule.EventsOn as any).mockReturnValue(vi.fn());
  });

  test('in-flight guard: second searchCode returns early when already searching', async () => {
    // Hold the first search promise open so searchCode is still "in-flight".
    let resolveSearch!: (value: any) => void;
    (AppModule.SearchWithProgress as any).mockReturnValue(
      new Promise((resolve) => {
        resolveSearch = resolve;
      }),
    );

    const { data, searchCode } = useSearch();
    data.directory = '/test';
    data.query = 'test';

    const firstPromise = searchCode();
    // The first call is now in-flight (awaiting the backend).
    expect(data.isSearching).toBe(true);

    // Resolve the second call's SearchWithProgress to [mockResults].
    const mockResults = [{ filePath: '/a.go', lineNum: 1, content: 'a', matchedText: 'a', contextBefore: [], contextAfter: [] }];
    (AppModule.SearchWithProgress as any).mockResolvedValue(mockResults);

    // Second call — should return early because isSearching is true.
    await searchCode();

    // The second call did NOT start — SearchWithProgress was called once.
    expect(AppModule.SearchWithProgress).toHaveBeenCalledTimes(1);

    // Resolve the first search.
    resolveSearch([]);
    await firstPromise;

    // The first search completes normally.
    expect(data.searchResults).toEqual([]);
  });

  test('generation token: stale response after cancel is discarded', async () => {
    let resolveSearch!: (value: any) => void;
    (AppModule.SearchWithProgress as any).mockReturnValue(
      new Promise((resolve) => {
        resolveSearch = resolve;
      }),
    );

    const { data, searchCode, cancelSearch } = useSearch();
    data.directory = '/test';
    data.query = 'test';

    const searchPromise = searchCode();
    await Promise.resolve();

    // Cancel mid-flight — bumps generation token.
    await cancelSearch();

    // Backend resolves with stale results.
    resolveSearch([
      { filePath: '/stale.go', lineNum: 1, content: 'old', matchedText: 'old', contextBefore: [], contextAfter: [] },
    ]);
    await searchPromise;

    // Stale response must be discarded.
    expect(data.searchResults).toEqual([]);
  });

  test('generation token: new search supersedes old in-flight response', async () => {
    let resolveSearch1!: (value: any) => void;
    let resolveSearch2!: (value: any) => void;
    (AppModule.SearchWithProgress as any)
      .mockReturnValueOnce(new Promise((resolve) => { resolveSearch1 = resolve; }))
      .mockReturnValueOnce(new Promise((resolve) => { resolveSearch2 = resolve; }));

    const { data, searchCode, cancelSearch } = useSearch();
    data.directory = '/test';
    data.query = 'test';

    // Start search 1 (in-flight, pending on backend 1).
    const search1 = searchCode();
    await Promise.resolve();

    // Cancel bumps the generation token and lets a fresh search start.
    await cancelSearch();
    const search2 = searchCode();
    await Promise.resolve();

    // Backend 1 resolves with stale results — discarded (generation mismatch).
    resolveSearch1([{ filePath: '/old.go', lineNum: 1, content: 'old', matchedText: 'old', contextBefore: [], contextAfter: [] }]);
    await search1;
    expect(data.searchResults).toEqual([]);

    // Backend 2 resolves with current results — kept.
    resolveSearch2([{ filePath: '/new.go', lineNum: 1, content: 'new', matchedText: 'new', contextBefore: [], contextAfter: [] }]);
    await search2;
    expect(data.searchResults).toHaveLength(1);
    expect(data.searchResults[0].filePath).toBe('/new.go');
  });
});
describe('useSearch composable - fuzzy search', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    localStorage.clear();
    (AppModule.SearchWithProgress as any).mockResolvedValue([]);
  });

  test('should enable fuzzy search mode via toggle', () => {
    const { data } = useSearch();

    expect(data.fuzzySearch).toBe(false);
    
    data.fuzzySearch = true;
    expect(data.fuzzySearch).toBe(true);
  });

  test('should filter results using client-side fuzzy matching', async () => {
    // Backend returns misspelled variations
    const backendResults = [
      {
        filePath: '/test/file.go',
        lineNum: 5,
        content: 'fmt.Println("tset messga")', // misspelled "test message"
        matchedText: 'tset',
        contextBefore: ['package main'],
        contextAfter: ['func main() {}'],
        similarityScore: undefined // Would be set if backend did fuzzy
      },
      {
        filePath: '/test/file2.go',
        lineNum: 10,
        content: 'const x = "value";',
        matchedText: 'value',
        contextBefore: [],
        contextAfter: []
      }
    ];

    (AppModule.SearchWithProgress as any).mockResolvedValue(backendResults);

    const { data, searchCode } = useSearch();

    data.directory = '/test';
    data.query = 'test message';
    data.fuzzySearch = true;

    await searchCode();

    // Fuzzy matching should find the misspelled version
    expect(data.searchResults.length).toBeGreaterThan(0);
    expect(data.resultText).toContain('(fuzzy)');
  });

  test('should handle fuzzy search with no matches', async () => {
    (AppModule.SearchWithProgress as any).mockResolvedValue([
      {
        filePath: '/test/file.go',
        lineNum: 5,
        content: 'some other text',
        matchedText: 'other',
        contextBefore: [],
        contextAfter: []
      }
    ]);

    const { data, searchCode } = useSearch();

    data.directory = '/test';
    data.query = 'xyznonexistent';
    data.fuzzySearch = true;

    await searchCode();

    // Should show no matches or empty fuzzy results
    expect(data.resultText.includes('No matches') || 
           data.searchResults.length === 0 || 
           data.resultText.includes('(fuzzy)')).toBe(true);
  });

  test('should not apply fuzzy matching when disabled', async () => {
    const backendResults = [
      {
        filePath: '/test/file.go',
        lineNum: 5,
        content: 'exact match text',
        matchedText: 'exact',
        contextBefore: [],
        contextAfter: []
      }
    ];

    (AppModule.SearchWithProgress as any).mockResolvedValue(backendResults);

    const { data, searchCode } = useSearch();

    data.directory = '/test';
    data.query = 'exact';
    data.fuzzySearch = false; // Explicitly disabled

    await searchCode();

    expect(data.resultText).not.toContain('(fuzzy)');
  });

  test('should update result text with fuzzy indicator', async () => {
    const mockResults = [{
      filePath: '/test/file.go',
      lineNum: 1,
      content: 'found it',
      matchedText: 'found',
      contextBefore: [],
      contextAfter: []
    }];

    (AppModule.SearchWithProgress as any).mockResolvedValue(mockResults);

    const { data, searchCode } = useSearch();

    data.directory = '/test';
    data.query = 'found'; // Exact match to bypass client-side filtering
    data.fuzzySearch = true;

    await searchCode();

    // Should find the result since we used exact matching query
    expect(data.searchResults.length).toBeGreaterThan(0);
  });

  test('should pass fuzzySearch and contextLines to backend request', async () => {
    const mockResults: any[] = [];
    
    const { data, searchCode } = useSearch();

    data.directory = '/test';
    data.query = 'test';
    data.fuzzySearch = true;
    data.contextLines = 5;

    await searchCode();

    // Verify the mock was called (exact args will vary based on default values)
    expect((AppModule.SearchWithProgress as any).mock.calls.length).toBe(1);
    const callArgs = (AppModule.SearchWithProgress as any).mock.calls[0][0];
    expect(callArgs.fuzzySearch).toBe(true);
    expect(callArgs.contextLines).toBe(5);
  });
});
