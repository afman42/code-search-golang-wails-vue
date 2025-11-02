// Setup for jest tests
// This file is run before each test file

// Mock all wailsjs functions globally
jest.mock('../wailsjs/go/main/App', () => ({
  SearchCode: jest.fn(),
  GetDirectoryContents: jest.fn(),
  SelectDirectory: jest.fn(),
  ShowInFolder: jest.fn(),
  SearchWithProgress: jest.fn(),
}));

// Mock the runtime functions as well
jest.mock('../wailsjs/runtime', () => ({
  EventsOn: jest.fn(),
}));