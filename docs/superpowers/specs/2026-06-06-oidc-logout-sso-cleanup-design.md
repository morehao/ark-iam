# OIDC 登出时 SSO Session 清理设计

> 状态：已确认 | 日期：2026-06-06

## 背景

登出功能存在两个 bug：

1. **`post_logout_redirect_uri` 未被允许**：`platform-admin-web` 客户端的 seed SQL 缺少 `post_logout_redirect_uris` 列，导致数据库中为空数组，end_session 验证失败。
2. **登出后 SSO 自动登录**：`PersistentStore.TerminateSession()` 为空实现，登出时未清除 Redis 中的 SSO session。浏览器中 `iam_sso_session` cookie 仍然有效，再次点击"登录"时 SSO 自动完成认证，无需输入密码。

Bug 1 已在 seed SQL 中修复。本文档聚焦 Bug 2。

## 设计目标

- 使用 zitadel/oidc 库的标准扩展点完成 session 清理，不依赖路由层 hack
- 登出后用户必须重新输入密码才能登录（SSO session 被正确销毁）
- 代码改动最小化，每个改动点职责单一

## 设计

### 整体流程

```
用户点击登出
  → 前端: getEndSessionURL(idToken) → end_session 端点
    → zitadel/oidc: EndSession()
      → 验证 id_token_hint（签名、过期）
      → 验证 post_logout_redirect_uri（与客户端配置匹配）
      → 调用 Storage.TerminateSession(ctx, "person:N", clientID) ← 标准扩展点
          → parseOIDCSubject → personID（uint）
          → RevokeSessionsByPersonID → 删除 Redis 中该用户所有 SSO session
      → 302 → http://localhost:3000/login

用户再次点击"登录"
  → OIDC authorize 流程
  → Provider 发现用户未认证
  → 构造 SSO login URL → 浏览器重定向到 /sso-login?authRequestID=xxx
  → SSOLogin handler:
      → 读取 iam_sso_session cookie
      → ValidateSession → Redis 无此 session → 失败
      → SetCookie("iam_sso_session", "", -1, ...) ← 防御：清除孤儿 cookie
      → 302 → 登录页（需要输入账号密码）
```

### 三层防御

| 层级 | 位置 | 职责 |
|------|------|------|
| **主路径** | `PersistentStore.TerminateSession()` | 标准扩展点，清除 Redis SSO session |
| **防御路径** | `SSOLogin` handler | session 验证失败时清除孤儿 cookie |
| **兜底路径** | `/logged-out` handler | 无 post_logout_redirect_uri 时清除 cookie |

### 关键设计决策

**为什么不实现 `CanTerminateSessionFromRequest` 接口？**

`CanTerminateSessionFromRequest` 允许返回自定义 redirect URL，但仍然无法访问 `http.ResponseWriter` 来设置 `Set-Cookie`。它的价值和 `TerminateSession` 相同，不需要额外实现。

**为什么不在 end_session 响应中清除 cookie？**

zitadel/oidc 库在调用 `TerminateSession` 后直接 `http.Redirect(w, r, redirect, 302)`，不给 OP 机会在 302 响应中追加 `Set-Cookie`。这是库的设计局限，不是本设计的问题。

**为什么 cookie 不清除也能工作？**

即使 `iam_sso_session` cookie 残留在浏览器中，Redis 中对应的 session 已被删除。SSOLogin 在 `ValidateSession` 时会失败，用户被重定向到登录页——同时 cookie 被清除。所以用户体验不受影响。

## 改动清单

### 1. 移除中间件 hack — `router/oidc.go`

**删除内容**：
- `endSessionCleanup` 中间件函数
- `parseJWTSub` 辅助函数
- `parsePersonIDFromSubject` 辅助函数
- `"encoding/base64"`、`"encoding/json"`、`"strings"` import（如无其他使用）
- `oidcGroup.Use(endSessionCleanup)` 注册行

**保留内容**：
- `/logged-out` handler 中的 cookie 清除
- 其余路由注册不变

### 2. TerminateSession 加日志 — `persistent_store.go`

```go
func (s *PersistentStore) TerminateSession(ctx context.Context, userID string, clientID string) error {
    personID, err := parseOIDCSubject(userID)
    if err != nil {
        return nil
    }
    return NewSSOSessionStore().RevokeSessionsByPersonID(ctx, personID)
}
```

改动：添加日志（使用合适项目的日志库），帮助验证调用链是否正常工作。

改动：添加日志，帮助验证调用链是否正确工作。

### 3. SSOLogin 防御 — `ctroidc/oidc.go`

在 `SSOLogin` 方法中，`CompleteLoginBySession` 失败后清除 cookie：

```go
func (ctr *OIDCCtr) SSOLogin(ctx *gin.Context) {
    authRequestID := ctx.Query("authRequestID")
    if authRequestID == "" {
        ctx.Redirect(302, config.Conf.OIDC.FrontendLoginURL)
        return
    }

    sessionID, err := ctx.Cookie("iam_sso_session")
    if err != nil {
        frontendURL := config.Conf.OIDC.FrontendLoginURL + "?authRequestID=" + url.QueryEscape(authRequestID)
        ctx.Redirect(302, frontendURL)
        return
    }

    continueURL, err := ctr.oidcAuthSvc.CompleteLoginBySession(ctx.Request.Context(), authRequestID, sessionID)
    if err != nil {
        ctx.SetCookie("iam_sso_session", "", -1, "/", "", false, true)  // ← 新增
        frontendURL := config.Conf.OIDC.FrontendLoginURL + "?authRequestID=" + url.QueryEscape(authRequestID)
        ctx.Redirect(302, frontendURL)
        return
    }

    ctx.Redirect(302, continueURL)
}
```

改动：在 `CompleteLoginBySession` 失败分支添加一行 `SetCookie`。

### 4. 无需改动 — `sso_session.go`

反向索引（`SADD`/`SREM`）和 `RevokeSessionsByPersonID` 已在上一轮实现，保持不变。

## 不受影响的部分

- 前端 OIDC 流程（`oidc.ts`、`Login.tsx`、`AuthCallback.tsx`、`MainLayout.tsx`）
- OIDC Provider 初始化（`oidc.go`）
- 路由注册（`routes.go`）
- OAuth 客户端管理
- seed SQL（Bug 1 已修复）

## 验证方式

1. 编译通过：`make build APP=iam`
2. 单元测试通过：`go test ./apps/iam/internal/service/svcoidc/...`
3. 手动验证：
   - 登录 → 进入首页
   - 点击登出 → 跳转到登录页
   - 点击登录 → **出现登录表单，需要输入账号密码**（不再自动登录）
4. 日志验证：登出时后端日志中出现 `TerminateSession` 的调用记录

## 已知限制

- 浏览器的 `iam_sso_session` cookie 不会在登出响应中被立即清除，而是在下次 SSO 登录尝试失败时被清除
- 这是 zitadel/oidc 库的架构限制，不影响功能正确性
