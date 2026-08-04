import { test, expect } from '@playwright/test';

// E2E coverage for the file-explorer tree in the preview modal and the
// recent-search suggestions dropdown, both against the mocked Wails backend
// (VITE_WAILS_MOCK=1, see src/mocks/wailsMock.ts).

test.beforeEach(async ({ page }) => {
  // The mock FS has three files that all contain "hello":
  //   /mock/project/main.go, /mock/project/util.go, /mock/project/README.md
  await page.goto('/');
  await expect(page.locator('#directory')).toBeVisible();
});

test('file tree lists result files and opens a selected file', async ({ page }) => {
  await page.fill('#directory', '/mock/project');
  await page.fill('#query', 'hello');
  await page.getByRole('button', { name: 'Search Code' }).click();
  await expect(page.locator('.result-item').first()).toBeVisible();

  // Open the preview modal for the first result, then switch to the tree tab.
  await page.locator('.view-btn').first().click();
  await expect(page.locator('.modal-overlay')).toBeVisible();
  await page.locator('.tree-view-button').click();
  await expect(page.locator('.tree-view-panel')).toBeVisible();

  // All three mocked result files are present in the tree.
  await expect(page.locator('.tree-item-name', { hasText: 'main.go' })).toBeVisible();
  await expect(page.locator('.tree-item-name', { hasText: 'util.go' })).toBeVisible();
  await expect(page.locator('.tree-item-name', { hasText: 'README.md' })).toBeVisible();

  // Clicking a file loads it in the modal and returns to the file tab.
  await page.locator('.tree-item-name', { hasText: 'util.go' }).click();
  await expect(page.locator('.modal-title')).toContainText('util.go');
  await expect(page.locator('.code-container')).toContainText('helper');
  await expect(
    page.locator('.tree-view-button'),
  ).not.toHaveClass(/active/);
});

test('suggestions appear on focus, select a query, and close on outside click', async ({ page }) => {
  await page.addInitScript(() => {
    localStorage.setItem(
      'codeSearchRecentSearches',
      JSON.stringify([
        { query: 'hello', extension: 'go', directory: '/mock/project' },
        { query: 'fuzzy', extension: '', directory: '/mock/project' },
      ]),
    );
  });
  // beforeEach already navigated; reload so the seeded localStorage applies.
  await page.reload();
  await expect(page.locator('#directory')).toBeVisible();

  await page.fill('#directory', '/mock/project');

  // Focusing the query input opens the dropdown with recent queries.
  await page.locator('#query').focus();
  await expect(page.locator('.search-suggestions')).toBeVisible();
  await expect(page.locator('.suggestion-query', { hasText: 'hello' })).toBeVisible();

  // Selecting a suggestion fills the query and runs the search.
  await page.locator('.suggestion-item', { hasText: 'hello' }).click();
  await expect(page.locator('#query')).toHaveValue('hello');
  await expect(page.locator('.result-item').first()).toBeVisible();

  // Reopen and verify outside-click and Escape both close the dropdown.
  await page.locator('#query').focus();
  await expect(page.locator('.search-suggestions')).toBeVisible();
  await page.locator('#directory').click();
  await expect(page.locator('.search-suggestions')).toHaveCount(0);

  await page.locator('#query').focus();
  await expect(page.locator('.search-suggestions')).toBeVisible();
  await page.locator('#query').press('Escape');
  await expect(page.locator('.search-suggestions')).toHaveCount(0);
});
