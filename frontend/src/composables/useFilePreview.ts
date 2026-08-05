// Singleton file-preview state. Any component can call openFile() to pop the
// code preview modal — used by SymbolSearch to navigate to a symbol's
// file:line, and by SearchResults for the existing "View" button.
//
// Keeping this as a module-level singleton (like toastManager) means the
// CodeModal mounted in CodeSearch.vue is the single source of truth for the
// preview, regardless of which component triggered it.

import { ref } from "vue";
import { toErrorMessage } from "@/utils/errorUtils";
import { ReadFile } from "@wails/go/main/App";
import { toastManager } from "@/composables/useToast";

export interface FilePreviewState {
  isVisible: boolean;
  filePath: string;
  fileContent: string;
  query: string;
  files: string[];
  initialLine: number | null;
}

const state = ref<FilePreviewState>({
  isVisible: false,
  filePath: "",
  fileContent: "",
  query: "",
  files: [],
  initialLine: null,
});

let loadingPromise: Promise<void> | null = null;

async function loadFileContent(filePath: string): Promise<string> {
  return await ReadFile(filePath);
}

export function useFilePreview() {
  function openFile(
    filePath: string,
    options?: {
      fileContent?: string;
      query?: string;
      files?: string[];
      initialLine?: number;
    },
  ): Promise<void> {
    const isSameFile = state.value.filePath === filePath;

    state.value = {
      isVisible: true,
      filePath,
      // Keep old content if opening the same file (avoids flash + re-highlight).
      fileContent: isSameFile
        ? state.value.fileContent
        : (options?.fileContent ?? ""),
      query: options?.query ?? "",
      files: options?.files ?? [],
      initialLine: options?.initialLine ?? null,
    };

    // If content wasn't passed AND it's a new file, load it from the backend.
    if (!options?.fileContent && !isSameFile) {
      loadingPromise = loadFileContent(filePath)
        .then((content) => {
          state.value.fileContent = content;
        })
        .catch((err: unknown) => {
          toastManager.error(
            toErrorMessage(err, "Could not open file"),
            "File Preview Error",
          );
          closePreview();
        });
    }

    return loadingPromise ?? Promise.resolve();
  }

  function closePreview() {
    state.value = {
      ...state.value,
      isVisible: false,
      initialLine: null,
    };
  }

  return {
    previewState: state,
    openFile,
    closePreview,
  };
}
