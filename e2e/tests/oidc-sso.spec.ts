import { test, expect } from '@playwright/test';
import {
  rp1Login,
  rp1SSOLogin,
  logoutFromAdmin,
  adminDirectLogin,
  adminSSOLogin,
  rp1Logout,
  callEndSession,
  clearSSOCookie,
  verifyRedirectedToLoginWeb,
} from '../helpers/oidc-helpers';

test.describe('OIDC SSO E2E', () => {
  test('RP1 首次登录：访问 SSO 测试应用 → 自动跳转 login-web → 填写凭证 → 回调展示首页', async ({ page }) => {
    await rp1Login(page);
  });

  test('Admin 直接登录：访问管理平台 → 自动跳转 login-web → 填写凭证 → 进入仪表盘', async ({ page }) => {
    await adminDirectLogin(page);
  });

  test('Admin 登录后 → SSO 测试应用免密登录（同一 context：共享 SSO session）', async ({ page, context }) => {
    await adminDirectLogin(page);
    const rp1Page = await context.newPage();
    await rp1SSOLogin(rp1Page);
    await rp1Page.close();
  });

  test('Admin 登出后 → SSO 测试应用仍可 SSO 免密登录（SSO session 未被清除）', async ({ page, context }) => {
    await adminDirectLogin(page);
    const rp1Page = await context.newPage();
    await rp1SSOLogin(rp1Page);
    await logoutFromAdmin(page);
    // Admin 登出只清除自身 token，SSO session 仍然有效，RP1 可以直接 SSO
    await rp1SSOLogin(rp1Page);
    await rp1Page.close();
  });

  test('Admin 直接登录 → 登出 → 重新登录（完整认证）', async ({ page }) => {
    await adminDirectLogin(page);
    await logoutFromAdmin(page);
    // 登出后 SSO session 仍保留，重新登录走 SSO 免密流程
    // 但 end_session 可能清除 SSO cookie 有延迟，使用完整登录流程
    await adminDirectLogin(page);
  });

  test('RP1 登录后 → Admin 管理平台静默 SSO 免密登录', async ({ page, context }) => {
    await rp1Login(page);
    const adminPage = await context.newPage();
    await adminSSOLogin(adminPage);
    await adminPage.close();
  });

  test('RP1 登录 → Admin SSO → Admin 登出 → RP1 仍可 SSO 免密登录', async ({ page, context }) => {
    await rp1Login(page);
    const adminPage = await context.newPage();
    await adminSSOLogin(adminPage);
    await logoutFromAdmin(adminPage);
    // Admin 登出只清除自身 token，SSO session 仍然有效
    await rp1SSOLogin(page);
    await adminPage.close();
  });

  test('双向 SSO：Admin 登录 → RP1 SSO → Admin 登出 → RP1 仍可 SSO 免密登录', async ({ page, context }) => {
    await adminDirectLogin(page);
    const rp1Page = await context.newPage();
    await rp1SSOLogin(rp1Page);
    await logoutFromAdmin(page);
    // Admin 登出只清除自身 token，SSO session 仍然有效
    await rp1SSOLogin(rp1Page);
    await rp1Page.close();
  });

  test('RP1 自身登出后再次 SSO 免密登录（RP 登出只清自身 token，SSO session 仍在）', async ({ page }) => {
    await rp1Login(page);
    await rp1Logout(page);
    await rp1SSOLogin(page);
  });

  test('RP1 自身登出后，Admin 的 SSO session 仍有效', async ({ page, context }) => {
    await rp1Login(page);
    const adminPage = await context.newPage();
    await adminSSOLogin(adminPage);
    await rp1Logout(page);
    await adminSSOLogin(adminPage);
    await adminPage.close();
  });

  test('Admin 登出（logout）后重新登录应重定向到 login-web', async ({ page }) => {
    await adminDirectLogin(page);
    await logoutFromAdmin(page);
    // 登出后访问 Admin，由于 SSO session 仍可能存在，可能走 SSO 免密或重定向到 login-web
    await page.goto('http://localhost:3000/', { waitUntil: 'domcontentloaded', timeout: 15000 });
    const url = page.url();
    // 登出后应跳转到 login-web 或 Admin 自带登录页
    const needsLogin = url.includes('localhost:3003') || url.includes('/login');
    expect(needsLogin).toBe(true);
  });

  test('/end_session 后当前页面需重新认证（全局清除由 multi-rp-sso 覆盖）', async ({ page }) => {
    await adminDirectLogin(page);
    await callEndSession(page);
    await verifyRedirectedToLoginWeb(page);
  });

  test('/logged-out 清除 cookie 后需重新输入凭证', async ({ page }) => {
    await rp1Login(page);
    await clearSSOCookie(page);
    await verifyRedirectedToLoginWeb(page);
  });
});