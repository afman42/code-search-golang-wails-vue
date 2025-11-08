import { useSearch } from '../../../src/composables/useSearch';

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
import * as AppModule from '../../../wailsjs/go/main/App';
import * as RuntimeModule from '../../../wailsjs/runtime';

describe('useSearch composable', () => {
  beforeEach(() => {
    // Reset all mocks but preserve the main functionality
    jest.clearAllMocks();

    // Clear localStorage
    localStorage.clear();

    // Set default return values for mocked Wails functions
    (AppModule.SearchWithProgress as jest.MockedFunction<any>).mockResolvedValue([]);
    (AppModule.SelectDirectory as jest.MockedFunction<any>).mockResolvedValue('/selected/directory');
    (AppModule.ShowInFolder as jest.MockedFunction<any>).mockResolvedValue(undefined);
    (AppModule.CancelSearch as jest.MockedFunction<any>).mockResolvedValue(undefined);
    (AppModule.ReadFile as jest.MockedFunction<any>).mockResolvedValue('file content');
    (AppModule.ValidateDirectory as jest.MockedFunction<any>).mockResolvedValue(true);
    
    // Mock EventsOn to return a cleanup function
    (RuntimeModule.EventsOn as jest.MockedFunction<any>).mockReturnValue(jest.fn());
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
    expect(data.searchSubdirs).toBe(true);
    expect(data.resultText).toBe('Please enter search parameters below 👇');
    expect(data.searchResults).toEqual([]);
    expect(data.truncatedResults).toBe(false);
    expect(data.isSearching).toBe(false);
    expect(data.showProgress).toBe(false);
    expect(data.minFileSize).toBe(0);
    expect(data.excludePatterns).toEqual([]);
    expect(data.recentSearches).toEqual([]);
    expect(data.error).toBeNull();
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
    (AppModule.SearchWithProgress as jest.MockedFunction<any>).mockResolvedValue(mockResults);

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
    (AppModule.SearchWithProgress as jest.MockedFunction<any>).mockResolvedValue(mockResults);

    data.directory = '/test';
    data.query = 'testQuery';
    data.extension = 'js';

    await searchCode();

    expect(data.recentSearches).toEqual([{ query: 'testQuery', extension: 'js' }]);
    expect(JSON.parse(localStorage.getItem('codeSearchRecentSearches') || '[]'))
      .toEqual([{ query: 'testQuery', extension: 'js' }]);
  });

  test('should handle directory selection', async () => {
    const { data, selectDirectory } = useSearch();

    await selectDirectory();

    expect(data.directory).toBe('/selected/directory');
  });

  test('should handle no search results', async () => {
    (AppModule.SearchWithProgress as jest.MockedFunction<any>).mockResolvedValue([]);

    const { data, searchCode } = useSearch();

    data.directory = '/test';
    data.query = 'nonexistent';

    await searchCode();

    expect(data.searchResults).toEqual([]);
    expect(data.resultText).toBe('No matches found');
  });

  test('should handle search errors', async () => {
    (AppModule.SearchWithProgress as jest.MockedFunction<any>).mockRejectedValue(new Error('Search failed'));

    const { data, searchCode } = useSearch();

    data.directory = '/test';
    data.query = 'test';

    await searchCode();

    expect(data.searchResults).toEqual([]);
    expect(data.resultText).toContain('Error: Search failed');
    expect(data.error).toContain('Search failed');
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
});