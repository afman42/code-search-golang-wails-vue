/**
 * Single source of truth for the frontend editor picker.
 *
 * Mirrors the Go backend's editorCatalog (system_integration.go): each row
 * drives one <option> in EditorSelect.vue, one entry in the binding-name
 * lookup used by OpenInEditorByName, and one availability flag read from
 * GetAvailableEditors. Adding an editor = one row here + one row in the
 * backend catalog.
 *
 * JetBrains routes through OpenInEditorByName too, but the backend resolves
 * the concrete JetBrains IDE by file extension, hence the "JetBrains" binding.
 *
 * SystemDefault ("default") is deliberately NOT cataloged: the backend exposes
 * OpenInDefaultEditor as a separate method (it dispatches to xdg-open /
 * explorer rather than a specific editor command), so it is handled as a
 * literal special case wherever it appears.
 */
export interface CatalogEditor {
  /** Frontend key: emitted as the <option> value and the availability JSON key */
  key: string;
  /** Human-readable label shown in the picker */
  label: string;
  /** Binding name passed to the backend's OpenInEditorByName dispatcher */
  binding: string;
}

export const EDITOR_CATALOG = [
  { key: "vscode", label: "VSCode", binding: "VSCode" },
  { key: "vscodium", label: "VSCodium", binding: "VSCodium" },
  { key: "sublime", label: "Sublime Text", binding: "Sublime" },
  { key: "jetbrains", label: "JetBrains IDE", binding: "JetBrains" },
  { key: "geany", label: "Geany", binding: "Geany" },
  { key: "goland", label: "GoLand", binding: "GoLand" },
  { key: "pycharm", label: "PyCharm", binding: "PyCharm" },
  { key: "intellij", label: "IntelliJ IDEA", binding: "IntelliJ" },
  { key: "webstorm", label: "WebStorm", binding: "WebStorm" },
  { key: "phpstorm", label: "PhpStorm", binding: "PhpStorm" },
  { key: "clion", label: "CLion", binding: "CLion" },
  { key: "rider", label: "Rider", binding: "Rider" },
  { key: "androidstudio", label: "Android Studio", binding: "AndroidStudio" },
  { key: "emacs", label: "Emacs", binding: "Emacs" },
  { key: "neovide", label: "Neovide", binding: "Neovide" },
  { key: "codeblocks", label: "Code::Blocks", binding: "CodeBlocks" },
  { key: "devcpp", label: "Dev-C++", binding: "DevCpp" },
  { key: "notepadplusplus", label: "Notepad++", binding: "NotepadPlusPlus" },
  { key: "visualstudio", label: "Visual Studio", binding: "VisualStudio" },
  { key: "eclipse", label: "Eclipse", binding: "Eclipse" },
  { key: "netbeans", label: "NetBeans", binding: "NetBeans" },
  { key: "neovim", label: "Neovim", binding: "Neovim" },
  { key: "vim", label: "Vim", binding: "Vim" },
] as const satisfies readonly CatalogEditor[];

/** Every selectable editor key, including the System Default pseudo-entry. */
export type EditorKey = (typeof EDITOR_CATALOG)[number]["key"] | "default";
