import { test, expect } from '@playwright/test';
import { CONFIG } from '../config';
import {
  fillLoginFormAndSubmit,
  verifyRp1HomePage,
  verifyTokenDetails,
  verifyPlatformAdminSSO,
} from '../helpers/oidc-helpers';

test.describe('OIDC SSO E2E', () => {
  test('RP1 首次登录：点击"使用 IAM 登录" → 跳转登录页 → 填写凭证 → 回调展示项目管理面板', async ({ page, context }) => {
    // 1) 打开 RP1
    await page.goto(CONFIG.rp1Url, { waitUntil: 'networkidle' });
    expect(page.url()).toBe(CONFIG.rp1Url);

    // 2) 点击"使用 IAM 登录"按钮
    await page.click('button', { timeout: 5000 });
    await page.waitForTimeout(2000);

    // 3) 验证跳转到 login-web 登录页
    expect(page.url()).toContain(CONFIG.loginWebUrl);
    expect(page.url()).toContain('authRequestID=');

    // 4) 填写凭证并提交
    await page.waitForSelector('#identifier', { timeout: 5000 });
    await page.fill('#identifier', CONFIG.identifier);
    await page.fill('#password', CONFIG.password);

    // 等待 OIDC 回调完成
    await page.click('button[type="submit"]');
    await page.waitForURL((url) => url.hostname === 'localhost' && url.searchParams.has('code'), { timeout: 20000 });
    await page.waitForTimeout(2000);

    // 5) 验证 RP1 回调 URL 带 code/state
    const callbackUrl = page.url();
    expect(callbackUrl).toContain('code=');
    expect(callbackUrl).toContain('state=');

    // 6) 验证项目管理面板
    await verifyRp1HomePage(page);
  });

  test('RP1 Token 详情：查看 Token → 获取 UserInfo → 刷新 Token → 返回主页', async ({ page, context }) => {
    // 先完成登录
    await page.goto(CONFIG.rp1Url, { waitUntil: 'networkidle' });
    await page.click('button', { timeout: 5000 });
    await page.waitForTimeout(2000);

    await page.waitForSelector('#identifier', { timeout: 5000 });
    await page.fill('#identifier', CONFIG.identifier);
    await page.fill('#password', CONFIG.password);
    await page.click('button[type="submit"]');
    await page.waitForURL((url) => url.hostname === 'localhost' && url.searchParams.has('code'), { timeout: 20000 });
    await page.waitForTimeout(2000);

    // 验证凭证页面先加载
    await page.waitForFunction(() => document.body.innerText.includes('项目管理面板'), { timeout: 10000 });

    // 验证 Token 详情
    await verifyTokenDetails(page);
  });

  test('管理平台 SSO 自动登录：RP1 登录后 → 打开管理平台 → 点击 IAM 账号登录 → 自动认证进仪表盘', async ({ page, context }) => {
    // RP1 登录
    await page.goto(CONFIG.rp1Url, { waitUntil: 'networkidle' });
    await page.click('button', { timeout: 5000 });
    await page.waitForTimeout(2000);

    await page.waitForSelector('#identifier', { timeout: 5000 });
    await page.fill('#identifier', CONFIG.identifier);
    await page.fill('#password', CONFIG.password);
    await page.click('button[type="submit"]');
    await page.waitForURL((url) => url.hostname === 'localhost' && url.searchParams.has('code'), { timeout: 20000 });
    await page.waitForTimeout(2000);

    await page.waitForFunction(() => document.body.innerText.includes('项目管理面板'), { timeout: 10000 });

    // 在同一 context 中打开管理平台（cookie 自动共享）
    const adminPage = await context.newPage();

    // 验证管理平台 SSO
    await adminPage.goto(CONFIG.platformAdminUrl, { waitUntil: 'networkidle', timeout: 15000 });
    await adminPage.waitForTimeout(2000);

    // 验证登录页
    const loginText = await adminPage.evaluate(() => document.body.innerText);
    expect(loginText).toContain('IAM 管理平台');
    expect(loginText).toContain('IAM 账号登录');

    // 点击"IAM 账号登录"
    const iamLoginBtn = adminPage.locator('button', { hasText: 'IAM 账号登录' });
    await iamLoginBtn.click();
    await adminPage.waitForTimeout(3000);

    // 验证无登录表单出现（SSO 自动认证）
    const loginFormVisible = await adminPage.$('form input[type="text"]');
    expect(loginFormVisible).toBeNull();

    // 等待仪表盘加载
    try {
      await adminPage.waitForFunction(
        () => document.body.innerText.includes('仪表盘'),
        { timeout: 15000 }
      );
    } catch {}

    const adminText = await adminPage.evaluate(() => document.body.innerText);
    expect(adminText).toContain('仪表盘');
    expect(adminText).toContain('IAM 管理平台');
    expect(adminText).toContain('用户管理');
    expect(adminText).toContain('角色管理');
    expect(adminText).toContain('部门管理');
    expect(adminText).toContain('应用管理');
    expect(adminText).toContain('租户管理');
    expect(adminText).toContain('OAuth 客户端');

    expect(['用户总数', '角色总数', '部门总数', '应用总数'].every((k) => adminText.includes(k))).toBeTruthy();

    await adminPage.close();
  });
});
