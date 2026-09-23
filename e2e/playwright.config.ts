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
    baseURL: 'http://localhost:4002',
    headless: true,
    browserName: 'chromium',
    // 默认用 playwright 自带的 chromium（`npx playwright install chromium`）。
    // 本机/CI 已装 Chrome 或 Edge 时可用系统内核，省掉一次浏览器下载（下载被墙的环境里这是唯一出路）：
    //   E2E_BROWSER_CHANNEL=chrome pnpm test
    channel: (process.env.E2E_BROWSER_CHANNEL || undefined) as 'chrome' | 'msedge' | undefined,
    ignoreHTTPSErrors: true,
    viewport: { width: 1280, height: 720 },
  },
  retries: 0,
  workers: 1,
  reporter: 'list',
});
