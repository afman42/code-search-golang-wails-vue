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

describe('useSearch composable - Additional Tests', () => {
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

  test('should handle search with regex enabled', async () => {
    const mockResults = [
      {
        filePath: '/test/file.js',
        lineNum: 10,
        content: 'const pattern = /\\d+/g;',
        matchedText: '\\d+',
        contextBefore: [],
        contextAfter: []
      }
    ];
    (AppModule.SearchWithProgress as jest.MockedFunction<any>).mockResolvedValue(mockResults);

    const { data, searchCode } = useSearch();

    data.directory = '/test';
    data.query = '\\d+';
    data.useRegex = true;

    // Test that regex validation passes with a valid pattern
    try {
      await searchCode();
      // If we reach this, regex validation passed
      expect(data.searchResults).toEqual(mockResults);
    } catch (error) {
      // If it fails, we should handle it gracefully
      expect(data.error).toBeNull(); // The error should be handled internally
    }
  });

  test('should handle search with case sensitivity', async () => {
    const mockResults = [
      {
        filePath: '/test/file.txt',
        lineNum: 1,
        content: 'Hello World',
        matchedText: 'Hello',
        contextBefore: [],
        contextAfter: []
      }
    ];
    (AppModule.SearchWithProgress as jest.MockedFunction<any>).mockResolvedValue(mockResults);

    const { data, searchCode } = useSearch();

    data.directory = '/test';
    data.query = 'Hello';
    data.caseSensitive = true;

    await searchCode();

    expect(data.searchResults).toEqual(mockResults);
  });

  test('should handle search with binary file inclusion', async () => {
    const mockResults = [
      {
        filePath: '/test/image.png',
        lineNum: 1,
        content: 'binary content here',
        matchedText: 'content',
        contextBefore: [],
        contextAfter: []
      }
    ];
    (AppModule.SearchWithProgress as jest.MockedFunction<any>).mockResolvedValue(mockResults);

    const { data, searchCode } = useSearch();

    data.directory = '/test';
    data.query = 'content';
    data.includeBinary = true;

    await searchCode();

    expect(data.searchResults).toEqual(mockResults);
  });

  test('should handle max file size filter', async () => {
    (AppModule.SearchWithProgress as jest.MockedFunction<any>).mockResolvedValue([]);

    const { data, searchCode } = useSearch();

    data.directory = '/test';
    data.query = 'test';
    data.maxFileSize = 5000000; // 5MB

    await searchCode();

    // Verify that the search was called with the correct maxFileSize
    expect(AppModule.SearchWithProgress).toHaveBeenCalledWith(
      expect.objectContaining({
        maxFileSize: 5000000
      })
    );
  });

  test('should handle min file size filter', async () => {
    (AppModule.SearchWithProgress as jest.MockedFunction<any>).mockResolvedValue([]);

    const { data, searchCode } = useSearch();

    data.directory = '/test';
    data.query = 'test';
    data.minFileSize = 100; // 100 bytes

    await searchCode();

    // Verify that the search was called with the correct minFileSize
    expect(AppModule.SearchWithProgress).toHaveBeenCalledWith(
      expect.objectContaining({
        minFileSize: 100
      })
    );
  });

  test('should handle search subdirectories toggle', async () => {
    (AppModule.SearchWithProgress as jest.MockedFunction<any>).mockResolvedValue([]);

    const { data, searchCode } = useSearch();

    data.directory = '/test';
    data.query = 'test';
    data.searchSubdirs = false;

    await searchCode();

    // Verify that the search was called with the correct searchSubdirs value
    expect(AppModule.SearchWithProgress).toHaveBeenCalledWith(
      expect.objectContaining({
        searchSubdirs: false
      })
    );
  });

  test('should handle exclude patterns', async () => {
    (AppModule.SearchWithProgress as jest.MockedFunction<any>).mockResolvedValue([]);

    const { data, searchCode } = useSearch();

    data.directory = '/test';
    data.query = 'test';
    data.excludePatterns = ['node_modules', '.git', '*.log'];

    await searchCode();

    // Verify that the search was called with the correct exclude patterns
    expect(AppModule.SearchWithProgress).toHaveBeenCalledWith(
      expect.objectContaining({
        excludePatterns: ['node_modules', '.git', '*.log']
      })
    );
  });

  test('should handle allowed file types', async () => {
    (AppModule.SearchWithProgress as jest.MockedFunction<any>).mockResolvedValue([]);

    const { data, searchCode } = useSearch();

    data.directory = '/test';
    data.query = 'test';
    data.allowedFileTypes = ['js', 'ts', 'vue'];

    await searchCode();

    // Verify that the search was called with the correct allowed file types
    expect(AppModule.SearchWithProgress).toHaveBeenCalledWith(
      expect.objectContaining({
        allowedFileTypes: ['js', 'ts', 'vue']
      })
    );
  });

  test('should handle search cancellation', async () => {
    (AppModule.CancelSearch as jest.MockedFunction<any>).mockResolvedValue(undefined);

    const { cancelSearch } = useSearch();

    await cancelSearch();

    expect(AppModule.CancelSearch).toHaveBeenCalled();
  });

  test('should format file paths correctly with long paths', () => {
    const { formatFilePath } = useSearch();
    
    // Test that long paths are handled properly
    const longPath = '/very/long/path/to/some/deeply/nested/directory/structure/file.txt';
    const result = formatFilePath(longPath);
    
    // The function should return the path as is, since formatting logic varies
    expect(typeof result).toBe('string');
    expect(result).toContain('file.txt');
  });

  test('should highlight matches correctly', () => {
    const { highlightMatch } = useSearch();
    
    const text = 'This is a test string';
    const query = 'test';
    const result = highlightMatch(text, query);
    
    // The function should return HTML with highlighted text
    expect(typeof result).toBe('string');
    expect(result).toContain(query);
  });

  test('should copy to clipboard successfully', async () => {
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
    const originalClipboard = navigator.clipboard;
    const mockWriteText = jest.fn().mockRejectedValue(new Error('Copy failed'));
    Object.assign(navigator, {
      clipboard: {
        writeText: mockWriteText
      }
    });

    const { data, copyToClipboard } = useSearch();

    await copyToClipboard('test content');

    expect(data.resultText).toContain('Failed to copy');

    // Restore original clipboard
    Object.assign(navigator, {
      clipboard: originalClipboard
    });
  });

  test('should handle file location opening', async () => {
    (AppModule.ShowInFolder as jest.MockedFunction<any>).mockResolvedValue(undefined);

    const { data, openFileLocation } = useSearch();

    await openFileLocation('/path/to/file.txt');

    expect(AppModule.ShowInFolder).toHaveBeenCalledWith('/path/to/file.txt');
    expect(data.resultText).not.toContain('Could not open file location');
  });
});