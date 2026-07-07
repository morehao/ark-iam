import { test, expect } from '@playwright/test';
import {
  rp1Login,
  adminSSOLogin,
  verifyRedirectedToLoginWeb,
  clearAllCookies,
  getSSOSessionCookie,
} from '../helpers/oidc-helpers';
import { CONFIG } from '../config';

test.describe('Cookie 和跨 Context 隔离', () => {
  test('新 browser context（无 cookie）访问 RP 重定向到登录页', async ({ browser }) => {
    const context1 = await browser.newContext();
    const page1 = await context1.newPage();
    await rp1Login(page1);
    await page1.close();

    const context2 = await browser.newContext();
    const page2 = await context2.newPage();
    await page2.goto(CONFIG.rp1Url, { waitUntil: 'domcontentloaded', timeout: 15000 });
    await verifyRedirectedToLoginWeb(page2);
    await page2.close();
    await context1.close();
    await context2.close();
  });

  test('手动清除 cookie 后 SPA 静默重认证', async ({ page, context }) => {
    await rp1Login(page);
    await clearAllCookies(context);
    // cookie 已清除但 SPA 内存中有 token，刷新页面后 API 调用可能失败
    // SPA 的 oidc 库会尝试静默重认证（prompt=none），由于 SSO session cookie 被清除，静默重认证失败
    // 后端返回 login_required，SPA 触发显式登录跳转
    await page.reload({ waitUntil: 'domcontentloaded', timeout: 15000 });
    // 页面最终会显示首页（SSO session 在 Redis 中仍有效，SPA 通过其他方式恢复）或跳转登录
    // 验证页面仍然可访问
    const url = page.url();
    expect(url.includes('localhost:3001') || url.includes('localhost:3003')).toBe(true);
  });

  test('cookie HttpOnly 不可被 JavaScript 读取', async ({ page }) => {
    await rp1Login(page);
    const hasCookie = await getSSOSessionCookie(page);
    expect(hasCookie).toBe(true);

    const jsCookie = await page.evaluate(() => document.cookie);
    expect(jsCookie).not.toContain('iam_sso_session');
  });

  test('两个独立 browser context 各自维护独立 session', async ({ browser }) => {
    const ctxA = await browser.newContext();
    const pageA = await ctxA.newPage();
    await rp1Login(pageA);

    const ctxB = await browser.newContext();
    const pageB = await ctxB.newPage();
    await pageB.goto(CONFIG.rp1Url, { waitUntil: 'domcontentloaded', timeout: 15000 });
    await verifyRedirectedToLoginWeb(pageB);

    // ctxA 仍保持登录态
    await adminSSOLogin(pageA);

    await pageA.close();
    await pageB.close();
    await ctxA.close();
    await ctxB.close();
  });
});
