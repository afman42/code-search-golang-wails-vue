import { reactive, readonly } from 'vue';

interface Toast {
  id: string;
  title: string;
  message: string;
  type: 'success' | 'error' | 'warning' | 'info';
  duration: number;
  timer: number | null;
  paused: boolean;
  remaining: number;  // Remaining ms on the current timer (updated on pause)
  startedAt: number;  // Timestamp when the current timer was started
}

interface ToastOptions {
  title?: string;
  type?: 'success' | 'error' | 'warning' | 'info';
  duration?: number;
}

interface ToastStore {
  toasts: Toast[];
}

const state: ToastStore = reactive({
  toasts: [],
});

const TOAST_DEFAULT_DURATION = 5000; // 5 seconds

export function useToast() {
  // Function to create a unique ID for each toast
  const createId = () => {
    return Date.now().toString(36) + Math.random().toString(36).substr(2);
  };

  // Function to add a new toast
  const addToast = (message: string, options: ToastOptions = {}) => {
    const id = createId();
    const {
      title = 'Notification',
      type = 'info',
      duration = TOAST_DEFAULT_DURATION,
    } = options;

    const toast: Toast = {
      id,
      title,
      message,
      type,
      duration,
      timer: null,
      paused: false,
      remaining: duration,
      startedAt: Date.now(),
    };

    // Add the toast to the state
    state.toasts.push(toast);

    // Create a timer to remove the toast after the specified duration
    if (duration > 0) {
      toast.timer = window.setTimeout(() => {
        removeToast(toast);
      }, duration) as unknown as number;
    }

    return toast.id;
  };

  // Function to remove a toast
  const removeToast = (toast: Toast) => {
    // Clear the timer if it exists
    if (toast.timer) {
      clearTimeout(toast.timer);
    }

    // Find the mutable toast by ID (toast parameter may be a readonly proxy)
    const index = state.toasts.findIndex(t => t.id === toast.id);
    if (index > -1) {
      state.toasts.splice(index, 1);
    }
  };

  // Function to pause a toast timer (on hover)
  const pauseToast = (toast: Toast) => {
    // Find the mutable toast by ID (toast parameter may be a readonly proxy)
    const mutable = state.toasts.find(t => t.id === toast.id);
    if (!mutable || !mutable.timer || mutable.paused) return;
    clearTimeout(mutable.timer);
    // Track remaining time by subtracting elapsed from what was left
    const elapsed = Date.now() - mutable.startedAt;
    mutable.remaining = Math.max(0, mutable.remaining - elapsed);
    mutable.paused = true;
  };

  // Function to resume a toast timer (on mouse leave)
  const resumeToast = (toast: Toast) => {
    // Find the mutable toast by ID (toast parameter may be a readonly proxy)
    const mutable = state.toasts.find(t => t.id === toast.id);
    if (!mutable || !mutable.paused) return;
    mutable.startedAt = Date.now();
    mutable.timer = window.setTimeout(() => {
      removeToast(mutable);
    }, mutable.remaining) as unknown as number;
    mutable.paused = false;
  };

  // Convenience methods for different toast types
  const success = (message: string, title: string = 'Success') => {
    return addToast(message, { title, type: 'success' });
  };

  const error = (message: string, title: string = 'Error') => {
    return addToast(message, { title, type: 'error' });
  };

  const warning = (message: string, title: string = 'Warning') => {
    return addToast(message, { title, type: 'warning' });
  };

  const info = (message: string, title: string = 'Info') => {
    return addToast(message, { title, type: 'info' });
  };

  // Clear all toasts
  const clearAll = () => {
    state.toasts.forEach(toast => {
      if (toast.timer) {
        clearTimeout(toast.timer);
      }
    });
    state.toasts.splice(0, state.toasts.length);
  };

  return {
    toasts: readonly(state.toasts),
    addToast,
    removeToast,
    pauseToast,
    resumeToast,
    success,
    error,
    warning,
    info,
    clearAll,
  };
}

// Global toast instance to be used across the app
export const toastManager = useToast();