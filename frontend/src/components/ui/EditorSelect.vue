<template>
  <select
    class="editor-select"
    @change="$emit('editorSelect', $event)"
    title="Open in editor"
  >
    <option value="">Editor...</option>
    <option v-if="availableEditors.vscode" value="vscode">VSCode</option>
    <option v-if="availableEditors.vscodium" value="vscodium">VSCodium</option>
    <option v-if="availableEditors.sublime" value="sublime">Sublime Text</option>
    <option v-if="availableEditors.atom" value="atom">Atom</option>
    <option v-if="availableEditors.jetbrains" value="jetbrains">JetBrains</option>
    <option v-if="availableEditors.geany" value="geany">Geany</option>
    <option v-if="availableEditors.goland" value="goland">GoLand</option>
    <option v-if="availableEditors.pycharm" value="pycharm">PyCharm</option>
    <option v-if="availableEditors.intellij" value="intellij">IntelliJ IDEA</option>
    <option v-if="availableEditors.webstorm" value="webstorm">WebStorm</option>
    <option v-if="availableEditors.phpstorm" value="phpstorm">PhpStorm</option>
    <option v-if="availableEditors.clion" value="clion">CLion</option>
    <option v-if="availableEditors.rider" value="rider">Rider</option>
    <option v-if="availableEditors.androidstudio" value="androidstudio">Android Studio</option>
    <option v-if="availableEditors.emacs" value="emacs">Emacs</option>
    <option v-if="availableEditors.neovide" value="neovide">Neovide</option>
    <option v-if="availableEditors.codeblocks" value="codeblocks">Code::Blocks</option>
    <option v-if="availableEditors.devcpp" value="devcpp">Dev-C++</option>
    <option v-if="availableEditors.notepadplusplus" value="notepadplusplus">Notepad++</option>
    <option v-if="availableEditors.visualstudio" value="visualstudio">Visual Studio</option>
    <option v-if="availableEditors.eclipse" value="eclipse">Eclipse</option>
    <option v-if="availableEditors.netbeans" value="netbeans">NetBeans</option>
    <option v-if="availableEditors.neovim" value="neovim">Neovim</option>
    <option v-if="availableEditors.vim" value="vim">Vim</option>
    <option value="default">System Default</option>
  </select>
</template>

<script setup lang="ts">
import type { EditorAvailability } from "../../types/search";

defineProps<{
  availableEditors: EditorAvailability;
}>();

defineEmits<{
  editorSelect: [event: Event];
}>();
</script>