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

  test('手动清除 cookie 后需重新认证', async ({ page, context }) => {
    await rp1Login(page);
    await clearAllCookies(context);
    // cookie 已清除但 SPA 内存中有 token，刷新页面将触发 API 调用返回 401
    await page.reload({ waitUntil: 'networkidle', timeout: 15000 });
    // 清除 cookie 后 SPA 的 token 调用会失败，可能显示 token 失效或跳转到 login-web
    const url = page.url();
    const body = await page.evaluate(() => document.body.innerText);
    const needsAuth = url.includes('localhost:3003') || url.includes('/login') || body.includes('token已失效');
    expect(needsAuth).toBe(true);
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
