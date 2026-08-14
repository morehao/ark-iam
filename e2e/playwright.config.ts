import { defineConfig } from '@playwright/test';

export default defineConfig({
  testDir: './tests',
  timeout: 120000,
  expect: {
    timeout: 10000,
  },
  globalSetup: './global-setup.ts',
  globalTeardown: './global-teardown.ts',
  use: {
    baseURL: 'http://localhost:3002',
    headless: true,
    browserName: 'chromium',
    ignoreHTTPSErrors: true,
    viewport: { width: 1280, height: 720 },
  },
  retries: 0,
  workers: 1,
  reporter: 'list',
});
