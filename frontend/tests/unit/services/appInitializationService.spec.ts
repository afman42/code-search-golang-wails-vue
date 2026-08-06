import { describe, test, expect, vi, beforeEach } from "vitest";

// Mock the syntaxHighlightingService module so we can control
// isHighlightJsLoaded / loadHighlightJs behavior independently.
vi.mock("@/services/syntaxHighlightingService", () => ({
  loadHighlightJs: vi.fn().mockResolvedValue(true),
  isHighlightJsLoaded: vi.fn().mockReturnValue(false),
}));

import {
  loadHighlightJs,
  isHighlightJsLoaded,
} from '@/services';
import { initializeAppServices } from '@/services';

describe("appInitializationService", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  test("initializeAppServices calls loadHighlightJs when not already loaded", async () => {
    vi.mocked(isHighlightJsLoaded).mockReturnValue(false);

    await initializeAppServices();

    expect(isHighlightJsLoaded).toHaveBeenCalledTimes(1);
    expect(loadHighlightJs).toHaveBeenCalledTimes(1);
  });

  test("initializeAppServices skips loadHighlightJs when already loaded (idempotent)", async () => {
    vi.mocked(isHighlightJsLoaded).mockReturnValue(true);

    await initializeAppServices();

    expect(isHighlightJsLoaded).toHaveBeenCalledTimes(1);
    expect(loadHighlightJs).not.toHaveBeenCalled();
  });

  test("initializeAppServices does not throw on completion", async () => {
    vi.mocked(isHighlightJsLoaded).mockReturnValue(false);

    await expect(initializeAppServices()).resolves.toBeUndefined();
  });
});
