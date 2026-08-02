import { test, expect } from '@playwright/test';

// End-to-end coverage of the real UX flows against the mocked Wails backend
// (VITE_WAILS_MOCK=1). The mock serves a 3-file synthetic project that all
// contain the word "hello" (see src/mocks/wailsMock.ts).

test.beforeEach(async ({ page }) => {
  await page.goto('/');
  // App.vue shows StartupLoader until app-ready / IsAppReady(); the mock
  // resolves both, so the search form must appear.
  await expect(page.locator('#directory')).toBeVisible();
});

test('startup renders the main UI, not a black screen', async ({ page }) => {
  await expect(page.locator('#query')).toBeVisible();
  await expect(page.getByRole('button', { name: 'Search Code' })).toBeVisible();
  // Symbol search panel is present.
  await expect(page.locator('.symbol-input')).toBeVisible();
});

test('Search Code populates results', async ({ page }) => {
  await page.fill('#directory', '/mock/project');
  await page.fill('#query', 'hello');
  await page.getByRole('button', { name: 'Search Code' }).click();

  // Results container appears with matches from the mock FS.
  const results = page.locator('.results-container');
  await expect(results).toBeVisible();
  await expect(page.locator('.result-item').first()).toBeVisible();
  await expect(page.locator('.results-summary')).toContainText('matches');
  // The searching overlay must clear once the search completes.
  await expect(page.locator('.searching-overlay')).toHaveCount(0);
});

test('empty query keeps Search Code disabled', async ({ page }) => {
  await page.fill('#directory', '/mock/project');
  // Query empty -> button disabled (SearchForm binds :disabled to !query).
  await expect(page.getByRole('button', { name: 'Search Code' })).toBeDisabled();
  await page.fill('#query', 'hello');
  await expect(page.getByRole('button', { name: 'Search Code' })).toBeEnabled();
});

test('file preview modal opens with file content', async ({ page }) => {
  await page.fill('#directory', '/mock/project');
  await page.fill('#query', 'hello');
  await page.getByRole('button', { name: 'Search Code' }).click();
  await expect(page.locator('.result-item').first()).toBeVisible();

  await page.locator('.view-btn').first().click();

  const modal = page.locator('.modal-overlay');
  await expect(modal).toBeVisible();
  await expect(page.locator('.modal-title')).toContainText('File Preview');
  // Mock file content includes the word "package" (Go source) or README text.
  await expect(page.locator('.code-container')).toContainText('hello');

  // Close it.
  await page.locator('.modal-close-button').click();
  await expect(modal).toHaveCount(0);
});

test('symbol search returns matches for a directory', async ({ page }) => {
  // Directory must be set for symbol search to work (bindings require it).
  await page.fill('#directory', '/mock/project');
  await page.fill('.symbol-input', 'greet');
  await page.locator('.search-btn').click();

  await expect(page.locator('.symbol-result')).toHaveCount(1);
  await expect(page.locator('.symbol-name').first()).toHaveText('greet');
});

test('symbol search without a directory prompts to select one', async ({ page }) => {
  // No directory set -> guarded path shows an info message, never crashes.
  await page.fill('.symbol-input', 'greet');
  await page.locator('.search-btn').click();
  await expect(page.locator('.status-message')).toContainText('Select a directory');
  await expect(page.locator('.symbol-result')).toHaveCount(0);
});

test('case-sensitive option is honored', async ({ page }) => {
  await page.fill('#directory', '/mock/project');
  await page.fill('#query', 'HELLO');
  // Enable case-sensitive: uppercase "HELLO" should match nothing (mock FS
  // only contains lowercase "hello").
  await page.getByLabel('Case Sensitive').check();
  await page.getByRole('button', { name: 'Search Code' }).click();
  await expect(page.locator('#result')).toContainText('No matches found');
  await expect(page.locator('.result-item')).toHaveCount(0);
});
