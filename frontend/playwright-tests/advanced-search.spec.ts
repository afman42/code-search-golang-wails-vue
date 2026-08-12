import { test, expect, type Page } from '@playwright/test';

// Advanced search-flow coverage that requires the extended mock FS
// (VITE_WAILS_MOCK=1, see src/mocks/wailsMock.ts):
//   - /mock/project  — 3 small files, all contain "hello"
//   - /mock/big      — huge.go (60 lines, 11 "needle" hits) for pagination +
//                      match-navigation (CodeModal mounts nav controls at >50 lines)
//   - /mock/lib      — extra.go ("hello from lib") for multi-directory search
// The mock's buildResults mirrors the backend: directory-scoped, dedup roots,
// and excludePatterns honored by path component.

async function runSearch(page: Page, directory: string, query: string) {
  await page.fill('#directory', directory);
  await page.fill('#query', query);
  await page.getByRole('button', { name: 'Search Code' }).click();
}

test.beforeEach(async ({ page }) => {
  await page.goto('/');
  await expect(page.locator('#directory')).toBeVisible();
});

test('search is scoped to the selected directory', async ({ page }) => {
  // /mock/project has 4 "hello" hits; /mock/lib and /mock/big are other roots
  // and must NOT leak in.
  await runSearch(page, '/mock/project', 'hello');
  await expect(page.locator('.result-item').first()).toBeVisible();
  await expect(page.locator('.results-summary')).toContainText('4 matches');

  const paths = await page
    .locator('.result-item .file-path')
    .allTextContents();
  expect(paths.every((p) => p.startsWith('/mock/project/'))).toBe(true);
});

test('pagination splits results into pages of 10', async ({ page }) => {
  // huge.go carries 11 "needle" lines -> 2 pages.
  await runSearch(page, '/mock/big', 'needle');
  await expect(page.locator('.result-item').first()).toBeVisible();
  await expect(page.locator('.results-summary')).toContainText('11 matches');

  // Page 1: 10 items, pagination controls visible.
  await expect(page.locator('.result-item')).toHaveCount(10);
  await expect(page.locator('.pagination-controls').first()).toBeVisible();
  await expect(page.locator('.pagination-info').first()).toContainText(
    '1-10 of 11',
  );

  // Next -> page 2 shows the remaining 1.
  await page
    .locator('.pagination-actions')
    .first()
    .getByRole('button', { name: 'Next' })
    .click();
  await expect(page.locator('.pagination-info').first()).toContainText(
    '11-11 of 11',
  );
  await expect(page.locator('.result-item')).toHaveCount(1);
});

test('preview modal mounts match navigation for large files and jumps to a line', async ({ page }) => {
  await runSearch(page, '/mock/big', 'needle');
  await expect(page.locator('.result-item').first()).toBeVisible();

  await page.locator('.view-btn').first().click();
  await expect(page.locator('.modal-overlay')).toBeVisible();

  // huge.go is 60 lines (>50), so the nav controls mount with a match counter.
  await expect(page.locator('.navigation-controls')).toBeVisible();
  await expect(page.locator('.match-counter')).toContainText('/ 11');

  // Jump-to-line input scrolls to and flashes the target line.
  await page.locator('.line-input').fill('25');
  await page.locator('.line-jump-group .icon-button').click();
  await expect(page.locator('.highlighted-line[data-line="25"]')).toBeVisible();
});

test('multi-directory search merges results from an extra root', async ({ page }) => {
  await page.fill('#directory', '/mock/project');
  await page.fill('#query', 'hello');
  // Add /mock/lib as an extra directory. extra.go contains "hello from lib".
  await page.fill('#extra-dirs', '/mock/lib');
  await page.getByRole('button', { name: 'Search Code' }).click();

  await expect(page.locator('.result-item').first()).toBeVisible();
  // 4 project hits + 1 lib hit.
  await expect(page.locator('.results-summary')).toContainText('5 matches');

  const paths = await page
    .locator('.result-item .file-path')
    .allTextContents();
  expect(paths.some((p) => p.startsWith('/mock/lib/'))).toBe(true);
});

test('exclude pattern drops matching files from results', async ({ page }) => {
  await page.fill('#directory', '/mock/project');
  await page.fill('#query', 'hello');

  // Exclude util.go via a custom pattern (backend matches by path component).
  await page.locator('.custom-pattern-input').first().fill('util.go');
  await page
    .locator('.pattern-section')
    .first()
    .getByRole('button', { name: 'Add' })
    .click();
  await expect(page.locator('.pattern-tag')).toContainText('util.go');

  await page.getByRole('button', { name: 'Search Code' }).click();
  await expect(page.locator('.result-item').first()).toBeVisible();

  const paths = await page
    .locator('.result-item .file-path')
    .allTextContents();
  expect(paths.some((p) => p.includes('util.go'))).toBe(false);
  expect(paths.some((p) => p.includes('main.go'))).toBe(true);
});
