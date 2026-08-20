import { vi } from "vitest";
import { mount } from '@vue/test-utils';
import { SearchForm } from '@/components/ui';

// Mock the SearchState data
const mockData = {
  directory: '',
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
  recentSearches: [],
  error: null,
  availableEditors: [],
  editorDetectionStatus: {
    detectingEditors: false,
    detectionComplete: false,
    message: '',
    detectionProgress: 0,
    detectedEditors: []
  },
  knownTextExtensions: [],
  directories: [],
};

const mockSearchCode = vi.fn();
const mockSelectDirectory = vi.fn();
const mockCancelSearch = vi.fn();

describe('SearchForm.vue', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  test('renders all modular child components', () => {
    const wrapper = mount(SearchForm, {
      props: {
        data: mockData,
        searchCode: mockSearchCode,
        selectDirectory: mockSelectDirectory,
        cancelSearch: mockCancelSearch
      }
    });

    expect(wrapper.findComponent({ name: 'EditorStatusDisplay' }).exists()).toBe(true);
    expect(wrapper.findComponent({ name: 'DirectoryPicker' }).exists()).toBe(true);
    expect(wrapper.findComponent({ name: 'QueryInput' }).exists()).toBe(true);
    expect(wrapper.findComponent({ name: 'SearchOptions' }).exists()).toBe(true);
    expect(wrapper.findComponent({ name: 'PatternSelector' }).exists()).toBe(true);
    expect(wrapper.findComponent({ name: 'ActionButtons' }).exists()).toBe(true);
  });

  test('passes editor detection status to EditorStatusDisplay', () => {
    const wrapper = mount(SearchForm, {
      props: {
        data: mockData,
        searchCode: mockSearchCode,
        selectDirectory: mockSelectDirectory,
        cancelSearch: mockCancelSearch
      }
    });

    const editorStatus = wrapper.findComponent({ name: 'EditorStatusDisplay' });
    expect(editorStatus.props('editorDetectionStatus')).toEqual(mockData.editorDetectionStatus);
  });

  test('passes isSearching to ActionButtons', () => {
    const wrapper = mount(SearchForm, {
      props: {
        data: { ...mockData, isSearching: true },
        searchCode: mockSearchCode,
        selectDirectory: mockSelectDirectory,
        cancelSearch: mockCancelSearch
      }
    });

    const actionButtons = wrapper.findComponent({ name: 'ActionButtons' });
    expect(actionButtons.props('isSearching')).toBe(true);
  });

  test('disables child components when searching', () => {
    const wrapper = mount(SearchForm, {
      props: {
        data: { ...mockData, isSearching: true },
        searchCode: mockSearchCode,
        selectDirectory: mockSelectDirectory,
        cancelSearch: mockCancelSearch
      }
    });

    const directoryPicker = wrapper.findComponent({ name: 'DirectoryPicker' });
    expect(directoryPicker.props('disabled')).toBe(true);

    const queryInput = wrapper.findComponent({ name: 'QueryInput' });
    expect(queryInput.props('disabled')).toBe(true);
  });

  test('has main container styling', () => {
    const wrapper = mount(SearchForm, {
      props: {
        data: mockData,
        searchCode: mockSearchCode,
        selectDirectory: mockSelectDirectory,
        cancelSearch: mockCancelSearch
      }
    });

    expect(wrapper.find('.search-form').exists()).toBe(true);
  });

  test('calls searchCode when ActionButtons emits search', async () => {
    const wrapper = mount(SearchForm, {
      props: {
        data: mockData,
        searchCode: mockSearchCode,
        selectDirectory: mockSelectDirectory,
        cancelSearch: mockCancelSearch
      }
    });

    const actionButtons = wrapper.findComponent({ name: 'ActionButtons' });
    actionButtons.vm.$emit('search');
    await wrapper.vm.$nextTick();

    expect(mockSearchCode).toHaveBeenCalled();
  });

  test('calls cancelSearch when ActionButtons emits cancel', async () => {
    const wrapper = mount(SearchForm, {
      props: {
        data: { ...mockData, isSearching: true },
        searchCode: mockSearchCode,
        selectDirectory: mockSelectDirectory,
        cancelSearch: mockCancelSearch
      }
    });

    const actionButtons = wrapper.findComponent({ name: 'ActionButtons' });
    actionButtons.vm.$emit('cancel');
    await wrapper.vm.$nextTick();

    expect(mockCancelSearch).toHaveBeenCalled();
  });

  test('calls selectDirectory when DirectoryPicker emits select', async () => {
    const wrapper = mount(SearchForm, {
      props: {
        data: mockData,
        searchCode: mockSearchCode,
        selectDirectory: mockSelectDirectory,
        cancelSearch: mockCancelSearch
      }
    });

    const directoryPicker = wrapper.findComponent({ name: 'DirectoryPicker' });
    directoryPicker.vm.$emit('select', '/some/path');
    await wrapper.vm.$nextTick();

    expect(mockSelectDirectory).toHaveBeenCalled();
  });

  test('emits update:caseSensitive (and siblings) when SearchOptions updates', async () => {
    const wrapper = mount(SearchForm, {
      props: {
        data: mockData,
        searchCode: mockSearchCode,
        selectDirectory: mockSelectDirectory,
        cancelSearch: mockCancelSearch
      }
    });

    const options = wrapper.findComponent({ name: 'SearchOptions' });
    options.vm.$emit('update', {
      caseSensitive: true,
      useRegex: false,
      includeBinary: true,
      fuzzySearch: false,
      respectGitignore: true
    });
    await wrapper.vm.$nextTick();

    expect(wrapper.emitted('update:caseSensitive')![0]).toEqual([true]);
    expect(wrapper.emitted('update:useRegex')![0]).toEqual([false]);
    expect(wrapper.emitted('update:includeBinary')![0]).toEqual([true]);
    expect(wrapper.emitted('update:fuzzySearch')![0]).toEqual([false]);
    expect(wrapper.emitted('update:respectGitignore')![0]).toEqual([true]);
  });

  test('emits update:query when QueryInput updates', async () => {
    const wrapper = mount(SearchForm, {
      props: {
        data: mockData,
        searchCode: mockSearchCode,
        selectDirectory: mockSelectDirectory,
        cancelSearch: mockCancelSearch
      }
    });

    const queryInput = wrapper.findComponent({ name: 'QueryInput' });
    queryInput.vm.$emit('update', 'hello world');
    await wrapper.vm.$nextTick();

    expect(wrapper.emitted('update:query')![0]).toEqual(['hello world']);
  });

  test('emits update:directory when DirectoryPicker updates', async () => {
    const wrapper = mount(SearchForm, {
      props: {
        data: mockData,
        searchCode: mockSearchCode,
        selectDirectory: mockSelectDirectory,
        cancelSearch: mockCancelSearch
      }
    });

    const directoryPicker = wrapper.findComponent({ name: 'DirectoryPicker' });
    directoryPicker.vm.$emit('update', '/new/dir');
    await wrapper.vm.$nextTick();

    expect(wrapper.emitted('update:directory')![0]).toEqual(['/new/dir']);
  });

  test('emits update:directories when extra directories textarea changes', async () => {
    const wrapper = mount(SearchForm, {
      props: {
        data: mockData,
        searchCode: mockSearchCode,
        selectDirectory: mockSelectDirectory,
        cancelSearch: mockCancelSearch
      }
    });

    const textarea = wrapper.find('#extra-dirs');
    await textarea.setValue('/one/path\n  /two/path  \n\n/trailing-empty');
    await wrapper.vm.$nextTick();

    expect(wrapper.emitted('update:directories')![0]).toEqual([
      ['/one/path', '/two/path', '/trailing-empty']
    ]);
  });

  test('emits update:excludePatterns and update:allowedFileTypes when PatternSelector updates', async () => {
    const wrapper = mount(SearchForm, {
      props: {
        data: mockData,
        searchCode: mockSearchCode,
        selectDirectory: mockSelectDirectory,
        cancelSearch: mockCancelSearch
      }
    });

    const patternSelector = wrapper.findComponent({ name: 'PatternSelector' });
    patternSelector.vm.$emit('update', { exclude: ['node_modules'], allow: ['go', 'ts'] });
    await wrapper.vm.$nextTick();

    expect(wrapper.emitted('update:excludePatterns')![0]).toEqual([['node_modules']]);
    expect(wrapper.emitted('update:allowedFileTypes')![0]).toEqual([['go', 'ts']]);
  });

  test('emits update:excludePatterns with a new array when a pattern is removed', async () => {
    const initialData = { ...mockData, excludePatterns: ['node_modules', 'dist'] };
    const wrapper = mount(SearchForm, {
      props: {
        data: initialData,
        searchCode: mockSearchCode,
        selectDirectory: mockSelectDirectory,
        cancelSearch: mockCancelSearch
      }
    });

    const patternSelector = wrapper.findComponent({ name: 'PatternSelector' });
    patternSelector.vm.$emit('remove-pattern', 'exclude', 0);
    await wrapper.vm.$nextTick();

    // The emit must carry a NEW array (the prop object must never be mutated).
    expect(wrapper.emitted('update:excludePatterns')![0]).toEqual([['dist']]);
    expect(initialData.excludePatterns).toEqual(['node_modules', 'dist']);
  });

  test('emits update:query/update:extension/update:directory when a suggestion is selected', async () => {
    const wrapper = mount(SearchForm, {
      props: {
        data: mockData,
        searchCode: mockSearchCode,
        selectDirectory: mockSelectDirectory,
        cancelSearch: mockCancelSearch
      }
    });

    // Suggestions only render while the search input has focus.
    const queryInput = wrapper.findComponent({ name: 'QueryInput' });
    queryInput.vm.$emit('focus');
    await wrapper.vm.$nextTick();

    const suggestions = wrapper.findComponent({ name: 'SearchSuggestions' });
    expect(suggestions.exists()).toBe(true);
    suggestions.vm.$emit('select', { query: 'fmt.Println', extension: 'go', directory: '/proj' });
    await wrapper.vm.$nextTick();

    expect(wrapper.emitted('update:query')![0]).toEqual(['fmt.Println']);
    expect(wrapper.emitted('update:extension')![0]).toEqual(['go']);
    expect(wrapper.emitted('update:directory')![0]).toEqual(['/proj']);
    expect(mockSearchCode).toHaveBeenCalled();
  });

  test('emits update:recentSearches when a suggestion is removed', async () => {
    const wrapper = mount(SearchForm, {
      props: {
        data: mockData,
        searchCode: mockSearchCode,
        selectDirectory: mockSelectDirectory,
        cancelSearch: mockCancelSearch
      }
    });

    // Suggestions only render while the search input has focus.
    const queryInput = wrapper.findComponent({ name: 'QueryInput' });
    queryInput.vm.$emit('focus');
    await wrapper.vm.$nextTick();

    const suggestions = wrapper.findComponent({ name: 'SearchSuggestions' });
    expect(suggestions.exists()).toBe(true);
    suggestions.vm.$emit('remove', 'hello');
    await wrapper.vm.$nextTick();

    expect(wrapper.emitted('update:recentSearches')).toBeTruthy();
  });
});
