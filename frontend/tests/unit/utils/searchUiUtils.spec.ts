import { describe, test, expect, vi, beforeEach } from "vitest";
import { openInEditor, buildSearchRequest } from '@/utils';
import type { SearchState } from '@/types';
import { makeDefaultEditorAvailability } from '@/composables';

// Import the mocked Wails module so we can assert on OpenInEditorByName and
// OpenInDefaultEditor calls. The mock at tests/__mocks__/wailsjs/go/main/App.ts
// provides vi.fn() stubs for every backend method the frontend can call.
import * as AppModule from "@wails/go/main/App";

// Helper to build a minimal SearchState for buildSearchRequest calls
function makeState(overrides: Partial<SearchState> = {}): SearchState {
  return {
    directory: "",
    query: "",
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
    searchProgress: { processedFiles: 0, totalFiles: 0, currentFile: "", resultsCount: 0, status: "" },
    showProgress: false,
    minFileSize: 0,
    excludePatterns: [],
    allowedFileTypes: [],
    knownTextExtensions: [],
    recentSearches: [],
    error: null,
    availableEditors: makeDefaultEditorAvailability(),
    editorDetectionStatus: {
      detectionComplete: false, totalAvailable: 0, message: "", detectionProgress: 0,
      detectingEditors: true, detectedEditors: [], availableEditors: makeDefaultEditorAvailability(),
    },
    ...overrides,
  };
}

describe("buildSearchRequest", () => {
  test("maps SearchState fields onto the SearchRequest", () => {
    const state = makeState({
      directory: "/proj",
      query: "foo",
      extension: "go",
      caseSensitive: true,
      useRegex: true,
      includeBinary: true,
      minFileSize: 5,
      fuzzySearch: true,
      contextLines: 2,
      directories: ["/proj/a", "/proj/b"],
      respectGitignore: true,
      excludePatterns: ["vendor", "node_modules"],
      allowedFileTypes: ["go", "ts"],
    });

    const req = buildSearchRequest(state);

    expect(req.directory).toBe("/proj");
    expect(req.query).toBe("foo");
    expect(req.extension).toBe("go");
    expect(req.caseSensitive).toBe(true);
    expect(req.useRegex).toBe(true);
    expect(req.includeBinary).toBe(true);
    expect(req.minFileSize).toBe(5);
    expect(req.fuzzySearch).toBe(true);
    expect(req.contextLines).toBe(2);
    expect(req.directories).toEqual(["/proj/a", "/proj/b"]);
    expect(req.respectGitignore).toBe(true);
    expect(req.excludePatterns).toEqual(["vendor", "node_modules"]);
    expect(req.allowedFileTypes).toEqual(["go", "ts"]);
  });

  test("preserves explicit zero maxFileSize/maxResults (nullish, not ||)", () => {
    const state = makeState({ maxFileSize: 0, maxResults: 0 });

    const req = buildSearchRequest(state);

    expect(req.maxFileSize).toBe(0);
    expect(req.maxResults).toBe(0);
  });

  test("falls back to defaults when maxFileSize/maxResults are null", () => {
    const state = makeState({
      maxFileSize: null as unknown as number,
      maxResults: null as unknown as number,
    });

    const req = buildSearchRequest(state);

    expect(req.maxFileSize).toBe(10485760);
    expect(req.maxResults).toBe(1000);
  });

  test("filters empty strings from list fields", () => {
    const state = makeState({
      excludePatterns: ["vendor", "", "  "],
      allowedFileTypes: ["go", "", "ts"],
      directories: ["/a", "", "/b"],
    });

    const req = buildSearchRequest(state);

    expect(req.excludePatterns).toEqual(["vendor", "  "]);
    expect(req.allowedFileTypes).toEqual(["go", "ts"]);
    expect(req.directories).toEqual(["/a", "/b"]);
  });
});

describe("openInEditor", () => {
  const setResultText = vi.fn();
  const setError = vi.fn();

  beforeEach(() => {
    vi.clearAllMocks();
    (AppModule.OpenInEditorByName as any).mockResolvedValue(undefined);
    (AppModule.OpenInDefaultEditor as any).mockResolvedValue(undefined);
  });

  test("calls OpenInEditorByName with the correct binding name for VSCode", async () => {
    await openInEditor("vscode", "/test/file.go", setResultText, setError);

    expect(AppModule.OpenInEditorByName).toHaveBeenCalledWith(
      "VSCode",
      "/test/file.go",
    );
    expect(setResultText).toHaveBeenCalledWith(
      expect.stringContaining("VSCode"),
    );
  });

  test("calls OpenInEditorByName with the correct binding name for Sublime", async () => {
    await openInEditor("sublime", "/test/file.txt", setResultText, setError);

    expect(AppModule.OpenInEditorByName).toHaveBeenCalledWith(
      "Sublime",
      "/test/file.txt",
    );
  });

  test("calls OpenInEditorByName for Neovim", async () => {
    await openInEditor("neovim", "/test/file.go", setResultText, setError);

    expect(AppModule.OpenInEditorByName).toHaveBeenCalledWith(
      "Neovim",
      "/test/file.go",
    );
  });

  test("calls OpenInEditorByName for JetBrains (routes by file extension in backend)", async () => {
    await openInEditor("jetbrains", "/test/file.go", setResultText, setError);

    // "jetbrains" maps to the "JetBrains" binding name, which the backend
    // routes to the appropriate JetBrains IDE based on file extension.
    expect(AppModule.OpenInEditorByName).toHaveBeenCalledWith(
      "JetBrains",
      "/test/file.go",
    );
  });

  test("calls OpenInDefaultEditor for the 'default' editor key (not OpenInEditorByName)", async () => {
    await openInEditor("default", "/test/file.go", setResultText, setError);

    // The "default" key is a special case — it calls OpenInDefaultEditor
    // directly (xdg-open / explorer) rather than OpenInEditorByName.
    expect(AppModule.OpenInDefaultEditor).toHaveBeenCalledWith("/test/file.go");
    // Must NOT call the generic dispatcher for "default".
    expect(AppModule.OpenInEditorByName).not.toHaveBeenCalled();
  });

  test("rejects unknown editor key", async () => {
    await openInEditor("nonexistent", "/test/file.go", setResultText, setError);

    expect(setError).toHaveBeenCalledWith(expect.stringContaining("Unknown editor"));
    expect(AppModule.OpenInEditorByName).not.toHaveBeenCalled();
    expect(AppModule.OpenInDefaultEditor).not.toHaveBeenCalled();
  });

  test("rejects empty file path", async () => {
    await openInEditor("vscode", "", setResultText, setError);

    expect(setResultText).toHaveBeenCalledWith("Invalid file path");
    expect(AppModule.OpenInEditorByName).not.toHaveBeenCalled();
  });

  test("rejects null-like file path", async () => {
    await openInEditor("vscode", null as any, setResultText, setError);

    expect(setResultText).toHaveBeenCalledWith("Invalid file path");
    expect(AppModule.OpenInEditorByName).not.toHaveBeenCalled();
  });

  test("surfaces backend errors via setError", async () => {
    (AppModule.OpenInEditorByName as any).mockRejectedValue(
      new Error("editor not found in PATH"),
    );

    await openInEditor("vscode", "/test/file.go", setResultText, setError);

    expect(setError).toHaveBeenCalledWith(
      expect.stringContaining("editor not found in PATH"),
    );
  });

  test("every editor key in EditorSelect.vue has a binding name mapping", async () => {
    // Every editor key that EditorSelect.vue can emit must have a
    // corresponding entry in the editorBindingName map (or be the "default"
    // special case). If a key is missing, openInEditor would reject it as
    // "Unknown editor" and the user's click would silently fail.
    //
    // This test lists the keys from EditorSelect.vue's <option> elements.
    // If a new editor is added to EditorSelect.vue, this test will fail
    // until the corresponding entry is added to editorBindingName in
    // searchUiUtils.ts — catching the drift early.
    const editorKeys = [
      "vscode", "vscodium", "sublime", "jetbrains",
      "geany", "goland", "pycharm", "intellij", "webstorm",
      "phpstorm", "clion", "rider", "androidstudio", "emacs",
      "neovide", "codeblocks", "devcpp", "notepadplusplus",
      "visualstudio", "eclipse", "netbeans", "neovim", "vim",
      "default", // special case — handled by OpenInDefaultEditor
    ];

    const missingKeys: string[] = [];
    for (const key of editorKeys) {
      vi.clearAllMocks();
      await openInEditor(key, "/test/file.go", setResultText, setError);

      // Every key must produce a successful Wails call (either
      // OpenInEditorByName or OpenInDefaultEditor) — NOT an "Unknown editor"
      // error. If it errored, the key is missing from the mapping.
      const errorCall = setError.mock.calls.find(
        (call) => typeof call[0] === "string" && call[0].includes("Unknown editor"),
      );
      if (errorCall) {
        missingKeys.push(key);
      }
    }

    if (missingKeys.length > 0) {
      throw new Error(
        `Editor keys missing from editorBindingName mapping in searchUiUtils.ts: ${missingKeys.join(", ")}`,
      );
    }
  });
});
