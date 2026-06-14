<script lang="ts" setup>
import { ref, onMounted } from "vue";
import CodeSearch from "./components/CodeSearch.vue";
import StartupLoader from "./components/StartupLoader.vue";
import ToastNotification from "./components/ui/ToastNotification.vue";
import { EventsOn } from "../wailsjs/runtime";
import { IsAppReady } from "../wailsjs/go/main/App";
import { APP_READY_TIMEOUT } from "./constants/appConstants";

// Track whether the app is ready to show the main content
const isAppReady = ref(false);

// Function to set the app as ready
const setAppReady = () => {
  isAppReady.value = true;
};

// Register the app-ready listener at setup time — before onMounted — so the
// window in which the backend could emit the event without anyone listening is
// as small as possible. EventsOn (not EventsOnce) lets a re-emit still arrive.
const stopAppReadyListener = EventsOn("app-ready", () => {
  setAppReady();
  stopAppReadyListener?.();
});

onMounted(async () => {
  // Pull-based check to close the event race entirely: if the backend already
  // finished startup (and possibly emitted app-ready before our listener was
  // registered), IsAppReady() returns true and we show the UI immediately
  // instead of waiting for the event or the fallback timeout.
  try {
    if (await IsAppReady()) {
      setAppReady();
      stopAppReadyListener?.();
      return;
    }
  } catch (e) {
    // If the binding isn't available for any reason, fall back to the event +
    // timeout path below rather than getting stuck.
    console.warn("IsAppReady check failed, relying on event/timeout:", e);
  }

  // Set a timeout as fallback to ensure the app eventually loads even if the
  // event is missed and the readiness check was inconclusive.
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
