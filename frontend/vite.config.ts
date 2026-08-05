import { defineConfig } from "vite";
import vue from "@vitejs/plugin-vue";

// https://vitejs.dev/config/
export default defineConfig({
  plugins: [vue()],
  build: {
    cssCodeSplit: false,
    sourcemap: false,
    // Vite 8 uses Oxc for minification by default. "esbuild" is deprecated
    // and requires esbuild as a separate dependency.
    minify: true,
    rollupOptions: {
      output: {
        manualChunks(id) {
          if (id.includes("node_modules/highlight.js")) {
            return "highlightjs";
          }
          if (id.includes("node_modules/vue")) {
            return "vendor";
          }
        },
      },
    },
  },
});
