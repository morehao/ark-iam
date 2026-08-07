# IAM 系统设计文档

> 本文档描述一套**全新设计**的统一认证服务（IAM）方案，完全独立于仓库内任何已有实现，所有设计均以主流 IdP（Auth0、Okta、Keycloak、微软 Entra）的实践为基准。

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

```
                  ┌────────────────────────────────┐
    登录 ─────────►  Person 级中心会话 (Redis)         │
                   key: sso_session:{sid}           │
                   data: {person_id, created_at}    │ ← 只存"你是谁"，不存租户选择
                   TTL: 24h 滑动续期, HttpOnly+Secure │
                   └────────────────────────────────┘
                              │ 取 person_id
                              ▼
   ┌────────────────────────────────────────────────┐
   │ ID Token  (JWT, 面向 RP)                        │
   │ sub=租户内user_id, aud=client_id, profile/email │ —— 身份证明
   ├────────────────────────────────────────────────┤
   │ Access Token (JWT, 面向资源服务器)               │
   │ sub=租户内user_id, aud=目标资源, tenant_id,      │ —— 无状态，JWKS 本地验签
   │ user_id, scope                                   │
   ├────────────────────────────────────────────────┤
   │ Refresh Token (不透明随机串, DB 存哈希)           │ —— 单次使用轮换，可吊销
   │ 关联 person_id/tenant_id/user_id/application_id │
   └────────────────────────────────────────────────┘
```

- **access** 无状态 JWT，资源服务器用 JWKS 验签，几乎免查库。
- **refresh** 服务端哈希存储、单次使用轮换，支撑续期与吊销。

### 1.3 需求如何被满足

1. **一登录 → 其他应用免登录** ← 浏览器带 person 级 `sso_session` Cookie，应用 B 走 `/authorize`→`/sso-login` 时见中心会话直接放行（仅需选租户拿 token）。
2. **一退出 → 所有重新登录** ← 全局登出：清中心会话 + 吊销该 person 所有 refresh token；access 靠 TTL 几分钟内失效。
3. **多租户先选租户** ← 认证在后，选租户在发 token 前，token 携带租户上下文；单租户自动选，多租户手选。

---

## 2. 关键流程

### 2.1 token 声明边界（主流分工）

| Token | 内容 | 用途 |
|---|---|---|
| ID Token | `iss, sub(=租户内user_id), aud(client_id), exp, iat, auth_time, nonce, amr`, profile/email | end-user 身份证明，仅声明不作资源鉴权 |
| Access Token | `iss, sub, aud(目标API资源), client_id, exp, iat, scope, tenant_id, user_id` | 调用资源服务器凭证，无状态验签 |
| Refresh Token | 不透明随机串 → DB 存哈希，记 `person_id/tenant_id/user_id/application_id` | 续期，单次使用轮换，可吊销 |
| 中心会话 Cookie | 仅 `person_id + created_at` | SSO 物理载体，不存租户选择 |

> **sub 语义**：`sub` = 该 OIDC Client 在特定租户上下文内的稳定用户标识（源自 user_id），**不暴露 person_id**。同一自然人在不同租户接入同一应用时 sub 不同，符合 OIDC 规范"sub 在 issuer+audience 内稳定"。

### 2.2 登录 + 选租户（Authorization Code Flow）

```
用户 → RP → IAM /authorize(client_id, redirect_uri, scope, state, nonce)
        │
        ▼ 检查中心会话
    ┌───┴────────────┐
    │ sso_session?     │
    └───┬────────────┘
  有(person已认证)      无(未登录)
     │                    │ 302→前端登录页, 输入账号密码
     ▼                    ▼ 认证成功→建中心会话
      └───────┬────────────┘
              ▼
 查询该 person 可用租户列表 (user 记录 × 该应用 tenant_application 可见性)
      │
      ├── 0 个  → 读取【该应用】tenantPolicy.allowPersonCreateTenant:
      │           ├ true  → 跳转"创建租户"页, 成为 owner → 回流程
      │           └ false → 提示无权访问(401/403)
      ├── 1 个  → 前端自动使用该租户, 跳过选择
      └── 多个  → 停留租户选择页, 用户手选
              │
              ▼
 生成 code → 302 redirect_uri?code&state → RP 用 code 换 token
 (token 携带 sub=租户user_id, tenant_id, user_id)
```

**租户选择发生在"认证后、发 code 前"**；中心会话只记住"你是谁"，租户选择决定"这次签发哪个租户的 token"。

### 2.3 应用级租户策略（`tenantPolicy`）

"0 租户是否允许自助建租户"是**应用自身的产品决策**，因此作为应用（OIDC Client / product）级配置：

```json
{
  "tenantPolicy": {
    "allowPersonCreateTenant": true,
    "allowJoinByInvite": true
  }
}
```

- `allowPersonCreateTenant`：0 可用租户时是否允许该自然人在此应用语境下自助建租户。
- `allowJoinByInvite`：是否允许用户接受邀请加入已有租户（预留，可进 V1.1）。

### 2.4 切换租户

- 中心会话**不变**。
- 触发：前端/应用调用 IAM 切换租户接口，或授权请求携带租户提示（`login_hint`/自定义 `org` 参数）再走 `/authorize`。
- IAM 针对**新租户**重新签发 token（`sub`=新租户 user_id，`tenant_id`=新租户）。
- **不追溯旧租户 token**：其它应用已持有的旧租户 access token 在 TTL 内仍可用、可 refresh 续期；只有它们下次刷新/登录时才切到新租户上下文。

| 认证/发码 | 全局会话 | token |
|---|---|---|
| Person（全局认证）| per person | 租户上下文独立 |

### 2.5 全局登出（需求："一退出 → 所有重新登录"）

```
用户 → 任一 RP → IAM /end_session?id_token_hint&post_logout_redirect_uri
  │
  ▼
① 删除中心会话 (sso_session cookie + Redis 记录)
② 吊销该 person 在所有租户下的 refresh token (DB 置 revoked_at)
③ 302 → post_logout_redirect_uri
access token 无状态, 靠自身 TTL 在几分钟内失效（如需即时可加黑名单, 非 MVP）
```

效果：一处退出 → 中心会话清除 + 所有 refresh 吊销 → 其它应用刷新失败需重新认证。

---

## 3. 数据模型

### 3.1 实体（按领域）

**身份域（全局）**
- `person`(id, username UNIQUE, primary_email UNIQUE?, primary_phone UNIQUE?, password_encrypted, password_method, name, avatar, profile JSON, custom_data JSON, is_suspended, last_sign_in_at, created_at/updated_at)。认证单位，登录标识列分别唯一 + 索引。
- `user_identity`(person_id, connector_id, issuer, external_subject, provider; UNIQUE(issuer, external_subject))。外部 IdP 绑定载体（V1.1）。

**租户域**
- `tenant`(id, name, code UNIQUE, type[platform/customer], is_suspended, timestamps)。业务隔离边界。
- `user`(id, tenant_id, person_id, name, avatar, status, is_owner; UNIQUE(tenant_id, person_id))。授权单位，`sub` 派生源。

**应用域**
- `application`(id, name, type[web/spa/native/machine], is_third_party, visibility[public/private], tenant_policy JSON, status, timestamps)。**产品**，全局配置。
- `application_client`(id, application_id, client_id UNIQUE, client_secret_hash, name(如 "Web"/"App"), oauth_metadata JSON: redirect_uris/grant_types/response_types/token_endpoint_auth_method/require_pkce/default_scopes/access_token_ttl/refresh_token_ttl/allowed_origins/require_auth_time)。**OIDC 接入端**，一个产品 1:N 多个接入端。
- `tenant_application`(id, tenant_id, application_id, granted_scope, status; UNIQUE(tenant_id, application_id))。租户 ↔ 应用可用性显式关联。

**权限域（RBAC，MVP 最小化）**
- `role`、`user_role`、`scope`：MVP 内置 `openid/profile/email` + `user→role` 最小集，精细 scope→resource 后置。

**会话/令牌域**
- `session`(id, person_id, session_id UNIQUE, client_ip, created_at...)。会话审计（主存 Redis）。
- `refresh_token`(id, person_id, tenant_id, user_id, application_id, session_id, token_hash UNIQUE, client_type, client_ip, expired_at, revoked_at, last_rotated_at; 索引(person_id, tenant_id))。不透明随机串，单次轮换，按 person 批量吊销。

**连接器域（可选 V1.1）**
- `connector`(id, tenant_id, protocol[OIDC/LDAP/SAML/social], provider, status, config JSON, claim_mapping JSON, domain_policy JSON, allow_auto_create_user, allow_account_link, sync_profile)。

### 3.2 实体关系

```
person ──1:N──> user <──N:1── tenant
  │ N:1
  ├─1:N──> user_identity (外部IdP, 可选)
  │
tenant ──1:N──> tenant_application <──N:1── application
  │                                             │ 1:N
  │                                             application_client (OIDC接入端)
person ──1:N──> refresh_token ──N:1──> application_client
user   ──(关联)──> refresh_token(tenant_id+user_id)
```

- `user` = `person × tenant` JOIN 产物（多态身份）。
- `tenant_application` 决定租户选择可见性过滤的依据。
- product 与接入端解耦：对应主流 Auth0 的 `API(产品) → Application(client)`。

### 3.3 关键设计取舍

**Application 与 OIDC Client 的关系（两层模型）**

- **`application`（产品）**：业务产品，含租户可见性、`tenantPolicy`，管理员按产品管理。
- **`application_client`（接入端 = OIDC Client）**：一个产品可有多个接入端。
- **多端（web/安卓/iOS）默认共享一个 client**：三端差异全在 `oauth_metadata.redirect_uris`（各自带自己的 redirect_uri）与 PKCE。纯 SPA web + 安卓 + iOS 均 `token_endpoint_auth_method=none` + PKCE，无 secret 冲突，可共享。
- **何时拆 client**：出现"传统服务端 web（用 client_secret）"且同时存在 SPA/移动端时，因 `secret` vs `none` 无法共存于一个 client，才拆为 `web`(secret) 与 `spa`(PKCE/none) 两个 client。
- **扩展性**：所有行为差异收敛进 JSON metadata，不靠拆表；若未来需"多环境独立密钥"，在 `application_client` 下继续纵向增行即可，演进容易。

**`tenant_application` 采用显式关联表（方案 A）**：天然支持"按租户开通应用"、与"0 租户=无可用应用"语义契合。

---

## 4. 后台 API 与 OIDC 端点

### 4.1 访问面

| 面 | 面向 | 鉴权 |
|---|---|---|
| 认证/注册/SSO | 浏览器、未登录 | 白名单 |
| OIDC 协议端点 | 任意 client/RP | OIDC 内建 |
| 管理 API | 管理员、内部服务 | 统一 token（用户×租户上下文）|

### 4.2 路由前缀

```
/v1/iam/auth/*       登录/注册/我的租户/选切租户/刷新/登出 (公共+JWT)
/v1/iam/oidc/*       OIDC Provider 标准端点 (含 /.well-known/*)
/v1/iam/console/*    管理后台 API (统一 token 鉴权)
```

### 4.3 OIDC 协议端点

| 端点 | 说明 | 认证 |
|---|---|---|
| `GET /v1/iam/oidc/.well-known/openid-configuration` | Discovery | 无 |
| `GET /v1/iam/oidc/.well-known/jwks.json` | 公钥 | 无 |
| `GET\|POST /v1/iam/oidc/authorize` | 授权入口 | 无 |
| `GET /v1/iam/oidc/authorize/callback` | 租户选择完成→发 code→重定向 RP | 无 |
| `GET /v1/iam/oidc/sso-login` | SSO 自动登录检查（见 2.2/2.5）：有中心会话→自动认证；无→重定向前端登录页 | 中心会话 Cookie |
| `POST /v1/iam/oidc/oauth/token` | code换/refresh/client_credentials | client 认证 |
| `GET\|POST /v1/iam/oidc/userinfo` | 用户信息 | access token |
| `GET\|POST /v1/iam/oidc/end_session` | 全局登出 | id_token_hint |
| `POST /v1/iam/oidc/revoke` | 吊销 token | client 认证 |

**协议库（不造轮子）**
- OIDC **Provider(OP)**：`github.com/zitadel/oidc/v3`（auth code + PKCE + refresh + custom 端点）
- 对接外部 IdP 的 **RP/校验**：`github.com/coreos/go-oidc/v3`
- 密码哈希：`golang.org/x/crypto`（argon2id/bcrypt）
- Web 框架：`gin-gonic/gin`；JWT/JWKS：`go-jose` 或 zitadel 自带；会话/缓存：`github.com/redis/go-redis/v9`

### 4.4 统一 token 管理面鉴权（重要修正）

- **所有登录产生同一种 token，不区分"平台管理员/租户管理员"身份**。一个用户有多身份，但对 IAM 都是同一种登录。
- 权限判定**不由 token 类型决定，而由"当前 user 在该 tenant 上下文下拥有的能力"决定**（`is_owner` 或 V1.1 的 `role`）。
- "平台管理员"与"租户管理员"非两类 token，而是同一用户分别在"平台租户上下文"与"客户租户上下文"下继承到的不同权限；**切换租户上下文，权限随之切换，token 形态不变**。管理 API 复用第 2 节同一套 token。
- **API Key 与用户一对一**：API Key 归属某个 user，相当于该用户授权机器用其身份调用接口。鉴权：`x-api-key` → 解析对应用户 + 租户上下文 → 按该用户权限判定。无需独立"机器身份/服务账号"模型；吊销 API Key 即解权，不影响用户本人。

简化模型一句话：

> 统一的身份系统：所有登录一种 token，权限由"用户 × 租户上下文"决定；API Key 只是某用户的可吊销具名凭证。

### 4.5 管理 API 覆盖

- 身份：person 注册/查询/挂起、user 创建/邀请/加入/列表
- 租户：tenant 创建/列表/配置、tenant_application 开通/关闭/查看
- 应用：application 创建、oauth_metadata/tenant_policy 更新、secret 轮换、启停
- 令牌：refresh 列表/吊销、会话管理、审计查询

---

## 5. 安全设计

### 5.1 密码
- argon2id（首选）或 bcrypt(cost≥10)，不存明文；`password_method` 记录算法便于迁移。
- 登录限流 + 失败锁定 + 验证码阈值（可配置）。
- 改密/挂起/删除后吊销该 person 全部 session + refresh（强制拉登）。

### 5.2 密钥
- `client_secret_hash`(SHA-256) 存储，明文仅创建/轮换时返回。
- 双 secret 并存过渡期 + 到期自动作废，避免改密即断。
- OIDC 签名私钥不落库或加密落库，JWKS 暴露公钥，支持多 kid 轮换（旧 kid 保留到未过期 token 失效）。

### 5.3 协议安全
- **PKCE**：SPA/移动端 `token_endpoint_auth_method=none` + 强制 `require_pkce`，S256。
- **redirect_uri** 精确白名单（scheme/端口/path 完全一致），防开放重定向。
- **state/nonce** 防 CSRF/重放；授权码一次性、短 TTL、绑 client、校验 code_verifier。
- **token 存储提示**：RP 用后端 httpOnly cookie/BFF 或 SPA 内存+PKCE refresh；禁 localStorage 存 access。
- **logout**：校验 id_token_hint + post_logout_redirect_uri 白名单，防 open-redirect/login CSRF。

### 5.4 token 生命周期
| 令牌 | 生命周期 |
|---|---|
| Access | 短 TTL(如 1h, 可按 client)；RS256 无状态验签 |
| Refresh | 长 TTL(7d~30d)；单次使用轮换；可吊销；与中心会话绑定随全局登出吊销 |
| ID | 短 TTL(1h)，仅身份声明 |
| 中心会话 | 24h 滑动续期；HttpOnly+Secure+SameSite=Lax |

### 5.5 审计与可观测
- `audit_log`：登录成败、选租户、切租户、应用创建/密钥轮换、管理操作、API Key 使用（actor、tenant、client_id、ip、ua、ts、result）。
- 登录失败/高频 IP 告警；关键管理操作留痕。
- 结构化日志 + request_id 贯穿；指标（登录 QPS、token 校验延迟）。

### 5.6 多租户隔离
- 租户内查询强制携带 `tenant_id` 过滤（中间件注入，防越权）。
- 授权/签发严格经 `tenant_application` 过滤。
- `sub`=租户内 user id，天然隔离不同租户用户标识。

---

## 6. 非功能 / 部署 / 演进

### 6.1 非功能需求
- **可用性**：无状态 API 水平扩展；中心会话/限流放 Redis；DB 主从；签名私钥多实例一致。
- **性能**：JWT 无状态验签（JWKS 本地缓存）+ 索引；鉴权路径免查库；核心端点近线性。
- **扩展性**：签名轮换、JWKS 多 kid；refresh 单次轮换配合原子 SQL 避免多实例竞态。
- **安全/合规**：TLS、审计保留、租户隔离、敏感字段哈希/加密。

### 6.2 部署（MVP 先行）
```
[浏览器/RP] --TLS--> [IAM 单体服务(Gin)] --+--> MySQL(业务/refresh/审计)
                                          +--> Redis(中心会话/限流)
```
- 签名私钥由环境变量/密钥卷提供。
- 独立 IdP 服务拆分（授权服务与管理/业务分离）为演进选项，不进 MVP。

### 6.3 技术栈
```
Web框架        : gin-gonic/gin
OIDC Provider  : github.com/zitadel/oidc/v3
对接外部IdP(RP): github.com/coreos/go-oidc/v3
密码哈希       : golang.org/x/crypto (argon2id/bcrypt)
JWT/JWKS       : github.com/go-jose/go-jose 或 zitadel 自带 (RS256)
数据库         : MySQL/PostgreSQL + GORM
缓存/会话      : Redis (github.com/redis/go-redis/v9)
```

### 6.4 演进路径

| 阶段 | 内容 |
|---|---|
| **MVP** | 密码登录/注册/选租户/单租户自动选/应用级 `allowPersonCreateTenant`/标准 OIDC AuthCode+PKCE/全局登出/应用+user 管理/租户隔离/API Key(M2M) |
| **V1.1** | 外部 IdP(Connector: 企业微信/Google/GitHub)/login_hint 租户提示/RBAC admin 角色/登录限流+验证码/邀请加入 |
| **V1.2** | Backend-Channel 登出(RP 配合)/密钥全自动轮换/scope 精细控制/审计查询台 |

### 6.5 范围边界（不过度设计）
- 登出 MVP 用"清会话+吊销 refresh，access 靠 TTL"；Backend-Channel 通知后置。
- scope/RBAC MVP 只内置 openid/profile/email + user→role 最小集。
- Connector 第三方登录首版不做，只做账号密码。

---

## 7. 待确认清单

- [x] 认证后、发 token 前选租户
- [x] person 级中心会话 + 租户上下文 token
- [x] 全局登出 = 清会话 + 吊销 refresh，access 靠 TTL
- [x] 应用全局配置 + tenant_application 显式多对多
- [x] 管理员集中配置 OAuth Client
- [x] 切租户签发新 token，不追溯旧 token
- [x] 账号产生 = 注册 + 邀请 + 管理员创建
- [x] `sub` = 租户内 user 标识（不暴露 person）
- [x] 应用级 `tenantPolicy.allowPersonCreateTenant`（0 租户可配置）
- [x] application(产品) + application_client(OIDC 接入端, 1:N) 两层模型
- [x] 统一 token 管理面鉴权：权限 = 用户 × 租户上下文；API Key 与用户一对一
- [x] OIDC 库：zitadel/oidc (OP) + coreos/go-oidc (RP) + bcrypt/argon2id
