import type { EditorAvailability, EditorDetectionStatus } from "@/types";
import { EventsOn } from "@wails/runtime";
import { GetEditorDetectionStatus } from "@wails/go/main/App";
import { asRecord } from "@/utils";

export function makeDefaultEditorAvailability(): EditorAvailability {
  return {
    vscode: false,
    vscodium: false,
    sublime: false,
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
    (payload: unknown) => {
      const e = asRecord(payload);
      editorDetectionStatus.detectionComplete = false;
      editorDetectionStatus.totalAvailable = 0;
      editorDetectionStatus.message =
        typeof e.message === "string" ? e.message : "Starting editor detection...";
      editorDetectionStatus.detectionProgress = 0;
      editorDetectionStatus.detectingEditors = true;
      editorDetectionStatus.detectedEditors = [];
      editorDetectionStatus.availableEditors = availableEditors;
    },
  );

  const cleanupProgress = EventsOn(
    "editor-detection-progress",
    (payload: unknown) => {
      const e = asRecord(payload);
      editorDetectionStatus.message =
        typeof e.message === "string" ? e.message : "Detecting editors...";
      editorDetectionStatus.detectionProgress =
        typeof e.progress === "number" ? Math.round(e.progress) : 0;

      if (e.available && typeof e.editor === "string") {
        if (!editorDetectionStatus.detectedEditors.includes(e.editor)) {
          editorDetectionStatus.detectedEditors.push(e.editor);
        }
      }
    },
  );

  const cleanupComplete = EventsOn(
    "editor-detection-complete",
    (payload: unknown) => {
      const e = asRecord(payload);
      const totalFound = typeof e.totalFound === "number" ? e.totalFound : 0;
      editorDetectionStatus.detectionComplete = true;
      editorDetectionStatus.totalAvailable = totalFound;
      editorDetectionStatus.message = `Detection complete! Found ${totalFound} editor(s).`;
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

// startEditorDetection wires up the full editor-detection lifecycle for a
// reactive EditorAvailability + EditorDetectionStatus pair: it subscribes to
// the live start/progress/complete events AND pulls the current status once so
// a detection that already completed before subscription is still reflected at
// first paint (#15). Returns a cleanup that releases the event subscriptions.
//
// Extracted from useSearch so search state and editor detection are separate
// concerns; useSearch just owns the reactive fields and delegates here.
export function startEditorDetection(
  availableEditors: EditorAvailability,
  editorDetectionStatus: EditorDetectionStatus,
): () => void {
  const cleanup = subscribeToEditorDetectionEvents(availableEditors, editorDetectionStatus);

  // Pull-based status check closes the race where "editor-detection-complete"
  // fired before the listeners above registered. Non-fatal on failure.
  void (async () => {
    try {
      const status = await GetEditorDetectionStatus();
      if (status) {
        if (status.availableEditors) {
          Object.assign(availableEditors, status.availableEditors);
          editorDetectionStatus.availableEditors = availableEditors;
        }
        editorDetectionStatus.totalAvailable = status.totalAvailable || 0;
        if (status.totalAvailable !== undefined) {
          editorDetectionStatus.message = `Detection complete! Found ${status.totalAvailable} editor(s).`;
        }
        editorDetectionStatus.detectionComplete = true;
        editorDetectionStatus.detectingEditors = false;
      }
    } catch (error: unknown) {
      console.error("Failed to fetch editor detection status:", error);
    }
  })();

  return cleanup;
}