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

describe('useSearch composable', () => {
  beforeEach(() => {
    // Clear mocks that are set up globally in setup.ts
    jest.clearAllMocks();
    
    // Clear localStorage
    localStorage.clear();
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

  test('should handle localStorage errors gracefully', () => {
    // Mock localStorage to throw an error
    const originalGetItem = localStorage.getItem;
    Object.defineProperty(window.localStorage, 'getItem', {
      value: jest.fn(() => {
        throw new Error('Storage error');
      }),
      writable: true,
    });

    const { data } = useSearch();
    expect(data.recentSearches).toEqual([]);
    
    // Restore original method
    Object.defineProperty(window.localStorage, 'getItem', {
      value: originalGetItem,
      writable: true,
    });
  });

  test('should save recent searches to localStorage', async () => {
    const { data, searchCode } = useSearch();
    
    // Mock successful search
    const mockResults = [];
    (require('../../../wailsjs/go/main/App').SearchWithProgress as jest.MockedFunction<any>).mockResolvedValue(mockResults);
    
    data.query = 'test';
    data.directory = '/test';
    data.extension = 'ts';
    
    await searchCode();
    
    expect(localStorage.getItem('codeSearchRecentSearches')).toContain('test');
  });

  test('should format file paths correctly', () => {
    const { formatFilePath } = useSearch();
    
    expect(formatFilePath('/path/to/file.txt')).toBe('to/file.txt');
    expect(formatFilePath('file.txt')).toBe('file.txt');
    expect(formatFilePath('')).toBe('');
    expect(formatFilePath(null as any)).toBe('');
    expect(formatFilePath('/')).toBe('/');
    expect(formatFilePath('path/to/')).toBe('to');
  });

  test('should format file paths with Windows-style separators', () => {
    const { formatFilePath } = useSearch();
    
    expect(formatFilePath('C:\\path\\to\\file.txt')).toBe('to/file.txt');
    expect(formatFilePath('path\\to\\file.txt')).toBe('to/file.txt');
  });

  test('should highlight matches in text (case insensitive)', () => {
    const { highlightMatch } = useSearch();
    
    expect(highlightMatch('Hello world', 'hello')).toBe('<mark class="highlight">Hello</mark> world');
    expect(highlightMatch('Hello world', 'World')).toBe('Hello <mark class="highlight">world</mark>');
    expect(highlightMatch('', 'test')).toBe('');
    expect(highlightMatch('test', '')).toBe('test');
    expect(highlightMatch('Hello world', 'nonexistent')).toBe('Hello world');
  });

  test('should highlight matches in text (case sensitive)', () => {
    const { data, highlightMatch } = useSearch();
    
    data.caseSensitive = true;
    
    expect(highlightMatch('Hello world', 'hello')).toBe('Hello world'); // No match due to case sensitivity
    expect(highlightMatch('Hello world', 'Hello')).toBe('<mark class="highlight">Hello</mark> world');
  });

  test('should highlight regex patterns', () => {
    const { data, highlightMatch } = useSearch();
    
    data.useRegex = true;
    
    expect(highlightMatch('Hello123 world', '\\d+')).toBe('Hello<mark class="highlight">123</mark> world');
  });

  test('should sanitize strings for XSS prevention', () => {
    const { sanitizeString } = useSearch();
    
    expect(sanitizeString('<script>alert("xss")</script>')).toBe('&lt;script&gt;alert("xss")&lt;/script&gt;');
    expect(sanitizeString('normal text')).toBe('normal text');
    expect(sanitizeString('')).toBe('');
    expect(sanitizeString('<div>Hello</div>')).toBe('&lt;div&gt;Hello&lt;/div&gt;');
    expect(sanitizeString(null as any)).toBe('');
  });

  test('should validate search inputs', async () => {
    const { data, searchCode } = useSearch();
    
    // Test without directory
    await searchCode();
    expect(data.error).toBe('Directory is required');
    
    // Test without query
    data.directory = '/test';
    await searchCode();
    expect(data.error).toBe('Query is required');
    
    // Test with invalid max file size
    data.query = 'test';
    data.maxFileSize = -1;
    await searchCode();
    expect(data.error).toBe('Invalid max file size');
    
    // Test with invalid min file size
    data.maxFileSize = 1000;
    data.minFileSize = -1;
    await searchCode();
    expect(data.error).toBe('Invalid min file size');
    
    // Test with invalid max results
    data.minFileSize = 0;
    data.maxResults = 0;
    await searchCode();
    expect(data.error).toBe('Invalid max results');
  });

  test('should perform successful search with results', async () => {
    const mockResults = [
      { 
        filePath: '/test/file1.go', 
        lineNum: 5, 
        content: 'fmt.Println("hello")',
        matchedText: 'hello',
        contextBefore: [],
        contextAfter: []
      }
    ];
    (require('../../../wailsjs/go/main/App').SearchWithProgress as jest.MockedFunction<any>).mockResolvedValue(mockResults);
    
    const { data, searchCode } = useSearch();
    
    data.directory = '/test';
    data.query = 'hello';
    
    await searchCode();
    
    expect(data.isSearching).toBe(false);
    expect(data.showProgress).toBe(false);
    expect(data.searchResults).toEqual(mockResults);
    expect(data.resultText).toBe('Found 1 matches');
  });

  test('should handle empty search results', async () => {
    (require('../../../wailsjs/go/main/App').SearchWithProgress as jest.MockedFunction<any>).mockResolvedValue([]);
    
    const { data, searchCode } = useSearch();
    
    data.directory = '/test';
    data.query = 'nonexistent';
    
    await searchCode();
    
    expect(data.searchResults).toEqual([]);
    expect(data.resultText).toBe('No matches found');
  });

  test('should handle backend errors', async () => {
    const error = new Error('Search failed');
    (require('../../../wailsjs/go/main/App').SearchWithProgress as jest.MockedFunction<any>).mockRejectedValue(error);
    
    const { data, searchCode } = useSearch();
    
    data.directory = '/test';
    data.query = 'test';
    
    await searchCode();
    
    expect(data.searchResults).toEqual([]);
    expect(data.resultText).toBe('Error: Search failed');
    expect(data.error).toBe('Search failed');
  });

  test('should handle null results from backend gracefully', async () => {
    (require('../../../wailsjs/go/main/App').SearchWithProgress as jest.MockedFunction<any>).mockResolvedValue(null);
    
    const { data, searchCode } = useSearch();
    
    data.directory = '/test';
    data.query = 'test';
    
    await searchCode();
    
    expect(data.searchResults).toEqual([]);
    expect(data.resultText).toBe('No matches found');
  });

  test('should handle undefined results from backend gracefully', async () => {
    (require('../../../wailsjs/go/main/App').SearchWithProgress as jest.MockedFunction<any>).mockResolvedValue(undefined);
    
    const { data, searchCode } = useSearch();
    
    data.directory = '/test';
    data.query = 'test';
    
    await searchCode();
    
    expect(data.searchResults).toEqual([]);
    expect(data.resultText).toBe('No matches found');
  });

  test('should add search to recent searches after successful search', async () => {
    (require('../../../wailsjs/go/main/App').SearchWithProgress as jest.MockedFunction<any>).mockResolvedValue([]);
    
    const { data, searchCode } = useSearch();
    
    data.directory = '/test';
    data.query = 'testQuery';
    data.extension = 'js';
    
    await searchCode();
    
    expect(data.recentSearches).toEqual([{ query: 'testQuery', extension: 'js' }]);
    expect(JSON.parse(localStorage.getItem('codeSearchRecentSearches') || '[]'))
      .toEqual([{ query: 'testQuery', extension: 'js' }]);
  });

  test('should limit recent searches to 5 items', async () => {
    (require('../../../wailsjs/go/main/App').SearchWithProgress as jest.MockedFunction<any>).mockResolvedValue([]);
    
    const { data, searchCode } = useSearch();
    
    data.directory = '/test';
    
    // Add 6 searches to test the limit
    for (let i = 1; i <= 6; i++) {
      data.query = `query${i}`;
      await searchCode();
    }
    
    expect(data.recentSearches).toHaveLength(5);
    // The most recent search should be first
    expect(data.recentSearches[0]).toEqual({ query: 'query6', extension: '' });
    // The oldest should be removed
    expect(data.recentSearches).not.toContainEqual({ query: 'query1', extension: '' });
  });

  test('should not duplicate recent searches', async () => {
    (require('../../../wailsjs/go/main/App').SearchWithProgress as jest.MockedFunction<any>).mockResolvedValue([]);
    
    const { data, searchCode } = useSearch();
    
    data.directory = '/test';
    data.query = 'duplicate';
    data.extension = 'ts';
    
    // First search
    await searchCode();
    expect(data.recentSearches).toHaveLength(1);
    
    // Second search with same query and extension
    await searchCode();
    expect(data.recentSearches).toHaveLength(1);
    
    // Search with same query but different extension
    data.extension = 'js';
    await searchCode();
    expect(data.recentSearches).toHaveLength(2);
  });

  test('should handle directory selection successfully', async () => {
    (require('../../../wailsjs/go/main/App').SelectDirectory as jest.MockedFunction<any>).mockResolvedValue('/selected/directory');
    
    const { data, selectDirectory } = useSearch();
    
    await selectDirectory();
    
    expect(data.directory).toBe('/selected/directory');
    expect(data.error).toBeNull();
  });

  test('should handle directory selection cancellation', async () => {
    (require('../../../wailsjs/go/main/App').SelectDirectory as jest.MockedFunction<any>).mockResolvedValue('');
    
    const { data, selectDirectory } = useSearch();
    
    await selectDirectory();
    
    expect(data.directory).toBe('');
    expect(data.error).toBeNull();
  });

  test('should handle directory selection errors', async () => {
    const error = new Error('Directory picker not implemented');
    (require('../../../wailsjs/go/main/App').SelectDirectory as jest.MockedFunction<any>).mockRejectedValue(error);
    
    const { data, selectDirectory } = useSearch();
    
    await selectDirectory();
    
    expect(data.resultText).toContain('Directory selection failed');
    expect(data.error).toContain('Directory selection failed');
  });

  test('should handle directory selection with "not implemented" error', async () => {
    const error = new Error('not implemented');
    (require('../../../wailsjs/go/main/App').SelectDirectory as jest.MockedFunction<any>).mockRejectedValue(error);
    
    const { data, selectDirectory } = useSearch();
    
    await selectDirectory();
    
    expect(data.resultText).toContain('not available on this platform');
  });

  test('should handle directory selection with "no suitable directory picker" error', async () => {
    const error = new Error('no suitable directory picker');
    (require('../../../wailsjs/go/main/App').SelectDirectory as jest.MockedFunction<any>).mockRejectedValue(error);
    
    const { data, selectDirectory } = useSearch();
    
    await selectDirectory();
    
    expect(data.resultText).toContain('No directory picker found');
  });

  test('should validate regex patterns', async () => {
    const { data, searchCode } = useSearch();
    
    data.directory = '/test';
    data.query = '['; // Invalid regex
    data.useRegex = true;
    
    await searchCode();
    
    expect(data.resultText).toContain('Invalid regex pattern');
    expect(data.error).toContain('Invalid regex');
    expect(data.isSearching).toBe(false);
    expect(data.showProgress).toBe(false);
  });

  test('should handle progress updates during search', async () => {
    (require('../../../wailsjs/go/main/App').SearchWithProgress as jest.MockedFunction<any>).mockResolvedValue([]);
    
    const { data, searchCode } = useSearch();
    
    data.directory = '/test';
    data.query = 'test';
    
    await searchCode();
    
    // The final state should be updated after search completes
    expect(data.isSearching).toBe(false);
  });

  test('should handle progress updates with completed status', async () => {
    (require('../../../wailsjs/go/main/App').SearchWithProgress as jest.MockedFunction<any>).mockResolvedValue([]);
    
    const { data, searchCode } = useSearch();
    
    data.directory = '/test';
    data.query = 'test';
    
    await searchCode();
    
    expect(data.resultText).toContain('Search completed!');
  });

  test('should handle clipboard copy in secure context', async () => {
    const originalClipboard = navigator.clipboard;
    const mockWriteText = jest.fn().mockResolvedValue(undefined);
    Object.assign(navigator, {
      clipboard: {
        writeText: mockWriteText
      }
    });
    
    const { data, copyToClipboard } = useSearch();
    
    await copyToClipboard('test content');
    
    expect(mockWriteText).toHaveBeenCalledWith('test content');
    expect(data.resultText).not.toContain('Failed to copy');
    
    // Restore original clipboard
    Object.assign(navigator, {
      clipboard: originalClipboard
    });
  });

  test('should handle clipboard copy failures', async () => {
    const mockWriteText = jest.fn().mockRejectedValue(new Error('Copy failed'));
    Object.assign(navigator, {
      clipboard: {
        writeText: mockWriteText
      }
    });
    
    const { data, copyToClipboard } = useSearch();
    
    await copyToClipboard('test content');
    
    expect(data.resultText).toContain('Failed to copy text to clipboard');
    expect(data.error).toBe('Clipboard error');
  });

  test('should handle clipboard copy in insecure context (fallback)', async () => {
    // Temporarily remove the clipboard API
    const originalClipboard = navigator.clipboard;
    Object.assign(navigator, {
      clipboard: undefined
    });
    
    const { copyToClipboard } = useSearch();
    
    await copyToClipboard('test content');
    
    // Restore clipboard
    Object.assign(navigator, {
      clipboard: originalClipboard
    });
    
    // This test checks that no error is thrown in fallback scenario
  });

  test('should validate numeric inputs correctly', async () => {
    const { data, searchCode } = useSearch();
    
    data.directory = '/test';
    data.query = 'test';
    
    // Test invalid maxFileSize
    data.maxFileSize = -1;
    await searchCode();
    expect(data.error).toBe('Invalid max file size');
    
    // Test invalid minFileSize
    data.maxFileSize = 1000;
    data.minFileSize = -5;
    await searchCode();
    expect(data.error).toBe('Invalid min file size');
    
    // Test invalid maxResults
    data.minFileSize = 0;
    data.maxResults = 0;
    await searchCode();
    expect(data.error).toBe('Invalid max results');
    
    // Test valid inputs should not error
    data.maxResults = 500;
    mockSearchWithProgress.mockResolvedValue([]);
    await searchCode();
    // No error should be set
    expect(data.error).toBeNull();
  });

  test('should handle file location opening successfully', async () => {
    (require('../../../wailsjs/go/main/App').ShowInFolder as jest.MockedFunction<any>).mockResolvedValue(undefined);
    
    const { data, openFileLocation } = useSearch();
    
    await openFileLocation('/path/to/file.txt');
    
    expect((require('../../../wailsjs/go/main/App').ShowInFolder as jest.MockedFunction<any>)).toHaveBeenCalledWith('/path/to/file.txt');
    expect(data.resultText).not.toContain('Could not open file location');
  });

  test('should handle file location opening errors', async () => {
    const error = new Error('Could not open folder');
    (require('../../../wailsjs/go/main/App').ShowInFolder as jest.MockedFunction<any>).mockRejectedValue(error);
    
    const { data, openFileLocation } = useSearch();
    
    await openFileLocation('/path/to/file.txt');
    
    expect(data.resultText).toContain('Could not open file location: Could not open folder');
    expect(data.error).toContain('Open folder error');
  });

  test('should handle invalid file path in openFileLocation', async () => {
    const { data, openFileLocation } = useSearch();
    
    await openFileLocation('');
    
    expect(data.resultText).toBe('Invalid file path');
    expect(mockShowInFolder).not.toHaveBeenCalled();
  });

  test('should handle copyToClipboard with empty or invalid text', async () => {
    const { copyToClipboard } = useSearch();
    
    // Test with empty string
    await copyToClipboard('');
    // Should not throw an error
    
    // Test with null
    await copyToClipboard(null as any);
    // Should not throw an error
    
    // Test with undefined
    await copyToClipboard(undefined as any);
    // Should not throw an error
  });

  test('should format complex file paths correctly', () => {
    const { formatFilePath } = useSearch();
    
    expect(formatFilePath('/home/user/projects/my-app/src/main.go')).toBe('src/main.go');
    expect(formatFilePath('/home/user/projects/my-app/src/components/CodeSearch.vue')).toBe('CodeSearch.vue');
    expect(formatFilePath('C:/Users/Name/Documents/file.txt')).toBe('Documents/file.txt');
    expect(formatFilePath('relative/path/to/some/file.js')).toBe('file.js');
  });

  test('should process exclude patterns correctly in search', async () => {
    (require('../../../wailsjs/go/main/App').SearchWithProgress as jest.MockedFunction<any>).mockResolvedValue([]);
    
    const { data, searchCode } = useSearch();
    
    data.directory = '/test';
    data.query = 'test';
    data.excludePatterns = ['node_modules', '.git', '*.log'];
    
    await searchCode();
    
    // Verify that the search request was made with the correct exclude patterns
    expect((require('../../../wailsjs/go/main/App').SearchWithProgress as jest.MockedFunction<any>)).toHaveBeenCalledWith(
      expect.objectContaining({
        excludePatterns: ['node_modules', '.git', '*.log']
      })
    );
  });

  test('should filter empty exclude patterns', async () => {
    (require('../../../wailsjs/go/main/App').SearchWithProgress as jest.MockedFunction<any>).mockResolvedValue([]);
    
    const { data, searchCode } = useSearch();
    
    data.directory = '/test';
    data.query = 'test';
    data.excludePatterns = ['node_modules', '', '.git', '   ', '*.log'];
    
    await searchCode();
    
    // Verify that empty patterns are filtered out
    expect((require('../../../wailsjs/go/main/App').SearchWithProgress as jest.MockedFunction<any>)).toHaveBeenCalledWith(
      expect.objectContaining({
        excludePatterns: ['node_modules', '.git', '*.log']
      })
    );
  });
});