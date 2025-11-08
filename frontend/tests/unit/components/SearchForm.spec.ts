import { mount } from '@vue/test-utils';
import SearchForm from '../../../src/components/ui/SearchForm.vue';

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
  searchSubdirs: true,
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
  error: null
};

const mockSearchCode = jest.fn();
const mockSelectDirectory = jest.fn();
const mockCancelSearch = jest.fn();

describe('SearchForm.vue', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  test('renders search form controls properly', () => {
    const wrapper = mount(SearchForm, {
      props: {
        data: mockData,
        searchCode: mockSearchCode,
        selectDirectory: mockSelectDirectory,
        cancelSearch: mockCancelSearch
      }
    });
    
    // Check that the search controls exist
    expect(wrapper.find('input#directory').exists()).toBe(true);
    expect(wrapper.find('input#query').exists()).toBe(true);
    expect(wrapper.find('input#extension').exists()).toBe(true);
    expect(wrapper.find('input#case-sensitive').exists()).toBe(true);
    expect(wrapper.find('input#regex-search').exists()).toBe(true);
    expect(wrapper.find('input#include-binary').exists()).toBe(true);
    expect(wrapper.find('input#search-subdirs').exists()).toBe(true);
    expect(wrapper.find('input#min-filesize').exists()).toBe(true);
    expect(wrapper.find('input#max-filesize').exists()).toBe(true);
    expect(wrapper.find('input#max-results').exists()).toBe(true);
    expect(wrapper.find('select#exclude-patterns').exists()).toBe(true);
    expect(wrapper.find('button.search-btn').exists()).toBe(true);
    expect(wrapper.find('button.select-dir').exists()).toBe(true);
  });

  test('allows user to input directory, query, extension, and toggles', async () => {
    const wrapper = mount(SearchForm, {
      props: {
        data: mockData,
        searchCode: mockSearchCode,
        selectDirectory: mockSelectDirectory
      }
    });
    
    const directoryInput = wrapper.find('input#directory');
    const queryInput = wrapper.find('input#query');
    const extensionInput = wrapper.find('input#extension');
    const caseSensitiveCheckbox = wrapper.find('input#case-sensitive');
    const regexCheckbox = wrapper.find('input#regex-search');
    
    await directoryInput.setValue('/test/directory');
    await queryInput.setValue('testQuery');
    await extensionInput.setValue('go');
    await caseSensitiveCheckbox.setChecked(true);
    await regexCheckbox.setChecked(true);
    
    expect(directoryInput.element.value).toBe('/test/directory');
    expect(queryInput.element.value).toBe('testQuery');
    expect(extensionInput.element.value).toBe('go');
    expect(caseSensitiveCheckbox.element.checked).toBe(true);
    expect(regexCheckbox.element.checked).toBe(true);
  });

  test('calls searchCode when search button is clicked', async () => {
    const wrapper = mount(SearchForm, {
      props: {
        data: mockData,
        searchCode: mockSearchCode,
        selectDirectory: mockSelectDirectory
      }
    });
    
    await wrapper.find('button.search-btn').trigger('click');
    
    expect(mockSearchCode).toHaveBeenCalled();
  });

  test('calls selectDirectory when browse button is clicked', async () => {
    const wrapper = mount(SearchForm, {
      props: {
        data: mockData,
        searchCode: mockSearchCode,
        selectDirectory: mockSelectDirectory
      }
    });
    
    await wrapper.find('button.select-dir').trigger('click');
    
    expect(mockSelectDirectory).toHaveBeenCalled();
  });
});