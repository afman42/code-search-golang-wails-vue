<template>
  <div v-if="detectingEditors" class="editor-detection-status">
    <div class="detection-animation">
      <div class="spinner"></div>
      <span>{{ message }}</span>
    </div>
    <div class="detection-progress">
      <div class="progress-bar">
        <div
          class="progress-fill"
          :style="{ width: detectionProgress + '%' }"
        ></div>
      </div>
      <span class="progress-text">
        {{ Math.round(detectionProgress) }}%
      </span>
    </div>
  </div>

  <div
    v-else-if="detectionComplete"
    class="editor-detection-status completed"
  >
    <div class="detection-result">
      <span class="status-icon">✓</span>
      <span>{{ message }}</span>
    </div>
    <div v-if="detectedEditors.length > 0" class="detected-editors-list">
      <span>Found editors: {{ detectedEditors.join(", ") }}</span>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, defineOptions } from 'vue';

defineOptions({
  name: 'EditorStatusDisplay',
});

interface Props {
  editorDetectionStatus?: {
    detectingEditors: boolean;
    detectionComplete: boolean;
    message: string;
    detectionProgress: number;
    detectedEditors: string[];
  };
}

const props = withDefaults(defineProps<Props>(), {
  editorDetectionStatus: () => ({
    detectingEditors: false,
    detectionComplete: false,
    message: '',
    detectionProgress: 0,
    detectedEditors: [],
  }),
});

const detectingEditors = computed(() => props.editorDetectionStatus?.detectingEditors);
const detectionComplete = computed(() => props.editorDetectionStatus?.detectionComplete);
const message = computed(() => props.editorDetectionStatus?.message || '');
const detectionProgress = computed(() => props.editorDetectionStatus?.detectionProgress || 0);
const detectedEditors = computed(() => props.editorDetectionStatus?.detectedEditors || []);
</script>

<style scoped>
.editor-detection-status {
  margin-bottom: 1rem;
  padding: 0.75rem;
  background-color: #f8f9fa;
  border-radius: 0.25rem;
  text-align: center;
}

.editor-detection-status.completed {
  background-color: #d4edda;
  border-color: #c3e6cb;
}

.detection-animation {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 0.5rem;
  font-size: 0.9rem;
  color: #6c757d;
}

.spinner {
  display: inline-block;
  width: 16px;
  height: 16px;
  border: 2px solid rgba(0, 0, 0, 0.1);
  border-radius: 50%;
  border-top-color: #28a745;
  animation: spin 1s ease-in-out infinite;
}

@keyframes spin {
  to {
    transform: rotate(360deg);
  }
}

.detection-progress {
  margin-top: 0.5rem;
}

.progress-bar {
  height: 8px;
  background-color: #e9ecef;
  border-radius: 4px;
  overflow: hidden;
  margin-bottom: 0.25rem;
}

.progress-fill {
  height: 100%;
  background-color: #28a745;
  transition: width 0.3s ease;
}

.progress-text {
  font-size: 0.75rem;
  color: #495057;
}

.detection-result {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 0.5rem;
  font-size: 0.9rem;
}

.status-icon {
  color: #28a745;
  font-weight: bold;
}

.detected-editors-list {
  margin-top: 0.5rem;
  font-size: 0.85em;
  color: #495057;
  line-height: 1.4;
}
</style>
