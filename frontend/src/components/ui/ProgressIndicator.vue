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
  </div>
</template>

<script setup lang="ts">
import type { SearchState } from '../../types/search';

// Define props with TypeScript
interface Props {
  data: SearchState;
  formatFilePath: (filePath: string) => string;
}
const props = defineProps<Props>();
</script>

<style scoped>
.progress-container {
  max-width: 600px;
  margin: 1.5rem auto;
  padding: 0 20px;
}

.progress-bar {
  width: 100%;
  height: 20px;
  background-color: #ecf0f1;
  border-radius: 10px;
  overflow: hidden;
  margin-bottom: 8px;
  box-shadow: inset 0 1px 3px rgba(0,0,0,0.2);
}

.progress-fill {
  height: 100%;
  background: linear-gradient(to right, #3498db, #2980b9);
  transition: width 0.3s ease;
  border-radius: 10px;
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
  color: #7f8c8d;
  margin-bottom: 5px;
}

.current-file {
  font-size: 0.85em;
  color: #95a5a6;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
</style>