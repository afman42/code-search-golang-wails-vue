// Setup for jest tests
// This file is run before each test file

// Mock IntersectionObserver for CodeModal component
global.IntersectionObserver = class IntersectionObserver {
  constructor(callback: any, options?: any) {
    this.callback = callback;
    this.options = options;
  }

  callback: any;
  options: any;

  observe() {
    // Mock implementation
  }

  unobserve() {
    // Mock implementation
  }

  disconnect() {
    // Mock implementation
  }

  static toString() {
    return 'function IntersectionObserver() { [native code] }';
  }
};

// Mock document.execCommand for clipboard functionality
Object.defineProperty(document, 'execCommand', {
  value: jest.fn(() => true),
  writable: true,
});

// Ensure that the modules are properly mocked before each test
beforeEach(() => {
  jest.resetAllMocks();
});