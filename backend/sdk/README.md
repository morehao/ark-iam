# ark-iam/sdk —— RP 侧 OIDC 能力 SDK

面向**外部应用（RP）**的 OIDC 接入 SDK，也是本仓内置应用（`auth`/`platformadmin`/`tenantadmin`/`rpapi`）
使用的同一份实现（dogfooding：内置应用能用的能力，外部应用同样可用）。

- **框架无关**：只依赖 Go 标准库与 `github.com/golang-jwt/jwt/v5`（测试另加 `testify`）。
  没有 gin/gorm/redis/go-oidc/go-jose 依赖，由 `scripts/check-deps.sh` 强制校验（`make sdk-check-deps`）。
  框架绑定（gin 中间件等）由消费方自己写 20 行——见下文「Gin 接入」。
- **不共享 IAM 的数据库与 Redis**：验签全程本地完成（JWKS 预取 + 内存 kid 查表），
  业务请求路径上**绝不访问 OP**。
- **fail-closed**：任何无法建立密钥集/校验器的情况都必须落到 401，不允许静默放行。

```
sdk/
├── contract/     # claim 契约（双端单一事实源）、错误分类、back-channel logout 事件常量
└── rp/           # RP 侧能力
    ├── keys.go       # KeySource：JWKS 预取 / kid 查表 / 限频刷新 / 磁盘兜底
    ├── verifier.go   # 本地验签 → Identity（person/machine、租户、scope、代操作主体）
    ├── client_credentials.go  # M2M 令牌自动续期（距 exp 60s 换新）
    ├── introspect/   # RFC 7662 introspection 客户端
    ├── revoke/       # RFC 7009 撤销客户端
    ├── userinfo/     # userinfo 客户端（展示用途，勿放鉴权/审计关键路径）
    ├── logout/       # back-channel logout 接收端（jti 去重 + 会话撤销）
    └── directory/    # 只读目录客户端（TTL 缓存 / ETag 304 / 批量 / single-flight）
```

## 安装

本仓 `go.work` 已包含 `./sdk`。外部应用按需引入：

```
require github.com/morehao/ark-iam/sdk v0.0.0
```

## 1. 本地验签（唯一鉴权路径）

```go
import (
    "github.com/morehao/ark-iam/sdk/rp"
    "github.com/morehao/ark-iam/sdk/contract"
)

// 传 issuer：SDK 用标准 discovery 解析 jwks_uri（也可直接给显式端点，如
// "https://iam.example.com/oidc/keys" 或 ".../jwks.json"，会原样使用）。
keys, err := rp.NewKeys(ctx, "https://iam.example.com/oidc")
if err != nil { /* 启动即失败，别把服务放起来 */ }
defer keys.Close()

verifier := rp.NewVerifier(keys,
    rp.WithIssuer("https://iam.example.com/oidc"),
    rp.WithAudiences("your-client-id"), // aud 收紧：拒绝其它 client 的令牌
)
// DefaultLeeway = 30s，DefaultAlgorithms = RS256，均按需覆盖。

identity, err := verifier.Verify(ctx, rawToken)
switch {
case err == nil:
case errors.Is(err, contract.ErrExpired):          // 401
case errors.Is(err, contract.ErrBadAudience):      // 401
case errors.Is(err, contract.ErrKeySourceUnavailable): // 503（密钥集不可用，别当成 401 掩盖）
default:                                            // 401
}
```

`rp.Identity` 字段：

| 字段 | 含义 |
|---|---|
| `PersonID` | 自然人 ID（人令牌；`sub` 去掉 `person:` 前缀） |
| `UserID` | **机器主体/租户成员主键**，写审计操作者列的唯一口径 |
| `TenantID` | 租户作用域，**一律来自令牌**，不接受入参指定 |
| `ClientID` | 令牌的 `client_id` |
| `IsMachine` | 是否机器凭证（`token_usage=machine`） |
| `SessionID` | SSO 会话标识（`sid`），用于会话粒度登出 |
| `Scopes` | 标准 `scope` 声明（空格分隔），`HasScope("...")` |
| `ActorID` | 代操作主体（`act.sub`） |

**`tenant_id` 不是令牌有效性条件**：签发侧在「自然人 → 租户映射不唯一」时不写该 claim，
因此验签会照常通过、`Identity.TenantID` 为空。需要租户的接口必须自行判定并返回 **403**，
不得依赖 `dbclient.ErrTenantScopeMissing` 冒成 500。

## 2. Gin 接入（消费方自写，约 20 行）

```go
func OIDCAuth(verifier *rp.Verifier) gin.HandlerFunc {
    return func(c *gin.Context) {
        raw := strings.TrimPrefix(c.GetHeader("Authorization"), "Bearer ")
        if raw == "" { c.AbortWithStatusJSON(401, gin.H{"code": 401}); return }
        identity, err := verifier.Verify(c.Request.Context(), raw)
        if err != nil { c.AbortWithStatusJSON(401, gin.H{"code": 401}); return }
        c.Set("identity", identity)
        c.Next()
    }
}
```

本仓内置应用的参考实现见 `backend/pkg/middleware/oidc_auth.go`（含 aud/issuer 收紧、
SSO 会话活性校验、`user_id` 回填等本仓特有逻辑）。

## 3. 机器凭证（M2M）

```go
tc, err := rp.NewTokenClient(rp.ClientCredentialsConfig{
    TokenURL:     "https://iam.example.com/oidc/oauth/token",
    ClientID:     "your-client-id",
    ClientSecret: "…",
    Scopes:       []string{"directory.read"},
}, nil)

client, err := directory.New(directory.Config{
    BaseURL: "https://iam.example.com/v1/rp",
    Tokens:  tc, // 距 exp 60s 自动换新
})
```

也可用 API Key 换 token（`POST /oidc/oauth/token`，`grant_type=client_credentials`），
机器令牌的 `user_id` = API Key 归属的机器主体，审计口径与 IAM 自身一致。

> ⚠️ **目录 API 只接受 API Key 形态的机器令牌。**
> 普通 OIDC 客户端（真实 `client_id`/`client_secret`）的纯 `client_credentials` 只产出
> `client_id` claim，**不含** `token_usage=machine`，会被本 SDK 的验签按"既非自然人亦非机器"
> 以 `contract.ErrMissingClaim` 拒绝（HTTP 401）。原因是目录数据按租户隔离，而纯
> `client_credentials` 没有任何租户归属（租户上下文只能来自 API Key 的归属主体）。
> 因此外部应用取目录数据请统一走上面这段 `NewTokenClient`（内部即 API Key 换 token），
> 并在客户端上授好 `directory.read` scope 与 rpapi 的 `aud`。

## 4. 只读目录（推荐：后台展示取姓名/部门）

```go
member, err := client.Member(ctx, userID)
members, err := client.Members(ctx, []string{id1, id2}) // ≤100，批量
tree, err := client.DepartmentTree(ctx)
roles, err := client.Roles(ctx) // roles 上限 500，client.RolesTruncated() 查询是否截断
```

- 缓存：成员 60s / 部门树与角色 300s；ETag + `If-None-Match` → 304（省带宽，不省 DB 查询）。
- 降级：**仅**连接失败/超时/5xx 返回 stale（`StaleOnError`，上限 5 分钟）；
  **401/403/404 一律透传、绝不降级**——否则客户端被撤销或 scope 被收紧后会用旧缓存继续放行。
- 无缓存且目录不可用 → `directory.ErrUnavailable`。
- ⚠️ **绝不用 `Member.Status` 做放行/拒绝判定**：挂起与否由令牌侧决定，目录只服务展示。

## 5. 单点登出（back-channel logout）

外部应用需实现 `POST <backChannelLogoutPath>`（URI 在客户端注册时声明）：

```go
receiver := logout.NewReceiver(
    logout.WithKeySource(keys),
    logout.WithIssuer("https://iam.example.com/oidc"),
    logout.WithClientID("your-client-id"),
    logout.WithSessionRevoker(func(ctx context.Context, claims *contract.LogoutTokenClaims) error {
        return revokeSessions(ctx, claims.SessionID) // 先撤销，成功了再记录 jti
    }),
)
http.Handle("/bc-logout", receiver) // 正确状态码：200/400/500
```

`receiver.Handle(ctx, tokenStr)` 是唯一实现，gin 壳只做参数搬运（见 `backend/pkg/oidckit`）。

## 6. 验证与自检

```bash
make test-sdk          # cd backend/sdk && go test ./...
make sdk-check-deps    # 依赖白名单（拒绝 gorm/redis/gin/go-oidc/go-jose）
```

## 7. `/v1/auth/*` 不是 RP 契约

`/v1/auth/*` 是 OP 自身的业务 API（服务登录前端），**不是**给 RP 的正式契约。
长期能力请用 `/v1/rp/directory/*`（见 `docs/design/api-reference.md`）。
