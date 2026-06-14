import type {
  EditorAvailability,
  EditorDetectionStatus,
} from "../../src/types/search";
import { makeDefaultEditorAvailability } from "../../src/composables/useEditorDetection";

export const makeEditorAvailability = (
  overrides: Partial<EditorAvailability> = {},
): EditorAvailability => ({
  ...makeDefaultEditorAvailability(),
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