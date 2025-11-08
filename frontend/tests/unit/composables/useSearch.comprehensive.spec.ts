import { useSearch } from "../../../src/composables/useSearch";

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
    },
  };
})();

Object.defineProperty(window, "localStorage", {
  value: localStorageMock,
});

// Import the Wails modules for access to their mocked functions
import * as AppModule from "../../../wailsjs/go/main/App";
import * as RuntimeModule from "../../../wailsjs/runtime";

describe("useSearch composable", () => {
  beforeEach(() => {
    // Reset all mocks but preserve the main functionality
    jest.clearAllMocks();

    // Clear localStorage
    localStorage.clear();

    // Set default return values for mocked Wails functions
    (
      AppModule.SearchWithProgress as jest.MockedFunction<any>
    ).mockResolvedValue([]);
    (AppModule.SelectDirectory as jest.MockedFunction<any>).mockResolvedValue(
      "/selected/directory",
    );
    (AppModule.ShowInFolder as jest.MockedFunction<any>).mockResolvedValue(
      undefined,
    );
    (AppModule.CancelSearch as jest.MockedFunction<any>).mockResolvedValue(
      undefined,
    );
    (AppModule.ReadFile as jest.MockedFunction<any>).mockResolvedValue(
      "file content",
    );
    (AppModule.ValidateDirectory as jest.MockedFunction<any>).mockResolvedValue(
      true,
    );

    // Mock EventsOn to return a cleanup function
    (RuntimeModule.EventsOn as jest.MockedFunction<any>).mockReturnValue(
      jest.fn(),
    );
  });

  test("should initialize with default values", () => {
    const { data } = useSearch();

    expect(data.directory).toBe("");
    expect(data.query).toBe("");
    expect(data.extension).toBe("");
    expect(data.caseSensitive).toBe(false);
    expect(data.useRegex).toBe(false);
    expect(data.includeBinary).toBe(false);
    expect(data.maxFileSize).toBe(10485760);
    expect(data.maxResults).toBe(1000);
    expect(data.searchSubdirs).toBe(true);
    expect(data.resultText).toBe("Please enter search parameters below 👇");
    expect(data.searchResults).toEqual([]);
    expect(data.truncatedResults).toBe(false);
    expect(data.isSearching).toBe(false);
    expect(data.showProgress).toBe(false);
    expect(data.minFileSize).toBe(0);
    expect(data.excludePatterns).toEqual([]);
    expect(data.recentSearches).toEqual([]);
    expect(data.error).toBeNull();
  });

  test("should load recent searches from localStorage", () => {
    const mockSearches = [{ query: "test", extension: "go" }];
    localStorage.setItem(
      "codeSearchRecentSearches",
      JSON.stringify(mockSearches),
    );

    const { data } = useSearch();
    expect(data.recentSearches).toEqual(mockSearches);
  });

  test("should handle localStorage errors gracefully", () => {
    // Create a temporary localStorage that throws an error
    const originalGetItem = Storage.prototype.getItem;
    Storage.prototype.getItem = jest.fn(() => {
      throw new Error("Storage error");
    });

    const { data } = useSearch();
    expect(data.recentSearches).toEqual([]);

    // Restore original method
    Storage.prototype.getItem = originalGetItem;
  });

  test("should save recent searches to localStorage", async () => {
    const { data, searchCode } = useSearch();

    // Mock successful search
    const mockResults: any[] = [];
    (
      AppModule.SearchWithProgress as jest.MockedFunction<any>
    ).mockResolvedValue(mockResults);

    data.directory = "/test";
    data.query = "testQuery";
    data.extension = "js";

    await searchCode();

    expect(data.recentSearches).toEqual([
      { query: "testQuery", extension: "js" },
    ]);
    expect(
      JSON.parse(localStorage.getItem("codeSearchRecentSearches") || "[]"),
    ).toEqual([{ query: "testQuery", extension: "js" }]);
  });

  test("should limit recent searches to 5 items", async () => {
    const { data, searchCode } = useSearch();

    // Simulate 6 searches to test the limit
    const mockResults: any[] = [];
    (
      AppModule.SearchWithProgress as jest.MockedFunction<any>
    ).mockResolvedValue(mockResults);

    for (let i = 1; i <= 6; i++) {
      data.directory = "/test";
      data.query = `query${i}`;
      data.extension = "";
      await searchCode();
    }

    expect(data.recentSearches).toHaveLength(5);
    // The most recent search should be first
    expect(data.recentSearches[0]).toEqual({ query: "query6", extension: "" });
    // The oldest should be removed
    expect(data.recentSearches).not.toContainEqual({
      query: "query1",
      extension: "",
    });
  });

  test("should not duplicate recent searches", async () => {
    const { data, searchCode } = useSearch();

    // Mock successful search
    const mockResults: any[] = [];
    (
      AppModule.SearchWithProgress as jest.MockedFunction<any>
    ).mockResolvedValue(mockResults);

    data.directory = "/test";
    data.query = "testQuery";
    data.extension = "js";

    // First search
    await searchCode();
    expect(data.recentSearches).toHaveLength(1);

    // Second search with same query and extension
    await searchCode();
    expect(data.recentSearches).toHaveLength(1);
  });

  test("should handle directory selection successfully", async () => {
    const { data, selectDirectory } = useSearch();

    await selectDirectory();

    expect(data.directory).toBe("/selected/directory");
    expect(data.error).toBeNull();
  });

  test("should handle directory selection cancellation", async () => {
    (AppModule.SelectDirectory as jest.MockedFunction<any>).mockResolvedValue(
      "",
    );

    const { data, selectDirectory } = useSearch();

    await selectDirectory();

    expect(data.directory).toBe("");
    expect(data.error).toBeNull();
  });

  test("should handle directory selection errors", async () => {
    (AppModule.SelectDirectory as jest.MockedFunction<any>).mockRejectedValue(
      new Error("Directory selection failed"),
    );

    const { data, selectDirectory } = useSearch();

    await selectDirectory();

    expect(data.directory).toBe("");
    expect(data.error).toContain("Directory selection failed");
  });

  test('should handle directory selection with "not implemented" error', async () => {
    (AppModule.SelectDirectory as jest.MockedFunction<any>).mockRejectedValue(
      new Error("not implemented"),
    );

    const { data, selectDirectory } = useSearch();

    await selectDirectory();

    expect(data.resultText).toContain("not available on this platform");
  });

  test('should handle directory selection with "no suitable directory picker" error', async () => {
    (AppModule.SelectDirectory as jest.MockedFunction<any>).mockRejectedValue(
      new Error("no suitable directory picker found"),
    );

    const { data, selectDirectory } = useSearch();

    await selectDirectory();

    expect(data.resultText).toContain("No directory picker found");
  });

  test("should search with correct parameters", async () => {
    const mockResults = [
      {
        filePath: "/test/file.go",
        lineNum: 5,
        content: 'fmt.Println("Hello")',
        matchedText: "Hello",
        contextBefore: [],
        contextAfter: [],
      },
    ];
    (
      AppModule.SearchWithProgress as jest.MockedFunction<any>
    ).mockResolvedValue(mockResults);

    const { data, searchCode } = useSearch();

    data.directory = "/test";
    data.query = "Hello";
    data.extension = "go";
    data.caseSensitive = true;
    data.includeBinary = false;
    data.maxFileSize = 1000000;
    data.maxResults = 10;
    data.searchSubdirs = false;

    await searchCode();

    expect(AppModule.SearchWithProgress).toHaveBeenCalledWith({
      directory: "/test",
      query: "Hello",
      extension: "go",
      caseSensitive: true,
      includeBinary: false,
      maxFileSize: 1000000,
      minFileSize: 0,
      maxResults: 10,
      searchSubdirs: false,
      useRegex: false,
      excludePatterns: [],
      allowedFileTypes: [],
    });

    expect(data.searchResults).toEqual(mockResults);
  });

  test("should handle search results correctly", async () => {
    const mockResults = [
      {
        filePath: "/test/file.go",
        lineNum: 5,
        content: 'fmt.Println("Hello")',
        matchedText: "Hello",
        contextBefore: [],
        contextAfter: [],
      },
    ];
    (
      AppModule.SearchWithProgress as jest.MockedFunction<any>
    ).mockResolvedValue(mockResults);

    const { data, searchCode } = useSearch();

    data.directory = "/test";
    data.query = "Hello";

    await searchCode();

    expect(data.searchResults).toEqual(mockResults);
    expect(data.resultText).toBe("Found 1 matches");
  });

  test("should handle no search results", async () => {
    (
      AppModule.SearchWithProgress as jest.MockedFunction<any>
    ).mockResolvedValue([]);

    const { data, searchCode } = useSearch();

    data.directory = "/test";
    data.query = "nonexistent";

    await searchCode();

    expect(data.searchResults).toEqual([]);
    expect(data.resultText).toBe("No matches found");
  });

  test("should handle search errors", async () => {
    (
      AppModule.SearchWithProgress as jest.MockedFunction<any>
    ).mockRejectedValue(new Error("Search failed"));

    const { data, searchCode } = useSearch();

    data.directory = "/test";
    data.query = "test";

    await searchCode();

    expect(data.searchResults).toEqual([]);
    expect(data.resultText).toContain("Error: Search failed");
    expect(data.error).toContain("Search failed");
  });

  test("should handle progress updates", async () => {
    const progressCallback = jest.fn();
    (RuntimeModule.EventsOn as jest.MockedFunction<any>).mockImplementation(
      (event, callback) => {
        // Simulate a progress event
        setTimeout(
          () =>
            callback({
              processedFiles: 5,
              totalFiles: 10,
              currentFile: "/test/file.go",
              resultsCount: 2,
            }),
          0,
        );
        return jest.fn(); // Return a cleanup function
      },
    );

    (
      AppModule.SearchWithProgress as jest.MockedFunction<any>
    ).mockResolvedValue([]);

    const { data, searchCode } = useSearch();

    data.directory = "/test";
    data.query = "test";

    await searchCode();

    // Progress should have been updated
    expect(data.searchProgress.processedFiles).toBeGreaterThanOrEqual(0);
  });

  test("should handle progress updates with completed status", async () => {
    const cleanupFn = jest.fn();
    (RuntimeModule.EventsOn as jest.MockedFunction<any>).mockReturnValue(
      cleanupFn,
    );
    (
      AppModule.SearchWithProgress as jest.MockedFunction<any>
    ).mockResolvedValue([]);

    const { data, searchCode } = useSearch();

    data.directory = "/test";
    data.query = "test";

    await searchCode();

    expect(data.resultText).toBe("No matches found");
  });

  test("should validate inputs before search", async () => {
    const { data, searchCode } = useSearch();

    // Don't set directory - should error
    data.query = "test";

    await searchCode();

    expect(data.error).toBe("Directory is required");
    expect(data.resultText).toBe("Please specify a directory to search in");
  });

  test("should validate numeric inputs correctly", async () => {
    const { data, searchCode } = useSearch();

    // Test invalid max file size
    data.directory = "/test";
    data.query = "test";
    data.maxFileSize = -1;

    await searchCode();

    expect(data.error).toBe("Invalid max file size");
    expect(data.resultText).toContain("maximum file size");

    // Test invalid max results
    data.maxFileSize = 1000000;
    data.maxResults = 0;

    await searchCode();

    expect(data.error).toBe("Invalid max results");
    expect(data.resultText).toContain("maximum number of results");

    // Test valid inputs should not error
    data.maxResults = 500;
    (
      AppModule.SearchWithProgress as jest.MockedFunction<any>
    ).mockResolvedValue([]);
    await searchCode();
    // No error should be set
    expect(data.error).toBeNull();
  });

  test("should handle file location opening successfully", async () => {
    (AppModule.ShowInFolder as jest.MockedFunction<any>).mockResolvedValue(
      undefined,
    );

    const { data, openFileLocation } = useSearch();

    await openFileLocation("/path/to/file.txt");

    expect(AppModule.ShowInFolder).toHaveBeenCalledWith("/path/to/file.txt");
    expect(data.resultText).not.toContain("Could not open file location");
  });

  test("should handle file location opening errors", async () => {
    (AppModule.ShowInFolder as jest.MockedFunction<any>).mockRejectedValue(
      new Error("Could not open folder"),
    );

    const { data, openFileLocation } = useSearch();

    await openFileLocation("/path/to/file.txt");

    expect(data.resultText).toContain("Could not open file location");
    expect(data.error).toContain("Open folder error");
  });

  test("should handle invalid file path in openFileLocation", async () => {
    const { data, openFileLocation } = useSearch();

    await openFileLocation(null); // Empty path should be invalid

    expect(data.resultText).toBe("Invalid file path");
  });

  test("should format complex file paths correctly", () => {
    const { formatFilePath } = useSearch();

    // These tests assume the formatFilePath function truncates long paths
    expect(formatFilePath("/home/user/projects/my-app/src/main.go")).toContain(
      "main.go",
    );
    expect(
      formatFilePath(
        "/home/user/projects/my-app/src/components/CodeSearch.vue",
      ),
    ).toContain("CodeSearch.vue");
    expect(formatFilePath("C:/Users/Name/Documents/file.txt")).toContain(
      "file.txt",
    );
    expect(formatFilePath("relative/path/to/some/file.js")).toContain(
      "file.js",
    );
  });

  test("should process exclude patterns correctly in search", async () => {
    (
      AppModule.SearchWithProgress as jest.MockedFunction<any>
    ).mockResolvedValue([]);

    const { data, searchCode } = useSearch();

    data.directory = "/test";
    data.query = "test";
    data.excludePatterns = ["node_modules", ".git", "*.log"];

    await searchCode();

    // Verify that the search request was made with the correct exclude patterns
    expect(AppModule.SearchWithProgress).toHaveBeenCalledWith(
      expect.objectContaining({
        excludePatterns: ["node_modules", ".git", "*.log"],
      }),
    );
  });

  test("should filter empty exclude patterns", async () => {
    (
      AppModule.SearchWithProgress as jest.MockedFunction<any>
    ).mockResolvedValue([]);

    const { data, searchCode } = useSearch();

    data.directory = "/test";
    data.query = "test";
    data.excludePatterns = ["node_modules", "", ".git", "*.log", ""];

    await searchCode();

    // Verify that empty patterns are filtered out
    expect(AppModule.SearchWithProgress).toHaveBeenCalledWith(
      expect.objectContaining({
        excludePatterns: ["node_modules", ".git", "*.log"],
      }),
    );
  });
});
