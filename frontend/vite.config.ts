import { defineConfig } from "vite";
import vue from "@vitejs/plugin-vue";
import * as path from "node:path";
import { fileURLToPath } from "node:url";
import tsconfigPaths from "vite-tsconfig-paths";

const __dirname = fileURLToPath(new URL(".", import.meta.url));

// https://vitejs.dev/config/
// Aliases (@/, @wails/) are defined in tsconfig.json `paths` and mirrored here
// in resolve.alias so both the type checker and the bundler resolve them.
export default defineConfig({
  plugins: [vue(), tsconfigPaths()],
  resolve: {
    alias: {
      "@": path.resolve(__dirname, "./src"),
      "@wails": path.resolve(__dirname, "./wailsjs"),
    },
  },
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
