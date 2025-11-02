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

  test('should format file paths correctly', () => {
    const { formatFilePath } = useSearch();
    
    expect(formatFilePath('/path/to/file.txt')).toBe('to/file.txt');
    expect(formatFilePath('file.txt')).toBe('file.txt');
    expect(formatFilePath('')).toBe('');
    expect(formatFilePath(null as any)).toBe('');
  });

  test('should sanitize strings for XSS prevention', () => {
    const { sanitizeString } = useSearch();
    
    expect(sanitizeString('<script>alert("xss")</script>')).toBe('&lt;script&gt;alert("xss")&lt;/script&gt;');
    expect(sanitizeString('normal text')).toBe('normal text');
    expect(sanitizeString('')).toBe('');
  });

  test('should highlight matches in text', () => {
    const { highlightMatch } = useSearch();
    
    expect(highlightMatch('Hello world', 'Hello')).toBe('<mark class="highlight">Hello</mark> world');
    expect(highlightMatch('', 'test')).toBe('');
    expect(highlightMatch('test', '')).toBe('test');
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
    
    // Test with valid inputs
    data.query = 'test';
    data.maxFileSize = -1;
    await searchCode();
    expect(data.error).toBe('Invalid max file size');
  });
});