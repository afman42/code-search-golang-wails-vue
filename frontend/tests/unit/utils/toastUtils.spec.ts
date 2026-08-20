import { describe, test, expect, vi, beforeEach } from "vitest";
import { copyToClipboardWithToast, openFileLocationWithToast } from '@/utils';
import { toastManager } from '@/composables';
import { ShowInFolder } from "@wails/go/main/App";

// Mock the Wails binding
vi.mock("@wails/go/main/App", () => ({
  ShowInFolder: vi.fn(),
}));

// Spy on toastManager methods
vi.spyOn(toastManager, "success");
vi.spyOn(toastManager, "error");
vi.spyOn(toastManager, "info");

const mockedShowInFolder = vi.mocked(ShowInFolder);

describe("copyToClipboardWithToast", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  test("rejects empty text with error toast and returns false", async () => {
    const result = await copyToClipboardWithToast("");
    expect(result).toBe(false);
    expect(toastManager.error).toHaveBeenCalledWith("Cannot copy empty or invalid text", "Copy Error");
  });

  test("rejects non-string text with error toast", async () => {
    const result = await copyToClipboardWithToast(null as unknown as string);
    expect(result).toBe(false);
    expect(toastManager.error).toHaveBeenCalled();
  });

  test("uses clipboard API in secure context and shows success toast", async () => {
    const writeText = vi.fn().mockResolvedValue(undefined);
    Object.defineProperty(navigator, "clipboard", {
      value: { writeText },
      configurable: true,
    });
    Object.defineProperty(window, "isSecureContext", {
      value: true,
      configurable: true,
    });

    const result = await copyToClipboardWithToast("hello world");
    expect(writeText).toHaveBeenCalledWith("hello world");
    expect(result).toBe(true);
    expect(toastManager.success).toHaveBeenCalledWith("Copied to clipboard successfully", "Copy Success");
  });

  test("falls back to textarea+execCommand when no clipboard API", async () => {
    Object.defineProperty(navigator, "clipboard", {
      value: undefined,
      configurable: true,
    });
    Object.defineProperty(window, "isSecureContext", {
      value: false,
      configurable: true,
    });

    const result = await copyToClipboardWithToast("fallback text");
    expect(result).toBe(true);
    expect(toastManager.success).toHaveBeenCalled();
  });
});

describe("openFileLocationWithToast", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockedShowInFolder.mockReset();
  });

  test("handles invalid path with error toast, no throw", async () => {
    await openFileLocationWithToast("");
    expect(toastManager.error).toHaveBeenCalledWith("Invalid file path provided", "Open Folder Error");
  });

  test("handles null path with error toast, no throw", async () => {
    await openFileLocationWithToast(null as unknown as string);
    expect(toastManager.error).toHaveBeenCalled();
  });

  test("calls ShowInFolder and shows info toast on success", async () => {
    mockedShowInFolder.mockResolvedValue(undefined);

    await openFileLocationWithToast("/home/user/project/main.go");
    expect(ShowInFolder).toHaveBeenCalledWith("/home/user/project/main.go");
    expect(toastManager.info).toHaveBeenCalledWith(
      "Opened containing folder for: main.go",
      "Folder Opened",
    );
  });

  test("extracts filename with backslash separator", async () => {
    mockedShowInFolder.mockResolvedValue(undefined);

    // Now that the split uses /[/\\]/, backslash-only paths correctly
    // extract the filename.
    await openFileLocationWithToast("C:\\Users\\test\\app.ts");
    expect(toastManager.info).toHaveBeenCalledWith(
      "Opened containing folder for: app.ts",
      "Folder Opened",
    );
  });

  test("shows error toast on ShowInFolder failure, no re-throw", async () => {
    mockedShowInFolder.mockRejectedValue(new Error("Wails error"));

    await openFileLocationWithToast("/test/file.go");
    expect(toastManager.error).toHaveBeenCalledWith(
      expect.stringContaining("Wails error"),
      "Open Folder Error",
    );
  });
});
