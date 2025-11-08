import { reactive, readonly } from 'vue';

interface Toast {
  id: string;
  title: string;
  message: string;
  type: 'success' | 'error' | 'warning' | 'info';
  duration: number;
  timer: number | null;
  paused: boolean;
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

    // Remove the toast from the state
    const index = state.toasts.indexOf(toast);
    if (index > -1) {
      state.toasts.splice(index, 1);
    }
  };

  // Function to pause a toast timer (on hover)
  const pauseToast = (toast: Toast) => {
    if (toast.timer && !toast.paused) {
      clearTimeout(toast.timer);
      toast.paused = true;
    }
  };

  // Function to resume a toast timer (on mouse leave)
  const resumeToast = (toast: Toast) => {
    if (toast.paused) {
      const remainingTime = toast.duration;
      toast.timer = window.setTimeout(() => {
        removeToast(toast);
      }, remainingTime) as unknown as number;
      toast.paused = false;
    }
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
    state.toasts = [];
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