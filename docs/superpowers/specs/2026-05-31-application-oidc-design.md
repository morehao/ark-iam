# Application OIDC Provider 设计文档

- **日期**: 2026-05-31
- **状态**: 草案
- **作者**: AI 辅助设计

## 1. 背景与目标

### 1.1 当前问题

当前 `application` 表按照 OAuth2/OIDC 客户端注册模型设计（包含 `oidc_client_metadata`、`secret`、`type`、`is_third_party` 等字段），但认证流程完全未使用 Application：

- 认证流程是 `person → selectTenant → 签发 JWT`，令牌不含 `application_id`
- `oidc_client_metadata` JSON 字段存在但未被任何代码使用
- OIDC 功能在独立的 connector 子系统中，connector 与 application 无关联
- `secret` 明文存储在 DB 并在 API 响应中返回

### 1.2 目标

将 ark-iam 改造为完整的 OIDC 授权服务器（Identity Provider），使第三方应用可以通过标准 OIDC 协议对接 ark-iam 进行认证。

### 1.3 非目标

- 不涉及前端 UI 变化（仅后端 API）
- 不涉及组织/部门/角色/菜单模块的改动
- 不涉及 Connector 子系统改动（身份源集成保持不动）

## 2. 整体架构

```
                      ┌─────────────────────────────────────┐
                      │         第三方应用 / 客户端            │
                      │  (SPA / 后端服务 / 移动端 / 微服务)   │
                      └────────────▲─────────────────▲──────┘
                                   │  OIDC 协议        │
                      ┌────────────┴─────────────────┴──────┐
                      │        ark-iam（OIDC 授权服务器）     │
                      │                                      │
                      │  ┌──────────────────────────────┐   │
                      │  │  OIDC Provider 端点            │   │
                      │  │  /authorize . /token           │   │
                      │  │  /userinfo  . /jwks           │   │
                      │  │  /.well-known/...             │   │
                      │  └──────────────────────────────┘   │
                      │                                      │
                      │  ┌──────────────────────────────┐   │
                      │  │  Application（OIDC 客户端注册） │   │
                      │  │  client_id / redirect_uris    │   │
                      │  │  grant_types / scopes / 密钥   │   │
                      │  └──────────────────────────────┘   │
                      │                                      │
                      │  ┌──────────────────────────────┐   │
                      │  │  Connector（身份源代理）       │   │
                      │  │  GitHub / Google / WeChat     │   │
                      │  │  LDAP / OIDC                  │   │
                      │  └──────────────────────────────┘   │
                      └─────────────────────────────────────┘
```

### 2.1 核心概念分层

| 层次 | 组件 | 职责 |
|------|------|------|
| 认证输出 | OIDC Provider 端点 | 第三方应用通过标准 OIDC 协议对接 |
| 客户端管理 | Application | OIDC 客户端注册（client_id、密钥、回调地址、权限范围） |
| 认证引擎 | Person + User + Tenant | 用户身份验证、租户上下文解析 |
| 身份源 | Connector | 对接外部 IdP 实现社交登录/企业 SSO |

## 3. Application 模型设计

### 3.1 表结构

```sql
CREATE TABLE `application` (
    `id`                         BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    `tenant_id`                  BIGINT UNSIGNED NOT NULL DEFAULT 0,
    `client_id`                  VARCHAR(64) NOT NULL DEFAULT '',
    `name`                       VARCHAR(256) NOT NULL DEFAULT '',
    `description`                TEXT,
    `logo_url`                   VARCHAR(2048) NOT NULL DEFAULT '',
    `homepage_url`               VARCHAR(2048) NOT NULL DEFAULT '',

    -- OIDC 结构化配置
    `redirect_uris`              JSON NOT NULL DEFAULT ('[]'),
    `post_logout_redirect_uris`  JSON NOT NULL DEFAULT ('[]'),
    `grant_types`                JSON NOT NULL DEFAULT ('["authorization_code"]'),
    `response_types`             JSON NOT NULL DEFAULT ('["code"]'),
    `token_endpoint_auth_method` VARCHAR(32) NOT NULL DEFAULT 'client_secret_basic',
    `allowed_origins`            JSON NOT NULL DEFAULT ('[]'),
    `require_pkce`               TINYINT(1) NOT NULL DEFAULT 0,
    `require_auth_time`          TINYINT(1) NOT NULL DEFAULT 0,
    `default_scopes`             JSON NOT NULL DEFAULT ('["openid","profile"]'),
    `access_token_ttl`           BIGINT NOT NULL DEFAULT 3600,
    `refresh_token_ttl`          BIGINT NOT NULL DEFAULT 2592000,

    -- 业务属性
    `type`                       VARCHAR(32) NOT NULL DEFAULT 'first_party',
    `status`                     VARCHAR(32) NOT NULL DEFAULT 'enable',
    `is_third_party`             TINYINT(1) NOT NULL DEFAULT 0,

    `created_at`                 DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `updated_at`                 DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    `deleted_at`                 DATETIME DEFAULT NULL,
    `created_by`                 BIGINT UNSIGNED NOT NULL DEFAULT 0,
    `updated_by`                 BIGINT UNSIGNED NOT NULL DEFAULT 0,
    `deleted_by`                 BIGINT UNSIGNED NOT NULL DEFAULT 0,

    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_client_id` (`client_id`),
    KEY `idx_tenant_id` (`tenant_id`),
    KEY `idx_tenant_type` (`tenant_id`, `type`),
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
```

### 3.2 密钥表

```sql
CREATE TABLE `application_secret` (
    `id`              BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    `application_id`  BIGINT UNSIGNED NOT NULL DEFAULT 0,
    `name`            VARCHAR(256) NOT NULL DEFAULT '',
    `value_hash`      VARCHAR(256) NOT NULL DEFAULT '',
    `value_prefix`    VARCHAR(16) NOT NULL DEFAULT '',
    `expired_at`      DATETIME DEFAULT NULL,
    `revoked_at`      DATETIME DEFAULT NULL,
    `created_at`      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `updated_at`      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    `deleted_at`      DATETIME DEFAULT NULL,
    `created_by`      BIGINT UNSIGNED NOT NULL DEFAULT 0,
    `updated_by`      BIGINT UNSIGNED NOT NULL DEFAULT 0,
    `deleted_by`      BIGINT UNSIGNED NOT NULL DEFAULT 0,
    PRIMARY KEY (`id`),
    KEY `idx_application_id` (`application_id`),
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
```

### 3.3 Go 模型

```go
type ApplicationEntity struct {
    ID        uint   `gorm:"primarykey"`
    TenantID  uint   `gorm:"column:tenant_id"`
    ClientID  string `gorm:"column:client_id;uniqueIndex"`
    Name      string
    LogoURL   string `gorm:"column:logo_url"`
    HomepageURL string `gorm:"column:homepage_url"`
    Type      string
    Status    string
    IsThirdParty int8 `gorm:"column:is_third_party"`

    RedirectURIs            datatypes.JSON `gorm:"column:redirect_uris"`
    PostLogoutRedirectURIs  datatypes.JSON `gorm:"column:post_logout_redirect_uris"`
    GrantTypes              datatypes.JSON `gorm:"column:grant_types"`
    ResponseTypes           datatypes.JSON `gorm:"column:response_types"`
    TokenEndpointAuthMethod string         `gorm:"column:token_endpoint_auth_method"`
    AllowedOrigins          datatypes.JSON `gorm:"column:allowed_origins"`
    RequirePKCE             int8           `gorm:"column:require_pkce"`
    RequireAuthTime         int8           `gorm:"column:require_auth_time"`
    DefaultScopes           datatypes.JSON `gorm:"column:default_scopes"`
    AccessTokenTTL          int64          `gorm:"column:access_token_ttl"`
    RefreshTokenTTL         int64          `gorm:"column:refresh_token_ttl"`

    CreatedAt time.Time
    UpdatedAt time.Time
    DeletedAt gorm.DeletedAt
    CreatedBy uint
    UpdatedBy uint
    DeletedBy uint
}

type OIDCClientConfig struct {
    RedirectURIs            []string `json:"redirect_uris"`
    PostLogoutRedirectURIs  []string `json:"post_logout_redirect_uris"`
    GrantTypes              []string `json:"grant_types"`
    ResponseTypes           []string `json:"response_types"`
    TokenEndpointAuthMethod string   `json:"token_endpoint_auth_method"`
    AllowedOrigins          []string `json:"allowed_origins"`
    RequirePKCE             bool     `json:"require_pkce"`
    RequireAuthTime         bool     `json:"require_auth_time"`
    DefaultScopes           []string `json:"default_scopes"`
}
```

### 3.4 模型变更要点

| 变化 | 说明 |
|------|------|
| `client_id` 新增 | OIDC 核心标识，UUID 格式，`app_` 前缀 |
| `redirect_uris` 从 JSON 提取 | 结构化数组，白名单校验 |
| `secret` 从主表移出 | 移至 `application_secret` 表，哈希存储 |
| `grant_types` 新增 | 控制该应用支持的授权模式 |
| `token_endpoint_auth_method` 新增 | `client_secret_basic` / `client_secret_post` / `none` |
| `require_pkce` 新增 | SPA 应用强制 PKCE |
| `access_token_ttl` / `refresh_token_ttl` 新增 | 按应用配置令牌有效期 |

## 4. OIDC 认证流程

### 4.1 授权码流程（Authorization Code + PKCE）

```
1. 用户访问第三方 APP
2. APP 构造 OIDC 请求：
   GET /v1/iam/oidc/authorize?client_id=xxx&redirect_uri=...&response_type=code
                              &scope=openid+profile&state=xyz&code_challenge=...

3. ark-iam 验证请求：
   - client_id 存在且状态正常
   - redirect_uri 在白名单中
   - 用户未登录 → 返回登录页面

4. 用户认证：
   - 密码登录（本地 Person 验证）
   - 或通过 Connector SSO（GitHub/Google/LDAP）
   - Connector 验证 → 映射到 Person

5. 租户解析（关键步骤）：
   - Application.tenant_id 确定目标租户
   - 查找 Person 在目标租户下的 User 记录
   - 如不存在 → 报错或自动创建（由应用配置决定）
   - 如果 Person 在多个租户下有 User 记录 → 展示租户选择页

6. 签发 Authorization Code：
   - 存入 Redis，有效期 5 分钟
   - 绑定：{client_id, person_id, tenant_id, user_id, scope, code_challenge_method, code_challenge, nonce, redirect_uri}

7. 重定向回 APP：
   302 Location: {redirect_uri}?code=xxx&state=xyz

8. APP 后端交换 Token：
   POST /v1/iam/oidc/token
   Content-Type: application/x-www-form-urlencoded
   grant_type=authorization_code&code=xxx&client_id=xxx&client_secret=yyy&code_verifier=...

9. ark-iam 验证并签发 Token：
   - 验证 code 存在且未使用
   - 验证 client_id + client_secret
   - PKCE 验证 code_verifier
   - 验证 redirect_uri 匹配
   - 签发 ID Token + Access Token + Refresh Token

10. APP 验证 Token：
    - 使用 /jwks 获取公钥
    - 验证 ID Token 签名、iss、aud、exp、nonce
    - 可选：调用 /userinfo 获取用户信息
```

### 4.2 客户端凭证流程（Client Credentials）

```
1. 机器客户端请求：
   POST /v1/iam/oidc/token
   Content-Type: application/x-www-form-urlencoded
   grant_type=client_credentials&client_id=xxx&client_secret=yyy&scope=api:read

2. 验证 client_id + client_secret
3. 查找 application 关联的 scope/role 权限
4. 签发 Access Token（无 sub 声明，无用户关联）
5. 返回：{access_token, token_type, expires_in, scope}
```

### 4.3 Token 声明

**ID Token（JWT）：**
```json
{
  "iss": "https://iam.example.com",
  "sub": "person_789",
  "aud": ["client_id_xxx"],
  "exp": 1717000000,
  "iat": 1716996400,
  "auth_time": 1716996400,
  "nonce": "abc123",
  "tenant_id": "456",
  "user_id": 123,
  "name": "张三",
  "preferred_username": "zhangsan",
  "email": "zhang@example.com",
  "email_verified": true
}
```

**Access Token（JWT）：**
```json
{
  "iss": "https://iam.example.com",
  "sub": "person_789",
  "aud": ["client_id_xxx"],
  "client_id": "client_id_xxx",
  "scope": "openid profile email",
  "tenant_id": 456,
  "user_id": 123,
  "type": "Bearer",
  "exp": 1717000000,
  "iat": 1716996400,
  "jti": "unique-token-id"
}
```

## 5. OIDC 端点设计

### 5.1 路由表

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/.well-known/openid-configuration` | OIDC 发现文档 |
| GET | `/.well-known/jwks.json` | 公钥 |
| GET | `/v1/iam/oidc/authorize` | 授权码端点 |
| POST | `/v1/iam/oidc/token` | Token 端点 |
| GET | `/v1/iam/oidc/userinfo` | 用户信息端点 |
| POST | `/v1/iam/oidc/logout` | RP-initiated 登出 |
| POST | `/v1/iam/oidc/revoke` | Token 吊销 |
| POST | `/v1/iam/oidc/introspect` | Token 查询（可选） |

### 5.2 发现端点

```go
type OIDCDiscoveryResponse struct {
    Issuer                string   `json:"issuer"`
    AuthorizationEndpoint string   `json:"authorization_endpoint"`
    TokenEndpoint         string   `json:"token_endpoint"`
    UserinfoEndpoint      string   `json:"userinfo_endpoint"`
    JWKSURI               string   `json:"jwks_uri"`
    RegistrationEndpoint  string   `json:"registration_endpoint,omitempty"`
    ScopesSupported       []string `json:"scopes_supported"`
    ResponseTypesSupported []string `json:"response_types_supported"`
    GrantTypesSupported   []string `json:"grant_types_supported"`
    SubjectTypesSupported []string `json:"subject_types_supported"`
    IDTokenSigningAlgValuesSupported []string `json:"id_token_signing_alg_values_supported"`
    TokenEndpointAuthMethodsSupported []string `json:"token_endpoint_auth_methods_supported"`
    ClaimsSupported       []string `json:"claims_supported"`
    CodeChallengeMethodsSupported []string `json:"code_challenge_methods_supported"`
}
```

### 5.3 Token 端点请求/响应

**authorization_code grant:**
```
请求：
  grant_type=authorization_code
  code={authorization_code}
  redirect_uri={original_redirect_uri}
  client_id={client_id}
  client_secret={client_secret}

响应 200：
  {
    "access_token": "eyJhbGci...",
    "token_type": "Bearer",
    "expires_in": 3600,
    "refresh_token": "def456...",
    "id_token": "eyJraWQ..."
  }
```

**client_credentials grant:**
```
请求：
  grant_type=client_credentials
  client_id={client_id}
  client_secret={client_secret}
  scope={scope}

响应 200：
  {
    "access_token": "eyJhbGci...",
    "token_type": "Bearer",
    "expires_in": 3600,
    "scope": "api:read"
  }
```

**refresh_token grant:**
```
请求：
  grant_type=refresh_token
  refresh_token={refresh_token}
  client_id={client_id}
  client_secret={client_secret}

响应 200：
  {
    "access_token": "eyJhbGci...",
    "token_type": "Bearer",
    "expires_in": 3600,
    "refresh_token": "new_refresh_token",
    "id_token": "eyJraWQ..."
  }
```

## 6. Session 与状态管理

| 组件 | 生命周期 | 存储 | 说明 |
|------|---------|------|------|
| Authorization Code | 5 分钟，一次性 | Redis | 临时凭证，绑定 code_challenge、pkce 等信息 |
| Session Cookie | 24h（可配置） | Redis | 保持用户在 /authorize 页面的登录态，跨 app 共享 |
| Access Token | 1h（按应用配置） | 无状态 JWT | 资源访问凭证，不存 DB |
| Refresh Token | 30天（按应用配置） | DB (refresh_token 表) | 轮换模式，每次刷新旧 token 自动失效 |

## 7. 认证流程整合

### 7.1 当前流程（将被替代）

```
POST /v1/iam/auth/login     → PersonToken + tenant list
POST /v1/iam/auth/selectTenant → AccessToken + RefreshToken
```

### 7.2 新流程

```
GET  /v1/iam/oidc/authorize → 完整 OIDC 授权码流程
       ├── 未登录 → 展示登录页（密码 or Connector SSO）
       ├── 已登录 → 自动认证
       ├── 多租户 → 展示租户选择
       └── 签发 authorization code → 302 重定向
POST /v1/iam/oidc/token     → 交换 token（支持 3 种 grant_type）
GET  /v1/iam/oidc/userinfo  → 获取用户信息（Bearer Token）
```

## 8. 新增代码结构

```
internal/
├── service/
│   └── svcoidc/                # OIDC 核心服务（新增）
│       ├── oidc.go             # 主入口，流程编排
│       ├── authorize.go        # 授权码端点逻辑
│       ├── token.go            # Token 签发（ID/Access/Refresh）
│       ├── userinfo.go         # 用户信息端点
│       ├── discovery.go        # well-known + jwks
│       ├── session.go          # OIDC Session 管理
│       ├── client_auth.go      # 客户端认证
│       ├── scope.go            # Scope 校验与映射
│       └── validator.go        # 请求验证
├── controller/
│   └── ctroidc/                # OIDC 控制器（新增）
│       └── oidc.go
└── dto/
    └── dtooidc/                # OIDC DTO（新增）
        ├── request.go
        └── response.go
```

### 8.1 不受影响的模块

- `internal/controller/ctrpermission/application.go` — 应用 CRUD 需适配新字段
- `internal/service/svcpermission/application.go` — 同上
- `internal/controller/ctrauth/connector*.go` — Connector 保持不动
- `internal/service/svcauth/connector*.go` — 同上
- `pkg/code/` — 需新增 OIDC 相关错误码

## 9. 安全设计

| 安全措施 | 说明 |
|---------|------|
| Secret 哈希存储 | 创建时返回明文，DB 存 SHA-256，支持轮换 |
| PKCE 支持 | SPA/移动端强制 PKCE（S256） |
| CSRF 防护 | state 参数校验 |
| Nonce 防护 | ID Token 中包含 nonce，防止重放 |
| Redirect URI 白名单 | 严格匹配，支持通配符 |
| Token 轮换 | Refresh Token 每次刷新自动失效旧 token |
| Token 吊销 | 支持通过 revoke 端点主动吊销 |
| 请求速率限制 | /token 端点按 client_id 限流 |

## 10. 错误码设计

```go
const (
    OIDCInvalidRequest           = 100601
    OIDCUnauthorizedClient       = 100602
    OIDCAccessDenied             = 100603
    OIDCUnsupportedResponseType  = 100604
    OIDCInvalidScope             = 100605
    OIDCInvalidGrant             = 100606
    OIDCInvalidClient            = 100607
    OIDCServerError              = 100608
    OIDCTemporarilyUnavailable   = 100609
)
```

## 11. 实施步骤

### 第 1 步：Application 模型重构

- 修改 `model/application.go` 结构体
- 更新 SQL schema
- 更新 DAO 层查询
- 更新 Service 层 CRUD 逻辑
- `client_id` 自动生成逻辑
- `application_secret` 改为哈希存储
- 数据迁移脚本

### 第 2 步：OIDC 基础设施

- 实现 `svcoidc/discovery.go`（/.well-known 端点）
- 实现 JWT 签名与 JWKS 管理（复用已有密钥）
- 实现 `svcoidc/session.go`（authorization code + session 管理）
- 实现 `svcoidc/client_auth.go`（客户端认证）

### 第 3 步：Token 端点

- 实现 `POST /v1/iam/oidc/token`
- 支持 `authorization_code` grant
- 支持 `client_credentials` grant
- 支持 `refresh_token` grant
- ID Token + Access Token + Refresh Token 签发

### 第 4 步：Authorize 端点

- 实现 `GET /v1/iam/oidc/authorize`
- 登录页面集成（密码登录 + Connector SSO）
- 租户选择逻辑
- Authorization Code 签发

### 第 5 步：Userinfo + 登出 + 吊销

- 实现 `GET /v1/iam/oidc/userinfo`
- 实现 `POST /v1/iam/oidc/logout`
- 实现 `POST /v1/iam/oidc/revoke`
- 实现 `POST /v1/iam/oidc/introspect`（可选）
