import { fileURLToPath } from "node:url";
import { defineConfig } from "vitest/config";
import vue from "@vitejs/plugin-vue";

// Vitest configuration.
// The Wails-generated bindings under `wailsjs/` reach out to `window.go`/`window.runtime`,
// which don't exist in the test environment. We alias them to lightweight mocks so the
// composables and components can be exercised in isolation.
export default defineConfig({
  plugins: [vue()],
  resolve: {
    // Match any relative depth (../../ or ../../../) so every importer of the
    // Wails bindings resolves to the same mock module.
    alias: [
      { find: "@", replacement: fileURLToPath(new URL("./src", import.meta.url)) },
      {
        find: /^(?:\.\.\/)+wailsjs\/go\/main\/App$/,
        replacement: fileURLToPath(
          new URL("./tests/__mocks__/wailsjs/go/main/App.ts", import.meta.url),
        ),
      },
      {
        find: /^(?:\.\.\/)+wailsjs\/runtime$/,
        replacement: fileURLToPath(
          new URL("./tests/__mocks__/wailsjs/runtime/index.ts", import.meta.url),
        ),
      },
    ],
  },
  test: {
    globals: true,
    environment: "jsdom",
    setupFiles: ["./tests/setup.ts"],
    include: ["tests/**/*.spec.{ts,js}"],
    coverage: {
      provider: "v8",
      include: ["src/**/*.{ts,vue}"],
      exclude: ["src/main.ts", "src/**/types/*"],
    },
  },
});
