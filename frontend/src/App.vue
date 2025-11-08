<script lang="ts" setup>
import { ref, onMounted } from 'vue';
import CodeSearch from "./components/CodeSearch.vue";
import StartupLoader from "./components/StartupLoader.vue";
import ToastNotification from "./components/ui/ToastNotification.vue";
import { EventsOnce } from "../wailsjs/runtime";
import { APP_READY_TIMEOUT } from "./constants/appConstants";

// Track whether the app is ready to show the main content
const isAppReady = ref(false);

// Function to set the app as ready
const setAppReady = () => {
  isAppReady.value = true;
};

onMounted(() => {
  // Listen for the app-ready event from the backend
  const cleanup = EventsOnce("app-ready", () => {
    setAppReady();
  });

  // Set a timeout as fallback to ensure the app eventually loads
  setTimeout(() => {
    if (!isAppReady.value) {
      setAppReady();
    }
  }, APP_READY_TIMEOUT); // 3 seconds max timeout
});
</script>

<template>
  <div class="app-container">
    <!-- Show loader while app is initializing -->
    <StartupLoader v-if="!isAppReady" />
    
    <!-- Show main content when app is ready -->
    <CodeSearch v-else />
    
    <!-- Toast notifications -->
    <ToastNotification />
  </div>
</template>

<style scoped>
.app-container {
  height: 100vh;
  width: 100vw;
  position: relative;
}
</style>
