import { test, expect, type Page } from '@playwright/test';

// E2E coverage for search options and app-chrome flows that the existing specs
// don't touch: regex mode, maxResults truncation, theme toggle + persistence,
// copy-to-clipboard, and the preview-modal footer actions. All run against the
// mocked Wails backend (VITE_WAILS_MOCK=1, see src/mocks/wailsMock.ts), whose
// 3-file synthetic project all contain "hello".

test.beforeEach(async ({ page }) => {
  await page.goto('/');
  await expect(page.locator('#directory')).toBeVisible();
});

// Startup fires transient toasts (e.g. "Syntax Highlight Loaded") into a
// fixed, top-right container with a very high z-index that overlaps the
// fixed theme-toggle button. A real user click lands once they auto-dismiss;
// tests must wait for the container to empty before clicking top-right chrome.
async function waitForToastsToClear(page: Page) {
  await expect(page.locator('.toast-container .toast')).toHaveCount(0, {
    timeout: 10000,
  });
}

async function runSearch(page: Page, query: string) {
  await page.fill('#directory', '/mock/project');
  await page.fill('#query', query);
  await page.getByRole('button', { name: 'Search Code' }).click();
}

test('regex search matches by pattern, not substring', async ({ page }) => {
  // Enable Regex, search a pattern that matches the mock's func declarations
  // (main.go: main + greet, util.go: helper) but is NOT a literal substring.
  await page.getByLabel('Regex Search').check();
  await runSearch(page, 'func\\s+\\w+');

  await expect(page.locator('.result-item').first()).toBeVisible();
  await expect(page.locator('.results-summary')).toContainText('3 matches');
  // The literal pattern text itself never appears in the files — proof the
  // query ran as a regex, not a substring match.
  await expect(page.locator('.result-item')).toHaveCount(3);
});

test('invalid-substring regex query yields no matches', async ({ page }) => {
  // A regex that matches nothing in the mock FS -> "No matches found".
  await page.getByLabel('Regex Search').check();
  await runSearch(page, 'zzz\\d{4}');
  await expect(page.locator('#result')).toContainText('No matches found');
  await expect(page.locator('.result-item')).toHaveCount(0);
});

test('maxResults caps results and flags truncation', async ({ page }) => {
  await page.fill('#directory', '/mock/project');
  await page.fill('#query', 'hello');
  // Cap at 2. The mock FS yields 4 "hello" matches, so the cap must bite.
  await page.fill('#max-results', '2');
  await page.getByRole('button', { name: 'Search Code' }).click();

  await expect(page.locator('.result-item')).toHaveCount(2);
  await expect(page.locator('.results-summary')).toContainText('truncated');
});

test('theme toggle flips the theme and persists to localStorage', async ({ page }) => {
  await waitForToastsToClear(page);

  const html = page.locator('html');
  const initial = await html.getAttribute('data-theme');
  const next = initial === 'dark' ? 'light' : 'dark';

  await page.locator('.theme-toggle').click();
  await expect(html).toHaveAttribute('data-theme', next);

  // Persisted, so a reload keeps the chosen theme.
  const persisted = await page.evaluate(() =>
    localStorage.getItem('codeSearchTheme'),
  );
  expect(persisted).toBe(next);

  await page.reload();
  await expect(page.locator('#directory')).toBeVisible();
  await expect(page.locator('html')).toHaveAttribute('data-theme', next);
});

test('copy line writes the match content to the clipboard', async ({ page, context }) => {
  await context.grantPermissions(['clipboard-read', 'clipboard-write']);
  await runSearch(page, 'hello');
  await expect(page.locator('.result-item').first()).toBeVisible();

  await page.locator('.result-item .copy-btn').first().click();

  // The clipboard should hold the copied line. Content is the robust signal
  // (the toast is environment-dependent under headless clipboard policy).
  await expect
    .poll(() => page.evaluate(() => navigator.clipboard.readText()), {
      timeout: 5000,
    })
    .toContain('hello');
});

test('preview modal footer exposes jump-to-line, show-in-folder, and copy actions', async ({ page }) => {
  await runSearch(page, 'hello');
  await expect(page.locator('.result-item').first()).toBeVisible();

  await page.locator('.view-btn').first().click();
  await expect(page.locator('.modal-overlay')).toBeVisible();

  const footer = page.locator('.modal-footer');
  await expect(footer.getByText('Jump to Line')).toBeVisible();
  await expect(footer.getByText('Show in Folder')).toBeVisible();
  await expect(footer.getByText('Copy to Clipboard')).toBeVisible();

  // Show in Folder hits a no-op mock binding; it must not crash the modal.
  await footer.getByText('Show in Folder').click();
  await expect(page.locator('.modal-container')).toBeVisible();
});
