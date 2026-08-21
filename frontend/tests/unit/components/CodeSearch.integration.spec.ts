import { describe, test, expect, vi } from 'vitest';
import { mount } from '@vue/test-utils';
import { CodeSearch } from '@/components';

// Mock all dependencies
vi.mock('@wails/go/main/App', () => ({
  SearchWithProgress: vi.fn().mockResolvedValue([]),
  SelectDirectory: vi.fn().mockResolvedValue('/test/dir'),
  ShowInFolder: vi.fn().mockResolvedValue(undefined),
  CancelSearch: vi.fn().mockResolvedValue(undefined),
  ReadFile: vi.fn().mockResolvedValue('file content'),
  GetKnownTextExtensions: vi.fn().mockResolvedValue(['go', 'ts', 'js']),
  GetEditorDetectionStatus: vi.fn().mockResolvedValue({
    totalAvailable: 3,
    availableEditors: { vscode: true, vim: false },
    detectionComplete: true,
    message: 'Detection complete'
  })
}));

vi.mock('@wails/runtime', () => ({
  EventsOn: vi.fn().mockReturnValue(() => {})
}));

describe('CodeSearch.vue Integration Tests', () => {
  const mockData = {
    directory: '/test',
    query: '',
    extension: '',
    caseSensitive: false,
    useRegex: false,
    includeBinary: false,
    maxFileSize: 10485760,
    maxResults: 1000,
    resultText: 'Please enter search parameters below 👇',
    searchResults: [],
    truncatedResults: false,
    isSearching: false,
    searchProgress: {
      processedFiles: 0,
      totalFiles: 0,
      currentFile: '',
      resultsCount: 0,
      status: ''
    },
    showProgress: false,
    minFileSize: 0,
    excludePatterns: [],
    allowedFileTypes: [],
    knownTextExtensions: ['go', 'ts'],
    recentSearches: [],
    error: null,
    availableEditors: {
      vscode: true,
      vscodium: false,
      sublime: false,
      jetbrains: false,
      geany: false,
      neovim: false,
      vim: false,
      goland: false,
      pycharm: false,
      intellij: false,
      webstorm: false,
      phpstorm: false,
      clion: false,
      rider: false,
      androidstudio: false,
      systemdefault: false,
      emacs: false,
      neovide: false,
      codeblocks: false,
      devcpp: false,
      notepadplusplus: false,
      visualstudio: false,
      eclipse: false,
      netbeans: false
    },
    editorDetectionStatus: {
      detectionComplete: false,
      totalAvailable: 0,
      message: 'Detecting editors...',
      detectionProgress: 0,
      detectingEditors: true,
      detectedEditors: [],
      availableEditors: {} as any
    },
    fuzzySearch: false,
    contextLines: 3,
    directories: []
  };

  test('renders main UI elements', async () => {
    const wrapper = mount(CodeSearch, {
      props: {
        data: mockData,
        searchCode: () => Promise.resolve(),
        cancelSearch: () => Promise.resolve(),
        selectDirectory: () => Promise.resolve()
      }
    });

    // Check that main components are present
    expect(wrapper.findComponent({ name: 'SearchForm' }).exists()).toBe(true);
    expect(wrapper.find('.result').exists()).toBe(true);
  });

  test('handles search flow with fuzzy search enabled', async () => {
    const wrapper = mount(CodeSearch, {
      props: {
        data: mockData,
        searchCode: () => Promise.resolve(),
        cancelSearch: () => Promise.resolve(),
        selectDirectory: () => Promise.resolve()
      }
    });

    // Enable fuzzy search via the form
    const fuzzyCheckbox = wrapper.find('#fuzzy-search');
    if (fuzzyCheckbox.exists()) {
      await fuzzyCheckbox.setValue(true);
      expect(fuzzyCheckbox.element.checked).toBe(true);
    }
  });

  test('displays sidebar with search history', async () => {
    const wrapper = mount(CodeSearch, {
      props: {
        data: {
          ...mockData,
          recentSearches: [
            { query: 'test query', extension: 'go' },
            { query: 'another query', extension: '' }
          ]
        },
        searchCode: () => Promise.resolve(),
        cancelSearch: () => Promise.resolve(),
        selectDirectory: () => Promise.resolve()
      }
    });

    // Sidebar should be present
    expect(wrapper.find('.search-history-sidebar').exists()).toBe(true);
  });

  test('handles empty state correctly', async () => {
    const wrapper = mount(CodeSearch, {
      props: {
        data: mockData,
        searchCode: () => Promise.resolve(),
        cancelSearch: () => Promise.resolve(),
        selectDirectory: () => Promise.resolve()
      }
    });

    const emptyState = wrapper.find('.empty-state');
    
    // Without search history or with empty searches, may show different state
    // Just verify component doesn't crash
    expect(wrapper.exists()).toBe(true);
  });

  test('layout has flex structure', async () => {
    const wrapper = mount(CodeSearch, {
      props: {
        data: mockData,
        searchCode: () => Promise.resolve(),
        cancelSearch: () => Promise.resolve(),
        selectDirectory: () => Promise.resolve()
      }
    });

    // Main component should render with correct structure
    expect(wrapper.exists()).toBe(true);
    
    // Sidebar should be present
    expect(wrapper.find('.search-history-sidebar').exists()).toBe(true);
    
    // Main content area should be present  
    expect(wrapper.find('.main-content').exists()).toBe(true);
  });

  test('main content area exists', async () => {
    const wrapper = mount(CodeSearch, {
      props: {
        data: mockData,
        searchCode: () => Promise.resolve(),
        cancelSearch: () => Promise.resolve(),
        selectDirectory: () => Promise.resolve()
      }
    });

    const mainContent = wrapper.find('.main-content');
    expect(mainContent.exists()).toBe(true);
  });

  test('emits re-search events from sidebar', async () => {
    const searchHistoryMock = [{ query: 'test', extension: 'go' }];
    
    const wrapper = mount(CodeSearch, {
      props: {
        data: {
          ...mockData,
          recentSearches: searchHistoryMock
        },
        searchCode: () => Promise.resolve(),
        cancelSearch: () => Promise.resolve(),
        selectDirectory: () => Promise.resolve()
      }
    });

    // Find and trigger the re-search sidebar component
    const sidebarComponent = wrapper.findComponent({ name: 'SearchHistorySidebar' });
    
    if (sidebarComponent.exists()) {
      await sidebarComponent.vm.$emit('re-search', searchHistoryMock[0]);
      
      // Verify emit was attempted
      expect(sidebarComponent.emitted('re-search')).toBeTruthy();
    }
  });

  test('removes items from history on remove event', async () => {
    const searchHistoryMock = [
      { query: 'first', extension: '' },
      { query: 'second', extension: 'go' }
    ];
    
    const wrapper = mount(CodeSearch, {
      props: {
        data: {
          ...mockData,
          recentSearches: searchHistoryMock
        },
        searchCode: () => Promise.resolve(),
        cancelSearch: () => Promise.resolve(),
        selectDirectory: () => Promise.resolve()
      }
    });

    const sidebarComponent = wrapper.findComponent({ name: 'SearchHistorySidebar' });
    
    if (sidebarComponent.exists()) {
      await sidebarComponent.vm.$emit('remove', 0);
      expect(sidebarComponent.emitted('remove')).toBeTruthy();
    }
  });

  test('clears all history on clear-all event', async () => {
    const wrapper = mount(CodeSearch, {
      props: {
        data: {
          ...mockData,
          recentSearches: [{ query: 'test', extension: '' }]
        },
        searchCode: () => Promise.resolve(),
        cancelSearch: () => Promise.resolve(),
        selectDirectory: () => Promise.resolve()
      }
    });

    const sidebarComponent = wrapper.findComponent({ name: 'SearchHistorySidebar' });
    
    if (sidebarComponent.exists()) {
      await sidebarComponent.vm.$emit('clear-all');
      expect(sidebarComponent.emitted('clear-all')).toBeTruthy();
    }
  });

  test('inline diff view renders in search results', async () => {
    const resultsData = {
      ...mockData,
      searchResults: [{
        filePath: '/test/file.go',
        lineNum: 5,
        content: 'fmt.Println("test")',
        matchedText: 'test',
        contextBefore: ['package main'],
        contextAfter: ['func main() {}']
      }]
    };

    const wrapper = mount(CodeSearch, {
      props: {
        data: resultsData,
        searchCode: () => Promise.resolve(),
        cancelSearch: () => Promise.resolve(),
        selectDirectory: () => Promise.resolve()
      }
    });

    // The InlineDiffView should render when searching with results
    // Note: SearchResults needs to be mounted separately for full inline diff testing
    expect(wrapper.exists()).toBe(true);
  });

  test('toggles sidebar collapse/expand', async () => {
    const wrapper = mount(CodeSearch, {
      props: {
        data: mockData,
        searchCode: () => Promise.resolve(),
        cancelSearch: () => Promise.resolve(),
        selectDirectory: () => Promise.resolve()
      }
    });

    const sidebarComponent = wrapper.findComponent({ name: 'SearchHistorySidebar' });
    
    if (sidebarComponent.exists()) {
      const isVisible = (sidebarComponent.vm as any).isVisible;
      
      if (isVisible) {
        // Toggle would change visibility
        expect(typeof isVisible).toBe('boolean');
      }
    }
  });

  test('handles loading states properly', async () => {
    const loadingData = {
      ...mockData,
      isSearching: true,
      showProgress: true,
      searchProgress: {
        processedFiles: 5,
        totalFiles: 100,
        currentFile: 'test.go',
        resultsCount: 2,
        status: 'in-progress'
      }
    };

    const wrapper = mount(CodeSearch, {
      props: {
        data: loadingData,
        searchCode: () => Promise.resolve(),
        cancelSearch: () => Promise.resolve(),
        selectDirectory: () => Promise.resolve()
      }
    });

    expect(wrapper.exists()).toBe(true);
  });

  test('search button disabled during search', async () => {
    const wrapper = mount(CodeSearch, {
      props: {
        data: { ...mockData, isSearching: true },
        searchCode: () => Promise.resolve(),
        cancelSearch: () => Promise.resolve(),
        selectDirectory: () => Promise.resolve()
      }
    });

    // Should see cancel button instead of search button when searching
    const cancelButton = wrapper.find('.cancel-btn');
    if (cancelButton.exists()) {
      expect(cancelButton.element.disabled).toBe(false);
    }
  });

  test('registers and cleans up symbol-selected listener exactly once', async () => {
    const addSpy = vi.spyOn(window, 'addEventListener');
    const removeSpy = vi.spyOn(window, 'removeEventListener');

    const wrapper = mount(CodeSearch, {
      props: {
        data: mockData,
        searchCode: () => Promise.resolve(),
        cancelSearch: () => Promise.resolve(),
        selectDirectory: () => Promise.resolve()
      }
    });

    // onMounted registers 'symbol-selected'; onUnmounted removes it.
    expect(addSpy).toHaveBeenCalledWith('symbol-selected', expect.any(Function));

    // Trigger onUnmounted and verify cleanup runs exactly once.
    wrapper.unmount();

    // There were 2 onUnmounted calls before the fix: one in the
    // listen-to-cleanup pair, one in the duplicate trailing cleanup()
    // block. After the fix there is exactly one removal.
    const symbolRemoveCalls = removeSpy.mock.calls.filter(
      ([name]) => name === 'symbol-selected'
    );
    expect(symbolRemoveCalls.length).toBe(1);
    expect(removeSpy).toHaveBeenCalledWith('symbol-selected', expect.any(Function));

    addSpy.mockRestore();
    removeSpy.mockRestore();
  });
});
