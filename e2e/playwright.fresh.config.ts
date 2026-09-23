import { defineConfig } from '@playwright/test';

// 「全新库」验收配置：**刻意不挂 globalSetup/globalTeardown**。
//
// 主配置（playwright.config.ts）的 globalSetup 会先把系统初始化掉（多数用例都要求
// "已初始化"这个前提）；而本配置下的用例要验证的恰恰是**初始化本身**——运维在页面上
// 把三步向导走完。因此它要求调用方自己提供一个**全新库**的后端，跑完即应作废该库。
//
// 用法（见 README「全新库验收」）：
//   BOOTSTRAP_TOKEN=e2e-bootstrap-token APP_CONFIG_PATH=<指向空库的配置> <backend>
//   pnpm test:fresh
export default defineConfig({
  testDir: './tests-fresh',
  timeout: 120000,
  expect: { timeout: 10000 },
  use: {
    baseURL: 'http://localhost:4000',
    headless: true,
    browserName: 'chromium',
    // 同主配置：可用系统 Chrome/Edge 内核，省掉浏览器下载
    channel: (process.env.E2E_BROWSER_CHANNEL || undefined) as 'chrome' | 'msedge' | undefined,
    viewport: { width: 1280, height: 720 },
  },
  retries: 0,
  workers: 1,
  reporter: 'list',
});
