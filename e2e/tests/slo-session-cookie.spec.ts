import { test, expect } from '@playwright/test';
import { CONFIG } from '../config';

// 全局登出（SLO）场景：验证 SSO 会话 cookie 被清除、兄弟应用不再共享免密 SSO。
// 请求粒度 SSO 活性校验（WithOIDCSSOValidation）下的"刷新即实时登出"见
// real-time-logout.spec.ts；本用例聚焦 cookie 层面的全局登出证据。
test('全局登出后，SSO 会话失效且兄弟应用需重新认证', async ({ page, context }) => {
  const identifier = 'admin';
  const password = 'admin123';

  // 1. 首次访问 3001（管理平台），应跳转 login-web（3000）并带 authRequestID
  await page.goto('http://localhost:3001/', { waitUntil: 'domcontentloaded', timeout: 20000 });
  await page.waitForURL((url) => url.port === '3000' && url.pathname === '/login' && url.searchParams.has('authRequestID'), { timeout: 30000 });
  await page.fill('#identifier', identifier);
  await page.fill('#password', password);
  await page.click('button[type="submit"]');
  // 回到管理平台仪表盘
  await page.waitForURL((url) => url.port === '3001' && !url.pathname.includes('/auth/callback'), { timeout: 30000 });
  await expect(page.getByText('仪表盘', { exact: true }).first()).toBeVisible({ timeout: 30000 });

  // 2. 新开标签页访问 3002（租户管理平台），应免密登录（共享 SSO session）
  const rp1 = await context.newPage();
  await rp1.goto('http://localhost:3002/', { waitUntil: 'domcontentloaded', timeout: 20000 });
  await expect(rp1.getByText('组织管理', { exact: true }).first()).toBeVisible({ timeout: 30000 });
  // SSO session cookie 已建立
  const ssoCookies = await context.cookies('http://localhost:8100');
  expect(ssoCookies.some((c) => c.name === 'iam_sso_session')).toBe(true);
  console.log('>>> 3002 SSO 免密登录成功');
  // 3. 在 3001 全局登出
  await page.evaluate(() => { localStorage.clear(); sessionStorage.clear(); });
  await page.goto('http://localhost:3001/', { waitUntil: 'domcontentloaded', timeout: 20000 });
  await expect(page.getByText('仪表盘', { exact: true }).first()).toBeVisible({ timeout: 30000 });
  await page.locator('.ant-avatar').click();
  await page.waitForTimeout(500);
  await page.locator('.ant-dropdown-menu-item', { hasText: '退出登录' }).click();
  // 全局登出后应跳到 login-web（3000）
  await page.waitForURL((url) => url.port === '3000', { timeout: 20000 });
  console.log('>>> 3001 全局登出完成，跳到 login-web');

  // 4. 全局登出后 SSO session cookie 应被清除，兄弟应用不再共享免密 SSO
  const ssoAfterLogout = await rp1.context().cookies('http://localhost:8100');
  expect(ssoAfterLogout.some((c) => c.name === 'iam_sso_session')).toBe(false);
  console.log('>>> 全局登出后 iam_sso_session SSO cookie 已清除');

  // 5. 清理兄弟应用本地 OIDC 用户（模拟该端本地会话已脱身），再访问 3002
  //    必须走完整 login-web，无法再静默 SSO —— 证明 SSO 会话已全局失效
  await rp1.goto('http://localhost:3002/', { waitUntil: 'domcontentloaded', timeout: 20000 });
  await rp1.evaluate(() => { localStorage.clear(); sessionStorage.clear(); });
  await rp1.context().clearCookies();
  await rp1.goto('http://localhost:3002/', { waitUntil: 'domcontentloaded', timeout: 20000 });
  await rp1.waitForURL((url) => url.port === '3000' && url.pathname === '/login', { timeout: 30000 });
  console.log('>>> 3002 需重新认证（全局登出生效，SSO 不再免密）');
  await rp1.close();
});
