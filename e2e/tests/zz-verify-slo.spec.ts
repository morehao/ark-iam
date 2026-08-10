import { test, expect } from '@playwright/test';

test('全局登出后，兄弟应用刷新需重新认证', async ({ page, context }) => {
  const identifier = 'admin';
  const password = 'admin123';

  // 1. 首次访问 3000（管理平台），应跳转 login-web 并带 authRequestID
  await page.goto('http://localhost:3000/', { waitUntil: 'networkidle', timeout: 20000 });
  await page.waitForURL((url) => url.hostname === 'localhost' && url.port === '3003' && url.pathname === '/login' && url.searchParams.has('authRequestID'), { timeout: 30000 });
  // 填登录凭证
  await page.fill('#identifier', identifier);
  await page.fill('#password', password);
  await page.click('button[type="submit"]');
  // 回到管理平台仪表盘
  await page.waitForFunction(() => document.body.innerText.includes('仪表盘'), { timeout: 30000 });

  // 2. 新开标签页访问 3001（sso-test-app），应免密登录（共享 SSO session）
  const rp1 = await context.newPage();
  await rp1.goto('http://localhost:3001/', { waitUntil: 'networkidle', timeout: 20000 });
  await rp1.waitForFunction(() => document.body.innerText.includes('用户信息') && document.body.innerText.includes('SSO 测试应用'), { timeout: 30000 });
  console.log('>>> 3001 免密登录成功');

  // 3. 在 3000 全局登出
  await page.evaluate(() => { localStorage.clear(); });
  await page.goto('http://localhost:3000/', { waitUntil: 'networkidle', timeout: 20000 });
  await page.waitForFunction(() => document.body.innerText.includes('仪表盘'), { timeout: 30000 });
  await page.locator('.ant-avatar').click();
  await page.waitForTimeout(500);
  await page.locator('.ant-dropdown-menu-item', { hasText: '退出登录' }).click();
  // 全局登出后应跳到 login-web
  await page.waitForURL((url) => url.hostname === 'localhost' && url.port === '3003', { timeout: 20000 });
  console.log('>>> 3000 全局登出完成，跳到 login-web');

  // 4. 刷新 3001，应触发 SSO 探活从而需重新认证
  await rp1.reload({ waitUntil: 'networkidle', timeout: 20000 });
  // 等待其跳转到 login-web（带 authRequestID），证明已登出
  await rp1.waitForURL((url) => url.hostname === 'localhost' && url.port === '3003' && url.pathname === '/login' && url.searchParams.has('authRequestID'), { timeout: 30000 });
  console.log('>>> 3001 刷新后需重新认证（全局登出生效）');
  await rp1.close();
});
