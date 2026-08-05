<template>
  <div class="toast-container">
    <div
      v-for="toast in toasts"
      :key="toast.id"
      class="toast"
      :class="`toast--${toast.type}`"
      @mouseenter="pauseToast(toast)"
      @mouseleave="resumeToast(toast)"
    >
      <div class="toast__content">
        <div class="toast__icon">
          <svg
            v-if="toast.type === 'success'"
            xmlns="http://www.w3.org/2000/svg"
            width="16"
            height="16"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            stroke-width="2"
            stroke-linecap="round"
            stroke-linejoin="round"
          >
            <polyline points="20 6 9 17 4 12"></polyline>
          </svg>
          <svg
            v-else-if="toast.type === 'error'"
            xmlns="http://www.w3.org/2000/svg"
            width="16"
            height="16"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            stroke-width="2"
            stroke-linecap="round"
            stroke-linejoin="round"
          >
            <line x1="18" y1="6" x2="6" y2="18"></line>
            <line x1="6" y1="6" x2="18" y2="18"></line>
          </svg>
          <svg
            v-else-if="toast.type === 'warning'"
            xmlns="http://www.w3.org/2000/svg"
            width="16"
            height="16"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            stroke-width="2"
            stroke-linecap="round"
            stroke-linejoin="round"
          >
            <path d="M10.29 3.86L1.82 18a2 2 0 0 0 1.71 3h16.94a2 2 0 0 0 1.71-3L13.71 3.86a2 2 0 0 0-3.42 0z"></path>
          </svg>
          <svg
            v-else
            xmlns="http://www.w3.org/2000/svg"
            width="16"
            height="16"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            stroke-width="2"
            stroke-linecap="round"
            stroke-linejoin="round"
          >
            <circle cx="12" cy="12" r="10"></circle>
            <line x1="12" y1="16" x2="12" y2="12"></line>
            <line x1="12" y1="8" x2="12.01" y2="8"></line>
          </svg>
        </div>
        <div class="toast__text">
          <div class="toast__title">{{ toast.title }}</div>
          <div class="toast__message">{{ toast.message }}</div>
        </div>
        <button class="toast__close" @click="removeToast(toast)">
          <svg
            xmlns="http://www.w3.org/2000/svg"
            width="16"
            height="16"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            stroke-width="2"
            stroke-linecap="round"
            stroke-linejoin="round"
          >
            <line x1="18" y1="6" x2="6" y2="18"></line>
            <line x1="6" y1="6" x2="18" y2="18"></line>
          </svg>
        </button>
      </div>
      <div
        class="toast__progress"
        :style="{ animationDuration: `${toast.duration}ms` }"
      ></div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { useToast } from "@/composables/useToast";

// Get all toast functions from the composable
const { toasts, removeToast, pauseToast, resumeToast } = useToast();
</script>

<style scoped>
.toast-container {
  position: fixed;
  top: 20px;
  right: 20px;
  z-index: 10000;
  max-width: 400px;
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.toast {
  background: var(--color-bg);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-lg);
  box-shadow: var(--shadow-md);
  overflow: hidden;
  transform: translateX(0);
  transition: transform 0.3s ease, opacity 0.3s ease;
  animation: slideIn 0.3s ease;
  min-width: 300px;
}

.toast--success {
  border-left: 4px solid var(--color-success);
}

.toast--error {
  border-left: 4px solid var(--color-danger);
}

.toast--warning {
  border-left: 4px solid var(--color-warning);
}

.toast--info {
  border-left: 4px solid var(--color-accent);
}

.toast__content {
  display: flex;
  align-items: flex-start;
  padding: var(--space-3);
  gap: 10px;
}

.toast__icon {
  color: var(--color-text-secondary);
  flex-shrink: 0;
}

.toast--success .toast__icon {
  color: var(--color-success);
}

.toast--error .toast__icon {
  color: var(--color-danger);
}

.toast--warning .toast__icon {
  color: var(--color-warning);
}

.toast--info .toast__icon {
  color: var(--color-accent);
}

.toast__text {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: var(--space-1);
  min-width: 0;
}

.toast__title {
  font-weight: 600;
  color: var(--color-text-primary);
  font-size: 14px;
  margin: 0;
}

.toast__message {
  color: var(--color-text-secondary);
  font-size: 13px;
  margin: 0;
  word-wrap: break-word;
  word-break: break-word;
}

.toast__close {
  background: none;
  border: none;
  color: var(--color-text-muted);
  cursor: pointer;
  padding: 0;
  width: 24px;
  height: 24px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: var(--radius-sm);
  transition: background-color 0.2s;
  flex-shrink: 0;
}

.toast__close:hover {
  background-color: var(--color-bg-secondary);
  color: var(--color-text-primary);
}

.toast__progress {
  height: 2px;
  background-color: var(--color-border);
  animation: progress linear;
}

.toast--success .toast__progress {
  background-color: var(--color-success);
}

.toast--error .toast__progress {
  background-color: var(--color-danger);
}

.toast--warning .toast__progress {
  background-color: var(--color-warning);
}

.toast--info .toast__progress {
  background-color: var(--color-accent);
}

@keyframes slideIn {
  from {
    transform: translateX(100%);
    opacity: 0;
  }
  to {
    transform: translateX(0);
    opacity: 1;
  }
}

@keyframes progress {
  from {
    width: 100%;
  }
  to {
    width: 0%;
  }
}
</style>