# Application & OAuthClient 代码重构设计文档

- **日期**: 2026-06-01
- **状态**: 草案
- **关联文档**: `2026-06-01-application-model-separation-design.md`（模型拆分设计）

## 1. 背景与问题

前序设计 `2026-06-01-application-model-separation-design.md` 已完成数据库模型拆分（`application` / `tenant_application` / `oauth_client`），但后端代码存在以下问题：

1. **路由命名错配**：`/v1/iam/application/*` 实际操作 `oauth_client` 表，而 `/v1/iam/appDefinition/*` 才操作 `application` 表
2. **包名冗余**：`svcappdefinition` 中的 "definition" 是多余概念，应直接为 `svcapplication`
3. **DTO 字段错误**：OAuthClient CreateReq/UpdateReq 包含 `description`/`logoURL`/`homepageURL` 等属于 `application` 表的无效字段
4. **OAuthClient 缺少 app_id**：创建 OAuthClient 时未关联到 application
5. **Detail 响应过简**：OAuthClient Detail 只返回基础信息，缺少 OIDC 协议关键字段
6. **错误码混用**：OAuthClient 和 Application 共用同一段错误码（10073X）
7. **缺少 tenant_application CRUD**：Model/DAO 已存在，但无 Service/Controller/Router
8. **缺失枚举常量**：字典值未定义为常量

## 2. 路由设计

### 2.1 路由对照表

| 当前路由 | 新路由 | 包名 | 操作表 |
|---------|--------|------|--------|
| `/v1/iam/appDefinition/*` | `/v1/iam/application/*` | `svcapplication` | `application` |
| `/v1/iam/application/*` | `/v1/iam/oauthClient/*` | `svcoauthclient` | `oauth_client` |
| 无 | `/v1/iam/tenantApplication/*` | `svctenantapplication` | `tenant_application` |

### 2.2 包名对照

| 当前包名 | 新包名 |
|---------|--------|
| `svcappdefinition` | `svcapplication` |
| `ctrappdefinition` | `ctrapplication` |
| `dtoappdefinition` | `dtoapplication` |
| `svcoauthclient` | 不变 |
| `ctroauthclient` | 不变 |
| `dtooauthclient` | 不变 |

## 3. DTO 修正

### 3.1 Application DTO（原 appDefinition）

`CreateReq`、`UpdateReq`、`DetailResp`、`PageListItem` 保持原有字段不变（code、name、description、logoUrl、homepageUrl、type、status、sort）。

### 3.2 OAuthClient CreateReq

移除 `description`、`logoURL`、`homepageURL` 三个无效字段。新增 `appId` 字段关联到 application：

```go
type CreateReq struct {
    AppId       uint     `json:"appId" binding:"required"`     // 新增
    Name        string   `json:"name" binding:"required"`
    Type        string   `json:"type"`
    IsThirdParty int8    `json:"isThirdParty"`
    RedirectURIs            []string `json:"redirectURIs"`
    PostLogoutRedirectURIs  []string `json:"postLogoutRedirectURIs"`
    GrantTypes              []string `json:"grantTypes"`
    ResponseTypes           []string `json:"responseTypes"`
    TokenEndpointAuthMethod string   `json:"tokenEndpointAuthMethod"`
    AllowedOrigins          []string `json:"allowedOrigins"`
    RequirePKCE             int8     `json:"requirePKCE"`
    RequireAuthTime         int8     `json:"requireAuthTime"`
    DefaultScopes           []string `json:"defaultScopes"`
    AccessTokenTTL          int64    `json:"accessTokenTTL"`
    RefreshTokenTTL         int64    `json:"refreshTokenTTL"`
}
```

### 3.3 OAuthClient UpdateReq

同步移除 `description`、`logoURL`、`homepageURL`。

### 3.4 OAuthClient DetailResp

展开返回所有 oauth_client 表字段：

```go
type DetailResp struct {
    OAuthClientID               uint     `json:"oauthClientId"`
    TenantID                    uint     `json:"tenantId"`
    AppID                       uint     `json:"appId"`
    ClientID                    string   `json:"clientID"`
    Name                        string   `json:"name"`
    RedirectURIs                []string `json:"redirectURIs"`
    PostLogoutRedirectURIs      []string `json:"postLogoutRedirectURIs"`
    GrantTypes                  []string `json:"grantTypes"`
    ResponseTypes               []string `json:"responseTypes"`
    TokenEndpointAuthMethod     string   `json:"tokenEndpointAuthMethod"`
    AllowedOrigins              []string `json:"allowedOrigins"`
    RequirePKCE                 int8     `json:"requirePKCE"`
    RequireAuthTime             int8     `json:"requireAuthTime"`
    DefaultScopes               []string `json:"defaultScopes"`
    AccessTokenTTL              int64    `json:"accessTokenTTL"`
    RefreshTokenTTL             int64    `json:"refreshTokenTTL"`
    Type                        string   `json:"type"`
    IsThirdParty                int8     `json:"isThirdParty"`
    Status                      string   `json:"status"`
    CreatedAt                   string   `json:"createdAt"`
}
```

## 4. 错误码

为 OAuthClient 引入独立错误码段 100750-100759：

```go
const (
    OAuthClientCreateError         = 100750
    OAuthClientDeleteError         = 100751
    OAuthClientUpdateError         = 100752
    OAuthClientGetDetailError      = 100753
    OAuthClientGetPageListError    = 100754
    OAuthClientNotExistError       = 100755
    OAuthClientSecretCreateError   = 100756
    OAuthClientSecretGetListError  = 100757
    OAuthClientSecretDeleteError   = 100758
    OAuthClientSecretNotExistError = 100759
)
```

`svcapplication` 继续使用原 `ApplicationXxxError` 错误码（100730-100739），`svcoauthclient` 改用新 `OAuthClientXxxError`。

## 5. 枚举常量

### 5.1 model/application.go

```go
const (
    AppTypeFirstParty  = "first_party"
    AppTypeThirdParty  = "third_party"

    AppStatusEnable  = "enable"
    AppStatusDisable = "disable"
)
```

### 5.2 model/oauth_client.go

```go
const (
    OAuthClientTypeFirstParty  = "first_party"
    OAuthClientTypeThirdParty  = "third_party"

    OAuthClientStatusEnable  = "enable"
    OAuthClientStatusDisable = "disable"

    GrantTypeAuthorizationCode = "authorization_code"
    GrantTypeClientCredentials = "client_credentials"
    GrantTypeRefreshToken      = "refresh_token"

    TokenEndpointAuthMethodBasic = "client_secret_basic"
    TokenEndpointAuthMethodPost  = "client_secret_post"
    TokenEndpointAuthMethodNone  = "none"
)
```

## 6. 影响范围

- `backend/apps/iam/internal/service/svcappdefinition/` → 目录/包重命名为 `svcapplication`
- `backend/apps/iam/internal/controller/ctrappdefinition/` → 目录/包重命名为 `ctrapplication`
- `backend/apps/iam/internal/dto/dtoappdefinition/` → 目录/包重命名为 `dtoapplication`
- `backend/apps/iam/internal/router/app_definition.go` → 注册路径改为 `/application/*`
- `backend/apps/iam/internal/router/permission.go` → `applicationRouter` 注册路径改为 `/oauthClient/*`
- `backend/apps/iam/internal/router/router.go` → 函数名 `appDefinitionRouter` → `applicationRouter`
- `backend/pkg/code/permission.go` → 新增 OAuthClient 错误码段
- `backend/apps/iam/internal/dto/dtooauthclient/` → DTO 字段修正
- `backend/apps/iam/internal/service/svcoauthclient/` → 逻辑修正（appId、错误码、Detail 展开）
- TenantApplication 模块的新建

## 7. 实施顺序

1. 包重命名（svcappdefinition → svcapplication 等）+ 路由修改
2. 错误码拆分
3. OAuthClient DTO & Service 逻辑修正（移除无效字段、添加 appId、展开 Detail）
4. TenantApplication 模块新增
5. 枚举常量添加
6. 编译验证 + 测试
