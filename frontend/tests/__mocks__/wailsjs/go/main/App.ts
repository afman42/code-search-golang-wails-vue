// Mock for the Wails-generated App bindings used in tests.
//
// This mock mirrors the real Wails bindings in wailsjs/go/main/App.js. Every
// method exported by the backend's App struct that the frontend can call via
// Wails has a corresponding vi.fn() here, so tests that exercise any backend
// method don't hit a "function not found" error.
//
// The mock includes:
//   - The generic OpenInEditorByName dispatcher (the frontend's primary path)
//   - OpenInDefaultEditor (the "default" editor key's special case)
//   - ReadFileLog (used by fileUtils.ts for .log files)
//   - All individual OpenIn* methods (for backward compatibility — older
//     tests or code paths may still reference them directly)
import { vi } from "vitest";

// Core search and file methods
export const SearchCode = vi.fn();
export const GetDirectoryContents = vi.fn();
export const SelectDirectory = vi.fn();
export const ShowInFolder = vi.fn();
export const SearchWithProgress = vi.fn().mockResolvedValue([]);
export const CancelSearch = vi.fn();
export const ReadFile = vi.fn();
export const ReadFileLog = vi.fn();
export const ValidateDirectory = vi.fn();
export const GetEditorDetectionStatus = vi.fn();
export const GetAvailableEditors = vi.fn();
// Sample known-text extension list returned by the backend. The real
// binding returns the full ~150-entry sorted list from text_extensions.go;
// this subset is enough for SearchForm dropdown rendering tests.
export const GetKnownTextExtensions = vi.fn().mockResolvedValue([
  "bat", "c", "coffee", "cpp", "cs", "css", "dart", "go", "h", "hpp",
  "html", "java", "js", "json", "jsx", "kt", "lua", "md", "php", "pl",
  "py", "r", "rb", "rs", "rust", "scala", "sh", "sql", "swift", "toml",
  "ts", "tsx", "txt", "vue", "xml", "yaml", "yml",
]);
export const GetInitialLogs = vi.fn().mockResolvedValue([]);
export const GetNewLogs = vi.fn().mockResolvedValue([]);
export const IsAppReady = vi.fn().mockResolvedValue(true);

// Generic editor dispatcher — the frontend's primary path for opening files
// in named editors. Calls the backend's OpenInEditorByName(name, filePath)
// which looks up the editor command in the editorBindings map.
export const OpenInEditorByName = vi.fn().mockResolvedValue(undefined);

// Default editor — special case for the "default" editor key. Calls the
// OS default (xdg-open on Linux, explorer on Windows) rather than a named
// editor. NOT part of the editorBindings map.
export const OpenInDefaultEditor = vi.fn().mockResolvedValue(undefined);

// Individual OpenIn* methods — kept for backward compatibility. The
// frontend now routes through OpenInEditorByName, but these stubs ensure
// any test that still imports them directly doesn't fail.
export const OpenInVSCode = vi.fn().mockResolvedValue(undefined);
export const OpenInVSCodium = vi.fn().mockResolvedValue(undefined);
export const OpenInSublime = vi.fn().mockResolvedValue(undefined);
export const OpenInAtom = vi.fn().mockResolvedValue(undefined);
export const OpenInJetBrains = vi.fn().mockResolvedValue(undefined);
export const OpenInGeany = vi.fn().mockResolvedValue(undefined);
export const OpenInGoland = vi.fn().mockResolvedValue(undefined);
export const OpenInPyCharm = vi.fn().mockResolvedValue(undefined);
export const OpenInIntelliJ = vi.fn().mockResolvedValue(undefined);
export const OpenInWebStorm = vi.fn().mockResolvedValue(undefined);
export const OpenInPhpStorm = vi.fn().mockResolvedValue(undefined);
export const OpenInCLion = vi.fn().mockResolvedValue(undefined);
export const OpenInRider = vi.fn().mockResolvedValue(undefined);
export const OpenInAndroidStudio = vi.fn().mockResolvedValue(undefined);
export const OpenInEmacs = vi.fn().mockResolvedValue(undefined);
export const OpenInNeovide = vi.fn().mockResolvedValue(undefined);
export const OpenInCodeBlocks = vi.fn().mockResolvedValue(undefined);
export const OpenInDevCpp = vi.fn().mockResolvedValue(undefined);
export const OpenInNotepadPlusPlus = vi.fn().mockResolvedValue(undefined);
export const OpenInVisualStudio = vi.fn().mockResolvedValue(undefined);
export const OpenInEclipse = vi.fn().mockResolvedValue(undefined);
export const OpenInNetBeans = vi.fn().mockResolvedValue(undefined);
export const OpenInNeovim = vi.fn().mockResolvedValue(undefined);
export const OpenInVim = vi.fn().mockResolvedValue(undefined);
