# SSO 测试应用退出登录改造设计

## 1. 背景

当前 SSO 测试应用（`frontend/apps/sso-test-app/index.html`）已支持 OIDC 登录流程，但缺少退出登录功能。用户登录后只能通过"重新开始"（页面刷新）回到登录页，无法模拟真实的退出场景，不利于 SSO 流程测试。

## 2. 目标

- 在 SSO 测试应用中新增两种退出方式：本地退出和全局退出
- 更新 E2E 测试覆盖退出场景
- 更新相关文档

## 3. 设计

### 3.1 退出方式

| 方式 | 按钮文案 | 行为 | SSO Session |
|------|---------|------|-------------|
| 本地退出 | "退出当前应用" | 清除本地 token/state，回到登录页 | 保留 |
| 全局退出 | "从所有应用退出" | 重定向到 IAM `/end_session` 清除 SSO Session，回调回应用 | 清除 |

### 3.2 新增函数

```javascript
// 仅退出当前应用，保留 SSO Session
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
  window.location.href = CONFIG.issuer + '/end_session?' + params.toString()
}
```

### 3.3 UI 变更

**首页（`renderHomePage`）**——操作按钮区替换"重新开始"按钮：

```
旧：[前往 SSO 测试应用 2] [查看 Token 详情] [重新开始]
新：[前往 SSO 测试应用 2] [查看 Token 详情] [退出当前应用] [从所有应用退出]
```

**Token 详情页（`renderResult`）**——操作区底部新增两个按钮，替换返回主页区域的"重新开始"：

```
旧：[获取 UserInfo] [刷新 Token] [重新开始]
     [返回主页]

新：[获取 UserInfo] [刷新 Token]
     [退出当前应用] [从所有应用退出]
     [返回主页]
```

### 3.4 回调处理

全局退出后，IAM 回调到 `post_logout_redirect_uri`。应用初始化时检测到 URL 中无 `code` 也无 `error` 参数，正常展示登录页（无需额外处理）。

## 4. E2E 测试变更

### 4.1 helpers/oidc-helpers.ts 新增

```typescript
// 本地退出
export async function rp1LogoutLocal(page: Page): Promise<void>

// 全局退出
export async function rp1LogoutGlobal(page: Page): Promise<void>
```

### 4.2 新增测试用例

| 测试用例 | 覆盖内容 |
|----------|----------|
| RP1 本地退出 → 重新登录免密 | 本地退出 → 显示登录页 → 点"使用 IAM 登录" → SSO 免密进入主页 |
| RP1 全局退出 → 需重新认证 | 全局退出 → 显示登录页 → 点"使用 IAM 登录" → 需填写凭证 |

### 4.3 现有测试用例适配

因按钮文案变化，需调整 `oidc-helpers.ts` 中以下定位器：
- `clickByText(page, '查看 Token 详情')` — 文案不变，无需修改
- `clickByText(page, '返回主页')` — 文案不变，无需修改
- `verifyRp1RequiresLogin` — 检查"使用 IAM 登录"和"您尚未登录此应用"文案不变

## 5. 文档变更

### 5.1 e2e/README.md

测试场景表新增 2 行退出登录相关测试用例。

### 5.2 docs/oidc-sso-integration.md

无需修改（该文档描述 IAM Provider 接入规范，不涉及测试应用实现细节）。

## 6. 风险与约束

- 无框架依赖，纯 HTML/JS 内联实现
- 全局退出依赖 IAM `/end_session` 端点可用
- 退出后 sessionStorage 清理不彻底不会影响下次正常登录（OIDC 流程每次生成新的 state/verifier）
