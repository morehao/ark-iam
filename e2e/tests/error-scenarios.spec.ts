import { test, expect } from '@playwright/test';
import {
  rp1Login,
  fillLoginWebCredentials,
  verifyRedirectedToLoginWeb,
  clearSSOCookie,
} from '../helpers/oidc-helpers';
import { CONFIG } from '../config';

test.describe('错误和边界场景', () => {
  test('错误凭证登录失败', async ({ page }) => {
    await page.goto(CONFIG.rp1Url, { waitUntil: 'networkidle', timeout: 15000 });
    const url = page.url();
    if (url.includes('auth/callback') && url.includes('code=')) {
      await clearSSOCookie(page);
      await page.goto(CONFIG.rp1Url, { waitUntil: 'networkidle', timeout: 15000 });
    }
    await page.waitForURL(
      (u) => u.toString().includes('localhost:3003') && u.searchParams.has('authRequestID'),
      { timeout: 20000 }
    );
    await page.waitForSelector('#identifier', { timeout: 10000 });
    await page.fill('#identifier', 'wrong-user');
    await page.fill('#password', 'wrong-pass');
    await page.click('button[type="submit"]');
    const errorVisible = await page.waitForFunction(
      () => {
        const body = document.body.innerText;
        return body.includes('错误') || body.includes('失败') || body.includes('密码') || body.includes('用户名');
      },
      { timeout: 10000 }
    ).then(() => true).catch(() => false);
    expect(errorVisible).toBe(true);
  });

  test('无效 client_id 导致 OIDC 错误', async ({ page }) => {
    const badUrl = `${CONFIG.issuer}/authorize?client_id=invalid-client&redirect_uri=${encodeURIComponent(CONFIG.rp1Url)}&response_type=code&scope=openid`;
    await page.goto(badUrl, { waitUntil: 'networkidle', timeout: 15000 });
    const errorInUrl = page.url().includes('error=');
    const errorInBody = await page.evaluate(() => document.body.innerText).then(t => t.includes('error') || t.includes('Error'));
    expect(errorInUrl || errorInBody).toBe(true);
  });

  test('无效 redirect_uri 导致 OIDC 错误', async ({ page }) => {
    const badUrl = `${CONFIG.issuer}/authorize?client_id=sso-test-app&redirect_uri=${encodeURIComponent('http://evil.com/callback')}&response_type=code&scope=openid`;
    await page.goto(badUrl, { waitUntil: 'networkidle', timeout: 15000 });
    const errorInUrl = page.url().includes('error=');
    const errorInBody = await page.evaluate(() => document.body.innerText).then(t => t.includes('error') || t.includes('Error'));
    expect(errorInUrl || errorInBody).toBe(true);
  });

  test('多设备：同一用户在两个独立 context 各自独立 SSO session', async ({ browser }) => {
    const ctxA = await browser.newContext();
    const pageA = await ctxA.newPage();
    await rp1Login(pageA);

    const ctxB = await browser.newContext();
    const pageB = await ctxB.newPage();
    await pageB.goto(CONFIG.rp1Url, { waitUntil: 'domcontentloaded', timeout: 15000 });
    await verifyRedirectedToLoginWeb(pageB);

    await pageB.goto(`${CONFIG.issuer}/logged-out`, { waitUntil: 'networkidle', timeout: 10000 });
    await pageB.goto(CONFIG.rp1Url, { waitUntil: 'networkidle', timeout: 15000 });
    const urlB = pageB.url();
    if (urlB.includes('localhost:3003')) {
      await fillLoginWebCredentials(pageB);
      await pageB.waitForURL(
        (u) => u.toString().includes('localhost:3001') && !u.toString().includes('/auth/callback'),
        { timeout: 30000 }
      );
    }

    await pageA.close();
    await pageB.close();
    await ctxA.close();
    await ctxB.close();
  });
});
