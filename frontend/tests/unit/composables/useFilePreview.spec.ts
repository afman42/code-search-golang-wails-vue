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

  test('openFile defers initialLine until content loads for new file', async () => {
    const { previewState, openFile } = useFilePreview();
    const { ReadFile } = await import('@wails/go/main/App');

    // Make ReadFile resolve asynchronously so we can observe the
    // intermediate state (before content arrives).
    const { promise: readPromise, resolve: resolveRead } = Promise.withResolvers<string>();
    vi.mocked(ReadFile).mockReturnValueOnce(readPromise);

    const promise = openFile('/test/file.go', { initialLine: 42 });

    // Before ReadFile resolves: modal is visible, filePath set, but
    // fileContent is empty and initialLine must NOT be set yet.
    expect(previewState.value.isVisible).toBe(true);
    expect(previewState.value.filePath).toBe('/test/file.go');
    expect(previewState.value.fileContent).toBe('');
    expect(previewState.value.initialLine).toBe(null);

    // Now resolve the async content load.
    resolveRead('real content');
    await promise;

    // After content arrives: both content and initialLine are set.
    expect(previewState.value.fileContent).toBe('real content');
    expect(previewState.value.initialLine).toBe(42);
  });

  test('openFile sets initialLine immediately when content is provided', async () => {
    const { previewState, openFile } = useFilePreview();
    const { ReadFile } = await import('@wails/go/main/App');

    await openFile('/test/file.go', { fileContent: 'passed content', initialLine: 15 });

    // Content was provided synchronously, so initialLine is set right away.
    expect(previewState.value.fileContent).toBe('passed content');
    expect(previewState.value.initialLine).toBe(15);
    expect(ReadFile).not.toHaveBeenCalled();
  });

  test('openFile same file sets initialLine immediately (content already loaded)', async () => {
    const { previewState, openFile } = useFilePreview();

    // First open: content provided, no initialLine
    await openFile('/test/file.go', { fileContent: 'first content' });
    expect(previewState.value.initialLine).toBe(null);

    // Second open: same file, now with initialLine
    await openFile('/test/file.go', { initialLine: 30 });

    // Same file → content already present → initialLine set immediately.
    expect(previewState.value.fileContent).toBe('first content');
    expect(previewState.value.initialLine).toBe(30);
  });

  test('openFile different file resets initialLine to null until content loads', async () => {
    const { previewState, openFile } = useFilePreview();

    // First file with content + initialLine
    await openFile('/test/file1.go', { fileContent: 'content1', initialLine: 5 });
    expect(previewState.value.initialLine).toBe(5);

    // Switch to a different file with a new initialLine — content not provided.
    const { ReadFile } = await import('@wails/go/main/App');
    const { promise: readPromise, resolve: resolveRead } = Promise.withResolvers<string>();
    vi.mocked(ReadFile).mockReturnValueOnce(readPromise);

    const promise = openFile('/test/file2.go', { initialLine: 20 });

    // Before content loads: initialLine must be null (not 20).
    expect(previewState.value.initialLine).toBe(null);
    expect(previewState.value.fileContent).toBe('');

    resolveRead('content2');
    await promise;

    expect(previewState.value.fileContent).toBe('content2');
    expect(previewState.value.initialLine).toBe(20);
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
