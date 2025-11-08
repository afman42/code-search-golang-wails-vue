export default {
  preset: "ts-jest/presets/default-esm",
  testEnvironment: "jsdom",
  testEnvironmentOptions: {
    customExportConditions: ["node", "node-addons"],
  },
  setupFilesAfterEnv: ["<rootDir>/tests/setup.ts"],
  moduleFileExtensions: ["vue", "ts", "js", "json"],
  transform: {
    "^.+\\.vue$": "@vue/vue3-jest",
    "^.+\\.ts$": "ts-jest",
    "^.+\\.js$": "babel-jest",
  },
  moduleNameMapper: {
    "^@/(.*)$": "<rootDir>/src/$1",
    "^../wailsjs/go/main/App$": "<rootDir>/tests/__mocks__/wailsjs/go/main/App.js",
    "^../../wailsjs/go/main/App$": "<rootDir>/tests/__mocks__/wailsjs/go/main/App.js",
    "^../../../wailsjs/go/main/App$": "<rootDir>/tests/__mocks__/wailsjs/go/main/App.js",
    "^../wailsjs/runtime$": "<rootDir>/tests/__mocks__/wailsjs/runtime/index.js",
    "^../../wailsjs/runtime$": "<rootDir>/tests/__mocks__/wailsjs/runtime/index.js",
    "^../../../wailsjs/runtime$": "<rootDir>/tests/__mocks__/wailsjs/runtime/index.js",
    "\\.css$": "<rootDir>/tests/__mocks__/styleMock.js",
  },
  transformIgnorePatterns: ["/node_modules/(?!@vue)"],
  testMatch: ["**/tests/**/*.spec.(ts|js)?(x)"],
  collectCoverage: true,
  collectCoverageFrom: ["src/**/*.{ts,vue}", "!src/main.ts", "!src/**/types/*"],
  globals: {
    "ts-jest": {
      useESM: true,
    },
  },
};
