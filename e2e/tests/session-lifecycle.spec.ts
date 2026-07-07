import { test, expect } from '@playwright/test';
import {
  rp1Login,
  rp1SSOLogin,
  adminDirectLogin,
  verifyRedirectedToLoginWeb,
  waitForSSOSessionExpiry,
  clearSSOCookie,
} from '../helpers/oidc-helpers';
import { CONFIG } from '../config';

test.describe('Session 生命周期', () => {
  test('SSO session 在 TTL 内有效 — Admin 登录后可免密 SSO 到 RP1', async ({ page, context }) => {
    await adminDirectLogin(page);
    const rp1Page = await context.newPage();
    await rp1SSOLogin(rp1Page);
    await rp1Page.close();
  });

  test('SSO session 过期后需重新输入凭证', async ({ page }) => {
    await rp1Login(page);
    await waitForSSOSessionExpiry();
    // session 过期后，重新访问 RP —— 如果 SPA 有缓存的 token，API 调用会返回 401
    await page.goto(CONFIG.rp1Url, { waitUntil: 'domcontentloaded', timeout: 15000 });
    // 页面将进入 SPA 路由，然后 API 调用失败触发重新认证
    // 验证过程：URL 可能最终落在 RP 首页（token 已失效但 SPA 加载了静态资源）、login-web 或 login 页
    const url = page.url();
    const needsAuth = url.includes('localhost:3003') || url.includes('/login');
    // 即使 URL 回落到 RP 首页，token 也已失效（body 包含 token已失效）
    const body = await page.evaluate(() => document.body.innerText).catch(() => '');
    expect(needsAuth || body.includes('token已失效')).toBe(true);
  });

  test('SSO session 过期后 Admin 也需要重新认证', async ({ page }) => {
    await adminDirectLogin(page);
    await waitForSSOSessionExpiry();
    await page.goto(CONFIG.platformAdminUrl, { waitUntil: 'domcontentloaded', timeout: 15000 });
    const url = page.url();
    const needsAuth = url.includes('localhost:3003') || url.includes('/login');
    const body = await page.evaluate(() => document.body.innerText).catch(() => '');
    expect(needsAuth || body.includes('token已失效')).toBe(true);
  });

  test('AuthRequest TTL 超时后 login-web 页面不崩溃', async ({ page }) => {
    await page.goto(CONFIG.rp1Url, { waitUntil: 'networkidle', timeout: 15000 });
    const url = page.url();
    if (url.includes('auth/callback') && url.includes('code=')) {
      await clearSSOCookie(page);
      await page.goto(CONFIG.rp1Url, { waitUntil: 'networkidle', timeout: 15000 });
    }
    // 等待 authRequest 过期（35秒 > 30秒 authRequestTTL）
    await page.waitForTimeout(35000);
    // 页面不应白屏或 500
    const body = await page.evaluate(() => document.body.innerText);
    expect(body.length).toBeGreaterThan(0);
  });
});
