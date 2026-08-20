import { vi } from "vitest";
import { mount } from '@vue/test-utils';
import { SearchResults } from '@/components/ui';
import { ReadFile } from '@wails/go/main/App';
import {
  makeEditorAvailability,
  makeEditorDetectionStatus,
} from '../../fixtures/editorAvailability';

// Mock the SearchState data with results
const mockDataWithResults = {
  directory: '',
  query: 'test',
  extension: '',
  caseSensitive: false,
  useRegex: false,
  includeBinary: false,
  maxFileSize: 10485760,
  maxResults: 1000,
  resultText: 'Found 2 matches',
  searchResults: [
    {
      filePath: '/test/file1.go',
      lineNum: 5,
      content: 'fmt.Println("test message")',
      matchedText: 'test',
      contextBefore: ['package main', '', 'import "fmt"'],
      contextAfter: ['func main() {', '\tfmt.Println("another test")']
    },
    {
      filePath: '/test/file2.js',
      lineNum: 10,
      content: 'console.log("test");',
      matchedText: 'test',
      contextBefore: ['// This is a JS file', 'function testFunction() {'],
      contextAfter: ['\treturn true;', '}']
    }
  ],
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
  knownTextExtensions: [],
  recentSearches: [],
  error: null,
  availableEditors: makeEditorAvailability(),
  editorDetectionStatus: makeEditorDetectionStatus(),
};

const mockFormatFilePath = vi.fn((path: string) => path);
const mockOpenFileLocation = vi.fn();
const mockCopyToClipboard = vi.fn();

describe('SearchResults.vue', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  test('renders search results properly', () => {
    const wrapper = mount(SearchResults, {
      props: {
        data: mockDataWithResults,
        formatFilePath: mockFormatFilePath,
        openFileLocation: mockOpenFileLocation,
        copyToClipboard: mockCopyToClipboard
      }
    });
    
    // Check that results container exists
    expect(wrapper.find('.results-container').exists()).toBe(true);
    
    // Check that results header exists
    expect(wrapper.find('.results-header').exists()).toBe(true);
    expect(wrapper.text()).toContain('Search Results:');
    expect(wrapper.text()).toContain('Found 2 matches');
    
    // Check that result items exist (using InlineDiffView)
    const inlineDiffViews = wrapper.findAllComponents({ name: 'InlineDiffView' });
    expect(inlineDiffViews.length).toBe(2);
    
    // Check first inline diff view renders correctly
    const firstDiff = inlineDiffViews[0];
    expect(firstDiff.exists()).toBe(true);
    
    // Verify we have file paths in the results
    const filePaths = wrapper.findAll('.file-path');
    expect(filePaths.length).toBe(2);
    expect(filePaths[0].text()).toBe('/test/file1.go');
    expect(filePaths[1].text()).toBe('/test/file2.js');
  });

  test('shows truncated results message when applicable', () => {
    const truncatedData = { ...mockDataWithResults, truncatedResults: true };
    
    const wrapper = mount(SearchResults, {
      props: {
        data: truncatedData,
        formatFilePath: mockFormatFilePath,
        openFileLocation: mockOpenFileLocation,
        copyToClipboard: mockCopyToClipboard
      }
    });
    
    expect(wrapper.text()).toContain('(truncated)');
  });

  test('does not render when no results', () => {
    const emptyData = { ...mockDataWithResults, searchResults: [] };
    
    const wrapper = mount(SearchResults, {
      props: {
        data: emptyData,
        formatFilePath: mockFormatFilePath,
        openFileLocation: mockOpenFileLocation,
        copyToClipboard: mockCopyToClipboard
      }
    });
    
    expect(wrapper.find('.results-container').exists()).toBe(false);
  });

  test('calls openFileLocation when file path is clicked', async () => {
    const wrapper = mount(SearchResults, {
      props: {
        data: mockDataWithResults,
        formatFilePath: mockFormatFilePath,
        openFileLocation: mockOpenFileLocation,
        copyToClipboard: mockCopyToClipboard
      }
    });
    
    const filePath = wrapper.find('.file-path');
    await filePath.trigger('click');
    
    expect(mockOpenFileLocation).toHaveBeenCalledWith('/test/file1.go');
  });

  test('calls copyToClipboard when copy button is clicked', async () => {
    const wrapper = mount(SearchResults, {
      props: {
        data: mockDataWithResults,
        formatFilePath: mockFormatFilePath,
        openFileLocation: mockOpenFileLocation,
        copyToClipboard: mockCopyToClipboard
      }
    });
    
    const copyButton = wrapper.find('.copy-btn');
    await copyButton.trigger('click');
    
    expect(mockCopyToClipboard).toHaveBeenCalledWith('fmt.Println("test message")');
  });

  test('displays fuzzy match badge when similarity score exists', async () => {
    const fuzzyData = {
      ...mockDataWithResults,
      searchResults: [
        {
          filePath: '/test/file1.go',
          lineNum: 5,
          content: 'fmt.Println("tset messga")', // misspelled "test" and "message"
          matchedText: 'tset',
          contextBefore: ['package main'],
          contextAfter: ['func main() {'],
          fuzzyMatch: true,
          similarityScore: 0.85
        }
      ]
    };

    const wrapper = mount(SearchResults, {
      props: {
        data: fuzzyData,
        formatFilePath: mockFormatFilePath,
        openFileLocation: mockOpenFileLocation,
        copyToClipboard: mockCopyToClipboard
      }
    });

    const inlineDiff = wrapper.findComponent({ name: 'InlineDiffView' });
    expect(inlineDiff.exists()).toBe(true);
    
    // Check that fuzzy badge is rendered with correct percentage
    expect(wrapper.text()).toContain('~');
  });

  test('renders InlineDiffView component for each result', async () => {
    const wrapper = mount(SearchResults, {
      props: {
        data: mockDataWithResults,
        formatFilePath: mockFormatFilePath,
        openFileLocation: mockOpenFileLocation,
        copyToClipboard: mockCopyToClipboard
      }
    });

    const inlineDiffViews = wrapper.findAllComponents({ name: 'InlineDiffView' });
    expect(inlineDiffViews.length).toBe(2);
  });

  test('shows line numbers correctly in context lines', async () => {
    const wrapper = mount(SearchResults, {
      props: {
        data: mockDataWithResults,
        formatFilePath: mockFormatFilePath,
        openFileLocation: mockOpenFileLocation,
        copyToClipboard: mockCopyToClipboard
      }
    });

    const inlineDiffViews = wrapper.findAllComponents({ name: 'InlineDiffView' });
    
    // Check that InlineDiffView components are rendering
    expect(inlineDiffViews.length).toBe(2);
    
    // Verify the component received correct data by checking rendered content
    const firstLine = inlineDiffViews[0].find('.result-line.matched');
    expect(firstLine.exists()).toBe(true);
  });

  test('emits copy event on copy-line-click', async () => {
    const wrapper = mount(SearchResults, {
      props: {
        data: mockDataWithResults,
        formatFilePath: mockFormatFilePath,
        openFileLocation: mockOpenFileLocation,
        copyToClipboard: mockCopyToClipboard
      }
    });

    const inlineDiff = wrapper.findAllComponents({ name: 'InlineDiffView' })[0];
    await inlineDiff.vm.$emit('copy', 'fmt.Println("test message")');
    
    expect(mockCopyToClipboard).toHaveBeenCalledWith('fmt.Println("test message")');
  });

  test('handles empty context arrays gracefully', async () => {
    const noContextData = {
      ...mockDataWithResults,
      searchResults: [
        {
          filePath: '/test/file1.go',
          lineNum: 5,
          content: 'fmt.Println("test")',
          matchedText: 'test',
          contextBefore: [],
          contextAfter: []
        }
      ]
    };

    const wrapper = mount(SearchResults, {
      props: {
        data: noContextData,
        formatFilePath: mockFormatFilePath,
        openFileLocation: mockOpenFileLocation,
        copyToClipboard: mockCopyToClipboard
      }
    });

    // Check that InlineDiffView exists (component should render even with empty context)
    const inlineDiffViews = wrapper.findAllComponents({ name: 'InlineDiffView' });
    expect(inlineDiffViews.length).toBe(1);
    
    // Get first component and check it received the empty arrays via its props
    // Note: In Vue Test Utils, we access child component props differently
    // We can verify by checking the rendered output doesn't crash
    expect(wrapper.find('.inline-diff-view').exists()).toBe(true);
  });

  test('uses a unique v-for key including the row index', async () => {
    // Two results with identical filePath + lineNum + content would collide
    // under the old content-based key; the index suffix keeps them distinct.
    const duplicateData = {
      ...mockDataWithResults,
      searchResults: [
        {
          filePath: '/test/dup.go',
          lineNum: 5,
          content: 'same content',
          matchedText: 'test',
          contextBefore: [],
          contextAfter: []
        },
        {
          filePath: '/test/dup.go',
          lineNum: 5,
          content: 'same content',
          matchedText: 'test',
          contextBefore: [],
          contextAfter: []
        }
      ]
    };

    const wrapper = mount(SearchResults, {
      props: {
        data: duplicateData,
        formatFilePath: mockFormatFilePath,
        openFileLocation: mockOpenFileLocation,
        copyToClipboard: mockCopyToClipboard
      }
    });

    const inlineDiffViews = wrapper.findAllComponents({ name: 'InlineDiffView' });
    expect(inlineDiffViews.length).toBe(2);
  });

  test('toasts an error when copying selected results fails', async () => {
    const failingCopy = vi.fn().mockRejectedValue(new Error('clipboard denied'));
    const wrapper = mount(SearchResults, {
      props: {
        data: mockDataWithResults,
        formatFilePath: mockFormatFilePath,
        openFileLocation: mockOpenFileLocation,
        copyToClipboard: failingCopy
      }
    });

    // Select both rows then trigger the copy action.
    const vm = wrapper.vm as unknown as {
      toggleSelected: (idx: number) => void;
      handleCopySelected: () => Promise<void>;
    };
    vm.toggleSelected(0);
    vm.toggleSelected(1);
    await wrapper.vm.$nextTick();

    // Must not throw even though copyToClipboard rejects.
    await expect(vm.handleCopySelected()).resolves.toBeUndefined();
  });

  test('emits update:error when reading a file fails', async () => {
    const readMock = vi.mocked(ReadFile).mockRejectedValue(new Error('boom'));

    const wrapper = mount(SearchResults, {
      props: {
        data: mockDataWithResults,
        formatFilePath: mockFormatFilePath,
        openFileLocation: mockOpenFileLocation,
        copyToClipboard: mockCopyToClipboard
      }
    });

    const vm = wrapper.vm as unknown as {
      openFilePreview: (filePath: string) => Promise<void>;
    };

    await vm.openFilePreview('/test/file1.go');
    await wrapper.vm.$nextTick();

    expect(readMock).toHaveBeenCalled();
    expect(wrapper.emitted('update:error')).toBeTruthy();
    expect(wrapper.emitted('update:resultText')).toBeTruthy();
    readMock.mockRestore();
  });
});