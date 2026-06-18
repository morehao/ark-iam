# SSO 测试应用退出登录改造 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 在 SSO 测试应用中新增"本地退出"和"全局退出"两种退出方式，并更新 E2E 测试和文档。

**Architecture:** 纯 HTML/JS 内联实现，在单文件 `index.html` 中新增 `logoutLocal()` 和 `logoutGlobal()` 两个函数，并在首页和 Token 详情页 UI 中替换"重新开始"按钮为两个退出按钮。E2E 测试新增两个 helper 函数和两个测试用例。

**Tech Stack:** Vanilla HTML/JS, Playwright (E2E)

---

### Task 1: 在 index.html 新增 logoutLocal 和 logoutGlobal 函数

**Files:**
- Modify: `frontend/apps/sso-test-app/index.html:167`

- [ ] **Step 1: 在 renderLogin 函数前插入两个退出函数**

在 `function renderLogin()` 行（第 150 行）之前插入：

```javascript
function logoutLocal() {
  currentTokens = null
  window.currentTokens = null
  sessionStorage.removeItem('oidc_verifier')
  sessionStorage.removeItem('oidc_state')
  history.replaceState({}, '', '/')
  renderLogin()
}

// 全局退出：重定向到 IAM end_session
function logoutGlobal() {
  if (!currentTokens || !currentTokens.id_token) {
    logoutLocal()
    return
  }
  const params = new URLSearchParams({
    id_token_hint: currentTokens.id_token,
    post_logout_redirect_uri: CONFIG.redirectUri,
  })
  currentTokens = null
  window.currentTokens = null
  sessionStorage.removeItem('oidc_verifier')
  sessionStorage.removeItem('oidc_state')
  window.location.href = CONFIG.issuer + '/end_session?' + params.toString()
}
```

- [ ] **Step 2: 验证 HTML 语法**

```bash
node -e "const fs=require('fs');const h=fs.readFileSync('frontend/apps/sso-test-app/index.html','utf8');const s=h.indexOf('<script>')+8;const e=h.lastIndexOf('</script>');new Function('window','document','sessionStorage','history','crypto','location',h.slice(s,e))"
```

Expected: 无错误输出

- [ ] **Step 3: Commit**

```bash
git add frontend/apps/sso-test-app/index.html
git commit -m "feat(sso-test-app): add logoutLocal and logoutGlobal functions"
```

---

### Task 2: 修改 renderHomePage 按钮区

**Files:**
- Modify: `frontend/apps/sso-test-app/index.html:254-256`

- [ ] **Step 1: 替换 renderHomePage 中的"重新开始"按钮为两个退出按钮**

将 `renderHomePage` 函数中（第 254-256 行）：
```html
      <button class="btn btn-sm" style="background:#f5f5f5;color:#333" onclick="location.href='/'">
        🔁 重新开始
      </button>
```

替换为：
```html
      <button class="btn btn-sm" style="background:#f5f5f5;color:#333" onclick="logoutLocal()">
        🚪 退出当前应用
      </button>
      <button class="btn btn-sm" style="background:#ff4d4f;color:#fff" onclick="logoutGlobal()">
        🔐 从所有应用退出
      </button>
```

- [ ] **Step 2: Commit**

```bash
git add frontend/apps/sso-test-app/index.html
git commit -m "feat(sso-test-app): add logout buttons to home page"
```

---

### Task 3: 修改 renderResult Token 详情页按钮区

**Files:**
- Modify: `frontend/apps/sso-test-app/index.html:500-503`

- [ ] **Step 1: 替换 renderResult 操作区的"重新开始"按钮**

将 `renderResult` 函数中"操作"区域（第 500-503 行）的"重新开始"按钮移除，并在下方新增"退出"分区：

将：
```html
    <h2 style="margin-top:24px">🔄 操作</h2>
    <div class="btn-group">
      <button class="btn btn-success btn-sm" onclick="doFetchUserInfo()">
        📋 获取 UserInfo
      </button>
      <button class="btn btn-primary btn-sm" onclick="doRefresh()">
        🔄 刷新 Token
      </button>
      <button class="btn btn-sm" style="background:#f5f5f5;color:#333" onclick="location.href='/'">
        🔁 重新开始
      </button>
    </div>
```

替换为：
```html
    <h2 style="margin-top:24px">🔄 操作</h2>
    <div class="btn-group">
      <button class="btn btn-success btn-sm" onclick="doFetchUserInfo()">
        📋 获取 UserInfo
      </button>
      <button class="btn btn-primary btn-sm" onclick="doRefresh()">
        🔄 刷新 Token
      </button>
    </div>
    <h2 style="margin-top:16px">🚪 退出</h2>
    <div class="btn-group">
      <button class="btn btn-sm" style="background:#f5f5f5;color:#333" onclick="logoutLocal()">
        🚪 退出当前应用
      </button>
      <button class="btn btn-sm" style="background:#ff4d4f;color:#fff" onclick="logoutGlobal()">
        🔐 从所有应用退出
      </button>
    </div>
```

- [ ] **Step 2: Commit**

```bash
git add frontend/apps/sso-test-app/index.html
git commit -m "feat(sso-test-app): add logout buttons to token details page"
```

---

### Task 4: 新增 E2E helper 函数

**Files:**
- Modify: `e2e/helpers/oidc-helpers.ts:147`（文件末尾追加）

- [ ] **Step 1: 在 oidc-helpers.ts 末尾追加两个 helper 函数**

在文件末尾追加：

```typescript
export async function rp1LogoutLocal(page: Page): Promise<void> {
  await clickByText(page, '退出当前应用');
  await wait(1000);
  const body = await page.evaluate(() => document.body.innerText);
  expect(body).toContain('使用 IAM 登录');
  expect(body).toContain('您尚未登录此应用');
}

export async function rp1LogoutGlobal(page: Page): Promise<void> {
  await clickByText(page, '从所有应用退出');
  await page.waitForURL(
    (url) =>
      url.hostname === 'localhost' &&
      url.port === '3001' &&
      !url.searchParams.has('code') &&
      !url.searchParams.has('error'),
    { timeout: 30000 }
  );
  await wait(1000);
  const body = await page.evaluate(() => document.body.innerText);
  expect(body).toContain('使用 IAM 登录');
}
```

- [ ] **Step 2: 运行 TypeScript 类型检查**

```bash
npx tsc --noEmit --project e2e/tsconfig.json
```

Expected: 无类型错误

- [ ] **Step 3: Commit**

```bash
git add e2e/helpers/oidc-helpers.ts
git commit -m "test(e2e): add rp1LogoutLocal and rp1LogoutGlobal helpers"
```

---

### Task 5: 新增 E2E 测试用例

**Files:**
- Modify: `e2e/tests/oidc-sso.spec.ts:91`（文件末尾追加）

- [ ] **Step 1: 在测试文件末尾追加 2 个新测试用例**

在最后一个 `});` 之前追加：

```typescript
  test('RP1 本地退出 → 重新登录免密：退出当前应用 → 显示登录页 → 点"使用 IAM 登录" → SSO 免密进入主页', async ({ page }) => {
    await rp1Login(page);
    await rp1LogoutLocal(page);
    // SSO Session 仍有效，点击"使用 IAM 登录"应免密进入主页
    await page.click('button', { timeout: 5000 });
    try {
      await page.waitForURL(
        (url) =>
          url.hostname === 'localhost' &&
          url.port === '3001' &&
          !url.searchParams.has('authRequestID'),
        { timeout: 20000 }
      );
    } catch {}
    await page.waitForTimeout(2000);
    await verifyRp1HomePage(page);
  });

  test('RP1 全局退出 → 需重新认证：全局退出 → 显示登录页 → 需填写凭证', async ({ page }) => {
    await rp1Login(page);
    await rp1LogoutGlobal(page);
    // SSO Session 已清除，点击"使用 IAM 登录"应跳转到登录页
    await page.click('button', { timeout: 5000 });
    await page.waitForTimeout(2000);
    expect(page.url()).toContain(CONFIG.loginWebUrl);
    expect(page.url()).toContain('authRequestID=');
  });
```

- [ ] **Step 2: 更新 import 语句追加新 helper 函数导入**

将第 4-13 行的 import 更新为：

```typescript
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
  rp1LogoutLocal,
  rp1LogoutGlobal,
} from '../helpers/oidc-helpers';
```

- [ ] **Step 3: 运行 TypeScript 类型检查**

```bash
npx tsc --noEmit --project e2e/tsconfig.json
```

Expected: 无类型错误

- [ ] **Step 4: Commit**

```bash
git add e2e/tests/oidc-sso.spec.ts
git commit -m "test(e2e): add RP1 logout test cases"
```

---

### Task 6: 更新 E2E README 文档

**Files:**
- Modify: `e2e/README.md:7-13`

- [ ] **Step 1: 更新测试场景表**

将第 7-13 行的测试场景表：

```markdown
| 测试用例 | 覆盖内容 |
|----------|----------|
| RP1 首次登录 | 打开测试 RP → 跳转登录页 → 输入凭据 → 回调展示项目管理面板 |
| RP1 Token 详情 | 查看 Token → 获取 UserInfo → 刷新 Token → 返回主页 |
| 管理平台 SSO 自动登录 | RP1 登录后 → 打开管理平台 → 点击 IAM 账号登录 → 自动认证进仪表盘 |
| 管理平台登出后 SSO 已清除 | 登录 → 登出 → 再点登录应显示登录表单而非自动认证 |
```

替换为：

```markdown
| 测试用例 | 覆盖内容 |
|----------|----------|
| RP1 首次登录 | 打开测试 RP → 跳转登录页 → 输入凭据 → 回调展示项目管理面板 |
| RP1 Token 详情 | 查看 Token → 获取 UserInfo → 刷新 Token → 返回主页 |
| RP1 本地退出 | 退出当前应用 → 登录页 → 点"使用 IAM 登录" → SSO 免密进入主页 |
| RP1 全局退出 | 从所有应用退出 → 登录页 → 需重新填写凭证 |
| 管理平台 SSO 自动登录 | RP1 登录后 → 打开管理平台 → 点击 IAM 账号登录 → 自动认证进仪表盘 |
| 管理平台登出后 SSO 已清除 | 登录 → 登出 → 再点登录应显示登录表单而非自动认证 |
```

- [ ] **Step 2: Commit**

```bash
git add e2e/README.md
git commit -m "docs(e2e): add logout test scenarios to README"
```

---

### Verification

- [ ] **Step 7: 最终验证**

```bash
# 检查所有改动的文件语法
npx tsc --noEmit --project e2e/tsconfig.json
# 预期：无类型错误
```
