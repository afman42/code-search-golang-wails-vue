import type {
  EditorAvailability,
  EditorDetectionStatus,
} from "../../src/types/search";

// Builds a fully-populated EditorAvailability object with every editor flag set
// to false (override individual flags as needed). The components iterate over
// these fields with v-if, so the object must contain every key from the type.
export const makeEditorAvailability = (
  overrides: Partial<EditorAvailability> = {},
): EditorAvailability => ({
  vscode: false,
  vscodium: false,
  sublime: false,
  atom: false,
  jetbrains: false,
  geany: false,
  goland: false,
  pycharm: false,
  intellij: false,
  webstorm: false,
  phpstorm: false,
  clion: false,
  rider: false,
  androidstudio: false,
  systemdefault: true,
  emacs: false,
  neovide: false,
  codeblocks: false,
  devcpp: false,
  notepadplusplus: false,
  visualstudio: false,
  eclipse: false,
  netbeans: false,
  ...overrides,
});

export const makeEditorDetectionStatus = (
  overrides: Partial<EditorDetectionStatus> = {},
): EditorDetectionStatus => ({
  detectionComplete: true,
  totalAvailable: 0,
  message: "",
  detectionProgress: 100,
  detectingEditors: false,
  detectedEditors: [],
  availableEditors: makeEditorAvailability(),
  ...overrides,
});
