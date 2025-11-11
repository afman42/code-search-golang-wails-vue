import { toastManager } from "../composables/useToast";
// Utility functions for file operations and path formatting

/**
 * Formats a file path for display, truncating long paths
 * @param filePath - The full file path to format
 * @returns A formatted path string suitable for display
 */
export const formatFilePath = (filePath: string): string => {
  if (!filePath) return "";
  // Truncate long paths for better display
  if (filePath.length > 80) {
    const pathParts = filePath.split("/");
    if (pathParts.length > 5) {
      return "..." + pathParts.slice(-3).join("/");
    }
  }
  return filePath;
};

/**
 * Truncates a file path to show only the end portion
 * @param filePath - The full file path to truncate
 * @param maxLength - Maximum length of the truncated path (default 50)
 * @returns A truncated path string
 */
export const truncatePath = (
  filePath: string,
  maxLength: number = 50,
): string => {
  if (!filePath) return "";
  if (filePath.length <= maxLength) {
    return filePath;
  }
  return "..." + filePath.slice(-maxLength + 3); // +3 for the '...' prefix
};

// Handle editor selection and open file in selected editor
export const handleEditorSelect = async (event: Event, filePath: string) => {
  const target = event.target as HTMLSelectElement;
  const editor = target.value;

  // Reset the select to the placeholder option
  target.selectedIndex = 0;

  if (!editor) return; // If no editor selected, do nothing

  try {
    // Import the appropriate editor function based on selection
    if (filePath.endsWith(".log")) {
      const { ReadFileLog } = await import("../../wailsjs/go/main/App");
      filePath = await ReadFileLog(filePath);
    }
    switch (editor) {
      case "vscode":
        const { openInVSCode } = await import("./searchUiUtils");
        await openInVSCode(
          filePath,
          (text) => {
            toastManager.success(text, "VSCode Success");
          },
          (err) => {
            toastManager.error(err!, "VSCode Error");
          },
        );
        break;
      case "vscodium":
        const { openInVSCodium } = await import("./searchUiUtils");
        await openInVSCodium(
          filePath,
          (text) => {
            toastManager.success(text, "VSCodium Success");
          },
          (err) => {
            toastManager.error(err!, "VSCodium Error");
          },
        );
        break;
      case "sublime":
        const { openInSublime } = await import("./searchUiUtils");
        await openInSublime(
          filePath,
          (text) => {
            toastManager.success(text, "Sublime Success");
          },
          (err) => {
            toastManager.error(err!, "Sublime Error");
          },
        );
        break;
      case "atom":
        const { openInAtom } = await import("./searchUiUtils");
        await openInAtom(
          filePath,
          (text) => {
            toastManager.success(text, "Atom Success");
          },
          (err) => {
            toastManager.error(err!, "Atom Error");
          },
        );
        break;
      case "jetbrains":
        const { openInJetBrains } = await import("./searchUiUtils");
        await openInJetBrains(
          filePath,
          (text) => {
            toastManager.success(text, "Jetbrains Success");
          },
          (err) => {
            toastManager.error(err!, "Jetbrains Error");
          },
        );
        break;
      case "geany":
        const { openInGeany } = await import("./searchUiUtils");
        await openInGeany(
          filePath,
          (text) => {
            toastManager.success(text, "Geany Success");
          },
          (err) => {
            toastManager.error(err!, "Geany Error");
          },
        );
        break;

      case "goland":
        const { openInGoland } = await import("./searchUiUtils");
        await openInGoland(
          filePath,
          (text) => {
            toastManager.success(text, "Goland Success");
          },
          (err) => {
            toastManager.error(err!, "Goland Error");
          },
        );
        break;
      case "pycharm":
        const { openInPyCharm } = await import("./searchUiUtils");
        await openInPyCharm(
          filePath,
          (text) => {
            toastManager.success(text, "PyCharm Success");
          },
          (err) => {
            toastManager.error(err!, "PyCharm Error");
          },
        );
        break;
      case "intellij":
        const { openInIntelliJ } = await import("./searchUiUtils");
        await openInIntelliJ(
          filePath,
          (text) => {
            toastManager.success(text, "Intellij Success");
          },
          (err) => {
            toastManager.error(err!, "Intellij Error");
          },
        );
        break;
      case "webstorm":
        const { openInWebStorm } = await import("./searchUiUtils");
        await openInWebStorm(
          filePath,
          (text) => {
            toastManager.success(text, "Webstorm Success");
          },
          (err) => {
            toastManager.error(err!, "Webstorm Error");
          },
        );
        break;
      case "phpstorm":
        const { openInPhpStorm } = await import("./searchUiUtils");
        await openInPhpStorm(
          filePath,
          (text) => {
            toastManager.success(text, "Phpstorm Success");
          },
          (err) => {
            toastManager.error(err!, "Phpstorm Error");
          },
        );
        break;
      case "clion":
        const { openInCLion } = await import("./searchUiUtils");
        await openInCLion(
          filePath,
          (text) => {
            toastManager.success(text, "Clion Success");
          },
          (err) => {
            toastManager.error(err!, "Clion Error");
          },
        );
        break;
      case "rider":
        const { openInRider } = await import("./searchUiUtils");
        await openInRider(
          filePath,
          (text) => {
            toastManager.success(text, "Rider Success");
          },
          (err) => {
            toastManager.error(err!, "Rider Error");
          },
        );
        break;
      case "androidstudio":
        const { openInAndroidStudio } = await import("./searchUiUtils");
        await openInAndroidStudio(
          filePath,
          (text: string) => {
            toastManager.success(text, "Android Studio Success");
          },
          (err) => {
            toastManager.error(err!, "Android Studio Error");
          },
        );
        break;

      case "default":
        const { openInDefaultEditor } = await import("./searchUiUtils");
        await openInDefaultEditor(
          filePath,
          (text: string) => {
            toastManager.success(text, "Default Editor Success");
          },
          (err) => {
            toastManager.error(err!, "Default Editor Error");
          },
        );
        break;
      case "emacs":
        const { openInEmacs } = await import("./searchUiUtils");
        await openInEmacs(
          filePath,
          (text: string) => {
            toastManager.success(text, "Emacs Success");
          },
          (err) => {
            toastManager.error(err!, "Emacs Error");
          },
        );
        break;
      case "neovide":
        const { openInNeovide } = await import("./searchUiUtils");
        await openInNeovide(
          filePath,
          (text: string) => {
            toastManager.success(text, "Neovide Success");
          },
          (err) => {
            toastManager.error(err!, "Neovide Error");
          },
        );
        break;
      case "codeblocks":
        const { openInCodeBlocks } = await import("./searchUiUtils");
        await openInCodeBlocks(
          filePath,
          (text: string) => {
            toastManager.success(text, "Code Blocks Success");
          },
          (err) => {
            toastManager.error(err!, "Code Blocks Error");
          },
        );
        break;

      case "devcpp":
        const { openInDevCpp } = await import("./searchUiUtils");
        await openInDevCpp(
          filePath,
          (text: string) => {
            toastManager.success(text, "Dev-C++ Success");
          },
          (err) => {
            toastManager.error(err!, "Dev-C++ Error");
          },
        );
        break;
      case "notepadplusplus":
        const { openInNotepadPlusPlus } = await import("./searchUiUtils");
        await openInNotepadPlusPlus(
          filePath,
          (text: string) => {
            toastManager.success(text, "Notepad++ Success");
          },
          (err) => {
            toastManager.error(err!, "Notepad++ Error");
          },
        );
        break;
      case "visualstudio":
        const { openInVisualStudio } = await import("./searchUiUtils");
        await openInVisualStudio(
          filePath,
          (text: string) => {
            toastManager.success(text, "Visual Studio Success");
          },
          (err) => {
            toastManager.error(err!, "Visual Studio Error");
          },
        );
        break;
      case "eclipse":
        const { openInEclipse } = await import("./searchUiUtils");
        await openInEclipse(
          filePath,
          (text: string) => {
            toastManager.success(text, "Eclipse Success");
          },
          (err) => {
            toastManager.error(err!, "Eclipse Error");
          },
        );
        break;
      case "netbeans":
        const { openInNetBeans } = await import("./searchUiUtils");
        await openInNetBeans(
          filePath,
          (text: string) => {
            toastManager.success(text, "Netbeans Success");
          },
          (err) => {
            toastManager.error(err!, "Netbeans Error");
          },
        );
        break;
      default:
        toastManager.error(`Unknown editor: ${editor}`, "Editor Error");
    }
  } catch (error: any) {
    console.error(`Failed to open file in ${editor}:`, error);
    const errorMessage = error.message || "Unknown error";
    toastManager.error(
      `Could not open file in ${editor}: ${errorMessage}`,
      `${editor} Error`,
    );
  }
};
