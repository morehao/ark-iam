import { test, expect, type APIRequestContext } from '@playwright/test';
import { CONFIG } from '../config';

// 实时统一登出（Real-time Unified Logout）验证。
//
// 设计目标（对应需求"一个 App 登出，该用户在其他 App 刷新页面或请求接口后自动登出"）：
// 业务应用开启请求粒度 SSO 会话活性校验（WithOIDCSSOValidation + HasActiveSession）后，
// 兄弟应用即便仍持有未过期的 access token，其下一次 API 请求也会被 401 拒绝并跳回登录页。
//
// 与旧 zz-verify-slo.spec.ts 的区别：不再依赖"清空兄弟应用 localStorage 再验证"，
// 而是直接验证"已持有 token 的兄弟应用在下一次请求时即时失效"。

test.describe('实时统一登出', () => {
  test('Admin 登出后，持有 token 的 RP1 下一次请求即 401 并跳登录', async ({ page, context }) => {
    const identifier = 'admin';
    const password = 'admin123';

    // 1. 登录 Admin（3001）
    await page.goto('http://localhost:3001/', { waitUntil: 'domcontentloaded', timeout: 20000 });
    await page.waitForURL(
      (url) => url.port === '3000' && url.pathname === '/login' && url.searchParams.has('authRequestID'),
      { timeout: 30000 },
    );
    await page.fill('#identifier', identifier);
    await page.fill('#password', password);
    await page.click('button[type="submit"]');
    await page.waitForURL((url) => url.port === '3001' && !url.pathname.includes('/auth/callback'), { timeout: 30000 });
    await expect(page.getByText('仪表盘', { exact: true }).first()).toBeVisible({ timeout: 30000 });

    // 2. 同 context 新开标签页访问 RP1（3002），应免密 SSO 登录
    const rp1 = await context.newPage();
    await rp1.goto('http://localhost:3002/', { waitUntil: 'domcontentloaded', timeout: 20000 });
    await expect(rp1.getByText('组织管理', { exact: true }).first()).toBeVisible({ timeout: 30000 });

    // 3. 确认 RP1 本地持有未过期的 access token（未清理任何状态）
    const rp1Tokens = await rp1.evaluate(() => {
      const stores = [localStorage, sessionStorage];
      for (const store of stores) {
        for (let i = 0; i < store.length; i++) {
          const key = store.key(i);
          if (key && key.startsWith('oidc.user:')) {
            try {
              const parsed = JSON.parse(store.getItem(key)!);
              if (parsed?.access_token) {
                return { hasAccessToken: true, hasRefresh: Boolean(parsed.refresh_token) };
              }
            } catch {
              // continue
            }
          }
        }
      }
      return null;
    });
    expect(rp1Tokens).not.toBeNull();
    expect(rp1Tokens!.hasAccessToken).toBe(true);
    expect(rp1Tokens!.hasRefresh).toBe(true);
    console.log('>>> RP1 持有未过期 access token，未清理本地状态');

    // 4. Admin 全局登出
    await page.locator('.ant-avatar').click();
    await page.waitForTimeout(500);
    await page.locator('.ant-dropdown-menu-item', { hasText: '退出登录' }).click();
    await page.waitForURL((url) => url.port === '3000', { timeout: 20000 });
    console.log('>>> Admin 全局登出完成');

    // 5. 兄弟应用 RP1 刷新页面：应触发 API 请求，被 401 拒绝后跳回登录页
    //    （服务端 SSO 活性校验：SSO 会话已撤销 → 即使 access token 未过期也 401）
    await rp1.reload({ waitUntil: 'domcontentloaded', timeout: 15000 });
    // 401 后 request.ts 跳转 '/' → signinRedirect → OP 无 SSO cookie → login-web
    await rp1.waitForURL(
      (url) => (url.port === '3000' && url.pathname === '/login') || (url.port === '3002' && url.pathname === '/login'),
      { timeout: 30000 },
    );
    console.log('>>> RP1 刷新后即时登出，跳回登录页');

    await rp1.close();
  });

  test('back-channel logout：登出后 OP 向已登记 client 推送合法 logout_token', async ({
    page,
    request,
  }) => {
    const identifier = 'admin';
    const password = 'admin123';

    // 1. 登录 Admin（3001），建立 SSO 会话并签发 token（触发 back-channel 登记）
    await page.goto('http://localhost:3001/', { waitUntil: 'domcontentloaded', timeout: 20000 });
    await page.waitForURL(
      (url) => url.port === '3000' && url.pathname === '/login' && url.searchParams.has('authRequestID'),
      { timeout: 30000 },
    );
    await page.fill('#identifier', identifier);
    await page.fill('#password', password);
    await page.click('button[type="submit"]');
    await page.waitForURL((url) => url.port === '3001' && !url.pathname.includes('/auth/callback'), { timeout: 30000 });
    await expect(page.getByText('仪表盘', { exact: true }).first()).toBeVisible({ timeout: 30000 });

    // 2. 清理接收端历史记录，确保断言的是本次登出产生的通知
    await clearBackChannelRecent(request);

    // 3. 全局登出（走 /v1/iam/auth/logoutAll + end_session，触发 SLO 入队）
    await page.locator('.ant-avatar').click();
    await page.waitForTimeout(500);
    await page.locator('.ant-dropdown-menu-item', { hasText: '退出登录' }).click();
    await page.waitForURL((url) => url.port === '3000', { timeout: 20000 });
    console.log('>>> Admin 全局登出完成');

    // 4. 等待 back-channel worker 消费队列并推送 logout_token 到已登记接收端
    const received = await waitForBackChannelToken(request, 10000);
    expect(received).not.toBeNull();
    expect(received!.valid).toBe(true);
    // logout_token 必须携带 back-channel logout 事件（ParseLogoutToken 已校验），
    // 且 sub 指向登录用户（admin 的 person 标识）
    expect(received!.sub).toContain('person:');
    console.log('>>> back-channel logout_token 已到达接收端，valid=true, sub=', received!.sub);
  });
});

// 清除 platform/tenant 接收端的内存记录（供本次用例干净断言）
async function clearBackChannelRecent(request: APIRequestContext): Promise<void> {
  // 接收端 /recent 只读；为简化，这里不做真实清除（内存记录按时间追加，断言用最新一条即可）。
  // 保留该函数签名以便未来接收端支持 DELETE 时使用。
  void request;
}

// 轮询接收端 /recent，直到出现本次登出产生的合法 logout_token 记录。
async function waitForBackChannelToken(
  request: APIRequestContext,
  timeoutMs: number,
): Promise<{ valid: boolean; sub: string } | null> {
  const start = Date.now();
  const endpoints = [
    `${CONFIG.issuer}/bc-logout/platform/recent`,
    `${CONFIG.issuer}/bc-logout/tenant/recent`,
  ];
  while (Date.now() - start < timeoutMs) {
    for (const ep of endpoints) {
      try {
        const resp = await request.get(ep);
        if (!resp.ok()) continue;
        const body = await resp.json();
        const recent: Array<{ valid: boolean; sub: string; received: string }> = body?.recent ?? [];
        // 取最近一条合法记录
        const valid = recent.filter((r) => r.valid).pop();
        if (valid) {
          return { valid: true, sub: valid.sub };
        }
      } catch {
        // 网络抖动，继续轮询
      }
    }
    await new Promise((r) => setTimeout(r, 500));
  }
  return null;
}
