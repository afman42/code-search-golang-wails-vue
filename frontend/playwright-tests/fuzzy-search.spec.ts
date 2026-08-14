import { test, expect, type Page } from "@playwright/test";

// Fuzzy search E2E against the mocked Wails backend (VITE_WAILS_MOCK=1).
// The mock's buildResults (src/mocks/wailsMock.ts) mirrors the real backend's
// two-phase search:
//   phase 1 — exact matches, unchanged
//   phase 2 — fuzzy near-misses appended only when fuzzySearch is on AND
//             useRegex is off; a line qualifies when a len(query)-wide window
//             aligns >= max(1, floor(queryLen * 0.6)) chars positionally.
// The frontend (useSearch.ts) then re-scores candidates and flags them with
// the fuzzy badge via InlineDiffView's fuzzyMatchScore prop.
//
// /mock/project contains exactly 4 lines with "hello" (main.go:6, main.go:11,
// util.go:5, README.md:3) plus two lines that align >=60% with "hello" but do
// not contain it (util.go:3 "// helper returns...", util.go:4
// "func helper() string {" — the "hel" prefix).

async function runSearch(page: Page, directory: string, query: string) {
  await page.fill("#directory", directory);
  await page.fill("#query", query);
  await page.getByRole("button", { name: "Search Code" }).click();
}

test.describe("fuzzy-search", () => {
  test.beforeEach(async ({ page }) => {
    await page.goto("/");
    await expect(page.locator("#directory")).toBeVisible();
  });

  test("fuzzy search checkbox is present in the search options", async ({ page }) => {
    await expect(page.getByLabel("Fuzzy Search")).toBeVisible();
    // The fuzzy toggle sits alongside the other search options.
    await expect(page.getByLabel("Case Sensitive")).toBeVisible();
    await expect(page.getByLabel("Regex Search")).toBeVisible();
    await expect(page.getByLabel("Include Binary")).toBeVisible();
  });

  test("fuzzy off: only exact 'hello' hits appear (4 results, no badges)", async ({ page }) => {
    await runSearch(page, "/mock/project", "hello");
    await expect(page.locator(".result-item").first()).toBeVisible();
    await expect(page.locator(".results-summary")).toContainText("4 matches");
    // No near-miss lines and no fuzzy badge: every result is an exact hit.
    await expect(page.locator(".fuzzy-badge")).toHaveCount(0);
  });

  test("fuzzy on: near-miss lines are appended beyond exact matches", async ({ page }) => {
    await page.getByLabel("Fuzzy Search").check();
    await runSearch(page, "/mock/project", "hello");
    await expect(page.locator(".result-item").first()).toBeVisible();
    // 4 exact + 2 fuzzy-only lines in util.go (helper prefix aligns 3/5).
    await expect(page.locator(".results-summary")).toContainText("6 matches");
    // Only the two near-miss lines carry the fuzzy badge.
    await expect(page.locator(".fuzzy-badge")).toHaveCount(2);
  });

  test("fuzzy near-misses are flagged with the fuzzy badge", async ({ page }) => {
    // "helo" is a typo for "hello": zero exact hits, but near-miss lines
    // qualify as fuzzy candidates and are badged.
    await page.getByLabel("Fuzzy Search").check();
    await runSearch(page, "/mock/project", "helo");
    const resultItems = page.locator(".result-item");
    await expect(resultItems.first()).toBeVisible();
    await expect(resultItems).toHaveCount(6);
    // Every candidate is a non-exact fuzzy match -> all carry the badge.
    await expect(page.locator(".fuzzy-badge")).toHaveCount(6);
  });

  test("regex mode bypasses fuzzy matching", async ({ page }) => {
    // With regex on, fuzzy is ignored: only exact regex hits are returned.
    await page.getByLabel("Regex Search").check();
    await page.getByLabel("Fuzzy Search").check();
    await runSearch(page, "/mock/project", "hell.+o");
    const resultItems = page.locator(".result-item");
    await expect(resultItems.first()).toBeVisible();
    // "hello world" and "Says hello to the world." match hell.+o; "hello %s"
    // and `return "hello"` do not (no char after "hell" plus a final "o").
    await expect(resultItems).toHaveCount(2);
    await expect(page.locator(".fuzzy-badge")).toHaveCount(0);
  });

  test("typo query with fuzzy off returns no results", async ({ page }) => {
    await runSearch(page, "/mock/project", "helo");
    await expect(page.locator("#result")).toContainText("No matches found");
    await expect(page.locator(".result-item")).toHaveCount(0);
  });

  test("long queries raise the threshold and return no garbage candidates", async ({ page }) => {
    // 19-char query => threshold = floor(19 * 0.6) = 11 aligned chars. No mock
    // line gets close, so the fuzzy pass yields nothing instead of flooding.
    await page.getByLabel("Fuzzy Search").check();
    await runSearch(page, "/mock/project", "abcdefghijklmnopqrs");
    await expect(page.locator("#result")).toContainText("No matches found");
    await expect(page.locator(".result-item")).toHaveCount(0);
  });
});
