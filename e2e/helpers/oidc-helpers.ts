import { type Page, expect } from '@playwright/test';
import { CONFIG } from '../config';

const wait = (ms: number) => new Promise((r) => setTimeout(r, ms));

function rp1ClearAuth(): void {
  localStorage.removeItem('auth-storage');
}

export async function clickByText(page: Page, text: string): Promise<void> {
  const btn = page.locator('button', { hasText: text });
  await btn.click();
}

export async function verifyRp1HomePage(page: Page): Promise<void> {
  await page.waitForFunction(
    () => document.body.innerText.includes('用户信息'),
    { timeout: 10000 }
  );
  const body = await page.evaluate(() => document.body.innerText);
  expect(body).toContain('用户信息');
  expect(body).toContain('SSO 测试应用');
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
  await page.waitForURL(
    (url) =>
      url.hostname === 'localhost' &&
      url.port === '3000' &&
      !url.pathname.includes('/auth/callback') &&
      !url.pathname.includes('/login'),
    { timeout: 30000 }
  );
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
  await page.waitForURL(
    (url) => url.hostname === 'localhost' && url.searchParams.has('code'),
    { timeout: 20000 }
  );
  await page.waitForTimeout(2000);
  await verifyRp1HomePage(page);
}

export async function adminDirectLogin(page: Page): Promise<void> {
  await page.goto(CONFIG.platformAdminUrl, { waitUntil: 'networkidle', timeout: 15000 });
  await page.waitForURL(
    (url) => url.toString().includes(CONFIG.loginWebUrl),
    { timeout: 15000 }
  );
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
  await page.waitForURL(
    (url) => url.toString().includes(CONFIG.loginWebUrl),
    { timeout: 15000 }
  );
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
    await page.waitForURL(
      (url) => url.hostname === 'localhost' && url.port === '3001' && !url.searchParams.has('authRequestID'),
      { timeout: 20000 }
    );
  } catch {}
  await wait(2000);
  const body = await page.evaluate(() => document.body.innerText);
  if (!body.includes('用户信息') || !body.includes('SSO 测试应用')) {
    throw new Error('SSO test app SSO login failed: not on home page');
  }
}

export async function verifyRp1RequiresLogin(page: Page): Promise<void> {
  await page.addInitScript(rp1ClearAuth);
  await page.goto(CONFIG.rp1Url, { waitUntil: 'networkidle', timeout: 15000 });
  await wait(1000);
  const body = await page.evaluate(() => document.body.innerText);
  expect(body).toContain('IAM 账号登录');
  expect(body).not.toContain('用户信息');
}

export async function rp1Logout(page: Page): Promise<void> {
  const avatar = page.locator('.ant-avatar');
  await expect(avatar).toBeVisible({ timeout: 10000 });
  await avatar.click();
  await wait(500);
  const logoutItem = page.locator('.ant-dropdown-menu-item', { hasText: '退出登录' });
  await expect(logoutItem).toBeVisible({ timeout: 5000 });
  await logoutItem.click();
  // end_session 跳转回 RP1，非登录态直接展示登录页
  await page.waitForURL(
    (url) => url.hostname === 'localhost' && url.port === '3001' && !url.searchParams.has('code'),
    { timeout: 20000 }
  );
  await wait(1000);
  const body = await page.evaluate(() => document.body.innerText);
  expect(body).toContain('IAM 账号登录');
  expect(body).not.toContain('用户信息');
}
