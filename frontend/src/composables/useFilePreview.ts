// Singleton file-preview state. Any component can call openFile() to pop the
// code preview modal — used by SymbolSearch to navigate to a symbol's
// file:line, and by SearchResults for the existing "View" button.
//
// Keeping this as a module-level singleton (like toastManager) means the
// CodeModal mounted in CodeSearch.vue is the single source of truth for the
// preview, regardless of which component triggered it.

import { ref } from "vue";
import { toErrorMessage } from "@/utils";
import { ReadFile } from "@wails/go/main/App";
import { toastManager } from "./useToast";

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
    const initialLine = options?.initialLine ?? null;
    // Content is available right now if the caller passed it OR this is the
    // same file we already have loaded. In that case initialLine can be set
    // immediately — CodeModal's watcher will see content + line together.
    const hasContentNow = isSameFile || !!options?.fileContent;

    state.value = {
      isVisible: true,
      filePath,
      fileContent: isSameFile
        ? state.value.fileContent
        : (options?.fileContent ?? ""),
      query: options?.query ?? "",
      files: options?.files ?? [],
      initialLine: hasContentNow ? initialLine : null,
    };

    if (!options?.fileContent && !isSameFile) {
      // New file, no content provided — load from the backend. Only after
      // content is committed to state do we arm initialLine, so CodeModal's
      // watcher sees both in the same tick.
      loadingPromise = loadFileContent(filePath)
        .then((content) => {
          state.value.fileContent = content;
          if (initialLine) {
            state.value = { ...state.value, initialLine };
          }
        })
        .catch((err: unknown) => {
          toastManager.error(
            toErrorMessage(err, "Could not open file"),
            "File Preview Error",
          );
          closePreview();
        });
    } else if (initialLine && isSameFile) {
      // Same file already has content — re-arm initialLine with a new object
      // so Vue's reactivity detects the change and the watcher re-fires.
      state.value = { ...state.value, initialLine };
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
