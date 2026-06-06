import { type Page, expect } from '@playwright/test';
import { CONFIG } from '../config';

const wait = (ms: number) => new Promise((r) => setTimeout(r, ms));

export async function clickByText(page: Page, text: string): Promise<void> {
  const btn = page.locator('button', { hasText: text });
  await btn.click();
}

export async function verifyRp1HomePage(page: Page): Promise<void> {
  await page.waitForFunction(
    () => document.body.innerText.includes('项目管理面板'),
    { timeout: 10000 }
  );
  const body = await page.evaluate(() => document.body.innerText);
  expect(body).toContain('项目管理面板');
  expect(body).toContain('已通过 IAM SSO 登录');
}

export async function verifyTokenDetails(page: Page): Promise<void> {
  await clickByText(page, '查看 Token 详情');
  await wait(1000);
  const body = await page.evaluate(() => document.body.innerText);
  expect(body).toContain('access_token');
  expect(body).toContain('id_token');
  expect(body).toContain('refresh_token');

  await clickByText(page, '获取 UserInfo');
  await wait(2000);
  const userinfoText = await page.evaluate(() => document.body.innerText);
  expect(['"name"', '"username"', '"email"', '"sub"'].some((k) => userinfoText.includes(k))).toBeTruthy();

  const tokensBefore = await page.evaluate(() => (window as any).currentTokens?.access_token);
  await clickByText(page, '刷新 Token');
  await wait(3000);
  const tokensAfter = await page.evaluate(() => (window as any).currentTokens?.access_token);
  expect(!!tokensBefore && !!tokensAfter && tokensBefore !== tokensAfter).toBeTruthy();

  await clickByText(page, '返回主页');
  await wait(1000);
  const homeText = await page.evaluate(() => document.body.innerText);
  expect(homeText).toContain('项目管理面板');
}

export async function logoutFromAdmin(page: Page): Promise<void> {
  await page.waitForFunction(() => document.body.innerText.includes('仪表盘'), { timeout: 5000 });
  await wait(1000);
  const avatar = page.locator('.ant-avatar');
  await expect(avatar).toBeVisible({ timeout: 10000 });
  await avatar.click();
  await wait(500);
  const logoutItem = page.locator('.ant-dropdown-menu-item', { hasText: '退出登录' });
  await expect(logoutItem).toBeVisible({ timeout: 5000 });
  await logoutItem.click();
  await page.waitForURL((url) => url.toString().includes('/login'), { timeout: 15000 });
  await wait(2000);
}

async function fillLoginWebCredentials(page: Page): Promise<void> {
  await page.waitForSelector('#identifier', { timeout: 10000 });
  await page.fill('#identifier', CONFIG.identifier);
  await page.fill('#password', CONFIG.password);
  await page.click('button[type="submit"]');
}

async function waitForDashboard(page: Page): Promise<void> {
  await page.waitForURL((url) => url.hostname === 'localhost' && url.port === '3000' && !url.pathname.includes('/auth/callback') && !url.pathname.includes('/login'), { timeout: 30000 });
  await wait(2000);
  try {
    await page.waitForFunction(() => document.body.innerText.includes('仪表盘'), { timeout: 15000 });
  } catch {}
  const adminText = await page.evaluate(() => document.body.innerText);
  expect(adminText).toContain('仪表盘');
  expect(adminText).toContain('IAM 管理平台');
}

export async function rp1Login(page: Page): Promise<void> {
  await page.goto(CONFIG.rp1Url, { waitUntil: 'networkidle' });
  await page.click('button', { timeout: 5000 });
  await page.waitForTimeout(2000);

  expect(page.url()).toContain(CONFIG.loginWebUrl);
  expect(page.url()).toContain('authRequestID=');

  await fillLoginWebCredentials(page);
  await page.waitForURL((url) => url.hostname === 'localhost' && url.searchParams.has('code'), { timeout: 20000 });
  await page.waitForTimeout(2000);
  await verifyRp1HomePage(page);
}

export async function adminDirectLogin(page: Page): Promise<void> {
  await page.goto(CONFIG.platformAdminUrl, { waitUntil: 'networkidle', timeout: 15000 });

  // App silent auth 会跳转到 IAM → login-web
  // 等待 login-web 登录页面出现
  await page.waitForURL((url) => url.toString().includes(CONFIG.loginWebUrl), { timeout: 15000 });
  await fillLoginWebCredentials(page);
  await waitForDashboard(page);
}

export async function adminSSOLogin(page: Page): Promise<void> {
  await page.goto(CONFIG.platformAdminUrl, { waitUntil: 'domcontentloaded', timeout: 15000 });
  try {
    await page.waitForFunction(() => document.body.innerText.includes('仪表盘'), { timeout: 30000 });
  } catch {}
  await wait(2000);
  const adminUrl = page.url();
  const adminText = await page.evaluate(() => document.body.innerText);
  expect(adminUrl).not.toMatch(/\/login/);
  expect(adminText).not.toContain('IAM 账号登录');
  expect(adminText).toContain('仪表盘');
  expect(adminText).toContain('IAM 管理平台');
}

export async function adminRequiresLoginAfterLogout(page: Page): Promise<void> {
  await page.goto(CONFIG.platformAdminUrl, { waitUntil: 'networkidle', timeout: 15000 });
  // 静默授权会尝试 SSO，SSO session 已清除，应跳转到 login-web
  await page.waitForURL((url) => url.toString().includes(CONFIG.loginWebUrl), { timeout: 15000 });
  const finalUrl = page.url();
  const finalText = await page.evaluate(() => document.body.innerText);
  expect(finalUrl).toContain('login');
  expect(finalText).not.toContain('仪表盘');
}

export async function rp1SSOLogin(page: Page): Promise<void> {
  await page.goto(CONFIG.rp1Url, { waitUntil: 'networkidle', timeout: 15000 });
  await wait(1000);
  await page.click('button', { timeout: 5000 });
  try {
    await page.waitForURL((url) => url.hostname === 'localhost' && url.port === '3001' && !url.searchParams.has('authRequestID'), { timeout: 20000 });
  } catch {}
  await wait(2000);
  const body = await page.evaluate(() => document.body.innerText);
  if (!body.includes('项目管理面板') || !body.includes('已通过 IAM SSO 登录')) {
    throw new Error('SSO test app SSO login failed: not on home page');
  }
}

export async function verifyRp1RequiresLogin(page: Page): Promise<void> {
  await page.goto(CONFIG.rp1Url, { waitUntil: 'networkidle', timeout: 15000 });
  await wait(1000);
  const body = await page.evaluate(() => document.body.innerText);
  expect(body).toContain('使用 IAM 登录');
  expect(body).toContain('您尚未登录此应用');
  expect(body).not.toContain('项目管理面板');
}
