<template>
  <div class="startup-loader-overlay">
    <div class="startup-loader-content">
      <div class="startup-spinner">
        <div class="startup-spinner-ring"></div>
        <div class="startup-spinner-ring"></div>
        <div class="startup-spinner-ring"></div>
        <div class="startup-spinner-ring"></div>
      </div>
      <div class="startup-loader-text">Loading application...</div>
    </div>
  </div>
</template>

<script lang="ts">
import { defineComponent } from 'vue';

export default defineComponent({
  name: 'StartupLoader'
});
</script>

<style scoped>
.startup-loader-overlay {
  position: fixed;
  top: 0;
  left: 0;
  width: 100%;
  height: 100%;
  background-color: #f5f5f5;
  display: flex;
  justify-content: center;
  align-items: center;
  z-index: 9999;
}

.startup-loader-content {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 20px;
}

.startup-loader-text {
  font-size: 16px;
  color: #666;
  font-weight: 500;
}

.startup-spinner {
  width: 60px;
  height: 60px;
  position: relative;
}

.startup-spinner-ring {
  box-sizing: border-box;
  display: block;
  position: absolute;
  width: 52px;
  height: 52px;
  margin: 4px;
  border: 4px solid #3498db;
  border-radius: 50%;
  animation: startup-spinner-ring 1.2s cubic-bezier(0.5, 0, 0.5, 1) infinite;
  border-color: #3498db transparent transparent transparent;
}

.startup-spinner-ring:nth-child(1) {
  animation-delay: -0.45s;
}

.startup-spinner-ring:nth-child(2) {
  animation-delay: -0.3s;
}

.startup-spinner-ring:nth-child(3) {
  animation-delay: -0.15s;
}

@keyframes startup-spinner-ring {
  0% {
    transform: rotate(0deg);
  }
  100% {
    transform: rotate(360deg);
  }
}
</style>