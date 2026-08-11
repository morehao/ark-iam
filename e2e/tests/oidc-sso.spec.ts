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
  test('RP1 首次登录：访问 统一登录演示应用 → 自动跳转 login-web → 填写凭证 → 回调展示首页', async ({ page }) => {
    await rp1Login(page);
  });

  test('Admin 直接登录：访问管理平台 → 自动跳转 login-web → 填写凭证 → 进入仪表盘', async ({ page }) => {
    await adminDirectLogin(page);
  });

  test('Admin 登录后 → 统一登录演示应用免密登录（同一 context：共享 SSO session）', async ({ page, context }) => {
    await adminDirectLogin(page);
    const rp1Page = await context.newPage();
    await rp1SSOLogin(rp1Page);
    await rp1Page.close();
  });

  test('Admin 全局登出后 → 统一登录演示应用需重新认证', async ({ page, context }) => {
    await adminDirectLogin(page);
    const rp1Page = await context.newPage();
    await rp1SSOLogin(rp1Page);
    await logoutFromAdmin(page);
    // Admin 全局登出清除 SSO session，RP1 需重新认证
    await rp1Login(rp1Page);
    await rp1Page.close();
  });

  test('Admin 直接登录 → 登出 → 重新登录（全局登出后需完整认证）', async ({ page }) => {
    await adminDirectLogin(page);
    await logoutFromAdmin(page);
    // 全局登出已清除 SSO session，重新登录需完整认证流程
    await adminDirectLogin(page);
  });

  test('RP1 登录后 → Admin 管理平台静默 SSO 免密登录', async ({ page, context }) => {
    await rp1Login(page);
    const adminPage = await context.newPage();
    await adminSSOLogin(adminPage);
    await adminPage.close();
  });

  test('RP1 登录 → Admin SSO → Admin 全局登出 → 需重新认证', async ({ page, context }) => {
    await rp1Login(page);
    const adminPage = await context.newPage();
    await adminSSOLogin(adminPage);
    await logoutFromAdmin(adminPage);
    // Admin 全局登出清除 SSO session，RP1 需重新认证
    await rp1Login(page);
    await adminPage.close();
  });

  test('双向 SSO：Admin 登录 → RP1 SSO → Admin 全局登出 → RP1 需重新认证', async ({ page, context }) => {
    await adminDirectLogin(page);
    const rp1Page = await context.newPage();
    await rp1SSOLogin(rp1Page);
    await logoutFromAdmin(page);
    // Admin 全局登出清除 SSO session，RP1 需重新认证
    await rp1Login(rp1Page);
    await rp1Page.close();
  });

  test('RP1 自身全局登出后需重新登录', async ({ page }) => {
    await rp1Login(page);
    await rp1Logout(page);
    await rp1Login(page);
  });

  test('RP1 自身全局登出后，Admin 需重新认证', async ({ page, context }) => {
    await rp1Login(page);
    const adminPage = await context.newPage();
    await adminSSOLogin(adminPage);
    await rp1Logout(page);
    // RP1 全局登出清除 SSO session，Admin 需重新认证
    await adminDirectLogin(adminPage);
    await adminPage.close();
  });

  test('Admin 全局登出后再访问 Admin 需登录', async ({ page }) => {
    await adminDirectLogin(page);
    await logoutFromAdmin(page);
    // 全局登出后 SSO session 已清除，再次访问 Admin 需完整认证
    await adminDirectLogin(page);
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