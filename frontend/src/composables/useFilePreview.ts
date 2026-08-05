// Singleton file-preview state. Any component can call openFile() to pop the
// code preview modal — used by SymbolSearch to navigate to a symbol's
// file:line, and by SearchResults for the existing "View" button.
//
// Keeping this as a module-level singleton (like toastManager) means the
// CodeModal mounted in CodeSearch.vue is the single source of truth for the
// preview, regardless of which component triggered it.

import { ref } from "vue";
import { toErrorMessage } from "../utils/errorUtils";

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
  const { ReadFile } = await import("../../wailsjs/go/main/App");
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
    state.value = {
      isVisible: true,
      filePath,
      fileContent: options?.fileContent ?? "",
      query: options?.query ?? "",
      files: options?.files ?? [],
      initialLine: options?.initialLine ?? null,
    };

    // If content wasn't passed, load it from the backend.
    if (!options?.fileContent) {
      loadingPromise = loadFileContent(filePath)
        .then((content) => {
          state.value.fileContent = content;
        })
        .catch((err: unknown) => {
          import("./useToast").then(({ toastManager }) => {
            toastManager.error(
              toErrorMessage(err, "Could not open file"),
              "File Preview Error",
            );
          });
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
