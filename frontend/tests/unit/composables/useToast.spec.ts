import { vi } from "vitest";
import { useToast } from "../../../src/composables/useToast";

describe("useToast composable", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    vi.useFakeTimers();
    // Clear any toasts left from previous tests (shared state)
    const { clearAll } = useToast();
    clearAll();
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  test("should add a toast and auto-remove after duration", () => {
    const { toasts, addToast } = useToast();

    addToast("Test message", { duration: 3000 });

    expect(toasts.length).toBe(1);
    expect(toasts[0].message).toBe("Test message");
    expect(toasts[0].remaining).toBe(3000);

    vi.advanceTimersByTime(3000);

    expect(toasts.length).toBe(0);
  });

  test("should remove toast manually", () => {
    const { toasts, addToast, removeToast } = useToast();

    addToast("Remove me", { duration: 5000 });
    const toast = toasts[0];

    removeToast(toast);

    expect(toasts.length).toBe(0);
  });

  test("should pause toast and track remaining time", () => {
    const { toasts, addToast, pauseToast } = useToast();

    addToast("Pause test", { duration: 5000 });
    const toast = toasts[0];

    vi.advanceTimersByTime(2000);

    pauseToast(toast);

    expect(toast.paused).toBe(true);
    expect(toast.remaining).toBeLessThanOrEqual(3000);
    expect(toast.remaining).toBeGreaterThan(2000);
  });

  test("should resume toast with remaining time, not full duration", () => {
    const { toasts, addToast, pauseToast, resumeToast } = useToast();

    addToast("Resume test", { duration: 5000 });
    const toast = toasts[0];

    vi.advanceTimersByTime(3000);

    pauseToast(toast);
    const remainingAfterPause = toast.remaining;

    resumeToast(toast);

    expect(toast.paused).toBe(false);
    expect(toast.remaining).toBe(remainingAfterPause);

    vi.advanceTimersByTime(remainingAfterPause);
    expect(toasts.length).toBe(0);
  });

  test("should not extend lifetime by pausing and resuming multiple times", () => {
    const { toasts, addToast, pauseToast, resumeToast } = useToast();

    addToast("Multi pause test", { duration: 5000 });
    const toast = toasts[0];

    vi.advanceTimersByTime(1000);
    pauseToast(toast);
    const remaining1 = toast.remaining;
    resumeToast(toast);

    vi.advanceTimersByTime(2000);
    pauseToast(toast);
    const remaining2 = toast.remaining;
    resumeToast(toast);

    expect(remaining2).toBeLessThanOrEqual(2000);
    expect(remaining2).toBeGreaterThan(1000);

    vi.advanceTimersByTime(remaining2);
    expect(toasts.length).toBe(0);
  });

  test("should create toasts with correct types", () => {
    const { toasts, success, error, warning, info } = useToast();

    success("Success!");
    error("Error!");
    warning("Warning!");
    info("Info!");

    expect(toasts.length).toBe(4);
    expect(toasts[0].type).toBe("success");
    expect(toasts[1].type).toBe("error");
    expect(toasts[2].type).toBe("warning");
    expect(toasts[3].type).toBe("info");
  });

  test("should clear all toasts", () => {
    const { toasts, addToast, clearAll } = useToast();

    addToast("First");
    addToast("Second");
    addToast("Third");

    expect(toasts.length).toBe(3);

    clearAll();

    expect(toasts.length).toBe(0);
  });

  test("should generate unique IDs for each toast", () => {
    const { addToast } = useToast();

    const id1 = addToast("First");
    const id2 = addToast("Second");

    expect(id1).not.toBe(id2);
  });

  test("zero duration toast should not auto-remove", () => {
    const { toasts, addToast } = useToast();

    addToast("Persistent", { duration: 0 });
    expect(toasts.length).toBe(1);

    vi.advanceTimersByTime(10000);
    expect(toasts.length).toBe(1);
  });

  describe("edge cases", () => {
    test("removing a non-existent toast should not throw", () => {
      const { toasts, addToast, removeToast } = useToast();

      addToast("First");
      const realToast = toasts[0];

      // Remove a fake toast that isn't in the array
      expect(() => {
        removeToast({ id: 'non-existent', title: '', message: '', type: 'info', duration: 5000, timer: null, paused: false, remaining: 5000, startedAt: 0 });
      }).not.toThrow();

      // The real toast should still be there
      removeToast(realToast);
      expect(toasts.length).toBe(0);
    });

    test("pausing a non-existent toast should not throw", () => {
      const { pauseToast } = useToast();

      expect(() => {
        pauseToast({ id: 'ghost', title: '', message: '', type: 'info', duration: 5000, timer: null, paused: false, remaining: 5000, startedAt: 0 });
      }).not.toThrow();
    });

    test("resuming a non-existent toast should not throw", () => {
      const { resumeToast } = useToast();

      expect(() => {
        resumeToast({ id: 'ghost', title: '', message: '', type: 'info', duration: 5000, timer: null, paused: true, remaining: 5000, startedAt: 0 });
      }).not.toThrow();
    });

    test("pausing an already-paused toast should be idempotent", () => {
      const { toasts, addToast, pauseToast } = useToast();

      addToast("Test", { duration: 5000 });
      const toast = toasts[0];

      vi.advanceTimersByTime(1000);
      pauseToast(toast);
      const remainingAfterFirstPause = toast.remaining;

      // Pause again — should be a no-op
      pauseToast(toast);
      expect(toast.paused).toBe(true);
      expect(toast.remaining).toBe(remainingAfterFirstPause);
    });

    test("resuming a non-paused toast should be idempotent", () => {
      const { toasts, addToast, resumeToast } = useToast();

      addToast("Test", { duration: 5000 });
      const toast = toasts[0];

      // Resume when not paused — should be a no-op
      resumeToast(toast);
      expect(toast.paused).toBe(false);
      expect(toast.remaining).toBe(5000);
    });

    test("clearing all when already empty should not throw", () => {
      const { toasts, addToast, clearAll } = useToast();

      expect(toasts.length).toBe(0);
      expect(() => clearAll()).not.toThrow();
      expect(toasts.length).toBe(0);
    });

    test("rapid add and remove cycle", () => {
      const { toasts, addToast, removeToast } = useToast();

      for (let i = 0; i < 50; i++) {
        const id = addToast(`Toast ${i}`, { duration: 100 });
        const toast = toasts[toasts.length - 1];
        removeToast(toast);
      }

      expect(toasts.length).toBe(0);
    });

    test("concurrent toasts with staggered durations all auto-remove", () => {
      const { toasts, addToast } = useToast();

      addToast("200ms", { duration: 200 });
      addToast("400ms", { duration: 400 });
      addToast("600ms", { duration: 600 });

      expect(toasts.length).toBe(3);

      vi.advanceTimersByTime(200);
      expect(toasts.length).toBe(2);

      vi.advanceTimersByTime(200);
      expect(toasts.length).toBe(1);

      vi.advanceTimersByTime(200);
      expect(toasts.length).toBe(0);
    });
  });
});
