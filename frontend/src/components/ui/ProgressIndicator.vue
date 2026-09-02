<template>
  <div v-if="data.showProgress" class="progress-container">
    <div class="progress-bar">
      <div
        class="progress-fill"
        :style="{ width: data.searchProgress.totalFiles > 0 ?
          (data.searchProgress.processedFiles / data.searchProgress.totalFiles * 100) + '%' : '0%' }"
      ></div>
    </div>
    <div class="progress-info">
      <span>Processed: {{ data.searchProgress.processedFiles }} / {{ data.searchProgress.totalFiles }} files</span>
      <span>Results: {{ data.searchProgress.resultsCount }}</span>
    </div>
    <div v-if="data.searchProgress.currentFile" class="current-file">
      Processing: {{ formatFilePath(data.searchProgress.currentFile) }}
    </div>
    <!-- Unreadable files were previously counted but never shown, so a search
         that could not open half the tree looked identical to one that found
         nothing. failedPaths is a backend-capped sample, so the count can
         exceed the listed paths — say so rather than implying the list is
         complete. -->
    <div v-if="data.searchProgress.failedFiles > 0" class="failed-summary">
      <span class="failed-count">
        Skipped {{ data.searchProgress.failedFiles }} unreadable file(s)
      </span>
      <ul v-if="data.searchProgress.failedPaths.length > 0" class="failed-list">
        <li
          v-for="path in data.searchProgress.failedPaths"
          :key="path"
          :title="path"
        >
          {{ formatFilePath(path) }}
        </li>
      </ul>
      <span
        v-if="data.searchProgress.failedPaths.length > 0 &&
          data.searchProgress.failedFiles > data.searchProgress.failedPaths.length"
        class="failed-more"
      >
        …and {{ data.searchProgress.failedFiles - data.searchProgress.failedPaths.length }} more
      </span>
    </div>
  </div>
</template>

<script setup lang="ts">
import type { SearchState } from "@/types";

// Define props with TypeScript
interface Props {
  data: SearchState;
  formatFilePath: (filePath: string) => string;
}
defineProps<Props>();
</script>

<style scoped>
.progress-container {
  max-width: 600px;
  margin: 1.5rem auto;
  padding: 0 var(--space-5);
}

.progress-bar {
  width: 100%;
  height: 20px;
  background-color: var(--color-bg-tertiary);
  border-radius: var(--radius-lg);
  overflow: hidden;
  margin-bottom: var(--space-2);
  box-shadow: inset 0 1px 3px rgba(0,0,0,0.2);
}

.progress-fill {
  height: 100%;
  background: linear-gradient(to right, var(--color-accent), var(--color-accent-dark));
  transition: width 0.3s ease;
  border-radius: var(--radius-lg);
  position: relative;
  overflow: hidden;
}

.progress-fill::after {
  content: '';
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background-image: linear-gradient(
    -45deg,
    rgba(255, 255, 255, .2) 25%,
    transparent 25%,
    transparent 50%,
    rgba(255, 255, 255, .2) 50%,
    rgba(255, 255, 255, .2) 75%,
    transparent 75%
  );
  background-size: 30px 30px;
  animation: progress-shine 1.5s infinite linear;
  opacity: 0.3;
}

@keyframes progress-shine {
  0% {
    background-position: 0 0;
  }
  100% {
    background-position: 30px 30px;
  }
}

.progress-info {
  display: flex;
  justify-content: space-between;
  font-size: 0.9em;
  color: var(--color-text-muted);
  margin-bottom: 5px;
}

.current-file {
  font-size: 0.85em;
  color: var(--color-text-muted);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.failed-summary {
  margin-top: var(--space-2);
  padding: var(--space-2);
  border-left: 3px solid var(--color-warning);
  background: var(--color-bg-tertiary);
  border-radius: var(--radius-sm);
  font-size: 0.85em;
}

.failed-count {
  color: var(--color-warning);
  font-weight: 600;
}

.failed-list {
  margin: var(--space-1) 0 0;
  padding-left: var(--space-4);
  max-height: 6em;
  overflow-y: auto;
  color: var(--color-text-muted);
}

.failed-list li {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.failed-more {
  display: block;
  margin-top: var(--space-1);
  color: var(--color-text-muted);
}
</style>