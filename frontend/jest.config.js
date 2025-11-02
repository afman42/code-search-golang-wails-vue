export default {
  preset: 'ts-jest/presets/default-esm',
  testEnvironment: 'jsdom',
  setupFilesAfterEnv: ['<rootDir>/tests/setup.ts'],
  moduleFileExtensions: ['vue', 'ts', 'js', 'json'],
  transform: {
    '^.+\\.vue$': '@vue/vue3-jest',
    '^.+\\.ts$': 'ts-jest',
    '^.+\\.js$': 'babel-jest',
  },
  moduleNameMapper: {
    '^@/(.*)$': '<rootDir>/src/$1',
    '^../../wailsjs/go/main/App$': '<rootDir>/tests/__mocks__/wailsAppMock.js',
    '^../../wailsjs/runtime/runtime$': '<rootDir>/tests/__mocks__/wailsRuntimeMock.js',
  },
  transformIgnorePatterns: [
    '/node_modules/(?!@vue)',
  ],
  testMatch: ['**/tests/**/*.spec.(ts|js)?(x)'],
  collectCoverage: true,
  collectCoverageFrom: [
    'src/**/*.{ts,vue}',
    '!src/main.ts',
    '!src/**/types/*',
  ],
  globals: {
    'ts-jest': {
      useESM: true,
    },
  },
};