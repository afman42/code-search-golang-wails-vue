import { describe, test, expect, vi, beforeEach } from 'vitest';
import { useFilePreview } from '@/composables/useFilePreview';

// Mock the Wails ReadFile binding — must match the import path in useFilePreview.ts
vi.mock('@wails/go/main/App', () => ({
  ReadFile: vi.fn().mockResolvedValue('mock file content'),
}));

describe('useFilePreview', () => {
  beforeEach(() => {
    // Reset singleton state between tests — closePreview keeps old filePath,
    // so we need to force-clear the state to avoid isSameFile matching.
    const { previewState } = useFilePreview();
    previewState.value = {
      isVisible: false,
      filePath: '',
      fileContent: '',
      query: '',
      files: [],
      initialLine: null,
    };
    vi.clearAllMocks();
  });

  test('openFile sets isVisible and filePath', async () => {
    const { previewState, openFile } = useFilePreview();
    await openFile('/test/file.go', { initialLine: 10 });

    expect(previewState.value.isVisible).toBe(true);
    expect(previewState.value.filePath).toBe('/test/file.go');
    expect(previewState.value.initialLine).toBe(10);
  });

  test('openFile loads content from backend when not provided', async () => {
    const { previewState, openFile } = useFilePreview();
    await openFile('/test/file.go');

    expect(previewState.value.fileContent).toBe('mock file content');
  });

  test('openFile uses provided content without calling ReadFile', async () => {
    const { previewState, openFile } = useFilePreview();
    const { ReadFile } = await import('@wails/go/main/App');
    await openFile('/test/file.go', { fileContent: 'passed content' });

    expect(previewState.value.fileContent).toBe('passed content');
    expect(ReadFile).not.toHaveBeenCalled();
  });

  test('openFile same file retains existing content (no reload)', async () => {
    const { previewState, openFile } = useFilePreview();
    const { ReadFile } = await import('@wails/go/main/App');

    await openFile('/test/file.go', { fileContent: 'first content' });
    await openFile('/test/file.go', { initialLine: 25 });

    expect(previewState.value.fileContent).toBe('first content');
    expect(ReadFile).not.toHaveBeenCalled();
  });

  test('openFile different file clears old content and loads new', async () => {
    const { previewState, openFile } = useFilePreview();
    await openFile('/test/file1.go', { fileContent: 'content1' });
    expect(previewState.value.fileContent).toBe('content1');

    await openFile('/test/file2.go', { initialLine: 5 });
    expect(previewState.value.fileContent).toBe('mock file content');
    expect(previewState.value.filePath).toBe('/test/file2.go');
  });

  test('closePreview sets isVisible false and clears initialLine', async () => {
    const { previewState, openFile, closePreview } = useFilePreview();
    await openFile('/test/file.go', { initialLine: 10 });

    closePreview();

    expect(previewState.value.isVisible).toBe(false);
    expect(previewState.value.initialLine).toBe(null);
  });

  test('openFile passes query and files options', async () => {
    const { previewState, openFile } = useFilePreview();
    await openFile('/test/file.go', {
      query: 'search term',
      files: ['/a.go', '/b.go'],
    });

    expect(previewState.value.query).toBe('search term');
    expect(previewState.value.files).toEqual(['/a.go', '/b.go']);
  });
});
