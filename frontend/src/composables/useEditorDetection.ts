import { type EditorAvailability, type EditorDetectionStatus } from "../types/search";
import { EventsOn } from "../../wailsjs/runtime";

export function makeDefaultEditorAvailability(): EditorAvailability {
  return {
    vscode: false,
    vscodium: false,
    sublime: false,
    atom: false,
    jetbrains: false,
    geany: false,
    neovim: false,
    vim: false,
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
  };
}

export function makeDefaultEditorDetectionStatus(): EditorDetectionStatus {
  return {
    detectionComplete: false,
    totalAvailable: 0,
    message: "Initializing editor detection...",
    detectionProgress: 0,
    detectingEditors: true,
    detectedEditors: [],
    availableEditors: makeDefaultEditorAvailability(),
  };
}

export function subscribeToEditorDetectionEvents(
  availableEditors: EditorAvailability,
  editorDetectionStatus: EditorDetectionStatus,
): () => void {
  const cleanupStart = EventsOn(
    "editor-detection-start",
    (eventData: any) => {
      editorDetectionStatus.detectionComplete = false;
      editorDetectionStatus.totalAvailable = 0;
      editorDetectionStatus.message = eventData?.message || "Starting editor detection...";
      editorDetectionStatus.detectionProgress = 0;
      editorDetectionStatus.detectingEditors = true;
      editorDetectionStatus.detectedEditors = [];
      editorDetectionStatus.availableEditors = availableEditors;
    },
  );

  const cleanupProgress = EventsOn(
    "editor-detection-progress",
    (eventData: any) => {
      if (eventData) {
        editorDetectionStatus.message =
          eventData.message || "Detecting editors...";
        editorDetectionStatus.detectionProgress =
          Math.round(eventData.progress) || 0;

        if (eventData.available && eventData.editor) {
          if (
            !editorDetectionStatus.detectedEditors.includes(
              eventData.editor,
            )
          ) {
            editorDetectionStatus.detectedEditors.push(
              eventData.editor,
            );
          }
        }
      }
    },
  );

  const cleanupComplete = EventsOn(
    "editor-detection-complete",
    (eventData: any) => {
      editorDetectionStatus.detectionComplete = true;
      editorDetectionStatus.totalAvailable =
        eventData?.totalFound || 0;
      editorDetectionStatus.message = `Detection complete! Found ${eventData?.totalFound || 0} editor(s).`;
      editorDetectionStatus.detectionProgress = 100;
      editorDetectionStatus.detectingEditors = false;
    },
  );

  return () => {
    if (cleanupStart) cleanupStart();
    if (cleanupProgress) cleanupProgress();
    if (cleanupComplete) cleanupComplete();
  };
}