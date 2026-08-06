import { fileURLToPath } from "node:url";
import * as path from "node:path";
import { defineConfig } from "vitest/config";
import vue from "@vitejs/plugin-vue";

const __dirname = fileURLToPath(new URL(".", import.meta.url));

// Vitest configuration.
// The Wails-generated bindings under `wailsjs/` reach out to `window.go`/`window.runtime`,
// which don't exist in the test environment. We alias them to lightweight mocks so the
// composables and components can be exercised in isolation.
export default defineConfig({
  plugins: [vue()],
  resolve: {
    alias: [
      // Route Wails binding imports to the lightweight test mocks, regardless
      // of whether the source uses the @wails/* alias or a relative path.
      // These must come before the generic @wails rule (first-match-wins).
      { find: /^@wails\/go\/main\/App$/, replacement: fileURLToPath(new URL("./tests/__mocks__/wailsjs/go/main/App.ts", import.meta.url)) },
      { find: /^@wails\/runtime$/, replacement: fileURLToPath(new URL("./tests/__mocks__/wailsjs/runtime/index.ts", import.meta.url)) },
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
      // Generic aliases — mirror tsconfig.json paths so @/ and @wails/ resolve in tests.
      { find: "@", replacement: path.resolve(__dirname, "./src") },
      { find: "@wails", replacement: path.resolve(__dirname, "./wailsjs") },
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
