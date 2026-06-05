# OIDC SSO 功能设计

## 概述

为 IAM 的 OIDC 流程添加真正的服务端 SSO（Single Sign-On）能力。用户在首次登录 IAM IdP 后，访问任意 OIDC RP（Relying Party）无需重新输入凭据。

## 现状

当前 OIDC 流程中，每次授权请求都经过前端登录页（`/oidc/login`），用户必须手动输入凭据。后端无会话机制，`LoginURL` 直接返回前端地址。

## 核心方案

在 `/oidc/login` 成功后，后端设置 HTTP-only cookie `iam_sso_session`。后续 RP 的授权请求到达时，先通过服务端端点 `/sso-login` 检查此 cookie，有则自动完成认证、无需前端参与。

## 架构

### SSO 流程

**第一次登录（RP1）：**
```
[RP1:3001] → GET /authorize → zitadel 创建 auth request
  → 调用 LoginURL → /sso-login?authRequestID=ar-xxx
  → 无 cookie → 302 到前端登录页 :3000/oidc/login?authRequestID=ar-xxx
  → 用户输入凭据 → POST /oidc/login
  → 认证成功 → 设置 iam_sso_session cookie → 返回 continueURL
  → 前端 302 到 continueURL → zitadel callback → code → RP1 换令牌
```

**第二次登录（RP2，SSO）：**
```
[RP2:3002] → GET /authorize → zitadel 创建 auth request
  → 调用 LoginURL → /sso-login?authRequestID=ar-yyy
  → 有有效 cookie → 自动 CompleteAuthRequest
  → 302 到 continueURL → zitadel callback → code → RP2 换令牌
  → 用户始终看不到登录表单
```

## 改动清单

### 1. 后端新增：SSO Session 管理

**文件:** `backend/apps/iam/internal/service/svcoidc/sso_session.go`

- Redis 存储 SSO session
- Key: `iam:oidc:sso_session:{sessionID}`
- Value: `{"personID": 1}`
- TTL: 86400s（24h），可通过 `config.OIDC.SessionTTL` 配置
- sessionID 使用 `crypto/rand` 生成 UUID

接口：
```go
type SSOSessionStore interface {
    CreateSession(ctx context.Context, personID uint) (string, error)
    ValidateSession(ctx context.Context, sessionID string) (uint, error)
    RevokeSession(ctx context.Context, sessionID) error
}
```

### 2. 后端修改：LoginURL

**文件:** `backend/apps/iam/internal/service/svcoidc/client.go`

将 `LoginURL` 从返回前端地址改为返回后端 SSO 检测端点：

```go
func (c *OIDCClient) LoginURL(id string) string {
    return c.issuer + "/sso-login?authRequestID=" + url.QueryEscape(id)
}
```

`issuer` 即 `http://localhost:8099/v1/iam/oidc`。

### 3. 后端新增：/sso-login 端点

**文件:** `backend/apps/iam/internal/service/svcoidc/routes.go`

新增路由：

```go
oidcGroup.GET("/sso-login", ctr.SSOLogin)
```

**控制器 `ctroidc/oidc.go`：**

```go
func (ctr *OIDCCtr) SSOLogin(ctx *gin.Context) {
    authRequestID := ctx.Query("authRequestID")
    if authRequestID == "" {
        ctx.Redirect(http.StatusFound, config.Conf.OIDC.FrontendLoginURL)
        return
    }

    sessionID, err := ctx.Cookie("iam_sso_session")
    if err != nil {
        // 无 cookie，重定向到前端登录页
        ctx.Redirect(http.StatusFound, config.Conf.OIDC.FrontendLoginURL+"?authRequestID="+url.QueryEscape(authRequestID))
        return
    }

    continueURL, err := ctr.oidcAuthSvc.CompleteLoginBySession(ctx, authRequestID, sessionID)
    if err != nil {
        // session 无效，重定向到前端登录页
        ctx.Redirect(http.StatusFound, config.Conf.OIDC.FrontendLoginURL+"?authRequestID="+url.QueryEscape(authRequestID))
        return
    }

    ctx.Redirect(http.StatusFound, continueURL)
}
```

**服务层 `svcoidc/oidc.go`：**

```go
func (svc *oidcAuthSvc) CompleteLoginBySession(ctx context.Context, authRequestID string, sessionID string) (string, error) {
    personID, err := svc.ssoSessionStore.ValidateSession(ctx, sessionID)
    if err != nil {
        return "", err
    }

    if _, err := svc.provider.Storage.AuthRequestByID(ctx, authRequestID); err != nil {
        return "", err
    }

    authTime := time.Now()
    if err := svc.provider.Storage.CompleteAuthRequest(authRequestID, buildOIDCSubject(personID), authTime, []string{"sso"}, ""); err != nil {
        return "", err
    }

    return svc.provider.BuildAuthCallbackURL(ctx, authRequestID), nil
}
```

### 4. 后端修改：Login 设置 cookie

**文件:** `backend/apps/iam/internal/controller/ctroidc/oidc.go`

`Login` 控制器在 `CompleteLogin` 成功后设置 session cookie：

```go
func (ctr *OIDCCtr) Login(ctx *gin.Context) {
    var req dtooidc.OIDCLoginReq
    if err := ctx.ShouldBindJSON(&req); err != nil {
        gincontext.Fail(ctx, err)
        return
    }
    resp, err := ctr.oidcAuthSvc.CompleteLogin(ctx, &req)
    if err != nil {
        gincontext.Fail(ctx, err)
        return
    }

    // 设置 SSO session cookie
    if resp.SessionID != "" {
        ctx.SetCookie("iam_sso_session", resp.SessionID, 86400, "/", "", false, true)
    }

    gincontext.Success(ctx, resp)
}
```

**DTO 响应增加 SessionID：**

```go
type OIDCLoginResp struct {
    ContinueURL string `json:"continueURL"`
    SessionID   string `json:"sessionID,omitempty"`
}
```

服务层在 `CompleteLogin` 成功后创建 session：

```go
sessionID, err := svc.ssoSessionStore.CreateSession(ctx, personEntity.ID)
// 非致命错误，不影响登录
```

### 5. 前端新增：第二个 RP

**文件:** `frontend/apps/sso-test-app-2/`

复制 `sso-test-app`，仅改动：
- `package.json`: 包名 `@ark-iam/sso-test-app-2`
- `vite.config.ts`: 端口 `3002`

### 6. 数据库种子数据

**文件:** `backend/scripts/sql/iam_seed_data.sql`

`test-rp-client` 的 `redirect_uris` 增加 `http://localhost:3002/`：

```sql
'["http://localhost:3001/","http://localhost:3002/"]'
```

### 7. 配置

**文件:** `backend/apps/iam/config/config.go`

`OIDC` 结构体增加：

```go
type OIDC struct {
    // ... 现有字段
    SessionTTL int `yaml:"sessionTTL"`  // SSO session TTL，秒，默认86400
}
```

### 8. 文档

**文件:** `OIDC_SELF_TEST.md`

- 更新流程图，新增 `sso-login` 节点
- 新增 Step 6a "SSO 验证" — 双 RP 跨应用 SSO 测试
- 现有 Step 6 改为 Step 6b
- 更新环境速查表新增 RP2

## 边界情况

| 场景 | 行为 |
|------|------|
| cookie 不存在 | 302 到前端登录页，走原有流程 |
| cookie 过期/无效 | 302 到前端登录页，走原有流程 |
| authRequestID 不存在 | 302 到前端登录页（无参数） |
| auth request 已过期 | CompleteAuthRequest 返回错误 → 302 到前端登录页 |
| 多标签页同浏览器 | 共享 cookie，均自动 SSO |
| session 撤销 | 删除 Redis key → 下次请求强制登录 |
