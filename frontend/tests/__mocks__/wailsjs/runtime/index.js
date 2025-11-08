// Mock for Wails Runtime module
export const EventsOn = jest.fn().mockReturnValue(jest.fn());
export const EventsOnce = jest.fn().mockReturnValue(jest.fn());
export const EventsOff = jest.fn();
export const EventsEmit = jest.fn();