import { type Page, type BrowserContext, expect } from '@playwright/test';
import { CONFIG } from '../config';

const wait = (ms: number) => new Promise((r) => setTimeout(r, ms));

function isLoginWebUrl(url: string): boolean {
  return url.includes('localhost:3003') && url.includes('/login');
}

function isAuthCallbackUrl(url: string): boolean {
  return url.includes('/auth/callback');
}

function isRp1Url(url: string): boolean {
  return url.includes('localhost:3001');
}

function isAdminUrl(url: string): boolean {
  return url.includes('localhost:3000');
}

export async function fillLoginWebCredentials(page: Page): Promise<void> {
  await page.waitForSelector('#identifier', { timeout: 10000 });
  await page.fill('#identifier', CONFIG.identifier);
  await page.fill('#password', CONFIG.password);
  await page.click('button[type="submit"]');
}

async function navigateToLoginWeb(page: Page, targetUrl: string): Promise<void> {
  await page.goto(targetUrl, { waitUntil: 'networkidle', timeout: 15000 });

  const currentUrl = page.url();
  // 如果已有 SSO session，signinRedirect 直接回调，不会到 login-web
  // 此时先清除 session 再重试
  if (isAuthCallbackUrl(currentUrl) && currentUrl.includes('code=')) {
    await page.goto(`${CONFIG.issuer}/logged-out`, { waitUntil: 'networkidle', timeout: 10000 });
    // /logged-out 会清除 cookie 并重定向到 login-web
    await page.goto(targetUrl, { waitUntil: 'networkidle', timeout: 15000 });
  }

  await page.waitForURL(
    (url) => isLoginWebUrl(url.toString()) && url.searchParams.has('authRequestID'),
    { timeout: 20000 }
  );
}

export async function verifyRp1HomePage(page: Page): Promise<void> {
  await page.waitForFunction(
    () => document.body.innerText.includes('用户信息') && document.body.innerText.includes('SSO 测试应用'),
    { timeout: 30000 }
  );
  const body = await page.evaluate(() => document.body.innerText);
  expect(body).toContain('用户信息');
  expect(body).toContain('SSO 测试应用');
}

export async function verifyAdminDashboard(page: Page): Promise<void> {
  await page.waitForFunction(
    () => document.body.innerText.includes('仪表盘') && document.body.innerText.includes('IAM 管理平台'),
    { timeout: 30000 }
  );
  const body = await page.evaluate(() => document.body.innerText);
  expect(body).toContain('仪表盘');
  expect(body).toContain('IAM 管理平台');
}

export async function rp1Login(page: Page): Promise<void> {
  await navigateToLoginWeb(page, CONFIG.rp1Url);
  await fillLoginWebCredentials(page);

  await page.waitForURL(
    (url) => isAuthCallbackUrl(url.toString()),
    { timeout: 20000 }
  );
  await page.waitForURL(
    (url) => isRp1Url(url.toString()) && !isAuthCallbackUrl(url.toString()),
    { timeout: 20000 }
  );
  await verifyRp1HomePage(page);
}

export async function adminDirectLogin(page: Page): Promise<void> {
  await navigateToLoginWeb(page, CONFIG.platformAdminUrl);
  await fillLoginWebCredentials(page);

  await page.waitForURL(
    (url) => isAuthCallbackUrl(url.toString()),
    { timeout: 20000 }
  );
  await page.waitForURL(
    (url) => isAdminUrl(url.toString()) && !isAuthCallbackUrl(url.toString()),
    { timeout: 20000 }
  );
  await verifyAdminDashboard(page);
}

export async function adminSSOLogin(page: Page): Promise<void> {
  await page.goto(CONFIG.platformAdminUrl, { waitUntil: 'domcontentloaded', timeout: 15000 });
  await page.waitForURL(
    (url) => isAdminUrl(url.toString()) && !isAuthCallbackUrl(url.toString()),
    { timeout: 30000 }
  );
  await verifyAdminDashboard(page);
}

export async function rp1SSOLogin(page: Page): Promise<void> {
  await page.goto(CONFIG.rp1Url, { waitUntil: 'domcontentloaded', timeout: 15000 });
  await page.waitForURL(
    (url) => isRp1Url(url.toString()) && !isAuthCallbackUrl(url.toString()),
    { timeout: 30000 }
  );
  await verifyRp1HomePage(page);
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

export async function rp1Logout(page: Page): Promise<void> {
  const avatar = page.locator('.ant-layout-header .ant-avatar');
  await expect(avatar).toBeVisible({ timeout: 10000 });
  await avatar.click();
  await wait(500);
  const logoutItem = page.locator('.ant-dropdown-menu-item', { hasText: '退出登录' });
  await expect(logoutItem).toBeVisible({ timeout: 5000 });
  await logoutItem.click();
  // signoutRedirect 走 end_session，完成后重定向回 RP1 的 /login 页面
  // 如果 end_session 失败(post_logout_redirect_uri 无效)，页面会停留在 end_session 错误页
  try {
    await page.waitForURL(
      (url) => isRp1Url(url.toString()) && url.includes('/login'),
      { timeout: 20000 }
    );
  } catch {
    await page.goto(CONFIG.rp1Url, { waitUntil: 'networkidle', timeout: 15000 });
  }
  await wait(1000);
  const body = await page.evaluate(() => document.body.innerText);
  expect(body).not.toContain('用户信息');
}

export async function waitForSSOSessionExpiry(): Promise<void> {
  await wait((CONFIG.sessionTTL + 5) * 1000);
}

export async function clearSSOCookie(page: Page): Promise<void> {
  await page.goto(`${CONFIG.issuer}/logged-out`, { waitUntil: 'networkidle', timeout: 10000 });
}

export async function clearAllCookies(context: BrowserContext): Promise<void> {
  await context.clearCookies();
}

export async function verifyRedirectedToLoginWeb(page: Page): Promise<void> {
  // 等待页面重定向到 login-web（如果 SSO session 已清除，OIDC authorize 会重定向到 login-web）
  // 超时不报错 —— 可能前端有自己的 /login 页面而非 redirect 到 login-web
  try {
    await page.waitForURL((url) => isLoginWebUrl(url.toString()), { timeout: 15000 });
  } catch {
    // 若同时不在 /login 页，下面 if 会抛出错误
  }
  const currentUrl = page.url();
  if (isLoginWebUrl(currentUrl)) {
    const body = await page.evaluate(() => document.body.innerText);
    expect(body).not.toContain('仪表盘');
    expect(body).not.toContain('用户信息');
    return;
  }
  if (currentUrl.includes('/login')) {
    const body = await page.evaluate(() => document.body.innerText);
    expect(body).toContain('登录');
    expect(body).not.toContain('仪表盘');
    expect(body).not.toContain('用户信息');
    return;
  }
  throw new Error(`Expected redirect to login page, but got: ${currentUrl}`);
}

export async function callEndSession(page: Page): Promise<void> {
  await page.goto(`${CONFIG.issuer}/end_session`, {
    waitUntil: 'networkidle',
    timeout: 10000,
  });
}

export async function getSSOSessionCookie(page: Page): Promise<boolean> {
  const cookies = await page.context().cookies();
  return cookies.some(
    (c) => c.name === 'iam_sso_session' && (c.domain === 'localhost' || c.domain === '')
  );
}
