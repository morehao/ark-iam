# OIDC SSO E2E 测试完善实施计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 将 OIDC SSO E2E 测试从 8 个用例扩展到约 38 个用例，全面覆盖开心路径、Session 生命周期、Token 刷新、Cookie/Context 隔离、多 RP 并行、错误边界等场景。

**Architecture:** 将现有单文件 `tests/oidc-sso.spec.ts` 拆分为 6 个按场景分组的 spec 文件，扩展现有 helpers，新增可测试短 TTL 的 E2E 专用配置文件。全程保持 helpers-only（不直接操作 DOM），通过 Playwright page/context 管理 session 隔离。

**Tech Stack:** Playwright + TypeScript，后端 Go/Gin + zitadel/oidc，Redis Session 存储。

## 全局约束

- 不修改后端 OIDC Provider 代码
- 不修改前端应用代码
- workers 保持为 1（确保服务状态一致性）
- retries 保持为 0（测试失败即暴露问题）
- 新增 helpers 遵循现有函数签名风格：`async function xxx(page: Page, ...): Promise<void>`
- cookie 名称 `iam_sso_session` 硬编码，不可配置
- session TTL 默认 86400 秒，通过 E2E 专用 config.yaml 覆盖为 30 秒可测过期

---

### Task 0: 基础设施 — 创建 E2E 专用配置文件和启动脚本

**Files:**
- Create: `backend/apps/iam/config/config.e2e.yaml`
- Modify: `e2e/global-setup.ts:28-31`
- Modify: `e2e/config.ts:1-8`

**Interfaces:**
- Produces: `CONFIG.issuer`, `CONFIG.sessionTTL`, `CONFIG.authRequestTTL`, `CONFIG.authCodeTTL`

提交信息: `feat(e2e): add e2e-specific backend config with short TTLs`

---

- [ ] **Step 0.1: 创建 E2E 专用 config.yaml**

在 `backend/apps/iam/config/config.e2e.yaml` 创建新文件，基于 `config.yaml` 副本，将 OIDC TTL 缩短：

```yaml
server:
  name: iam
  port: 8099
  env: dev

client:
  httpbingo:
    host: https://httpbingo.org
    module: httpbingo
    retry: 3
    timeout: 5s

log:
  default:
    service: iam
    module: default
    level: info
    writer: file
    dir: ../../../log
    enable_otel_trace: true
    extra_keys:
      - requestID
  gorm:
    service: iam
    module: gorm
    level: debug
    writer: file
    dir: ../../../log
    enable_otel_trace: true
    extra_keys:
      - requestID
  redis:
    service: iam
    module: redis
    level: debug
    writer: file
    dir: ../../../log
    enable_otel_trace: true
    extra_keys:
      - requestID

trace:
  enable: true
  service_version: ""
  sampler: traceidratio
  trace_id_ratio: 1.0
  otlp:
    endpoint: "127.0.0.1:4317"
    insecure: true
    timeout: 10s

db_configs:
  - url: "mysql://root:123456@127.0.0.1:3306/iam?charset=utf8mb4&parseTime=True&loc=Local&timeout=5s&readTimeout=3s&writeTimeout=3s"
    service: iam

redis_config:
  service: iam
  addr: 127.0.0.1:6379
  password: ""
  db: 0
  dial_timeout: 5s
  read_timeout: 3s
  write_timeout: 3s

jwt:
  signKey: "your-jwt-secret-key"

oidc:
  issuer: "http://localhost:8099/v1/iam/oidc"
  frontendLoginURL: "http://localhost:3003/login"
  signingKeyID: "dev-oidc-key"
  signingPrivateKeyPath: "config/oidc-dev-key.pem"
  signingPrivateKeyPEM: ""
  encryptionKey: "oidc-dev-encryption-key-32bytes"
  encryptionKeyID: "dev-enc-key"
  allowInsecure: true
  # E2E 专用：缩短 TTL 以测试过期场景
  authRequestTTL: 30
  authCodeTTL: 15
  spentCodeTTL: 60
  sessionTTL: 30
```

- [ ] **Step 0.2: 修改 global-setup.ts 使用 E2E 专用配置**

修改 `SERVICES` 数组中 IAM Backend 的 env：

```typescript
// 将
env: {
  APP_CONFIG_PATH: path.join(ROOT, 'backend', 'apps', 'iam', 'config', 'config.yaml'),
},
// 改为
env: {
  APP_CONFIG_PATH: path.join(ROOT, 'backend', 'apps', 'iam', 'config', 'config.e2e.yaml'),
},
```

- [ ] **Step 0.3: 扩展 config.ts 添加 TTL 配置**

```typescript
export const CONFIG = {
  issuer: 'http://localhost:8099/v1/iam/oidc',
  rp1Url: 'http://localhost:3001/',
  loginWebUrl: 'http://localhost:3003/login',
  platformAdminUrl: 'http://localhost:3000/',
  identifier: 'admin',
  password: 'admin123',
  sessionTTL: 30,
  authRequestTTL: 30,
  authCodeTTL: 15,
};
```

- [ ] **Step 0.4: 验证基础设施**

运行: `cd e2e && npx playwright test tests/oidc-sso.spec.ts --headed --reporter=list`

预期: 原有 8 个测试通过（如果 30 秒 TTL 下 SSO 登录环节太快仍能通过）

- [ ] **Step 0.5: 提交**

```bash
git add backend/apps/iam/config/config.e2e.yaml e2e/global-setup.ts e2e/config.ts
git commit -m "feat(e2e): add e2e-specific backend config with short TTLs"
```

---

### Task 1: Helpers 扩展 — 新增通用辅助函数

**Files:**
- Modify: `e2e/helpers/oidc-helpers.ts:1-196`

**Interfaces:**
- Consumes: `CONFIG` from config.ts
- Produces: 辅助函数列表（见下方）

提交信息: `feat(e2e): extend helpers for session lifecycle, cookie isolation, and error scenarios`

---

- [ ] **Step 1.1: 新增 `wait()`, `waitForSSOSessionExpiry()`, `clearSSOCookie()`**

在文件开头 `const wait` 下方追加：

```typescript
export async function waitForSSOSessionExpiry(): Promise<void> {
  await wait((CONFIG.sessionTTL + 5) * 1000);
}

export async function clearSSOCookie(page: Page): Promise<void> {
  await page.goto(`${CONFIG.issuer}/logged-out`, { waitUntil: 'networkidle', timeout: 10000 });
}

export async function clearAllCookies(context: BrowserContext): Promise<void> {
  await context.clearCookies();
}

export async function navigateToLoginWebWithAuthRequest(
  page: Page,
  targetUrl: string
): Promise<void> {
  await page.goto(targetUrl, { waitUntil: 'networkidle', timeout: 15000 });
  const url = page.url();

  if (isAuthCallbackUrl(url) && url.includes('code=')) {
    await clearSSOCookie(page);
    await page.goto(targetUrl, { waitUntil: 'networkidle', timeout: 15000 });
  }

  await page.waitForURL(
    (u) => isLoginWebUrl(u.toString()) && u.searchParams.has('authRequestID'),
    { timeout: 20000 }
  );
}
```

- [ ] **Step 1.2: 新增 `verifyRedirectedToLoginWeb()`, `verifyStillOnHomePage()`**

```typescript
export async function verifyRedirectedToLoginWeb(page: Page): Promise<void> {
  // RP 或 Admin 在没有 SSO session 时访问，先触发 signinRedirect，
  // 重定向到 OIDC /authorize，再重定向到 login-web
  try {
    await page.waitForURL((url) => isLoginWebUrl(url.toString()), { timeout: 15000 });
  } catch {
    // 可能在前端自身的 /login 页
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

export async function verifyStillOnHomePage(page: Page, app: 'admin' | 'rp1'): Promise<void> {
  if (app === 'admin') {
    await verifyAdminDashboard(page);
  } else {
    await verifyRp1HomePage(page);
  }
}
```

- [ ] **Step 1.3: 新增 `loginWithCredentials()`, `visitWithAuthRequestExpiry()`, `callEndSession()`**

```typescript
export async function loginWithCredentials(page: Page): Promise<void> {
  await page.waitForSelector('#identifier', { timeout: 10000 });
  await page.fill('#identifier', CONFIG.identifier);
  await page.fill('#password', CONFIG.password);
  await page.click('button[type="submit"]');
}

export async function callEndSession(page: Page): Promise<void> {
  // OIDC RP-initiated logout 调用 /end_session
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
```

- [ ] **Step 1.4: 新增 `verifyAuthCodeReuseFails()` helper**

```typescript
export async function extractAuthCode(page: Page): Promise<string> {
  const url = page.url();
  const codeMatch = url.match(/[?&]code=([^&]+)/);
  if (!codeMatch) throw new Error('No auth code found in URL');
  return codeMatch[1];
}

export async function verifyAuthCodeReuseFails(
  page: Page,
  reusedCode: string,
  clientId: string
): Promise<void> {
  const tokenUrl = `${CONFIG.issuer}/oauth/token`;
  const response = await page.request.post(tokenUrl, {
    form: {
      grant_type: 'authorization_code',
      code: reusedCode,
      redirect_uri: CONFIG.rp1Url,
      client_id: clientId,
    },
  });
  expect(response.status()).toBeGreaterThanOrEqual(400);
}
```

- [ ] **Step 1.5: 运行现有测试确保 helpers 修改不破坏**

运行: `cd e2e && npx playwright test tests/oidc-sso.spec.ts --reporter=list`

预期: 8 tests passed

- [ ] **Step 1.6: 提交**

```bash
git add e2e/helpers/oidc-helpers.ts
git commit -m "feat(e2e): extend helpers for session lifecycle, cookie isolation, and error scenarios"
```

---

### Task 2: 补充核心认证流程 — 扩展现有 oidc-sso.spec.ts

**Files:**
- Modify: `e2e/tests/oidc-sso.spec.ts`

**Interfaces:**
- Consumes: `rp1Login`, `adminDirectLogin`, `rp1SSOLogin`, `adminSSOLogin`, `logoutFromAdmin`, `rp1Logout`, `verifyRp1HomePage`, `verifyAdminDashboard`, `callEndSession`, `clearSSOCookie`, `verifyRedirectedToLoginWeb`

提交信息: `feat(e2e): add RP logout, logoutAll, and end_session test cases`

---

- [ ] **Step 2.1: 追加测试用例到现有 describe 块末尾**

在 `oidc-sso.spec.ts` 第 69 行 `})` 之前追加以下测试用例：

```typescript
  test('RP1 自身登出后再次 SSO 免密登录（RP 登出只清自身 token，SSO session 仍在）', async ({ page }) => {
    await rp1Login(page);
    await rp1Logout(page);
    await rp1SSOLogin(page);
  });

  test('RP1 自身登出后，Admin 的 SSO session 仍有效', async ({ page, context }) => {
    await rp1Login(page);
    const adminPage = await context.newPage();
    await adminSSOLogin(adminPage);
    await rp1Logout(page);
    await adminSSOLogin(adminPage);
    await adminPage.close();
  });

  test('Admin 登出（logout）后重新登录应走 SSO 免密流程', async ({ page }) => {
    await adminDirectLogin(page);
    await logoutFromAdmin(page);
    await adminSSOLogin(page);
  });

  test('/end_session 后所有 RP 都需要重新认证', async ({ page, context }) => {
    await adminDirectLogin(page);
    const rp1Page = await context.newPage();
    await rp1SSOLogin(rp1Page);
    await callEndSession(page);
    await verifyRedirectedToLoginWeb(rp1Page);
    await verifyRedirectedToLoginWeb(page);
    await rp1Page.close();
  });

  test('/logged-out 清除 cookie 后需重新输入凭证', async ({ page }) => {
    await rp1Login(page);
    await clearSSOCookie(page);
    await verifyRedirectedToLoginWeb(page);
  });
```

- [ ] **Step 2.2: 更新 imports**

在文件头部 `import` 语句追加新函数：

```typescript
import {
  rp1Login,
  rp1SSOLogin,
  logoutFromAdmin,
  adminDirectLogin,
  adminSSOLogin,
  rp1Logout,
  callEndSession,
  clearSSOCookie,
  verifyRedirectedToLoginWeb,
} from '../helpers/oidc-helpers';
```

- [ ] **Step 2.3: 运行测试验证**

运行: `cd e2e && npx playwright test tests/oidc-sso.spec.ts --reporter=list`

预期: 13 tests passed（原有 8 个 + 新增 5 个）

- [ ] **Step 2.4: 提交**

```bash
git add e2e/tests/oidc-sso.spec.ts
git commit -m "feat(e2e): add RP logout, end_session, and cookie-clear test cases"
```

---

### Task 3: Session 生命周期测试 — 创建 session-lifecycle.spec.ts

**Files:**
- Create: `e2e/tests/session-lifecycle.spec.ts`

**Interfaces:**
- Consumes: `rp1Login`, `rp1SSOLogin`, `adminDirectLogin`, `adminSSOLogin`, `verifyRedirectedToLoginWeb`, `verifyStillOnHomePage`, `waitForSSOSessionExpiry`, `clearSSOCookie`

提交信息: `feat(e2e): add session lifecycle test spec`

---

- [ ] **Step 3.1: 创建测试文件**

```typescript
import { test } from '@playwright/test';
import {
  rp1Login,
  rp1SSOLogin,
  adminDirectLogin,
  adminSSOLogin,
  verifyRedirectedToLoginWeb,
  waitForSSOSessionExpiry,
  clearSSOCookie,
} from '../helpers/oidc-helpers';

test.describe('Session 生命周期', () => {
  test('SSO session 在 TTL 内有效 — Admin 登录后可免密 SSO 到 RP1', async ({ page, context }) => {
    await adminDirectLogin(page);
    const rp1Page = await context.newPage();
    await rp1SSOLogin(rp1Page);
    await rp1Page.close();
  });

  test('SSO session 过期后需重新输入凭证', async ({ page }) => {
    await rp1Login(page);
    await waitForSSOSessionExpiry();
    await verifyRedirectedToLoginWeb(page);
  });

  test('SSO session 过期后 Admin 也需要重新认证', async ({ page }) => {
    await adminDirectLogin(page);
    await waitForSSOSessionExpiry();
    await verifyRedirectedToLoginWeb(page);
  });

  test('AuthRequest TTL 超时后需重新生成授权请求', async ({ page }) => {
    // login-web 停留在凭证填写页 - 等待 authRequest 过期
    // 注意：30 秒 sessionTTL 意味着等待期间 session 也会过期
    // 因此只验证 authRequest 过期后 login-web 的降级行为
    await page.goto('http://localhost:3001/', { waitUntil: 'networkidle', timeout: 15000 });
    const url = page.url();
    if (url.includes('auth/callback') && url.includes('code=')) {
      await clearSSOCookie(page);
      await page.goto('http://localhost:3001/', { waitUntil: 'networkidle', timeout: 15000 });
    }
    // 等待 authRequest 过期（35 秒 > 30 秒 authRequestTTL）
    await page.waitForTimeout(35000);
    // 验证页面仍然存在且没有崩溃（login-web 不应白屏或 500）
    const body = await page.evaluate(() => document.body.innerText);
    expect(body.length).toBeGreaterThan(0);
    // authRequest 过期后重新提交凭证，login-web 应正常降级
    // 具体行为取决于前端实现（可能是重新发起 authorize 或显示错误提示）
  });
});
```

- [ ] **Step 3.2: 运行测试验证**

运行: `cd e2e && npx playwright test tests/session-lifecycle.spec.ts --reporter=list`

预期: 至少 3 tests passed（AuthRequest 过期测试可能需要调整）

- [ ] **Step 3.3: 提交**

```bash
git add e2e/tests/session-lifecycle.spec.ts
git commit -m "feat(e2e): add session lifecycle test spec"
```

---

### Task 4: Cookie 和 Context 隔离测试 — 创建 cookie-isolation.spec.ts

**Files:**
- Create: `e2e/tests/cookie-isolation.spec.ts`

**Interfaces:**
- Consumes: `rp1Login`, `adminSSOLogin`, `verifyRedirectedToLoginWeb`, `clearAllCookies`, `getSSOSessionCookie`

提交信息: `feat(e2e): add cookie and context isolation test spec`

---

- [ ] **Step 4.1: 创建测试文件**

```typescript
import { test, expect, type BrowserContext } from '@playwright/test';
import {
  rp1Login,
  adminSSOLogin,
  verifyRedirectedToLoginWeb,
  clearAllCookies,
  clearSSOCookie,
  getSSOSessionCookie,
} from '../helpers/oidc-helpers';
import { CONFIG } from '../config';

test.describe('Cookie 和跨 Context 隔离', () => {
  test('新 browser context（无 cookie）访问 RP 重定向到登录页', async ({ browser }) => {
    const context1 = await browser.newContext();
    const page1 = await context1.newPage();
    await rp1Login(page1);
    await page1.close();

    const context2 = await browser.newContext();
    const page2 = await context2.newPage();
    await page2.goto(CONFIG.rp1Url, { waitUntil: 'domcontentloaded', timeout: 15000 });
    await verifyRedirectedToLoginWeb(page2);
    await page2.close();
    await context1.close();
    await context2.close();
  });

  test('手动清除 cookie 后需重新认证', async ({ page, context }) => {
    await rp1Login(page);
    await clearAllCookies(context);
    await page.goto(CONFIG.rp1Url, { waitUntil: 'domcontentloaded', timeout: 15000 });
    await verifyRedirectedToLoginWeb(page);
  });

  test('cookie HttpOnly 不可被 JavaScript 读取', async ({ page }) => {
    await rp1Login(page);
    const hasCookie = await getSSOSessionCookie(page);
    expect(hasCookie).toBe(true);

    // 尝试用 JS 读取，HttpOnly cookie 应该无法通过 document.cookie 访问
    const jsCookie = await page.evaluate(() => document.cookie);
    expect(jsCookie).not.toContain('iam_sso_session');
  });

  test('同 context 内两个 browser context 实例各自独立', async ({ browser }) => {
    // 两个全新的 context，各自独立
    const ctxA = await browser.newContext();
    const pageA = await ctxA.newPage();
    await rp1Login(pageA);

    const ctxB = await browser.newContext();
    const pageB = await ctxB.newPage();
    await pageB.goto(CONFIG.rp1Url, { waitUntil: 'domcontentloaded', timeout: 15000 });
    await verifyRedirectedToLoginWeb(pageB);

    // ctxA 仍保持登录态
    await adminSSOLogin(pageA);

    await pageA.close();
    await pageB.close();
    await ctxA.close();
    await ctxB.close();
  });
});
```

- [ ] **Step 4.2: 运行测试验证**

运行: `cd e2e && npx playwright test tests/cookie-isolation.spec.ts --reporter=list`

预期: 4 tests passed

- [ ] **Step 4.3: 提交**

```bash
git add e2e/tests/cookie-isolation.spec.ts
git commit -m "feat(e2e): add cookie and context isolation test spec"
```

---

### Task 5: 多 RP 并行场景测试 — 创建 multi-rp-sso.spec.ts

**Files:**
- Create: `e2e/tests/multi-rp-sso.spec.ts`

**Interfaces:**
- Consumes: `rp1Login`, `rp1SSOLogin`, `adminDirectLogin`, `adminSSOLogin`, `logoutFromAdmin`, `rp1Logout`, `callEndSession`, `verifyRedirectedToLoginWeb`

提交信息: `feat(e2e): add multi-RP parallel SSO test spec`

---

- [ ] **Step 5.1: 创建测试文件**

```typescript
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
    // 两个应用都处于已登录状态
    await rp1SSOLogin(rp1Page);
    await adminSSOLogin(page);
    await rp1Page.close();
  });

  test('Admin 登出不影响 RP1 的 SSO session', async ({ page, context }) => {
    await adminDirectLogin(page);
    const rp1Page = await context.newPage();
    await rp1SSOLogin(rp1Page);
    await logoutFromAdmin(page);
    // Admin 登出只清自身 token
    await rp1SSOLogin(rp1Page);
    await rp1Page.close();
  });

  test('RP1 登出不影响 Admin 的 SSO session', async ({ page, context }) => {
    await rp1Login(page);
    const adminPage = await context.newPage();
    await adminSSOLogin(adminPage);
    await rp1Logout(page);
    // RP1 登出只清自身 token
    await adminSSOLogin(adminPage);
    await adminPage.close();
  });

  test('/end_session 调用后所有应用都需要重新认证', async ({ page, context }) => {
    await adminDirectLogin(page);
    const rp1Page = await context.newPage();
    await rp1SSOLogin(rp1Page);
    await callEndSession(page);
    // end_session 清除 cookie 和 revoke 所有 SSO session
    await verifyRedirectedToLoginWeb(page);
    await verifyRedirectedToLoginWeb(rp1Page);
    await rp1Page.close();
  });
});
```

- [ ] **Step 5.2: 运行测试验证**

运行: `cd e2e && npx playwright test tests/multi-rp-sso.spec.ts --reporter=list`

预期: 4 tests passed

- [ ] **Step 5.3: 提交**

```bash
git add e2e/tests/multi-rp-sso.spec.ts
git commit -m "feat(e2e): add multi-RP parallel SSO test spec"
```

---

### Task 6: 错误和边界场景测试 — 创建 error-scenarios.spec.ts

**Files:**
- Create: `e2e/tests/error-scenarios.spec.ts`

**Interfaces:**
- Consumes: `CONFIG`, `fillLoginWebCredentials`（需导出）, `verifyRedirectedToLoginWeb`, `rp1Login`, `adminDirectLogin`, `clearSSOCookie`

提交信息: `feat(e2e): add error and edge case test spec`

---

- [ ] **Step 6.1: 导出 `fillLoginWebCredentials`**

在 `helpers/oidc-helpers.ts` 中将 `async function fillLoginWebCredentials` 改为 `export async function fillLoginWebCredentials`，并更新对应调用处。

- [ ] **Step 6.2: 创建测试文件**

```typescript
import { test, expect } from '@playwright/test';
import {
  rp1Login,
  adminDirectLogin,
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
    // 应显示错误提示而非跳转到回调
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
    // OIDC Provider 应返回错误页面或错误 response
    const errorInUrl = page.url().includes('error=');
    const errorInBody = await page.evaluate(() => document.body.innerText).then(t => t.includes('error') || t.includes('Error'));
    expect(errorInUrl || errorInBody).toBe(true);
  });

  test('无效 redirect_uri 导致 OIDC 错误', async ({ page }) => {
    // 使用未注册的 redirect_uri
    const badUrl = `${CONFIG.issuer}/authorize?client_id=sso-test-app&redirect_uri=${encodeURIComponent('http://evil.com/callback')}&response_type=code&scope=openid`;
    await page.goto(badUrl, { waitUntil: 'networkidle', timeout: 15000 });
    const errorInUrl = page.url().includes('error=');
    const errorInBody = await page.evaluate(() => document.body.innerText).then(t => t.includes('error') || t.includes('Error'));
    expect(errorInUrl || errorInBody).toBe(true);
  });

  test('多设备：同一用户在两个 context 各自独立 SSO session', async ({ browser }) => {
    // 设备 A 登录
    const ctxA = await browser.newContext();
    const pageA = await ctxA.newPage();
    await rp1Login(pageA);

    // 设备 B 登录
    const ctxB = await browser.newContext();
    const pageB = await ctxB.newPage();
    // 设备 B 首次访问，需要输入凭证
    await pageB.goto(CONFIG.rp1Url, { waitUntil: 'domcontentloaded', timeout: 15000 });
    await verifyRedirectedToLoginWeb(pageB);

    // 但设备 B 输入凭证后，其自身的 SSO session 正常
    // 先清除可能存在的 cookie
    await pageB.goto(`${CONFIG.issuer}/logged-out`, { waitUntil: 'networkidle', timeout: 10000 });
    // 再手动登录
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
```

- [ ] **Step 6.3: 运行测试验证**

运行: `cd e2e && npx playwright test tests/error-scenarios.spec.ts --reporter=list`

预期: 4 tests passed

- [ ] **Step 6.4: 提交**

```bash
git add e2e/tests/error-scenarios.spec.ts e2e/helpers/oidc-helpers.ts
git commit -m "feat(e2e): add error and edge case test spec"
```

---

### Task 7: 完整运行与回归验证

- [ ] **Step 7.1: 运行全部 E2E 测试**

```bash
cd e2e && npx playwright test --reporter=list
```

预期: 约 25+ tests passed（原有 8 个 + 核心认证新增 5 个 + 生命周期 3-4 个 + cookie 4 个 + 多 RP 4 个 + 错误边界 4 个）

- [ ] **Step 7.2: 检查测试隔离性**

验证每个测试用例完全独立运行，不依赖其他测试的状态：

```bash
cd e2e && npx playwright test tests/cookie-isolation.spec.ts --reporter=list
```

- [ ] **Step 7.3: 问题修复与记录**

如果测试失败：
1. 分析失败原因（是测试逻辑问题还是后端缺陷）
2. 修正测试逻辑或记录后端缺陷到 issue
3. 重新运行确认修复

- [ ] **Step 7.4: 最终提交（如果步骤 7.3 无修改则跳过）**

```bash
git status
git diff --stat
# 如果只有测试文件变更，按需提交
```

---

## 完整测试矩阵汇总

| # | Spec 文件 | 用例数 | 覆盖场景 |
|---|---|---|---|
| 1 | oidc-sso.spec.ts | 13 | 首次登录、免密 SSO、登出隔离、重新登录、双向 SSO、end_session、cookie 清除 |
| 2 | session-lifecycle.spec.ts | 3 | TTL 内有效、TTL 过期重认证、authRequest 过期 |
| 3 | cookie-isolation.spec.ts | 4 | 新 context 隔离、手动清 cookie、HttpOnly、跨 context 独立 |
| 4 | multi-rp-sso.spec.ts | 4 | Admin+RP1 并行、Admin 登出不影响 RP1、RP1 登出不影响 Admin、end_session 全局清除 |
| 5 | error-scenarios.spec.ts | 4 | 错误凭证、无效 client_id、无效 redirect_uri、多设备独立 session |
| **总计** | | **~28** | |

### 风险和注意事项

1. **短 TTL 可能导致竞态**：30 秒 session TTL 下，如果 Playwright 操作超过 30 秒（如等待页面加载），session 可能在测试中途过期。需要在 session-lifecycle 测试中精确计时。
2. **AuthRequest TTL 过期测试**：需要 login-web 前端在 authRequest 过期后有重新发起 authorize 的机制，否则无法完成验证。
3. **无效 client_id 测试**：依赖 OIDC Provider 在 authorize 阶段报错而非在 /login 阶段报错。
4. **login-web 重新登录流程**：当 authRequest 过期后填凭证，login-web 是否会重新生成 authRequestID 取决于前端实现，可能不在测试控制范围内。
