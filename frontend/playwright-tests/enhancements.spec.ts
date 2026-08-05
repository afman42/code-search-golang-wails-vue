import { test, expect } from '@playwright/test';

// E2E coverage for new features: symbol navigation, multi-select/export,
// diff markers, multi-directory search, and log viewer search — all against
// the mocked Wails backend (VITE_WAILS_MOCK=1).

test.beforeEach(async ({ page }) => {
  await page.goto('/');
  await expect(page.locator('#directory')).toBeVisible();
});

test('symbol click opens code preview modal', async ({ page }) => {
  await page.fill('#directory', '/mock/project');
  await page.fill('.symbol-input', 'greet');
  await page.locator('.search-btn').click();

  await expect(page.locator('.symbol-result')).toHaveCount(1);

  // Click the symbol result — should open the preview modal.
  await page.locator('.symbol-result').first().click();
  await expect(page.locator('.modal-overlay')).toBeVisible();
  await expect(page.locator('.modal-title')).toContainText('File Preview');
});

test('symbol click shows file content in preview', async ({ page }) => {
  await page.fill('#directory', '/mock/project');
  await page.fill('.symbol-input', 'main');
  await page.locator('.search-btn').click();

  await expect(page.locator('.symbol-result')).toHaveCount(1);
  await page.locator('.symbol-result').first().click();

  // Modal should show code content from the mock file.
  await expect(page.locator('.modal-overlay')).toBeVisible();
  await expect(page.locator('.code-container')).toBeVisible();
  // The mock main.go contains "hello world".
  await expect(page.locator('.code-container')).toContainText('hello');
});

test('diff markers render on search results (+/-)', async ({ page }) => {
  await page.fill('#directory', '/mock/project');
  await page.fill('#query', 'hello');
  await page.getByRole('button', { name: 'Search Code' }).click();

  await expect(page.locator('.result-item').first()).toBeVisible();

  // Match line should have a + diff marker.
  await expect(page.locator('.diff-plus').first()).toBeVisible();
  // Context lines (before/after) should have - diff markers.
  await expect(page.locator('.diff-minus').first()).toBeVisible();
});

test('batch export buttons are present', async ({ page }) => {
  await page.fill('#directory', '/mock/project');
  await page.fill('#query', 'hello');
  await page.getByRole('button', { name: 'Search Code' }).click();

  await expect(page.locator('.result-item').first()).toBeVisible();

  // Export buttons should be visible.
  await expect(page.getByText('Export CSV')).toBeVisible();
  await expect(page.getByText('Export JSON')).toBeVisible();
});

test('multi-select checkboxes toggle and show count', async ({ page }) => {
  await page.fill('#directory', '/mock/project');
  await page.fill('#query', 'hello');
  await page.getByRole('button', { name: 'Search Code' }).click();

  await expect(page.locator('.result-item').first()).toBeVisible();

  // Check the first result checkbox.
  await page.locator('.result-checkbox').first().check();
  await expect(page.locator('.selected-count')).toContainText('1 selected');

  // Uncheck it.
  await page.locator('.result-checkbox').first().uncheck();
  await expect(page.locator('.selected-count')).toHaveCount(0);
});

test('select all checkbox selects all visible results', async ({ page }) => {
  await page.fill('#directory', '/mock/project');
  await page.fill('#query', 'hello');
  await page.getByRole('button', { name: 'Search Code' }).click();

  await expect(page.locator('.result-item').first()).toBeVisible();

  // Click "Select All (page)".
  await page.getByText('Select All (page)').click();

  // All visible result checkboxes should be checked.
  const checkboxes = page.locator('.result-checkbox');
  const count = await checkboxes.count();
  for (let i = 0; i < count; i++) {
    await expect(checkboxes.nth(i)).toBeChecked();
  }
});

test('extra directories textarea is present and editable', async ({ page }) => {
  const textarea = page.locator('#extra-dirs');
  await expect(textarea).toBeVisible();

  // Type a path — should be editable.
  await textarea.fill('/another/path\n/third/path');
  await expect(textarea).toHaveValue('/another/path\n/third/path');
});

test('log viewer search input is present when expanded', async ({ page }) => {
  // Expand the log viewer.
  await page.locator('.log-toggle-button').click();
  await expect(page.locator('.log-content-wrapper')).toBeVisible();

  // Search input should be visible.
  await expect(page.locator('.log-search-input')).toBeVisible();

  // Auto-scroll toggle should be visible.
  await expect(page.locator('.autoscroll-label')).toBeVisible();
});

test('load all symbols shows progress and results', async ({ page }) => {
  await page.fill('#directory', '/mock/project');

  await page.locator('.fetch-all-btn').click();

  // "Load All Symbols" stores results in allSymbols and shows the
  // Quick Access section with .quick-item elements.
  await expect(page.locator('.quick-item').first()).toBeVisible({ timeout: 5000 });
  // Mock returns 3 symbols, recentlySeenSymbols shows last 5.
  await expect(page.locator('.quick-item')).toHaveCount(3);

  // Status message should confirm indexing.
  await expect(page.locator('.status-message')).toContainText('Indexed');
});
