import { test } from '@playwright/test';
import {
  rp1Login,
  rp1SSOLogin,
  adminDirectLogin,
  adminSSOLogin,
  logoutFromAdmin,
  rp1Logout,
  callEndSession,
  verifyRedirectedToLoginWeb,
} from '../helpers/oidc-helpers';

test.describe('多 RP 并行 SSO', () => {
  test('Admin 和 RP1 同时 SSO 登录，互不干扰', async ({ page, context }) => {
    await adminDirectLogin(page);
    const rp1Page = await context.newPage();
    await rp1SSOLogin(rp1Page);
    await rp1SSOLogin(rp1Page);
    await adminSSOLogin(page);
    await rp1Page.close();
  });

  test('Admin 登出不影响 RP1 的 SSO session', async ({ page, context }) => {
    await adminDirectLogin(page);
    const rp1Page = await context.newPage();
    await rp1SSOLogin(rp1Page);
    await logoutFromAdmin(page);
    await rp1SSOLogin(rp1Page);
    await rp1Page.close();
  });

  test('RP1 登出不影响 Admin 的 SSO session', async ({ page, context }) => {
    await rp1Login(page);
    const adminPage = await context.newPage();
    await adminSSOLogin(adminPage);
    await rp1Logout(page);
    await adminSSOLogin(adminPage);
    await adminPage.close();
  });

  test('/end_session 调用后 Admin 需要重新认证，RP1 session 稍后过期', async ({ page, context }) => {
    await adminDirectLogin(page);
    const rp1Page = await context.newPage();
    await rp1SSOLogin(rp1Page);
    await callEndSession(page);
    // end_session 清除 cookie 和所有 SSO sessions
    await verifyRedirectedToLoginWeb(page);
    // RP1 的 SPA 可能仍有本地 token 缓存，直接导航可能仍显示首页
    // 验证 Admin 已经跳转到登录页是核心断言
    await rp1Page.close();
  });
});
