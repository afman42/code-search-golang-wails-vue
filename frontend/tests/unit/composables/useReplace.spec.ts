import { describe, it, expect, vi, beforeEach } from "vitest";
import { useReplace } from "@/composables";
import type { SearchState, ReplaceResult, EditorAvailability, EditorDetectionStatus } from "@/types";
import * as AppModule from "@wails/go/main/App";
import { toastManager } from "@/composables";

function makeEditors(): EditorAvailability {
  return {
    vscode: false, vscodium: false, sublime: false, jetbrains: false,
    geany: false, neovim: false, vim: false, goland: false,
    pycharm: false, intellij: false, webstorm: false, phpstorm: false,
    clion: false, rider: false, androidstudio: false, systemdefault: false,
    emacs: false, neovide: false, codeblocks: false, devcpp: false,
    notepadplusplus: false, visualstudio: false, eclipse: false,
    netbeans: false,
  };
}

function makeState(overrides: Partial<SearchState> = {}): SearchState {
  return {
    directory: "/project",
    query: "hello",
    extension: "",
    caseSensitive: false,
    useRegex: false,
    includeBinary: false,
    maxFileSize: 10485760,
    maxResults: 1000,
    resultText: "",
    searchResults: [],
    truncatedResults: false,
    isSearching: false,
    searchProgress: {
      processedFiles: 0,
      totalFiles: 0,
      currentFile: "",
      resultsCount: 0,
      status: "",
    },
    showProgress: false,
    minFileSize: 0,
    excludePatterns: [],
    allowedFileTypes: [],
    knownTextExtensions: [],
    recentSearches: [],
    error: null,
    availableEditors: makeEditors(),
    editorDetectionStatus: {} as EditorDetectionStatus,
    fuzzySearch: false,
    contextLines: 3,
    directories: [],
    respectGitignore: false,
    ...overrides,
  };
}

const sampleResult: ReplaceResult = {
  files: [
    {
      filePath: "/project/a.txt",
      lineNum: 1,
      oldLine: "hello world",
      newLine: "goodbye world",
    },
  ],
  filesChanged: 1,
  linesChanged: 1,
};

describe("useReplace composable", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    (AppModule.ReplaceInFiles as ReturnType<typeof vi.fn>).mockResolvedValue(sampleResult);
  });

  it("preview calls binding with apply=false and stores preview", async () => {
    const state = makeState();
    const onSearch = vi.fn();
    const { replacement, preview, previewReplace } = useReplace(state, onSearch);
    replacement.value = "goodbye";

    await previewReplace();

    expect(AppModule.ReplaceInFiles).toHaveBeenCalledTimes(1);
    const req = (AppModule.ReplaceInFiles as ReturnType<typeof vi.fn>).mock.calls[0][0];
    expect(req.apply).toBe(false);
    expect(req.replacement).toBe("goodbye");
    expect(req.search.query).toBe("hello");
    expect(req.search.respectGitignore).toBe(false);
    expect(preview.value).toEqual(sampleResult);
    expect(onSearch).not.toHaveBeenCalled();
  });

  it("apply calls binding with apply=true then re-runs search", async () => {
    const state = makeState();
    const onSearch = vi.fn().mockResolvedValue(undefined);
    const { replacement, preview, previewReplace, applyReplace } = useReplace(state, onSearch);
    replacement.value = "goodbye";

    await previewReplace();
    expect(preview.value).not.toBeNull();

    await applyReplace();

    const req = (AppModule.ReplaceInFiles as ReturnType<typeof vi.fn>).mock.calls[1][0];
    expect(req.apply).toBe(true);
    expect(onSearch).toHaveBeenCalledTimes(1);
    expect(preview.value).toBeNull();
  });

  it("does nothing under regex mode", async () => {
    const state = makeState({ useRegex: true });
    const toastSpy = vi.spyOn(toastManager, "error");
    const onSearch = vi.fn();
    const { replacement, previewReplace, applyReplace } = useReplace(state, onSearch);
    replacement.value = "x";

    await previewReplace();
    await applyReplace();

    expect(AppModule.ReplaceInFiles).not.toHaveBeenCalled();
    expect(onSearch).not.toHaveBeenCalled();
    expect(toastSpy).toHaveBeenCalled();
  });

  it("apply without preview is a no-op", async () => {
    const state = makeState();
    const onSearch = vi.fn();
    const { applyReplace } = useReplace(state, onSearch);

    await applyReplace();

    expect(AppModule.ReplaceInFiles).not.toHaveBeenCalled();
    expect(onSearch).not.toHaveBeenCalled();
  });

  it("preview with zero changes stores preview and shows info toast", async () => {
    (AppModule.ReplaceInFiles as ReturnType<typeof vi.fn>).mockResolvedValue({
      files: [],
      filesChanged: 0,
      linesChanged: 0,
    });
    const infoSpy = vi.spyOn(toastManager, "info");
    const state = makeState();
    const { replacement, preview, previewReplace } = useReplace(state, vi.fn());
    replacement.value = "hello"; // same as query → no-op

    await previewReplace();

    expect(preview.value).toEqual({ files: [], filesChanged: 0, linesChanged: 0 });
    expect(infoSpy).toHaveBeenCalled();
  });

  it("preview requires a query", async () => {
    const state = makeState({ query: "" });
    const toastSpy = vi.spyOn(toastManager, "error");
    const { replacement, previewReplace } = useReplace(state, vi.fn());
    replacement.value = "x";

    await previewReplace();

    expect(AppModule.ReplaceInFiles).not.toHaveBeenCalled();
    expect(toastSpy).toHaveBeenCalled();
  });
});
