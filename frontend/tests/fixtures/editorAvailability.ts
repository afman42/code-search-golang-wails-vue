import type {
  EditorAvailability,
  EditorDetectionStatus,
} from "@/types/search";
import { makeDefaultEditorAvailability } from "@/composables/useEditorDetection";

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