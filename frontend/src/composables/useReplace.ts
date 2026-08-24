import { ref } from "vue";
import { ReplaceInFiles as GoReplaceInFiles } from "@wails/go/main/App";
import { main } from "@wails/go/models";
import type { SearchState, ReplaceRequest, ReplaceResult } from "@/types";
import { toastManager } from "./useToast";
import { buildSearchRequest, toErrorMessage } from "@/utils";

// Build the replace request from the current search state. Reuses the exact
// same search parameters the user searched with, so replace operates on the
// same directory, filters, and case-sensitivity.
function buildReplaceRequest(data: SearchState, replacement: string, apply: boolean): ReplaceRequest {
  return {
    search: buildSearchRequest(data),
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
    // Separate in-flight flags per action so the UI can show a spinner on the
  // exact button doing the work (preview vs apply).
  const isPreviewing = ref(false);
  const isApplying = ref(false);
  // Snapshot of the replacement text at preview time. Apply uses this instead
  // of the live replacement.value so it replaces exactly what was previewed,
  // even if the user edits the field between preview and apply.
  let previewedReplacement = "";

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
        isPreviewing.value = true;
    try {
      previewedReplacement = replacement.value;
      const result = await GoReplaceInFiles(
        new main.ReplaceRequest(buildReplaceRequest(data, previewedReplacement, false)),
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
            isPreviewing.value = false;
    }
  };

  const applyReplace = async () => {
    if (guardRegex()) return;
    if (!preview.value || preview.value.filesChanged === 0) return;
        isApplying.value = true;
    try {
      const result = await GoReplaceInFiles(
        new main.ReplaceRequest(buildReplaceRequest(data, previewedReplacement, true)),
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
            isApplying.value = false;
    }
  };

  const clearPreview = () => {
    preview.value = null;
    previewedReplacement = "";
  };

  return {
    replacement,
    preview,
        isPreviewing,
    isApplying,
    previewReplace,
    applyReplace,
    clearPreview,
  };
}
