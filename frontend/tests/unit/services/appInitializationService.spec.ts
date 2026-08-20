import { describe, test, expect, vi, beforeEach } from "vitest";

// Mock the syntaxHighlightingService module so we can control
// loadHighlightJs behavior independently. loadHighlightJs is idempotent
// (self-guards), so initializeAppServices just delegates to it.
vi.mock("@/services/syntaxHighlightingService", () => ({
  loadHighlightJs: vi.fn().mockResolvedValue(true),
  isHighlightJsLoaded: vi.fn().mockReturnValue(false),
}));

import { loadHighlightJs } from '@/services';
import { initializeAppServices } from '@/services';

describe("appInitializationService", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  test("initializeAppServices loads syntax highlighting", async () => {
    await initializeAppServices();

    expect(loadHighlightJs).toHaveBeenCalledTimes(1);
  });

  test("initializeAppServices does not throw on completion", async () => {
    await expect(initializeAppServices()).resolves.toBeUndefined();
  });
});
