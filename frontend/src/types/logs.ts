// Log entry types for the log streaming composable.

export interface LogEntry {
  timestamp: string;
  level: string;
  message: string;
}
