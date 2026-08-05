import { describe, test, expect, vi, beforeEach } from 'vitest';
import {
  makeDefaultEditorAvailability,
  makeDefaultEditorDetectionStatus,
  subscribeToEditorDetectionEvents,
  startEditorDetection,
} from '../../../src/composables/useEditorDetection';
import { EventsOn } from '../../../wailsjs/runtime';
import { GetEditorDetectionStatus } from '../../../wailsjs/go/main/App';
import type { EditorAvailability, EditorDetectionStatus } from '../../../src/types/search';

// Hoisted shared cleanup spy so the EventsOn mock can return a callable cleanup
// that we can assert against when the returned teardown is invoked.
const mocks = vi.hoisted(() => ({
  cleanupSpy: vi.fn(),
}));

// Mock the Wails runtime EventsOn — must match the import path in useEditorDetection.ts.
// Each subscription returns the shared cleanup spy so the teardown can call it.
vi.mock('../../../wailsjs/runtime', () => ({
  EventsOn: vi.fn(() => mocks.cleanupSpy),
}));

// Mock the Wails App binding used for the pull-based status check.
vi.mock('../../../wailsjs/go/main/App', () => ({
  GetEditorDetectionStatus: vi.fn(),
}));

/** Retrieve the payload callback registered for a given event name. */
function getEventCallback(eventName: string): ((payload: unknown) => void) | undefined {
  for (const call of vi.mocked(EventsOn).mock.calls) {
    if (call[0] === eventName) {
      return call[1] as (payload: unknown) => void;
    }
  }
  return undefined;
}

/** Flush the macrotask queue so the pull-based async status check resolves. */
function flush(): Promise<void> {
  const { promise, resolve } = Promise.withResolvers<void>();
  setTimeout(resolve, 0);
  return promise;
}

describe('useEditorDetection', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    vi.mocked(GetEditorDetectionStatus).mockResolvedValue({} as Record<string, unknown>);
  });

  describe('makeDefaultEditorAvailability', () => {
    test('returns all editors false except systemdefault: true', () => {
      const avail = makeDefaultEditorAvailability();

      expect(avail.systemdefault).toBe(true);
      for (const [key, value] of Object.entries(avail)) {
        if (key === 'systemdefault') continue;
        expect(value).toBe(false);
      }
    });

    test('returns a fresh object each call (not cached)', () => {
      const a = makeDefaultEditorAvailability();
      const b = makeDefaultEditorAvailability();
      expect(a).not.toBe(b);
      expect(a).toEqual(b);
      a.vscode = true;
      expect(b.vscode).toBe(false);
    });
  });

  describe('makeDefaultEditorDetectionStatus', () => {
    test('returns detectionComplete=false, detectingEditors=true, progress=0, empty detectedEditors', () => {
      const status = makeDefaultEditorDetectionStatus();

      expect(status.detectionComplete).toBe(false);
      expect(status.detectingEditors).toBe(true);
      expect(status.detectionProgress).toBe(0);
      expect(status.detectedEditors).toEqual([]);
      expect(status.totalAvailable).toBe(0);
      expect(status.availableEditors).toEqual(makeDefaultEditorAvailability());
    });

    test('detectedEditors is a fresh array each call', () => {
      const a = makeDefaultEditorDetectionStatus();
      const b = makeDefaultEditorDetectionStatus();
      expect(a.detectedEditors).not.toBe(b.detectedEditors);
    });
  });

  describe('subscribeToEditorDetectionEvents', () => {
    test('registers listeners for start, progress, and complete and returns a teardown', () => {
      const avail = makeDefaultEditorAvailability();
      const status = makeDefaultEditorDetectionStatus();

      const cleanup = subscribeToEditorDetectionEvents(avail, status);

      expect(typeof cleanup).toBe('function');
      expect(vi.mocked(EventsOn)).toHaveBeenCalledTimes(3);
      expect(vi.mocked(EventsOn)).toHaveBeenCalledWith('editor-detection-start', expect.any(Function));
      expect(vi.mocked(EventsOn)).toHaveBeenCalledWith('editor-detection-progress', expect.any(Function));
      expect(vi.mocked(EventsOn)).toHaveBeenCalledWith('editor-detection-complete', expect.any(Function));
    });

    test('editor-detection-start sets detectingEditors=true, progress=0, clears detectedEditors', () => {
      const avail = makeDefaultEditorAvailability();
      const status = makeDefaultEditorDetectionStatus();
      status.detectedEditors = ['vscode', 'sublime'];
      status.detectionProgress = 73;
      status.detectionComplete = true;
      status.totalAvailable = 9;

      subscribeToEditorDetectionEvents(avail, status);
      const cb = getEventCallback('editor-detection-start');
      expect(cb).toBeDefined();
      cb?.({ message: 'Starting now' });

      expect(status.detectionComplete).toBe(false);
      expect(status.detectingEditors).toBe(true);
      expect(status.detectionProgress).toBe(0);
      expect(status.detectedEditors).toEqual([]);
      expect(status.totalAvailable).toBe(0);
      expect(status.message).toBe('Starting now');
      expect(status.availableEditors).toBe(avail);
    });

    test('editor-detection-start falls back to default message when payload lacks one', () => {
      const avail = makeDefaultEditorAvailability();
      const status = makeDefaultEditorDetectionStatus();
      subscribeToEditorDetectionEvents(avail, status);

      getEventCallback('editor-detection-start')?.({});

      expect(status.message).toBe('Starting editor detection...');
    });

    test('editor-detection-start tolerates non-object payload', () => {
      const avail = makeDefaultEditorAvailability();
      const status = makeDefaultEditorDetectionStatus();
      subscribeToEditorDetectionEvents(avail, status);

      expect(() => getEventCallback('editor-detection-start')?.(null)).not.toThrow();
      expect(status.detectionProgress).toBe(0);
      expect(status.detectedEditors).toEqual([]);
      expect(status.message).toBe('Starting editor detection...');
    });

    test('editor-detection-progress rounds progress to int and pushes new editors', () => {
      const avail = makeDefaultEditorAvailability();
      const status = makeDefaultEditorDetectionStatus();
      subscribeToEditorDetectionEvents(avail, status);
      const cb = getEventCallback('editor-detection-progress');

      cb?.({ progress: 33.7, editor: 'vscode', available: true, message: 'Found vscode' });

      expect(status.detectionProgress).toBe(34);
      expect(status.detectedEditors).toEqual(['vscode']);
      expect(status.message).toBe('Found vscode');
    });

    test('editor-detection-progress dedups already-present editors', () => {
      const avail = makeDefaultEditorAvailability();
      const status = makeDefaultEditorDetectionStatus();
      subscribeToEditorDetectionEvents(avail, status);
      const cb = getEventCallback('editor-detection-progress');

      cb?.({ progress: 10, editor: 'vscode', available: true });
      cb?.({ progress: 40, editor: 'vscode', available: true });

      expect(status.detectedEditors).toEqual(['vscode']);
      expect(status.detectionProgress).toBe(40);
    });

    test('editor-detection-progress pushes distinct editors in order', () => {
      const avail = makeDefaultEditorAvailability();
      const status = makeDefaultEditorDetectionStatus();
      subscribeToEditorDetectionEvents(avail, status);
      const cb = getEventCallback('editor-detection-progress');

      cb?.({ progress: 10, editor: 'vscode', available: true });
      cb?.({ progress: 50, editor: 'sublime', available: true });
      cb?.({ progress: 80, editor: 'vim', available: true });

      expect(status.detectedEditors).toEqual(['vscode', 'sublime', 'vim']);
    });

    test('editor-detection-progress does not push when available is falsey', () => {
      const avail = makeDefaultEditorAvailability();
      const status = makeDefaultEditorDetectionStatus();
      subscribeToEditorDetectionEvents(avail, status);
      const cb = getEventCallback('editor-detection-progress');

      cb?.({ progress: 30, editor: 'vim', available: false });

      expect(status.detectedEditors).toEqual([]);
      expect(status.detectionProgress).toBe(30);
    });

    test('editor-detection-progress does not push when editor is missing or non-string', () => {
      const avail = makeDefaultEditorAvailability();
      const status = makeDefaultEditorDetectionStatus();
      subscribeToEditorDetectionEvents(avail, status);
      const cb = getEventCallback('editor-detection-progress');

      cb?.({ progress: 30, available: true });
      cb?.({ progress: 40, editor: 42, available: true });

      expect(status.detectedEditors).toEqual([]);
    });

    test('editor-detection-progress defaults progress to 0 for non-number and message to default', () => {
      const avail = makeDefaultEditorAvailability();
      const status = makeDefaultEditorDetectionStatus();
      subscribeToEditorDetectionEvents(avail, status);

      getEventCallback('editor-detection-progress')?.({});

      expect(status.detectionProgress).toBe(0);
      expect(status.message).toBe('Detecting editors...');
    });

    test('editor-detection-complete sets complete=true, detecting=false, progress=100, totalAvailable=totalFound', () => {
      const avail = makeDefaultEditorAvailability();
      const status = makeDefaultEditorDetectionStatus();
      subscribeToEditorDetectionEvents(avail, status);
      const cb = getEventCallback('editor-detection-complete');

      cb?.({ totalFound: 5 });

      expect(status.detectionComplete).toBe(true);
      expect(status.detectingEditors).toBe(false);
      expect(status.detectionProgress).toBe(100);
      expect(status.totalAvailable).toBe(5);
      expect(status.message).toBe('Detection complete! Found 5 editor(s).');
    });

    test('editor-detection-complete defaults totalFound to 0 when missing or non-number', () => {
      const avail = makeDefaultEditorAvailability();
      const status = makeDefaultEditorDetectionStatus();
      subscribeToEditorDetectionEvents(avail, status);
      const cb = getEventCallback('editor-detection-complete');

      cb?.({});
      expect(status.totalAvailable).toBe(0);
      expect(status.message).toBe('Detection complete! Found 0 editor(s).');

      cb?.({ totalFound: 'many' });
      expect(status.totalAvailable).toBe(0);
    });

    test('returned teardown invokes every EventsOn cleanup', () => {
      const avail = makeDefaultEditorAvailability();
      const status = makeDefaultEditorDetectionStatus();
      const cleanup = subscribeToEditorDetectionEvents(avail, status);

      // 3 subscriptions registered, each returning the shared cleanup spy.
      expect(vi.mocked(EventsOn)).toHaveBeenCalledTimes(3);
      mocks.cleanupSpy.mockClear();

      cleanup();

      expect(mocks.cleanupSpy).toHaveBeenCalledTimes(3);
    });
  });

  describe('startEditorDetection', () => {
    test('subscribes to events and calls GetEditorDetectionStatus once', async () => {
      const avail = makeDefaultEditorAvailability();
      const status = makeDefaultEditorDetectionStatus();

      const cleanup = startEditorDetection(avail, status);

      expect(typeof cleanup).toBe('function');
      expect(vi.mocked(EventsOn)).toHaveBeenCalledTimes(3);
      // The pull-based status check fires asynchronously.
      await flush();
      expect(vi.mocked(GetEditorDetectionStatus)).toHaveBeenCalledTimes(1);
      cleanup();
    });

    test('applies pulled status: merges availableEditors, sets totalAvailable and complete flags', async () => {
      const returned = {
        availableEditors: { vscode: true, systemdefault: false },
        totalAvailable: 3,
      };
      vi.mocked(GetEditorDetectionStatus).mockResolvedValue(
        returned as unknown as Record<string, unknown>,
      );

      const avail = makeDefaultEditorAvailability();
      const status = makeDefaultEditorDetectionStatus();
      startEditorDetection(avail, status);

      await flush();

      expect(avail.vscode).toBe(true);
      expect(avail.systemdefault).toBe(false);
      expect(status.availableEditors).toBe(avail);
      expect(status.totalAvailable).toBe(3);
      expect(status.detectionComplete).toBe(true);
      expect(status.detectingEditors).toBe(false);
      expect(status.message).toBe('Detection complete! Found 3 editor(s).');
    });

    test('handles status without availableEditors (skips merge, still completes)', async () => {
      vi.mocked(GetEditorDetectionStatus).mockResolvedValue({} as Record<string, unknown>);

      const avail = makeDefaultEditorAvailability();
      const status = makeDefaultEditorDetectionStatus();
      startEditorDetection(avail, status);

      await flush();

      expect(status.detectionComplete).toBe(true);
      expect(status.detectingEditors).toBe(false);
      expect(status.totalAvailable).toBe(0);
      // No totalAvailable field in payload -> message left untouched.
      expect(status.message).toBe(makeDefaultEditorDetectionStatus().message);
    });

    test('does not apply status when GetEditorDetectionStatus resolves null', async () => {
      vi.mocked(GetEditorDetectionStatus).mockResolvedValue(
        null as unknown as Record<string, unknown>,
      );

      const avail = makeDefaultEditorAvailability();
      const status = makeDefaultEditorDetectionStatus();
      startEditorDetection(avail, status);

      await flush();

      expect(status.detectionComplete).toBe(false);
      expect(status.detectingEditors).toBe(true);
    });

    test('swallows GetEditorDetectionStatus rejection and logs without throwing', async () => {
      vi.mocked(GetEditorDetectionStatus).mockRejectedValue(new Error('boom'));
      const errSpy = vi.spyOn(console, 'error').mockImplementation(() => {});

      const avail = makeDefaultEditorAvailability();
      const status = makeDefaultEditorDetectionStatus();

      expect(() => startEditorDetection(avail, status)).not.toThrow();

      await flush();

      expect(errSpy).toHaveBeenCalledWith(
        'Failed to fetch editor detection status:',
        expect.any(Error),
      );
      errSpy.mockRestore();
    });
  });
});
