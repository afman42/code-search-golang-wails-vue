// Mock for Wails App module
export const SearchCode = jest.fn();
export const GetDirectoryContents = jest.fn();
export const SelectDirectory = jest.fn();
export const ShowInFolder = jest.fn();
export const SearchWithProgress = jest.fn().mockResolvedValue([]);
export const CancelSearch = jest.fn();
export const ReadFile = jest.fn();
export const ValidateDirectory = jest.fn();