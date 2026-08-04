// Toast notification types.

export type ToastType = "success" | "error" | "warning" | "info";

export interface Toast {
  id: string;
  title: string;
  message: string;
  type: ToastType;
  duration: number;
  timer: number | null;
  paused: boolean;
  remaining: number; // Remaining ms on the current timer (updated on pause)
  startedAt: number; // Timestamp when the current timer was started
}

export interface ToastOptions {
  title?: string;
  type?: ToastType;
  duration?: number;
}

export interface ToastStore {
  toasts: Toast[];
}
