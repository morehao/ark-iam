import { test, expect } from '@playwright/test';
import { CONFIG } from '../config';
import {
  verifyRp1HomePage,
  verifyTokenDetails,
  logoutFromAdmin,
  adminDirectLogin,
  adminSSOLogin,
  adminRequiresLoginAfterLogout,
  rp1Login,
  rp1SSOLogin,
  verifyRp1RequiresLogin,
} from '../helpers/oidc-helpers';

test.describe('OIDC SSO E2E', () => {
  test('RP1 首次登录：点击"使用 IAM 登录" → 跳转登录页 → 填写凭证 → 回调展示项目管理面板', async ({ page }) => {
    await rp1Login(page);
  });

  test('RP1 Token 详情：查看 Token → 获取 UserInfo → 刷新 Token → 返回主页', async ({ page }) => {
    await rp1Login(page);
    await verifyTokenDetails(page);
  });

  test('管理平台 SSO 自动登录：RP1 登录后 → 打开管理平台 → 静默认证直接进仪表盘', async ({ page, context }) => {
    await rp1Login(page);
    const adminPage = await context.newPage();
    await adminSSOLogin(adminPage);
    await adminPage.close();
  });

  test('管理平台登出后 SSO session 已清除', async ({ page, context }) => {
    await rp1Login(page);
    const adminPage = await context.newPage();
    await adminSSOLogin(adminPage);
    await logoutFromAdmin(adminPage);
    // 登出后，SSO session 应被清除，管理员需要重新登录（跳转到 login-web）
    await adminRequiresLoginAfterLogout(adminPage);
    await adminPage.close();
  });

  test('Admin 直接登录：访问管理平台 → 静默授权跳转 login-web → 填写凭证 → 进入仪表盘', async ({ page }) => {
    await adminDirectLogin(page);
  });

  test('Admin 登录后 → SSO 测试应用免密登录（同一 context）', async ({ page, context }) => {
    await adminDirectLogin(page);
    const rp1Page = await context.newPage();
    await rp1SSOLogin(rp1Page);
    await rp1Page.close();
  });

  test('Admin 登出后 → SSO 测试应用需重新登录', async ({ page, context }) => {
    await adminDirectLogin(page);
    const rp1Page = await context.newPage();
    await rp1SSOLogin(rp1Page);
    await logoutFromAdmin(page);
    await verifyRp1RequiresLogin(rp1Page);
    await rp1Page.close();
  });

  test('RP1 登录后 → Admin 管理平台静默 SSO 免密登录', async ({ page, context }) => {
    await rp1Login(page);
    const adminPage = await context.newPage();
    await adminSSOLogin(adminPage);
    await adminPage.close();
  });

  test('RP1 登录 → Admin SSO → Admin 登出 → RP1 需重新登录', async ({ page, context }) => {
    await rp1Login(page);
    const adminPage = await context.newPage();
    await adminSSOLogin(adminPage);
    await logoutFromAdmin(adminPage);
    await verifyRp1RequiresLogin(page);
    await adminPage.close();
  });

  test('双向 SSO：Admin 登录 → RP1 SSO → Admin 登出 → RP1 需重登录；重新 Admin 登录 → RP1 再次 SSO', async ({ page, context }) => {
    // Round 1: Admin → RP1 SSO
    await adminDirectLogin(page);
    const rp1Page = await context.newPage();
    await rp1SSOLogin(rp1Page);
    await logoutFromAdmin(page);
    await verifyRp1RequiresLogin(rp1Page);

    // Round 2: 重新 Admin 登录 → RP1 SSO
    await adminDirectLogin(page);
    await rp1SSOLogin(rp1Page);
    await rp1Page.close();
  });
});
