# SSO E2E Playwright 迁移 & 前端应用精简 实现计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 删除 sso-test-app-2，将 e2e 测试从 puppeteer-core 迁移到 Playwright Test Runner，确保测试结束自动清理进程，更新相关文档。

**Architecture:** Playwright 测试通过 `globalSetup` 启动 4 个服务（IAM 后端 + 3 个前端），`globalTeardown` 通过进程树和端口查找清理进程。测试用单个 browser context 模拟 SSO session 共享（cookie 在同一个 context 中的 page 间自动传递）。

**Tech Stack:** `@playwright/test`, TypeScript, Node.js `child_process`

---

## 文件结构总览

| 文件 | 操作 |
|------|------|
| `frontend/apps/sso-test-app-2/` | 删除 |
| `frontend/package.json` | 修改：移除 `dev:sso2`，更新 `dev:all` |
| `e2e/oidc-e2e.js` | 删除 |
| `e2e/admin-e2e.js` | 删除 |
| `e2e/package.json` | 重写：Playwright 依赖 |
| `e2e/playwright.config.ts` | 新建：Playwright 配置 |
| `e2e/global-setup.ts` | 新建：启动服务 |
| `e2e/global-teardown.ts` | 新建：停止服务 |
| `e2e/helpers/oidc-helpers.ts` | 新建：测试辅助函数 |
| `e2e/tests/oidc-sso.spec.ts` | 新建：测试用例 |
| `Makefile` | 修改：新增 `e2e` 目标，更新 `help` |
| `OIDC_SELF_TEST.md` | 修改：移除 RP2 引用 |
| `e2e/README.md` | 重写：Playwright 文档 |

---

### Task 1: 删除 sso-test-app-2 前端应用

**Files:**
- Delete: `frontend/apps/sso-test-app-2/`
- Modify: `frontend/package.json`

- [ ] **Step 1: 删除 sso-test-app-2 目录**

```bash
rm -rf frontend/apps/sso-test-app-2
```

- [ ] **Step 2: 更新 frontend/package.json — 移除 dev:sso2 脚本**

Edit `frontend/package.json`：删除 `"dev:sso2"` 行，更新 `"dev:all"` 脚本。

```json
{
  "name": "ark-iam-frontend",
  "private": true,
  "version": "1.0.0",
  "type": "module",
  "scripts": {
    "dev": "pnpm --filter @ark-iam/platform-admin-web dev",
    "dev:web": "pnpm --filter @ark-iam/platform-admin-web dev",
    "dev:sso": "pnpm --filter @ark-iam/sso-test-app dev",
    "dev:log": "pnpm --filter @ark-iam/login-web dev",
    "dev:all": "pnpm run --parallel --filter @ark-iam/platform-admin-web --filter @ark-iam/sso-test-app --filter @ark-iam/login-web dev",
    "build:web": "pnpm --filter @ark-iam/platform-admin-web build",
    "test": "pnpm --filter @ark-iam/platform-admin-web test",
    "test:web": "pnpm --filter @ark-iam/platform-admin-web test"
  },
  "packageManager": "pnpm@11.1.0+sha512.0c44e842e5686b2c061a81adda8b2258bd8818e9704b2cf2c63d56b931a7b2e910092e085027003b96ca3911ab56a07f6df5abaed2be9925034cdd686a535b14"
}
```

- [ ] **Step 3: 提交**

```bash
git add frontend/apps/sso-test-app-2 frontend/package.json
git commit -m "chore: remove sso-test-app-2, update frontend scripts to 3 apps"
```

---

### Task 2: 清理旧 e2e 文件，初始化 Playwright

**Files:**
- Delete: `e2e/oidc-e2e.js`, `e2e/admin-e2e.js`
- Modify: `e2e/package.json`

- [ ] **Step 1: 删除旧 e2e 脚本**

```bash
rm e2e/oidc-e2e.js e2e/admin-e2e.js
```

- [ ] **Step 2: 重写 e2e/package.json**

```json
{
  "name": "@ark-iam/e2e",
  "version": "1.0.0",
  "private": true,
  "description": "OIDC SSO E2E tests with Playwright",
  "scripts": {
    "test": "npx playwright test",
    "test:headed": "npx playwright test --headed",
    "test:debug": "npx playwright test --debug"
  },
  "devDependencies": {
    "@playwright/test": "^1.48.0"
  }
}
```

- [ ] **Step 3: 安装依赖并安装 Chromium 浏览器**

```bash
cd e2e && npm install && npx playwright install chromium
```

- [ ] **Step 4: 提交**

```bash
git add e2e/oidc-e2e.js e2e/admin-e2e.js e2e/package.json
git commit -m "chore: remove old puppeteer e2e scripts, add @playwright/test dependency"
```

---

### Task 3: 编写 global-setup.ts — 服务启动

**Files:**
- Create: `e2e/global-setup.ts`

- [ ] **Step 1: 创建 e2e/global-setup.ts**

```typescript
import { spawn, type ChildProcess } from 'child_process';
import * as net from 'net';
import * as path from 'path';

const ROOT = path.resolve(__dirname, '..');
const BACKEND_ROOT = path.join(ROOT, 'backend');
const FRONTEND_ROOT = path.join(ROOT, 'frontend');

interface ServiceDef {
  name: string;
  port: number;
  cmd: string;
  args: string[];
  cwd: string;
  env?: Record<string, string>;
}

const SERVICES: ServiceDef[] = [
  {
    name: 'IAM Backend',
    port: 8099,
    cmd: 'go',
    args: ['run', './apps/iam/cmd'],
    cwd: path.join(ROOT, 'backend'),
    env: {
      APP_CONFIG_PATH: path.join(ROOT, 'backend', 'apps', 'iam', 'config', 'config.yaml'),
    },
  },
  {
    name: 'platform-admin-web',
    port: 3000,
    cmd: 'pnpm',
    args: ['--filter', '@ark-iam/platform-admin-web', 'dev'],
    cwd: FRONTEND_ROOT,
  },
  {
    name: 'sso-test-app',
    port: 3001,
    cmd: 'pnpm',
    args: ['--filter', '@ark-iam/sso-test-app', 'dev'],
    cwd: FRONTEND_ROOT,
  },
  {
    name: 'login-web',
    port: 3003,
    cmd: 'pnpm',
    args: ['--filter', '@ark-iam/login-web', 'dev'],
    cwd: FRONTEND_ROOT,
  },
];

async function checkPort(port: number): Promise<boolean> {
  return new Promise((resolve) => {
    const hosts = ['127.0.0.1', '::1'];
    let tried = 0;
    for (const host of hosts) {
      const socket = new net.Socket();
      socket.setTimeout(1500);
      socket.once('connect', () => { socket.destroy(); resolve(true); });
      socket.once('error', () => { socket.destroy(); });
      socket.once('timeout', () => { socket.destroy(); });
      socket.once('close', () => { tried++; if (tried === hosts.length) resolve(false); });
      socket.connect(port, host);
    }
  });
}

async function waitForPort(port: number, label: string, timeoutMs: number): Promise<boolean> {
  const start = Date.now();
  while (Date.now() - start < timeoutMs) {
    if (await checkPort(port)) return true;
    await new Promise((r) => setTimeout(r, 1000));
  }
  return false;
}

const children: ChildProcess[] = [];

async function globalSetup() {
  console.log('\n[globalSetup] checking and starting services...\n');

  const needStart: ServiceDef[] = [];

  for (const svc of SERVICES) {
    const running = await checkPort(svc.port);
    if (running) {
      console.log(`  ✅ ${svc.name} (port ${svc.port}) already running`);
    } else {
      console.log(`  ⏳ ${svc.name} (port ${svc.port}) starting...`);
      needStart.push(svc);
    }
  }

  if (needStart.length === 0) {
    console.log('  All services ready\n');
    process.env.E2E_SERVICE_CHILDREN = JSON.stringify([]);
    return;
  }

  await Promise.all(
    needStart.map(
      (svc) =>
        new Promise<void>((resolve, reject) => {
          const opts: any = {
            cwd: svc.cwd,
            stdio: ['ignore', 'pipe', 'pipe'],
            detached: true,
          };
          if (svc.env) {
            opts.env = { ...process.env, ...svc.env };
          }
          const child = spawn(svc.cmd, svc.args, opts);
          children.push(child);

          // collect stdout/stderr but don't print (keeps output clean)
          child.stdout?.on('data', () => {});
          child.stderr?.on('data', () => {});

          child.on('error', (err) => {
            console.error(`[${svc.name}] spawn error: ${err.message}`);
            reject(err);
          });

          waitForPort(svc.port, svc.name, 180000).then((ok) => {
            if (ok) {
              console.log(`  ✅ ${svc.name} ready`);
              resolve();
            } else {
              console.error(`  ❌ ${svc.name} timed out`);
              reject(new Error(`${svc.name} startup timeout`));
            }
          });
        })
    )
  );

  console.log('  All services ready\n');
  console.log('[globalSetup] complete\n');

  // store child PIDs for teardown via env
  process.env.E2E_SERVICE_CHILDREN = JSON.stringify(children.map((c) => c.pid));
}

export default globalSetup;
```

- [ ] **Step 2: 提交**

```bash
git add e2e/global-setup.ts
git commit -m "feat: add Playwright global-setup for service orchestration"
```

---

### Task 4: 编写 global-teardown.ts — 进程清理

**Files:**
- Create: `e2e/global-teardown.ts`

- [ ] **Step 1: 创建 e2e/global-teardown.ts**

```typescript
import { spawn, execSync } from 'child_process';

const PORTS = [8099, 3000, 3001, 3003];
const PORT_LABELS: Record<number, string> = {
  8099: 'IAM Backend',
  3000: 'platform-admin-web',
  3001: 'sso-test-app',
  3003: 'login-web',
};

function killByPort(port: number, label: string): Promise<void> {
  return new Promise((resolve) => {
    const child = spawn('lsof', ['-ti', `:${port}`], { stdio: ['ignore', 'pipe', 'pipe'] });
    let pidStr = '';
    child.stdout.on('data', (d: Buffer) => { pidStr += d.toString(); });
    child.on('close', () => {
      if (!pidStr.trim()) { resolve(); return; }
      const pids = pidStr.trim().split('\n').map((p) => parseInt(p)).filter(Boolean);
      for (const pid of pids) {
        try { process.kill(pid, 'SIGTERM'); console.log(`  ⏹ ${label} (PID ${pid})`); } catch {}
      }
      // after 2s, force kill remaining
      setTimeout(() => {
        for (const pid of pids) {
          try { process.kill(pid, 0); process.kill(pid, 'SIGKILL'); console.log(`  ☠ ${label} (PID ${pid}) force killed`); } catch {}
        }
        resolve();
      }, 2000);
    });
    child.on('error', () => resolve());
  });
}

// also kill child processes we started
async function killChildren() {
  try {
    const childrenJson = process.env.E2E_SERVICE_CHILDREN;
    if (childrenJson && childrenJson !== '[]') {
      const pids: number[] = JSON.parse(childrenJson);
      for (const pid of pids) {
        try { process.kill(-pid, 'SIGTERM'); } catch {}
      }
    }
  } catch {}
}

async function globalTeardown() {
  console.log('\n[globalTeardown] stopping services...\n');

  // first try to kill via process group
  await killChildren();

  // then clean up by port
  await Promise.all(PORTS.map((port) => killByPort(port, PORT_LABELS[port])));

  console.log('[globalTeardown] complete\n');
}

export default globalTeardown;
```

- [ ] **Step 2: 提交**

```bash
git add e2e/global-teardown.ts
git commit -m "feat: add Playwright global-teardown for process cleanup"
```

---

### Task 5: 编写 playwright.config.ts

**Files:**
- Create: `e2e/playwright.config.ts`

- [ ] **Step 1: 创建 e2e/playwright.config.ts**

```typescript
import { defineConfig } from '@playwright/test';

export default defineConfig({
  testDir: './tests',
  timeout: 120000,
  expect: {
    timeout: 10000,
  },
  globalSetup: './global-setup.ts',
  globalTeardown: './global-teardown.ts',
  use: {
    baseURL: 'http://localhost:3001',
    headless: true,
    browserName: 'chromium',
    ignoreHTTPSErrors: true,
    viewport: { width: 1280, height: 720 },
  },
  retries: 0,
  workers: 1,
  reporter: 'list',
});
```

- [ ] **Step 2: 提交**

```bash
git add e2e/playwright.config.ts
git commit -m "feat: add Playwright config for e2e tests"
```

---

### Task 6: 编写 oidc-helpers.ts — 测试辅助函数

**Files:**
- Create: `e2e/helpers/oidc-helpers.ts`

- [ ] **Step 1: 创建目录并编写 e2e/helpers/oidc-helpers.ts**

```bash
mkdir -p e2e/helpers e2e/tests
```

```typescript
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
```

- [ ] **Step 2: 提交**

```bash
git add e2e/helpers/oidc-helpers.ts
git commit -m "feat: add Playwright e2e helpers for OIDC SSO tests"
```

---

### Task 7: 编写 oidc-sso.spec.ts — 测试用例

**Files:**
- Create: `e2e/tests/oidc-sso.spec.ts`

- [ ] **Step 1: 创建 e2e/tests/oidc-sso.spec.ts**

```typescript
import { test, expect } from '@playwright/test';
import {
  CONFIG,
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
```

- [ ] **Step 2: 提交**

```bash
git add e2e/tests/oidc-sso.spec.ts
git commit -m "feat: add Playwright e2e tests for OIDC SSO (RP1 + platform admin)"
```

---

### Task 8: 更新 Makefile — e2e 目标

**Files:**
- Modify: `Makefile`

- [ ] **Step 1: 在 Makefile 中新增 e2e 目标**

在 `stop-frontend` 目标之后（第 250 行附近），新增 `e2e` 目标：

```makefile
# 运行 E2E 测试（需先安装依赖：cd e2e && npm install && npx playwright install chromium）
.PHONY: e2e
e2e:
	@echo "🧪 运行 E2E 测试..."
	@cd e2e && npx playwright test
```

在 `.PHONY` 声明（第 34 行）末尾追加 `e2e`：

```makefile
.PHONY: all build build-env clean run lint test swag codegen \
        docker-build docker-run check-image \
        list-apps deps tidy update-dep dev-frontend stop-frontend e2e help
```

在 `help` 目标的"测试 & 检查"部分，`make lint` 之后新增一行：

```makefile
	@echo "    make e2e                                运行 E2E 测试（Playwright）"
```

- [ ] **Step 2: 提交**

```bash
git add Makefile
git commit -m "feat: add make e2e target for Playwright tests"
```

---

### Task 9: 更新 OIDC_SELF_TEST.md — 移除 RP2 引用

**Files:**
- Modify: `OIDC_SELF_TEST.md`

- [ ] **Step 1: 更新种子数据表（第 33 行）**

删除第 33 行的 `http://localhost:3002/` 回调地址：将：
```
| 回调地址 | `http://localhost:3001/`, `http://localhost:3002/` |
```
改为：
```
| 回调地址 | `http://localhost:3001/`, `http://localhost:3000/` |
```

- [ ] **Step 2: 重写 Step 4（第 67-96 行）**

将：
```markdown
## Step 4 — 启动 SSO 测试应用

`sso-test-app` 和 `sso-test-app-2` 是 monorepo 下的子应用（`frontend/apps/sso-test-app/`、`frontend/apps/sso-test-app-2/`），基于 Vite 开发。

**从 monorepo 根目录同时启动 RP1 + RP2：**

```bash
cd frontend
pnpm dev:sso
pnpm dev:sso2
```

**或从子应用目录单独启动：**

```bash
cd frontend/apps/sso-test-app
pnpm dev

# 另一个终端
cd frontend/apps/sso-test-app-2
pnpm dev
```

| 应用 | 端口 | 说明 |
|------|------|------|
| sso-test-app | 3001 | SSO 测试 RP（首次手动登录） |
| sso-test-app-2 | 3002 | SSO 测试 RP（自动 SSO 登录） |

sso-test-app 和 sso-test-app-2 均无 API 代理，直接请求后端 `:8099`。
```

改为：

```markdown
## Step 4 — 启动 SSO 测试应用

`sso-test-app` 是 monorepo 下的子应用（`frontend/apps/sso-test-app/`），基于 Vite 开发。

**启动：**

```bash
cd frontend
pnpm dev:sso
```

**或从子应用目录单独启动：**

```bash
cd frontend/apps/sso-test-app
pnpm dev
```

| 应用 | 端口 | 说明 |
|------|------|------|
| sso-test-app | 3001 | SSO 测试 RP（首次手动登录） |

sso-test-app 无 API 代理，直接请求后端 `:8099`。
```

- [ ] **Step 3: 更新 Step 5 流程图（第 103-143 行）**

将整个 ASCII 流程图替换为简化版，移除 RP2 列：

```markdown
```
[测试RP:3001]                        [IAM后端:8099]
     |                                      |
     |-- 1. 点击"使用IAM登录" ------------>|
     |                                      |-- 2. GET /authorize -------------->
     |                                      |
     |                                      |<-- 3. 302 → /sso-login ------------
|<-- 4. 302 → :3003/login ------------|
|    ?authRequestID=ar-xxx             |
     |                                      |
     |-- 5. POST /oidc/login ------------->|
     |    (admin/admin123)                  |
     |                                      |
     |<-- 6. {continueURL, sessionID} ------|
     |    + Set-Cookie: iam_sso_session     |
     |                                      |
     |-- 7. 302 → continueURL ------------>|
     |<-- 8. 302 → :3001/?code=xxx --------|
     |                                      |
     |-- 9. POST /oauth/token ------------->|
     |<-- 10. {access_token, id_token} -----|
     |-- 10b. 展示"项目管理面板"主页       |
```
```

- [ ] **Step 4: 重写 Step 6（第 169-189 行）— "双 RP SSO 验证" → "管理平台 SSO 验证"**

将整个 Step 6 替换为：

```markdown
## Step 6 — 管理平台 SSO 自动登录验证

SSO 验证中，改以管理平台（platform-admin-web）作为第二个 RP，替代 sso-test-app-2。

### 操作步骤

1. ✅ 完成 Step 5（RP1 已成功登录，浏览器已有 `iam_sso_session` cookie）
2. 打开新标签页 `http://localhost:3000/`
3. 页面显示管理平台登录页，点击 **"IAM 账号登录"**
4. IAM 检测到已有 session cookie → **自动签发授权码**，重定向回管理平台
5. 管理平台自动完成令牌交换，展示 **仪表盘**

### 验证要点

- ✅ RP1 首次登录：跳转到 IAM 登录页 → 输入凭据 → 展示"项目管理面板"主页
- ✅ 管理平台 SSO 登录：点击"IAM 账号登录"后**无登录表单出现**，直接进入仪表盘
- ✅ 仪表盘展示统计信息（用户总数、角色总数、部门总数、应用总数）
- ✅ 侧边栏菜单完整（用户管理、角色管理、部门管理、应用管理、租户管理、OAuth 客户端）
```

- [ ] **Step 5: 更新 Step 7 SSO 原理说明（第 193-204 行）**

将表格中的 RP2 条目合并：

```markdown
| 验证点 | 说明 | 结果 |
|--------|------|------|
| 首次登录需凭据 | RP 首次访问 IAM IdP，无 session cookie，显示登录页 | ✅ |
| SSO 自动认证 | 管理平台访问 IAM IdP，携带 `iam_sso_session` cookie，自动完成认证 | ✅ |
| 身份令牌签发 | IAM 签发 id_token，含 sub(personID) 等声明 | ✅ |
| 跨应用令牌 | 同一用户在不同应用获取不同的授权码和令牌 | ✅ |

**SSO 的本质** — 用户只需要在 IAM（IdP）进行一次认证后，浏览器获得 `iam_sso_session` cookie。后续任意应用（测试 RP、管理平台）发起 OIDC 授权请求时，服务端 `/sso-login` 端点检测到有效 cookie，自动完成用户认证并生成授权码，用户全程无需干预。
```

- [ ] **Step 6: 更新 Step 8 — E2E 测试说明（第 208-271 行）**

将整个 Step 8 替换为 Playwright 版：

```markdown
## Step 8 — 自动化端到端测试

项目提供了基于 Playwright 的端到端自动化测试，模拟真实浏览器操作验证 OIDC SSO 完整流程。

### 前置条件

- 已完成 Step 1 ~ Step 4（数据库、后端、login-web、sso-test-app 全部启动并运行）
- 安装 Playwright：`cd e2e && npm install && npx playwright install chromium`

### 运行

```bash
# 如果服务未启动，Playwright 的 globalSetup 会自动启动所需服务
cd e2e

# 执行端到端测试（headless 模式）
npx playwright test

# 有头模式（可视化调试）
npx playwright test --headed

# 调试模式
npx playwright test --debug
```

### 输出示例

```
Running 3 tests using 1 worker

  ✓  RP1 首次登录 (25.3s)
  ✓  RP1 Token 详情 (22.1s)
  ✓  管理平台 SSO 自动登录 (28.6s)

  3 passed (76s)
```

退出码：全部通过返回 0，有失败返回 1（适合接入 CI）。

### 进程清理

测试通过 `globalSetup` 启动所需服务（如果未运行），`globalTeardown` 确保测试结束后（无论成功/失败）自动停止所有进程。无需手动清理。

### 常见失败排查

- **`RP1 首页加载` 失败** → 检查 `sso-test-app` 是否在 3001 端口运行
- **跳转到登录页失败** → 检查 `login-web` 是否在 3003 端口运行
- **`Token 刷新成功` 失败** → 检查 sso-test-app 的 scope 是否包含 `offline_access`
- **管理平台 SSO 登录失败** → 检查 `platform-admin-web` 是否在 3000 端口运行
```

- [ ] **Step 7: 更新环境速查表（第 292-305 行）**

移除 RP2 行：

```markdown
| 服务 | 地址 | 端口 | 说明 |
|------|------|------|------|
| MySQL | `127.0.0.1:3306` | 3306 | 数据库 iam |
| Redis | `127.0.0.1:6379` | 6379 | 缓存 |
| IAM 后端 | `http://localhost:8099` | 8099 | Gin HTTP 服务 |
| IAM 前端 | `http://localhost:3000` | 3000 | React SPA（管理端） |
| 独立登录页 | `http://localhost:3003` | 3003 | Vite + React（独立 IdP 登录页） |
| SSO 测试 RP | `http://localhost:3001` | 3001 | 静态 HTML 测试页 |
| OIDC Issuer | `http://localhost:8099/v1/iam/oidc` | - | OIDC Provider 根路径 |
| OIDC 登录页 | `http://localhost:3003/login` | - | 独立登录页服务（login-web） |
| SSO Session | `iam_sso_session` cookie | - | HTTP-only，Redis 存储，24h 过期 |
```

- [ ] **Step 8: 更新常见问题排查（第 283-337 行）**

移除涉及 `sso-test-app-2` / 端口 3002 的排查项：

原文本第 287 行：
```
- **`Token 刷新成功` 失败** → 检查 `sso-test-app` 和 `sso-test-app-2` 的 scope 是否包含 `offline_access`（已修复，源码默认带上）
```
改为：
```
- **`Token 刷新成功` 失败** → 检查 sso-test-app 的 scope 是否包含 `offline_access`（已修复，源码默认带上）
```

原文本第 333-337 行（自动化测试排查）：
```
**Q: 自动化测试用例不通过？**
A: 运行 `cd e2e && npm run test:oidc` 逐项查看失败项。常见原因：
- 服务未启动：检查 8099 / 3001 / 3002 / 3003 端口
- Chrome 路径不对：编辑 `e2e/oidc-e2e.js` 的 `CONFIG.chromePath`
- 端口被占用：清理占用进程或改端口
```
改为：
```
**Q: 自动化测试用例不通过？**
A: 运行 `cd e2e && npx playwright test` 逐项查看失败项。常见原因：
- 服务未启动：检查 8099 / 3001 / 3000 / 3003 端口
- 端口被占用：清理占用进程或改端口
```

- [ ] **Step 9: 提交**

```bash
git add OIDC_SELF_TEST.md
git commit -m "docs: update OIDC_SELF_TEST.md - remove RP2, migrate to Playwright e2e"
```

---

### Task 10: 重写 e2e/README.md

**Files:**
- Modify: `e2e/README.md`

- [ ] **Step 1: 重写 e2e/README.md**

```markdown
# E2E 自动化测试

本目录存放基于 Playwright 的 OIDC SSO 端到端测试，模拟真实浏览器操作验证完整业务流程。

## 测试场景

| 测试用例 | 覆盖内容 |
|----------|----------|
| RP1 首次登录 | 打开测试 RP → 跳转登录页 → 输入凭据 → 回调展示项目管理面板 |
| RP1 Token 详情 | 查看 Token → 获取 UserInfo → 刷新 Token → 返回主页 |
| 管理平台 SSO 自动登录 | RP1 登录后 → 打开管理平台 → 点击 IAM 账号登录 → 自动认证进仪表盘 |

测试基于 Playwright 的 browser context 自动共享 cookie，模拟真实的 SSO session 行为。

## 前置条件

- Node.js 18+
- MySQL + Redis 已运行
- 后端种子数据已导入（`admin` / `admin123` + `test-rp-client`）

## 安装

```bash
cd e2e
npm install
npx playwright install chromium
```

## 运行

```bash
# headless 模式（CI 推荐）
npm test

# 有头模式（可视化调试）
npm run test:headed

# 单步调试
npm run test:debug
```

测试自动管理服务生命周期：
- `globalSetup` — 检查并启动所需服务（IAM 后端 :8099、platform-admin-web :3000、sso-test-app :3001、login-web :3003）
- `globalTeardown` — 测试结束后强制清理所有进程（无论成功/失败）

## 配置

配置文件：`playwright.config.ts`

| 配置项 | 值 | 说明 |
|--------|-----|------|
| headless | true | headless 运行 |
| browser | chromium | 浏览器类型 |
| workers | 1 | 单 worker（确保服务状态一致） |
| timeout | 120s | 单个测试超时 |

## 项目集成

通过 Makefile：

```bash
make e2e
```
```

- [ ] **Step 2: 提交**

```bash
git add e2e/README.md
git commit -m "docs: update e2e README for Playwright migration"
```

---

### Task 11: 验证

- [ ] **Step 1: 确认所有文件已创建和修改**

```bash
ls -la e2e/
ls -la e2e/helpers/
ls -la e2e/tests/
git status
```

- [ ] **Step 2: TypeScript 编译检查**

```bash
cd e2e && npx tsc --noEmit --module commonjs --target es2020 --esModuleInterop --skipLibCheck playwright.config.ts global-setup.ts global-teardown.ts helpers/oidc-helpers.ts tests/oidc-sso.spec.ts
```

- [ ] **Step 3: 运行 E2E 测试（需 MySQL/Redis 可用）**

```bash
make e2e
```

Expected: 3 tests pass, processes cleaned up.

- [ ] **Step 4: 最终提交（如有修复）**

```bash
git add -A
git commit -m "chore: final verification fixes for e2e playwright migration"
```

---

## 注意事项

1. **Playwright 的 browser context 共享 cookie**：在同一个 `test()` 中使用 `context.newPage()` 创建的页面会自动共享 cookie（包括 `iam_sso_session`），无需手动 `setCookie`。
2. **globalTeardown 可靠性**：使用 SIGTERM + 2s 后 SIGKILL 双重保障，即使测试异常退出也会通过 `lsof` 按端口清理。
3. **backend config.yaml 路径**：`global-setup.ts` 通过 `APP_CONFIG_PATH` 环境变量指定，确保后端使用正确的配置。
4. **不变更内容**：login-web、platform-admin-web、sso-test-app 的代码逻辑完全不变。
