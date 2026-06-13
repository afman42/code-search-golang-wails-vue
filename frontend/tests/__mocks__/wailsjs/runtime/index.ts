// Mock for the Wails-generated runtime bindings used in tests.
import { vi } from "vitest";

export const EventsOn = vi.fn().mockReturnValue(vi.fn());
export const EventsOnce = vi.fn().mockReturnValue(vi.fn());
export const EventsOff = vi.fn();
export const EventsEmit = vi.fn();
