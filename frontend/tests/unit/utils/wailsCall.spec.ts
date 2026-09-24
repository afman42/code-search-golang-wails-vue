import { describe, expect, it, vi, beforeEach } from "vitest";
import { toastManager } from "@/composables/useToast";
import {
  numField,
  payloadRecord,
  strField,
  withWailsCall,
} from "@/utils/wailsCall";

beforeEach(() => {
  vi.clearAllMocks();
});

describe("wailsCall", () => {
  it("payloadRecord narrows unknown to record", () => {
    expect(payloadRecord(null)).toEqual({});
    expect(payloadRecord([1])).toEqual({});
    expect(payloadRecord({ a: 1 })).toEqual({ a: 1 });
  });

  it("numField/strField degrade to defaults", () => {
    expect(numField({ n: 3 }, "n")).toBe(3);
    expect(numField({}, "n")).toBe(0);
    expect(numField({ n: "x" }, "n")).toBe(0);
    expect(strField({ s: "hi" }, "s")).toBe("hi");
    expect(strField({}, "s")).toBe("");
  });

  it("withWailsCall passes success through", async () => {
    await expect(
      withWailsCall(async () => 42, { title: "T" }),
    ).resolves.toBe(42);
  });

  it("withWailsCall toasts and returns null on failure", async () => {
    const errSpy = vi.spyOn(toastManager, "error");
    const setStatus = vi.fn();
    const out = await withWailsCall(
      () => Promise.reject(new Error("boom")),
      { title: "Failed", setStatus },
    );
    expect(out).toBeNull();
    expect(errSpy).toHaveBeenCalledWith("boom", "Failed");
    expect(setStatus).toHaveBeenCalledWith("Error: boom", "error");
    errSpy.mockRestore();
  });
});
