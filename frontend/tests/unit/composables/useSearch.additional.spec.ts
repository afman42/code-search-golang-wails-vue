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

describe('useSearch composable - Additional Tests', () => {
  beforeEach(() => {
    localStorage.clear();
  });

  test('should handle exclude patterns as array', () => {
    const { data } = useSearch();
    
    expect(Array.isArray(data.excludePatterns)).toBe(true);
    expect(data.excludePatterns).toEqual([]);
    
    // Test setting exclude patterns
    data.excludePatterns = ['node_modules', '.git'];
    expect(data.excludePatterns).toEqual(['node_modules', '.git']);
  });

  test('should handle recent searches with localStorage', () => {
    const mockSearches = [{ query: 'test', extension: 'go' }];
    localStorage.setItem('codeSearchRecentSearches', JSON.stringify(mockSearches));
    
    const { data } = useSearch();
    expect(data.recentSearches).toEqual(mockSearches);
  });

  test('should handle localStorage errors gracefully', () => {
    // Mock localStorage to throw an error
    const originalGetItem = localStorage.getItem;
    const originalSetItem = localStorage.setItem;
    
    Object.defineProperty(window.localStorage, 'getItem', {
      value: jest.fn(() => {
        throw new Error('Storage error');
      }),
      writable: true,
    });
    
    Object.defineProperty(window.localStorage, 'setItem', {
      value: jest.fn(() => {
        throw new Error('Storage error');
      }),
      writable: true,
    });

    const { data } = useSearch();
    expect(data.recentSearches).toEqual([]);
    
    // Restore original methods
    Object.defineProperty(window.localStorage, 'getItem', {
      value: originalGetItem,
      writable: true,
    });
    
    Object.defineProperty(window.localStorage, 'setItem', {
      value: originalSetItem,
      writable: true,
    });
  });

  test('should sanitize strings for XSS prevention', () => {
    const { sanitizeString } = useSearch();
    
    expect(sanitizeString('<script>alert("xss")</script>')).toBe('&lt;script&gt;alert("xss")&lt;/script&gt;');
    expect(sanitizeString('normal text')).toBe('normal text');
    expect(sanitizeString('')).toBe('');
    expect(sanitizeString(null as any)).toBe('');
  });

  test('should validate search inputs correctly', async () => {
    const { data, searchCode } = useSearch();
    
    // Test without directory
    await searchCode();
    expect(data.error).toBe('Directory is required');
    
    // Test without query
    data.directory = '/test';
    await searchCode();
    expect(data.error).toBe('Query is required');
    
    // Test with valid inputs (but no actual search since we're not mocking)
    data.query = 'test';
    data.maxFileSize = -1;
    await searchCode();
    expect(data.error).toBe('Invalid max file size');
    
    // Test with valid min file size
    data.maxFileSize = 1000;
    data.minFileSize = -1;
    await searchCode();
    expect(data.error).toBe('Invalid min file size');
    
    // Test with valid max results
    data.minFileSize = 0;
    data.maxResults = 0;
    await searchCode();
    expect(data.error).toBe('Invalid max results');
  });

  test('should format file paths correctly', () => {
    const { formatFilePath } = useSearch();
    
    expect(formatFilePath('/path/to/file.txt')).toBe('to/file.txt');
    expect(formatFilePath('file.txt')).toBe('file.txt');
    expect(formatFilePath('')).toBe('');
    expect(formatFilePath(null as any)).toBe('');
    expect(formatFilePath('/')).toBe('/'); // Special case - root directory
    // Just verify it doesn't crash with trailing slash
    expect(typeof formatFilePath('path/to/')).toBe('string');
  });

  test('should highlight matches in text', () => {
    const { highlightMatch } = useSearch();
    
    expect(highlightMatch('Hello world', 'Hello')).toBe('<mark class="highlight">Hello</mark> world');
    expect(highlightMatch('', 'test')).toBe('');
    expect(highlightMatch('test', '')).toBe('test');
    expect(highlightMatch('Special chars [', '[')).toBe('Special chars <mark class="highlight">[</mark>');
  });

  test('should highlight matches with regex enabled', () => {
    const { data, highlightMatch } = useSearch();
    
    data.useRegex = true;
    expect(highlightMatch('Hello123 world', '\\d+')).toBe('Hello<mark class="highlight">123</mark> world');
  });

  test('should handle invalid regex patterns gracefully', () => {
    const { data, highlightMatch } = useSearch();
    
    data.useRegex = true;
    // This should not throw an error even with invalid regex
    // It should return the original text when regex fails
    expect(highlightMatch('test', '[')).toBe('test');
  });
});