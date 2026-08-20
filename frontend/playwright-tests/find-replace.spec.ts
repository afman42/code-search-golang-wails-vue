import { test, expect } from '@playwright/test';

// End-to-end coverage of Find & Replace against the mocked Wails backend
// (VITE_WAILS_MOCK=1). The mock's ReplaceInFiles mutates the in-memory
// synthetic project, so a re-search after Apply reflects the change.

test.beforeEach(async ({ page }) => {
  await page.goto('/');
  await expect(page.locator('#directory')).toBeVisible();
});

test('replace controls hidden under regex mode', async ({ page }) => {
  await page.fill('#directory', '/mock/project');
  await page.fill('#query', 'hello');
  await page.getByRole('button', { name: 'Search Code' }).click();
  await expect(page.locator('.result-item').first()).toBeVisible();

  // Default: replace row is present.
  await expect(page.locator('.replace-row')).toBeVisible();

  // Regex mode: backend rejects regex replace, so the row must hide.
  await page.getByLabel('Regex Search').check();
  await page.getByRole('button', { name: 'Search Code' }).click();
  await expect(page.locator('.replace-row')).toHaveCount(0);
});

test('preview then apply replaces matches and re-search reflects the change', async ({ page }) => {
  await page.fill('#directory', '/mock/project');
  await page.fill('#query', 'hello');
  await page.getByRole('button', { name: 'Search Code' }).click();
  await expect(page.locator('.result-item').first()).toBeVisible();

  // Preview Replace shows old → new diffs without writing.
  await page.fill('.replace-input', 'goodbye');
  await page.getByRole('button', { name: 'Preview Replace' }).click();
  await expect(page.locator('.replace-preview')).toBeVisible();
  await expect(page.locator('.replace-preview-item').first()).toBeVisible();
  await expect(page.locator('.replace-preview-header')).toContainText('line(s) to change');

  // Apply writes the changes; the composable re-runs the search afterwards.
  await page.getByRole('button', { name: /^Apply / }).click();
  // All "hello" became "goodbye" — the re-search finds no matches.
  await expect(page.locator('#result')).toContainText('No matches found');
  await expect(page.locator('.result-item')).toHaveCount(0);
  // Preview cleared after apply.
  await expect(page.locator('.replace-preview')).toHaveCount(0);
});
