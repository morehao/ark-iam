# SSO 登录流程重构设计文档

## 目标

用 `react-oidc-context` 替换自研的 SSO 实现，修复以下问题：
1. 静默登录使用 `window.location.replace` 导致页面闪烁和状态丢失
2. Token 刷新被动等待后端返回 401，多一次失败往返
3. 刷新失败跳转到 `/login` 而非 OIDC Provider
4. 登出时 `getEndSessionURL` 的 base URL 使用了 `window.location.origin` 而非 OIDC Provider 地址

## 替换范围

| 应用 | 操作 |
|------|------|
| `platform-admin-web` | 全面替换 |
| `sso-test-app` | 全面替换 |
| `login-web` | 不涉及 |
| 后端 | 本次不改 |

## 架构变更

### 替换前（当前自研）

```
App.tsx
  ├─ useEffect → silent PKCE redirect (window.location.replace)
  ├─ Login.tsx → 手动 buildAuthorizeURL
  ├─ AuthCallback.tsx → exchangeCodeForTokens
  └─ stores/authStore.ts → zustand persist 管理 token
       utils/oidc.ts → PKCE/Token/Logout 工具函数
       utils/request.ts → 被动等待 TokenExpired 再刷新
       MainLayout.tsx → 手动拼接 end_session URL + visibilitychange 检测
```

### 替换后（react-oidc-context）

```
main.tsx
  └─ <AuthProvider config={oidcConfig}>
       ├─ automaticSilentRenew: true (主动定时刷新)
       ├─ onSigninSilent: 刷新失败 → removeUser + signinRedirect
       └─ <App />
            ├─ useAuth() → 路由守卫
            ├─ request.ts → 请求拦截器注入 token
            └─ MainLayout.tsx → auth.signoutRedirect()
```

## 数据流

### 初始化流程

```
用户打开页面
  └─ AuthProvider 初始化
       ├─ 调用 getUser() 检查 User Store（localStorage）
       │    ├─ 有有效 user → auth.isAuthenticated = true → 渲染应用
       │    ├─ 有 user 但 access_token 过期 → 尝试 silentRenew
       │    ├─ refresh_token 失效 → signinRedirect()
       │    └─ 无 user → signinRedirect()
       └─ autoSignIn: true

App 路由守卫
  └─ useAuth().isAuthenticated
       ├─ true → 渲染 MainLayout
       └─ false → 当前在 /auth/callback → 显示加载中
                → 其他路径 → signinRedirect()
```

### 静默刷新流程

```
automaticSilentRenew: true
  └─ 内部定时器（60s 检查一次 exp）
       └─ access_token 剩余有效期 < 60s
            ├─ 用 refresh_token grant 请求 token endpoint
            │    └─ 成功 → 更新 user，触发 addUserLoaded 事件
            └─ 失败 → 触发 addSilentRenewError 事件
                 └─ removeUser() → isAuthenticated = false
                    └─ App 路由守卫检测 → signinRedirect()
```

### 登出流程

```
用户点击退出
  └─ auth.signoutRedirect()
       ├─ removeUser() 清空本地 user
       ├─ 调后端 logoutAllAPI 注销 refresh token
       └─ 跳转 OIDC Provider end_session 端点
            └─ Provider 清除 SSO Cookie → 重定向回 /login
```

### Token 注入

```
request 拦截器
  └─ 从 User 对象获取 access_token
       └─ Authorization: Bearer {access_token}
            └─ 每次请求前 automaticSilentRenew 已确保 token 有效
                 └─ 兜底：TokenExpired/TokenInvalid → removeUser() + signinRedirect()
```

### State 持久化

使用 `WebStorageStateStore({ store: localStorage })`，刷新页面后 token 仍然存在，AuthProvider 能快速恢复 session。

## 配置

### platform-admin-web

```ts
{
  authority: import.meta.env.VITE_OIDC_ISSUER || '/v1/iam/oidc',
  client_id: import.meta.env.VITE_OIDC_CLIENT_ID || 'platform-admin-web',
  redirect_uri: window.location.origin + '/auth/callback',
  post_logout_redirect_uri: window.location.origin + '/login',
  scope: 'openid profile email offline_access',
  automaticSilentRenew: true,
  monitorSession: false,
  loadUserInfo: true,
  userStore: new WebStorageStateStore({ store: localStorage }),
}
```

### sso-test-app

```ts
{
  authority: import.meta.env.VITE_OIDC_ISSUER || '/v1/iam/oidc',
  client_id: import.meta.env.VITE_OIDC_CLIENT_ID || 'test-rp-client',
  client_secret: 'my-test-client-secret',
  token_endpoint_auth_method: 'client_secret_basic',
  redirect_uri: window.location.origin + '/auth/callback',
  post_logout_redirect_uri: window.location.origin + '/login',
  scope: 'openid profile email offline_access',
  automaticSilentRenew: true,
  monitorSession: false,
  loadUserInfo: true,
  userStore: new WebStorageStateStore({ store: localStorage }),
}
```

## 错误处理

### Token 相关

| 场景 | 触发方式 | 处理 |
|------|---------|------|
| access_token 过期 | automaticSilentRenew 主动检测 | 定时用 refresh_token 刷新，请求不会失败 |
| refresh_token 失效 | silentRenew 失败 | removeUser() → signinRedirect() |
| 后端返回 TokenExpired（兜底） | 响应拦截器 | removeUser() → signinRedirect() |
| 后端返回 TokenInvalid/Unauthorized | 响应拦截器 | removeUser() → signinRedirect() |
| HTTP 401 | Axios error 拦截器 | removeUser() → signinRedirect() |

### 业务错误

Forbidden/PermissionDenied 等其他业务错误码处理逻辑保持不变。

## 文件变更清单

| 文件 | 操作 | 说明 |
|------|------|------|
| `main.tsx` | 修改 | 包裹 AuthProvider |
| `App.tsx` | 修改 | 简化路由守卫 |
| `Login.tsx` | 修改 | auth.signinRedirect() |
| `AuthCallback.tsx` | 删除 | AuthProvider 自动处理回调 |
| `authStore.ts` | 删除 | useAuth() 替代 |
| `oidc.ts` | 删除 | oidc-client-ts 内置替代 |
| `request.ts` | 修改 | 简化拦截器，删除 TokenExpired 处理 |
| `MainLayout.tsx` | 修改 | signoutRedirect() |
| `packages/shared/src/types/oidc.ts` | 删除 | 不再需要 |
| `packages/shared/src/types/auth.ts` | 修改 | 删除不需要的类型 |

## 关键决策

1. `automaticSilentRenew: true` + `onSigninSilent` 处理 session 失效
2. `monitorSession: false`（不实现 OIDC Session Management iframe 机制）
3. localStorage 持久化（`WebStorageStateStore`）
4. 保留响应拦截器兜底错误处理
5. 本次不改后端，`/v1/iam/oidc/session/status` 接口后续清理
6. `platform-admin-web` 和 `sso-test-app` 同步替换
