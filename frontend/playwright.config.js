import { defineConfig, devices } from '@playwright/test';

// E2E config drives the Vue frontend over HTTP with the Wails Go backend
// mocked (VITE_WAILS_MOCK=1, see src/mocks/wailsMock.ts). This exercises the
// real search -> results -> preview and symbol-search UX flows that unit tests
// cannot cover. Uses the system Google Chrome (channel: 'chrome') to avoid a
// bundled-browser download.
export default defineConfig({
  testDir: './playwright-tests',
  timeout: 30000,
  expect: { timeout: 5000 },
  use: {
    baseURL: 'http://localhost:5173',
    trace: 'on-first-retry',
    screenshot: 'only-on-failure',
    channel: 'chrome',
  },
  projects: [
    {
      name: 'chromium',
      use: {
        ...devices['Desktop Chrome'],
        channel: 'chrome',
        launchOptions: { args: ['--no-sandbox', '--disable-gpu'] },
      },
    },
  ],
  retries: 0,
  workers: 1,
  webServer: {
    command: 'VITE_WAILS_MOCK=1 node node_modules/vite/bin/vite.js --port 5173 --strictPort',
    url: 'http://localhost:5173',
    reuseExistingServer: true,
    timeout: 60000,
  },
});
