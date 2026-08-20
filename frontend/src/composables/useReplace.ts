import { ref } from "vue";
import { ReplaceInFiles as GoReplaceInFiles } from "@wails/go/main/App";
import type { main } from "@wails/go/models";
import type { SearchState, ReplaceRequest, ReplaceResult } from "@/types";
import { toastManager } from "./useToast";
import { toErrorMessage } from "@/utils";

// Build the replace request from the current search state. Reuses the exact
// same search parameters the user searched with, so replace operates on the
// same directory, filters, and case-sensitivity.
function buildReplaceRequest(data: SearchState, replacement: string, apply: boolean): ReplaceRequest {
  return {
    search: {
      directory: data.directory,
      query: data.query,
      extension: data.extension,
      caseSensitive: data.caseSensitive,
      includeBinary: data.includeBinary,
      maxFileSize: Number(data.maxFileSize) || 10485760,
      minFileSize: Number(data.minFileSize) || 0,
      maxResults: Number(data.maxResults) || 1000,
      useRegex: data.useRegex,
      excludePatterns: Array.isArray(data.excludePatterns)
        ? data.excludePatterns.filter((s) => s.length > 0)
        : [],
      allowedFileTypes: Array.isArray(data.allowedFileTypes)
        ? data.allowedFileTypes.filter((s) => s.length > 0)
        : [],
      fuzzySearch: data.fuzzySearch,
      contextLines: data.contextLines,
      directories: Array.isArray(data.directories)
        ? data.directories.filter((s) => s.length > 0)
        : [],
      respectGitignore: data.respectGitignore,
    },
    replacement,
    apply,
  };
}

// useReplace drives the Find & Replace flow: a literal replacement string, a
// dry-run preview (Apply=false — nothing written), and an explicit apply that
// re-runs the search afterward so results reflect the new file contents.
// Both actions are no-ops with a toast when regex mode is on (the backend
// rejects regex replace; the UI just prevents the round-trip).
export function useReplace(
  data: SearchState,
  onSearch: () => Promise<void>,
) {
  const replacement = ref("");
  const preview = ref<ReplaceResult | null>(null);
  const isReplacing = ref(false);

  const guardRegex = (): boolean => {
    if (data.useRegex) {
      toastManager.error(
        "Replace is disabled while regex search is on",
        "Replace Disabled",
      );
      return true;
    }
    return false;
  };

  const previewReplace = async () => {
    if (guardRegex()) return;
    if (!data.query) {
      toastManager.error("Search for something before replacing", "Replace");
      return;
    }
    isReplacing.value = true;
    try {
      const result = await GoReplaceInFiles(
        buildReplaceRequest(data, replacement.value, false) as unknown as main.ReplaceRequest,
      );
      preview.value = result;
      if (result.filesChanged === 0) {
        toastManager.info(
          "No changes needed — matches already contain the replacement text",
          "Replace Preview",
        );
      } else {
        toastManager.success(
          `Preview: ${result.linesChanged} changes in ${result.filesChanged} files`,
          "Replace Preview",
        );
      }
    } catch (error: unknown) {
      toastManager.error(toErrorMessage(error), "Replace Preview Failed");
    } finally {
      isReplacing.value = false;
    }
  };

  const applyReplace = async () => {
    if (guardRegex()) return;
    if (!preview.value || preview.value.filesChanged === 0) return;
    isReplacing.value = true;
    try {
      const result = await GoReplaceInFiles(
        buildReplaceRequest(data, replacement.value, true) as unknown as main.ReplaceRequest,
      );
      toastManager.success(
        `Replaced ${result.linesChanged} lines in ${result.filesChanged} files`,
        "Replace Applied",
      );
      preview.value = null;
      // Re-run the search so the result list reflects the changed files.
      await onSearch();
    } catch (error: unknown) {
      toastManager.error(toErrorMessage(error), "Replace Failed");
    } finally {
      isReplacing.value = false;
    }
  };

  const clearPreview = () => {
    preview.value = null;
  };

  return {
    replacement,
    preview,
    isReplacing,
    previewReplace,
    applyReplace,
    clearPreview,
  };
}
