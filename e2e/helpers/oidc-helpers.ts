import { type Page, expect } from '@playwright/test';

export const CONFIG = {
  issuer: 'http://localhost:8099/v1/iam/oidc',
  rp1Url: 'http://localhost:3001/',
  loginWebUrl: 'http://localhost:3003/login',
  platformAdminUrl: 'http://localhost:3000/',
  identifier: 'admin',
  password: 'admin123',
};

const wait = (ms: number) => new Promise((r) => setTimeout(r, ms));

/**
 * 点击页面中包含指定文本的按钮
 */
export async function clickByText(page: Page, text: string): Promise<void> {
  const btn = page.locator('button', { hasText: text });
  await btn.click();
}

/**
 * 在 login-web 登录页填写凭证并提交
 * 等待 OIDC 回调完成（URL 不包含 login/oidc/authRequestID）
 */
export async function fillLoginFormAndSubmit(page: Page): Promise<void> {
  await page.waitForSelector('#identifier', { timeout: 5000 });
  await page.fill('#identifier', CONFIG.identifier);
  await page.fill('#password', CONFIG.password);
  await page.click('button[type="submit"]');
  // 等待回调完成，不再是登录页 URL
  await page.waitForURL((url) => !url.toString().includes('/login'), { timeout: 20000 });
  await wait(1500);
}

/**
 * 验证 RP1 首页显示"项目管理面板"
 */
export async function verifyRp1HomePage(page: Page): Promise<void> {
  await page.waitForFunction(
    () => document.body.innerText.includes('项目管理面板'),
    { timeout: 10000 }
  );
  const body = await page.evaluate(() => document.body.innerText);
  expect(body).toContain('项目管理面板');
  expect(body).toContain('已通过 IAM SSO 登录');

  // 验证统计卡片
  const statCards = await page.$$('.stat-card');
  expect(statCards.length).toBe(4);
  expect(body).toContain('项目数');
  expect(body).toContain('任务数');
  expect(body).toContain('消息数');
  expect(body).toContain('团队数');
}

/**
 * 验证 Token 详情页
 */
export async function verifyTokenDetails(page: Page): Promise<void> {
  await clickByText(page, '查看 Token 详情');
  await wait(1000);

  const body = await page.evaluate(() => document.body.innerText);
  expect(body).toContain('access_token');
  expect(body).toContain('id_token');
  expect(body).toContain('refresh_token');

  // 验证 UserInfo 获取
  await clickByText(page, '获取 UserInfo');
  await wait(2000);
  const userinfoText = await page.evaluate(() => document.body.innerText);
  expect(['"name"', '"username"', '"email"', '"sub"'].some((k) => userinfoText.includes(k))).toBeTruthy();

  // 验证 Token 刷新
  const tokensBefore = await page.evaluate(() => (window as any).currentTokens?.access_token);
  await clickByText(page, '刷新 Token');
  await wait(3000);
  const tokensAfter = await page.evaluate(() => (window as any).currentTokens?.access_token);
  expect(!!tokensBefore && !!tokensAfter && tokensBefore !== tokensAfter).toBeTruthy();

  // 返回主页
  await clickByText(page, '返回主页');
  await wait(1000);
  const homeText = await page.evaluate(() => document.body.innerText);
  expect(homeText).toContain('项目管理面板');
}

/**
 * 验证管理平台 SSO 自动登录
 * 在同一 browser context 中打开管理平台，点击"IAM 账号登录"，
 * 凭借已存在的 iam_sso_session cookie 自动完成认证
 */
export async function verifyPlatformAdminSSO(page: Page): Promise<void> {
  await page.goto(CONFIG.platformAdminUrl, { waitUntil: 'networkidle', timeout: 15000 });
  await wait(2000);

  // 验证登录页
  const loginText = await page.evaluate(() => document.body.innerText);
  expect(loginText).toContain('IAM 管理平台');
  expect(loginText).toContain('IAM 账号登录');

  // 点击"IAM 账号登录"
  await clickByText(page, 'IAM 账号登录');
  await wait(3000);

  // 验证无登录表单出现（SSO 自动认证）
  const loginFormVisible = await page.$('form input[type="text"]');
  expect(loginFormVisible).toBeNull();

  // 等待仪表盘加载
  try {
    await page.waitForFunction(
      () => document.body.innerText.includes('仪表盘'),
      { timeout: 15000 }
    );
  } catch {}

  const adminText = await page.evaluate(() => document.body.innerText);
  expect(adminText).toContain('仪表盘');
  expect(adminText).toContain('IAM 管理平台');
  expect(adminText).toContain('用户管理');
  expect(adminText).toContain('角色管理');
  expect(adminText).toContain('部门管理');
  expect(adminText).toContain('应用管理');
  expect(adminText).toContain('租户管理');
  expect(adminText).toContain('OAuth 客户端');

  // 验证仪表盘统计卡片
  expect(['用户总数', '角色总数', '部门总数', '应用总数'].every((k) => adminText.includes(k))).toBeTruthy();
}
