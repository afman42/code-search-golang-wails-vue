// Mock for the Wails-generated App bindings used in tests.
import { vi } from "vitest";

export const SearchCode = vi.fn();
export const GetDirectoryContents = vi.fn();
export const SelectDirectory = vi.fn();
export const ShowInFolder = vi.fn();
export const SearchWithProgress = vi.fn().mockResolvedValue([]);
export const CancelSearch = vi.fn();
export const ReadFile = vi.fn();
export const ValidateDirectory = vi.fn();
export const GetEditorDetectionStatus = vi.fn();
export const GetAvailableEditors = vi.fn();
