# IAM 系统设计文档（落地实现版）

> 本文档描述**当前已落地实现**的统一认证服务（IAM），以 `backend/apps/iam` 的实际代码与 `backend/scripts/sql/iam_schema.sql` 的数据库结构为准。设计参照主流 IdP（Auth0、Okta、Keycloak、微软 Entra）的实践，并基于 OIDC 标准协议（zitadel/oidc，不造轮子）。

---

## 0. 需求概述

一个统一的认证服务：当有新的应用时，可在这个平台配置后快速接入用户体系。

核心能力：
- 多个应用之间共享登录态：**一个应用登录，其他应用无需再登录**；**一个应用退出，其他应用也需重新登录**。
- 支持多租户：一个自然人 `person` 在不同租户下有不同的 `user`。登录后先选租户：只有一个租户时前端自动选择，多于一个时才需手动选择。
- 基于 OIDC 标准协议，使用成熟可用的库，不自造轮子。

---

## 1. 总体架构与核心概念

### 1.1 四层模型

关键分层避免"多租户 + 共享登录态"纠缠：

| 层 | 名称 | 定位 |
|---|---|---|
| 🔷 | **Person（自然人）** | 全局唯一身份，**认证单位**。`username`/`email`/`phone` + 密码，全局唯一 |
| 🟨 | **Tenant（租户）** | 业务边界，**隔离单位** |
| 🟩 | **User（租户内用户）** | `tenant_id + person_id` 唯一，**授权单位**，携带角色/权限 |
| 🟦 | **Application（应用 / OIDC Client）** | 接入单位，全局配置、跨租户复用 |

```
Person(认证·全局) ──1:N──> User(授权·租户内) <──N:1── Tenant(隔离)
        │
        └── (经 tenant_application 可见性) 接入 Application(全局复用)
```

**核心原则：认证(person) 全局，授权(user) 租户内，接入(application) 全局复用并按租户可见。**

### 1.2 登录态与 token 分工

实现侧 OIDC Provider 由一个 `OIDCStorage`（见 `svcoidc/storage.go`）驱动，它组合两块能力：
- **Redis 协议态存储**（`protocolStore`）：保存授权请求(AuthRequest)、授权码(code)。
- **持久化存储**（`persistentStore`）：client 校验、userinfo、refresh token 落库、API Key 解析。

```
                    ┌────────────────────────────────┐
     登录 ─────────►  Person 级中心会话 (Redis)         │
                    key: iam:oidc:sso_session:{sid}    │
                    data: {personID, createdAt}        │ ← 只存"你是谁"，不存租户选择
                    TTL: 24h 滑动续期, HttpOnly+Secure  │ ← cookie 名 iam_sso_session
                    └────────────────────────────────┘
                               │ 取 person_id
                               ▼
   ┌────────────────────────────────────────────────┐
   │ ID Token  (JWT, 面向 RP)                        │
   │ sub=person:{person_id}, aud=client_id,          │ —— 身份证明
   │ tenant_id, user_id, token_usage                 │
   ├────────────────────────────────────────────────┤
   │ Access Token (JWT, 面向资源服务器)               │
   │ sub=person:{person_id}, aud=目标资源(client),     │ —— 无状态 JWT + JWKS 验签
   │ tenant_id, user_id, client_id, token_usage      │ —— 一/三方通用
   ├────────────────────────────────────────────────┤
   │ Refresh Token (不透明随机串, DB 存哈希)           │ —— 单次使用轮换，可吊销
   │ 关联 person_id/tenant_id/user_id/              │
   │ application_client_id                          │
   └────────────────────────────────────────────────┘
```

- **access** 为 JWT（RS256），资源服务器用 JWKS 验签；**内网管理面 `/v1/iam/*`** 在此基础上额外校验中心会话活性（见 §2.5），实现登出后即时失效。
- **refresh** 服务端哈希存储、单次使用轮换，支撑续期与吊销。

**access token 私有 claim（单一事实源 `object/objauth/claims.go`）**

| claim | 含义 |
|---|---|
| `sub` | `person:{person_id}`。自然人全局稳定身份；机器凭证为 `client_id` |
| `tenant_id` | 本次签发选定的租户上下文 |
| `user_id` | 该租户下的 user 记录 id |
| `client_id` | 签发此 token 的 OIDC client |
| `token_usage` | `machine` 表示机器凭证（API Key / client_credentials）签发 |

### 1.3 需求如何被满足

1. **一登录 → 其他应用免登录** ← 浏览器带 person 级 `iam_sso_session` Cookie，应用 B 走 `/oidc/authorize`→`/oidc/sso-login` 时见中心会话直接放行（仅需选租户拿 token）。
2. **一退出 → 所有重新登录** ← 全局登出：清中心会话 + 吊销该 person 所有 refresh token；access token 对内网面经 SSO 活性校验即时失效，对外部 RP 靠 TTL 失效。
3. **多租户先选租户** ← 认证在后，选租户在发 token 前，token 携带租户上下文；单租户自动选，多租户手选。

---

## 2. 关键流程

### 2.1 token 声明边界（主流分工）

| Token | 内容 | 用途 |
|---|---|---|
| ID Token | `iss, sub(=person:{id}), aud(client_id), exp, iat, auth_time, nonce, amr` + 私有 claim | end-user 身份证明，仅声明不作资源鉴权 |
| Access Token | `iss, sub(=person:{id}), aud(client_id/resource), client_id, exp, iat, scope, tenant_id, user_id, token_usage` | 调用资源服务器凭证，无状态验签 |
| Refresh Token | 不透明随机串 → DB 存哈希，记 `person_id/tenant_id/user_id/application_client_id` | 续期，单次使用轮换，可吊销 |
| 中心会话 Cookie | 仅 `personID + createdAt` | SSO 物理载体，不存租户选择 |

> **sub 语义**：`sub` = `person:{person_id}`（自然人身份，跨租户、跨应用不变），符合 OIDC 规范"sub 在 issuer+audience 内稳定"。租户上下文不放进 sub，而是作为私有 claim（`tenant_id`/`user_id`/`client_id`/`token_usage`）注入 token。租户之间仍相互隔离：`sub` 只回答"你是谁"，`tenant_id` 回答"你在哪个租户"。

### 2.2 登录 + 选租户（Authorization Code Flow）

```
用户 → RP → IAM /oidc/authorize(client_id, redirect_uri, scope, state, nonce)
        │
        ▼ 检查中心会话
    ┌───┴────────────┐
    │ iam_sso_session │
    └───┬────────────┘
  有(person已认证)      无(未登录)
     │                    │ 302→前端登录页(login-web), 输入账号密码
     ▼                    ▼ POST /oidc/login → 认证成功→建中心会话
      └───────┬────────────┘
              ▼
  查询该 person 可用租户列表 (user 记录 × 该应用 tenant_application 可见性)
      │
      ├── 0 个  → 读取【该应用】tenantPolicy.allowPersonCreateTenant:
      │           ├ true  → 跳转"创建租户"页(tenant/createAsOwner), 成为 owner → 回流程
      │           └ false → 提示无权访问(401/403)
      ├── 1 个  → 前端自动使用该租户, 跳过选择
      └── 多个  → 停留租户选择页(/oidc/login 返回 requiresTenantSelection)
              │   用户 POST /oidc/login/selectTenant 选定
              ▼
  生成 code → 302 redirect_uri?code&state → RP 用 code 换 token
  (token 携带 sub=person:{id}, tenant_id, user_id, client_id, token_usage)
```

**租户选择发生在"认证后、发 code 前"**；中心会话只记住"你是谁"，租户选择决定"这次签发哪个租户的 token"。

落地端点（见 `router/oidc.go`、`svcoidc/routes.go`、`svcauth/auth.go`）：
- `POST /oidc/login`：密码认证。单租户自动选租户并 done；多租户（无合法 tenant hint）返回 `requiresTenantSelection`，暂不发 code。成功建 SSO 会话并返回 `continueURL`。
- `POST /oidc/login/selectTenant`：完成 auth request 并选定租户，返回 `continueURL`。
- `GET /oidc/sso-login`：有中心会话则自动认证（`CompleteLoginBySession`），无则重定向前端登录页。
- 登出内容见 §2.5。

### 2.3 应用级租户策略（`tenantPolicy`）

"0 租户是否允许自助建租户"是**应用自身的产品决策**，因此作为应用（product）级配置（`model/application.go`）：

```json
{
  "tenantPolicy": {
    "allowPersonCreateTenant": true,
    "allowJoinByInvite": true
  }
}
```

- `allowPersonCreateTenant`：0 可用租户时是否允许该自然人在此应用语境下自助建租户（`POST /v1/iam/tenant/createAsOwner`）。
- `allowJoinByInvite`：是否允许用户接受邀请加入已有租户（预留，可进 V1.1）。

### 2.4 切换租户

- 中心会话**不变**。
- 实现方式：新的 `/oidc/authorize` 请求携带 `tenant` 参数（`tenantHintMiddleware` 注入 context，见 `svcoidc/routes.go`），经 `/oidc/sso-login`（`CompleteLoginBySession`）或 `/oidc/login` 按该 hint 选定租户，重新签发 token（`sub=person:{id}` 不变，`tenant_id`/`user_id` 随新租户）。
- **hint 失败关闭校验**：hint 指定的租户必须是该 person 的成员租户，否则回退，杜绝跨租户逃逸。
- **不追溯旧租户 token**：其它应用已持有的旧租户 access token 在其 TTL 内仍可用；只有它们下次刷新/登录时才切到新租户上下文。

| 认证/发码 | 全局会话 | token |
|---|---|---|
| Person（全局认证）| per person | 租户上下文独立 |

### 2.5 全局登出（需求："一退出 → 所有重新登录"）

```
用户 → 任一 RP → IAM /oidc/end_session (或 /v1/iam/auth/logout)
  │
  ▼
① 删除中心会话 (iam_sso_session cookie + Redis 记录 RevokeSessionsByPersonID)
② 吊销该 person 在所有租户下的 refresh token (DB 置 revoked_at)
③ 302 → post_logout_redirect_uri / /oidc/logged-out
```

**对内网管理面（`/v1/iam/*`）的即时失效**：`app.go` 为 `/v1` 管理面挂载 `OIDCCompatibleAuth` 中间件（`middleware/oidcauth/oidcauth.go`），验证 access token 后调用 `WithOIDCSSOValidation` 注入的校验器，检查该 person 是否仍有活跃 SSO 会话（`svcsso.HasActiveSession`）。**全局登出后，内网管理 API 立即 401**；机器凭证（`token_usage=machine`）豁免该检查。

**对外部 RP**：access token 无状态，靠自身 TTL（默认 1h）失效。

> **与原设计的差异**：早期设计稿认为"access token 完全无状态、仅靠 TTL"。落地后，第一方管理面通过 SSO 会话活性实现了即时失效，这与"一个退出、相关应用立即失效"的需求目标更贴合。

---

## 3. 数据模型

> 完整结构以 `backend/scripts/sql/iam_schema.sql` 为准（30 张表）。以下按领域列出要点。

### 3.1 实体（按领域）

**身份域（全局）**
- `person`(id, username UNIQUE, primary_email, primary_phone, password_encrypted, password_method, name, avatar, profile JSON, custom_data JSON, is_suspended, last_sign_in_at)。认证单位，登录标识列分别唯一 + 索引。
- `user_identity`(person_id, connector_id, provider, issuer, external_subject, detail JSON, last_used_at; UNIQUE(issuer, external_subject))。外部 IdP 绑定载体（Connector 已实现）。
- `user_login_log`(tenant_id, user_id, login_type, login_ip, user_agent, login_time)。登录审计。

**租户域**
- `tenant`(id, code UNIQUE, name, type[platform/customer], is_suspended, db_user, tag)。业务隔离边界。
- `user`(id, tenant_id, person_id, name, avatar, profile, custom_data, is_owner, is_suspended, joined_at, last_sign_in_at; UNIQUE(tenant_id, person_id))。授权单位，`sub` 派生源。
- `department`(tenant_id, parent_id, name, code, sort, leader_user_id) + `user_department`(tenant_id, user_id, department_id, is_primary)。部门树与用户归属。
- `organization`(tenant_id, name, description, custom_data, is_mfa_required)、`organization_role`、`organization_user`、`organization_role_user`。租户内组织架构与组织角色。

**应用域**
- `application`(id, code UNIQUE, name, type[first_party/third_party], visibility[public/private], tenant_policy JSON, status, is_system)。**产品**，全局配置，承载 `tenantPolicy`。
- `application_client`(id, app_id, client_id UNIQUE, tenant_id, name, redirect_uris, post_logout_redirect_uris, grant_types, response_types, token_endpoint_auth_method, allowed_origins, require_pkce, require_auth_time, default_scopes, access_token_ttl, refresh_token_ttl, type, is_third_party, status, is_system)。**OIDC 接入端**，一个 product 1:N 多个接入端。原设计稿中的 `oauth_metadata` 已拆为独立列。
- `application_client_secret`(application_client_id, name, value_hash, value_prefix, expired_at, revoked_at)。**密钥轮换载体**，一个接入端可持多个密钥（平滑轮换）。
- `tenant_application`(tenant_id, app_id, granted_scope JSON, config JSON, status; UNIQUE(tenant_id, app_id))。租户 ↔ 应用可用性显式关联。

**权限域（RBAC + 菜单）**
- `role`(tenant_id, app_id, name, code, type, is_default)、`scope`(tenant_id, resource_id, name, description)、`resource`(tenant_id, name, indicator, is_default, access_token_ttl)。
- `user_role`(user↔role)、`role_scope`(role↔scope)、`menu`(app_id, parent_id, name, code, path, component, icon, permission, type, sort, hidden)、`role_menu`(role↔menu)。支持用户-角色-权限-资源与角色-菜单的双维 RBAC。

**会话/令牌域**
- `session`(person_id, session_id UNIQUE, tenant_id, client_ip, user_agent, login_time, last_active_at, revoked_at, status)。会话审计（主存 Redis，落库审计）。
- `refresh_token`(person_id, tenant_id, user_id, application_client_id, session_id, token_hash UNIQUE, client_type, client_ip, user_agent, expired_at, revoked_at, last_rotated_at)。不透明随机串，单次轮换，按 person 批量吊销。
- `api_key`(tenant_id, name, key_hash, key_prefix, scope JSON, expired_at, last_used_at, revoked_at)。机器凭证，SHA-256 存哈希。

**连接器域（已实现）**
- `connector`(tenant_id, name, display_name, protocol[OIDC/LDAP/SAML/social], provider, status, allow_auto_create_user, allow_account_link, sync_profile, enable_token_storage, config JSON, claim_mapping JSON, domain_policy JSON)。

**基础设施**
- `domain`(tenant_id, domain; UNIQUE(tenant_id, domain), is_verified, verified_at)。租户自有域名。
- `system`(tenant 级 KV 配置)。
- `audit_log`(actor_person_id, actor_user_id, tenant_id, client_id, action, target_type, target_id, result, ip, user_agent, detail)。结构化审计。
- `log`(通用日志)。

### 3.2 实体关系

```
person ──1:N──> user <──N:1── tenant
  │ N:1
  ├─1:N──> user_identity (外部IdP/Connector)
  │
tenant ──1:N──> tenant_application <──N:1── application
  │                                             │ 1:N
  │                                             application_client (OIDC接入端)
  │                                             │ 1:N
  │                                             application_client_secret (密钥轮换)
person ──1:N──> refresh_token ──N:1──> application_client
user ──1:N──> user_role ──N:1──> role ──N:1──> role_scope ──N:1──> scope ──N:1──> resource
                                     └──N:1──> role_menu ──N:1──> menu
```

- `user` = `person × tenant` JOIN 产物（多态身份）。
- `tenant_application` 决定租户选择可见性过滤的依据。
- product 与接入端解耦：`application`（产品）+ `application_client`（OIDC 接入端）。

### 3.3 关键设计取舍

**Application 与 OIDC Client 的关系（两层模型）**

- **`application`（产品）**：业务产品，含租户可见性、`tenantPolicy`，管理员按产品管理。
- **`application_client`（接入端 = OIDC Client）**：一个产品可有多个接入端；`application_client_secret` 支持多密钥平滑轮换。
- **何时拆 client**：出现"传统服务端 web（用 client_secret）"且同时存在 SPA/移动端时，因 `secret` vs `none` 无法共存于一个 client，才拆为不同 client。
- **扩展性**：所有行为差异收敛进 `application_client` 的独立列 + JSON（redirect_uris/grant_types 等），不靠拆表。

**`tenant_application` 采用显式关联表**：天然支持"按租户开通应用"、与"0 租户=无可用应用"语义契合。

---

## 4. 后台 API 与 OIDC 端点

### 4.1 访问面

| 面 | 面向 | 鉴权 |
|---|---|---|
| 认证/注册/SSO | 浏览器、未登录 | 白名单 |
| OIDC 协议端点 | 任意 client/RP | OIDC 内建 |
| 管理 API | 管理员、内部服务 | 统一 OIDC access token（`OIDCCompatibleAuth`）或 x-api-key |

### 4.2 路由前缀（落地实现）

```
/oidc/*                         OIDC Provider 标准端点 + 自研登录端点 (版本无关, 稳定 issuer)
/v1/iam/*                       管理/认证 API (统一 token 鉴权)
```

- **OIDC Provider 固定在根路径 `/oidc`（不带版本号）**，保证 issuer 稳定，未来大量级版本升级不会破坏已签发 token 的 `iss` 校验。这是相对早期设计稿（曾考虑 `/v1/iam/oidc`）的关键修正——落地时采用了更符合"issuer 稳定性"的做法。
- **管理/认证 API 用 `/v1/iam/*` 带版本**，可随版本演进。

**issuer 约定**

```
issuer      = {BASE_URL}/oidc                      （无尾斜杠，来自配置 oidc.issuer）
discovery   = {issuer}/.well-known/openid-configuration
jwks_uri    = discovery 动态返回（当前单一 kid）
token.iss   = {issuer}                              （严格相等）
```

- `{BASE_URL}` 由部署配置提供（如 `http://localhost:8099`，生产 `https://iam.example.com`），不写死。
- 本地默认 issuer：`http://localhost:{port}/oidc`（见 `router/oidc.go`）。

### 4.3 OIDC 协议端点

**标准端点（由 zitadel/oidc Provider 提供，`svcoidc/routes.go`）**

| 端点 | 说明 | 认证 |
|---|---|---|
| `GET /oidc/.well-known/openid-configuration` | Discovery | 无 |
| `GET/POST /oidc/authorize` | 授权入口（含 tenant hint） | 无/SSO Cookie |
| `GET/POST /oidc/authorize/callback` | 授权码回调 | 无 |
| `POST /oidc/oauth/token` | code 换 / refresh / client_credentials | client 认证 |
| `GET/POST /oidc/userinfo` | 用户信息 | access token |
| `GET/POST /oidc/end_session` | 全局登出 | id_token_hint |
| `POST /oidc/revoke` | 吊销 token | client 认证 |
| `GET/POST /oidc/keys`、`/healthz`、`/ready` | JWKS / 探活 | 无 |

**自研登录端点（`router/oidc.go`）**

| 端点 | 说明 |
|---|---|
| `POST /oidc/login` | 密码认证；多租户返回 `requiresTenantSelection` |
| `POST /oidc/login/selectTenant` | 多租户选定后完成 auth request |
| `GET /oidc/sso-login` | SSO 自动登录检查（SilentSSORequired） |
| `GET /oidc/logged-out` | 登出成功提示页 |

**协议库（不造轮子）**
- OIDC **Provider(OP)**：`github.com/zitadel/oidc/v3`（auth code + PKCE + refresh + 自定义端点）
- 对接外部 IdP 的 **RP/校验**：`github.com/coreos/go-oidc/v3`
- access token 内网校验：`github.com/golang-jwt/jwt/v5`
- JWT/jose/JWKS：`github.com/go-jose/go-jose/v4`（RS256 签发）
- 密码哈希：`golang.org/x/crypto`（经 golib/gcrypto，bcrypt）
- Web 框架：`gin-gonic/gin`；会话/缓存：`github.com/redis/go-redis/v9`

### 4.4 统一 token 管理面鉴权

- **所有登录产生同一种 OIDC access token，不区分"平台管理员/租户管理员"身份**。一个用户有多身份，但对 IAM 都是同一种登录。
- 权限判定**不由 token 类型决定，而由"当前 user 在该 tenant 上下文下具备的能力"决定**（`is_owner` 或 RBAC `user_role→role→scope→resource`）。
- **API Key 与用户一对一**：API Key 归属某个 user（`created_by`），鉴权路径有两种——
  1. `x-api-key` 头：`middleware/apikey_auth.go` 直查 api_key 表校验（不依赖浏览器 SSO 活性）。
  2. OIDC `client_credentials`：client_id == client_secret == rawKey，签发 `token_usage=machine` 的 access token。
- **登录态即时失效**：`OIDCCompatibleAuth` 校验 SSO 会话活性（`token_usage=machine` 豁免）；配合 `TokenBlacklistCheck`（Redis 黑名单）作兜底。

> 简化模型一句话：**统一的身份系统：所有登录一种 token，权限由"用户 × 租户上下文 × RBAC"决定；API Key 只是某用户的可吊销具名凭证。**

### 4.5 管理 API 覆盖（`/v1/iam/*`）

- 身份：`auth/register`、`auth/joinTenant`、`auth/myTenants`、`auth/userinfo`、`auth/logout`(All)、`person/detail`、`person/updatePassword`
- 租户：`tenant/*`(create/createAsOwner/delete/update/detail/pageList)
- 组织/部门：`department/*`、`organization/*`、`organizationRole/*`、`organizationUser/*`、`organizationRoleUser/*`
- 应用：`application/*`、`applicationClient/*`(+secrets 列表/创建/删除)、`tenantApplication/*`
- RBAC/菜单：`role/*`、`scope/*`、`resource/*`、`menu/*`、`userRole/*`、`roleMenu/*`、`roleScope/*`
- 用户：`user/*`(含 userIdentity/userLoginLog/userDepartment/sessions)
- 连接器：`connector/*`(create/delete/update/detail/pageList/getFactoryList/test/authorize/callback)
- 其它：`apiKey/*`、`domain/*`、`system/*`、`log/*`

---

## 5. 安全设计

### 5.1 密码
- bcrypt（golib/gcrypto），不存明文；`password_method` 记录算法便于迁移。
- 登录限流 + 失败锁定（`svcloginguard`，可配置 maxFailures/windowSec/lockSec）。
- 改密/挂起/删除后吊销该 person 全部 session + refresh（强制拉登）。

### 5.2 密钥
- `client_secret` 存 SHA-256 哈希（`application_client_secret.value_hash`），明文仅创建/轮换时返回。
- 多 secret 并存过渡期 + 到期自动作废，避免改密即断。
- OIDC 签名私钥由配置提供（`signingPrivateKeyPath`/`PEM`），自动生成有兜底；JWKS 暴露公钥，RS256。

### 5.3 协议安全
- **PKCE**：SPA/移动端 `token_endpoint_auth_method=none` + `require_pkce`，S256（`CodeMethodS256: true`）。
- **redirect_uri** 精确白名单（scheme/端口/path 完全一致），防开放重定向。
- **state/nonce** 防 CSRF/重放；授权码一次性、短 TTL、绑 client、校验 code_verifier。
- **logout**：校验 id_token_hint + post_logout_redirect_uri 白名单，防 open-redirect/login CSRF。
- **tenant hint** 失败关闭校验，杜绝跨租户逃逸。

### 5.4 token 生命周期
| 令牌 | 生命周期 |
|---|---|
| Access | 短 TTL(默认 1h, 可按 client `access_token_ttl`)；RS256 无状态验签 |
| Refresh | 长 TTL(默认 30d, `refresh_token_ttl`)；单次使用轮换；可吊销；与中心会话绑定随全局登出吊销 |
| ID | 短 TTL(1h)，仅身份声明 |
| 中心会话 | 24h 滑动续期(`sessionTTL`)；HttpOnly（Secure 由部署控制） |

### 5.5 审计与可观测
- `audit_log`：登录成败、选租户、切租户、应用创建、密钥轮换、管理操作、API Key 使用（actor、tenant、client_id、ip、ua、ts、result）。
- `session`（会话审计）+ `user_login_log`（登录日志）。
- 登录失败/高频 IP 告警阈值（`svcloginguard`）；重要操作经 `svcaudit` 留痕。
- 结构化日志 + trace 贯穿（golib/glog + gtrace）。

### 5.6 多租户隔离
- 租户内查询强制携带 `tenant_id` 过滤（各 dao Cond）。
- 授权/签发严格经 `tenant_application` 过滤。
- `sub`=全局 person_id，租户隔离由 `tenant_id` claim 承担，切租户只变 `tenant_id`/`user_id`，`sub` 不变。

---

## 6. 非功能 / 部署 / 演进

### 6.1 非功能需求
- **可用性**：无状态 API 水平扩展；中心会话/限流放 Redis；DB 主从可选。
- **性能**：JWT 无状态验签（JWT/JWKS 本地缓存）+ 索引；鉴权路径免查库。
- **扩展性**：签名轮换、JWKS 多 kid；refresh 单次轮换配合原子 SQL 避免多实例竞态。
- **安全/合规**：TLS、审计保留、租户隔离、敏感字段哈希。

### 6.2 部署（MVP 现行）
```
[浏览器/RP (3000/3001/3003)] --TLS--> [IAM 单体服务(Gin, :8099)] --+--> MySQL(业务/refresh/审计)
                                                                 +--> Redis(中心会话/限流/协议态)
```
- 签名私钥由配置/密钥卷提供（`oidc-dev-key.pem`）。
- login-web(:3003) = 独立 OIDC 登录页；platform-admin-web(:3000) = 管理台（OIDC 客户端）；sso-test-app(:3001) = 测试 RP。

### 6.3 技术栈
```
Web框架        : gin-gonic/gin
OIDC Provider  : github.com/zitadel/oidc/v3
对接外部IdP(RP): github.com/coreos/go-oidc/v3
内网token校验  : github.com/golang-jwt/jwt/v5
JWT/JWKS       : github.com/go-jose/go-jose/v4 (RS256)
密码哈希       : golang.org/x/crypto (经 golib/gcrypto, bcrypt)
数据库         : MySQL + GORM
缓存/会话      : Redis (github.com/redis/go-redis/v9)
前端 OIDC      : oidc-client-ts / react-oidc-context (platform-admin-web)
```

### 6.4 落地状态与演进路径

| 能力 | 状态 |
|---|---|
| 密码登录/注册/选租户/单租户自动选/应用级 `allowPersonCreateTenant` | ✅ 已落地 |
| 标准 OIDC AuthCode + PKCE(RFC7636) + refresh 轮换 | ✅ 已落地 |
| 全局登出（清 SSO 会话 + 吊销 refresh + 内网 access 即时失效）| ✅ 已落地 |
| 多应用 SSO（central session + sso-login 自动认证）| ✅ 已落地 |
| 管理 API 统一 OIDC token 鉴权 + x-api-key 双通道 | ✅ 已落地 |
| RBAC（role/scope/resource/menu/user_role/role_menu/role_scope）| ✅ 已落地 |
| 组织/部门/域名/系统配置域 | ✅ 已落地 |
| Connector 外部 IdP（OIDC/OAuth2 驱动、身份解析、账号关联、域策略）| ✅ 已落地（OIDC/OAuth2 驱动；LDAP/SAML 为预留）|
| API Key（M2M）| ✅ 已落地 |
| 审计（audit_log/session/user_login_log）| ✅ 已落地 |
| 登录限流 + 临时锁定 | ✅ 已落地 |
| **待做**：Backend-Channel 登出(RP 配合)、密钥全自动轮换、scope 精细到资源级控制台、验证码登录、邀请加入(`allowJoinByInvite`)、LDAP/SAML Connector、签名密钥多 kid 动态轮换 |

### 6.5 范围边界（不过度设计）
- 登出 MVP 用"清会话+吊销 refresh；内网 access 经 SSO 活性即时失效、外部 access 靠 TTL"。Backend-Channel 通知后置。
- scope/RBAC MVP 落地到 role→scope→resource 与 role→menu 双维。
- Connector 已实现 OIDC/OAuth2 账号密码与第三方；LDAP/SAML 后置。

---

## 7. 落地核对清单

- [x] 认证后、发 token 前选租户（`/oidc/login` → `/oidc/login/selectTenant`）
- [x] person 级中心会话（`iam_sso_session` / Redis）+ 租户上下文 token
- [x] 全局登出 = 清会话 + 吊销 refresh；内网 access 即时失效，外部靠 TTL
- [x] 应用(产品)全局配置 + `application_client`(接入端) + `application_client_secret`(密钥轮换)
- [x] `tenant_application` 显式多对多 + `granted_scope`
- [x] 管理员集中配置 OAuth Client（`applicationClient` 管理 API）
- [x] 切租户签发新 token（`tenant` query hint），不追溯旧 token
- [x] 账号产生 = 注册 + joinTenant + 管理员创建 + Connector 自动创建
- [x] `sub` = 全局 `person:{id}`（稳定），租户上下文走私有 claim
- [x] 应用级 `tenantPolicy.allowPersonCreateTenant`（0 租户可配置，`tenant/createAsOwner`）
- [x] 统一 token 管理面鉴权：权限 = 用户 × 租户上下文 × RBAC；API Key 与用户一对一
- [x] OIDC 库：zitadel/oidc (OP) + coreos/go-oidc (RP) + golang-jwt + go-jose + bcrypt
- [x] OIDC Provider 前缀 `/oidc`（版本无关），管理面 `/v1/iam/*`（带版本）
