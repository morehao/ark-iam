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
    // session 过期后 SPA 页面上某些 API 调用会返回 401，触发 re-login
    // 验证页面已不再能正常访问用户信息（token 已过期）
    try {
      await page.waitForFunction(
        () => document.body.innerText.includes('token已失效'),
        { timeout: 20000 }
      );
    } catch {
      // 部分场景下前端静默重新登录了，验证 session 确实过期了
    }
    // 强制重新访问验证需要重新认证
    await page.goto(CONFIG.rp1Url, { waitUntil: 'networkidle', timeout: 15000 });
    // 由于 SSO session 也可过期，页面将重定向到 login-web
    const url = page.url();
    const needsAuth = url.includes('localhost:3003') || url.includes('/login');
    expect(needsAuth).toBe(true);
  });

  test('SSO session 过期后 Admin 也需要重新认证', async ({ page }) => {
    await adminDirectLogin(page);
    await waitForSSOSessionExpiry();
    try {
      await page.waitForFunction(
        () => document.body.innerText.includes('token已失效'),
        { timeout: 20000 }
      );
    } catch {
      // 可能已静默重新认证
    }
    await page.goto(CONFIG.platformAdminUrl, { waitUntil: 'networkidle', timeout: 15000 });
    const url = page.url();
    const needsAuth = url.includes('localhost:3003') || url.includes('/login');
    expect(needsAuth).toBe(true);
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
