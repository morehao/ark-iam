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

  test('Admin 全局登出后 RP1 需重新认证', async ({ page, context }) => {
    await adminDirectLogin(page);
    const rp1Page = await context.newPage();
    await rp1SSOLogin(rp1Page);
    await logoutFromAdmin(page);
    await rp1Login(rp1Page);
    await rp1Page.close();
  });

  test('RP1 全局登出后 Admin 需重新认证', async ({ page, context }) => {
    await rp1Login(page);
    const adminPage = await context.newPage();
    await adminSSOLogin(adminPage);
    await rp1Logout(page);
    await adminDirectLogin(adminPage);
    await adminPage.close();
  });

  test('/end_session 调用后 Admin 需要重新认证，RP1 也需重新认证', async ({ page, context }) => {
    await adminDirectLogin(page);
    const rp1Page = await context.newPage();
    await rp1SSOLogin(rp1Page);
    await callEndSession(page);
    // end_session 清除 cookie 和所有 SSO sessions
    await verifyRedirectedToLoginWeb(page);
    // RP1 也需重新认证
    await rp1Login(rp1Page);
    await rp1Page.close();
  });
});
