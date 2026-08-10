# IAM 数据模型与业务场景概览

> 本文档以 `backend/scripts/sql/iam_schema.sql`（共 30 张表）为权威基准，描述每个业务域的字段与场景。落地实现中 **OIDC Client 模型为 `application_client` + `application_client_secret`**，已无 `oauth_client` / `oauth_client_secret` / `application_secret` 旧表。

## 1. 租户与用户域

### tenant（租户表）

| 字段 | 业务含义 |
|------|---------|
| `name` | 租户名称，如 "Acme 公司" |
| `code` | 租户编码，UNIQUE，自动生成 |
| `type` | `customer`-客户租户, `platform`-平台租户 |
| `db_user` | 数据库用户 |
| `is_suspended` | 租户挂起后，该租户下所有用户无法登录 |
| `tag` | 租户标签 |

**场景**：SaaS 多租户隔离。每个租户有独立的组织架构、用户、应用、权限配置。`platform` 类型租户为平台运营方，可管理所有客户租户。

### person（自然人表）

全局唯一身份，跨租户共享。一个自然人可以加入多个租户。

| 字段 | 业务含义 |
|------|---------|
| `username` | 全局唯一用户名，UNIQUE |
| `primary_email` | 全局唯一邮箱，用于登录 |
| `primary_phone` | 全局唯一手机号 |
| `password_encrypted` | 加密后的密码（bcrypt） |
| `password_method` | 加密方式（bcrypt / argon2 等） |
| `is_suspended` | 自然人被挂起后，所有租户下的用户均无法登录 |
| `last_sign_in_at` | 最后登录时间 |
| `profile` | 扩展配置（JSON） |
| `custom_data` | 自定义数据（JSON，业务方自行使用） |

**场景**：
- 用户使用邮箱/手机号/用户名 + 密码登录 IAM（OIDC `/oidc/login`）。
- 一个人既是 A 公司的员工，又是 B 公司的外包人员，一个 person 对应多个 user。
- OIDC 令牌 `sub = person:<id>` 以 person.id 为稳定身份。

### user（租户用户表）

自然人在某个租户下的身份。

| 字段 | 业务含义 |
|------|---------|
| `tenant_id` + `person_id` | (一致) 一个自然人在一个租户下只有一个用户 |
| `name` | 租户内姓名，可与 person 层不同 |
| `avatar` | 租户内头像 |
| `is_owner` | 租户拥有者，拥有该租户全部权限 |
| `is_suspended` | 是否挂起 |
| `joined_at` | 加入租户时间 |
| `last_sign_in_at` | 最后登录时间 |

**场景**：
- 邀请/管理员创建用户时创建 user 记录。
- 用户切换租户时，本质是切换 user 上下文（变换 access token 的 `tenant_id`/`user_id`）。

### user_identity（自然人外部身份表）

绑定第三方身份（社交账号、企业微信、LDAP 等），由 Connector 写入。

| 字段 | 业务含义 |
|------|---------|
| `person_id` | 关联的自然人 |
| `connector_id` | 关联的连接器 |
| `provider` | 身份提供商，如 `wechat`, `dingtalk`, `ldap` |
| `issuer` | 身份签发方 URL |
| `external_subject` | 第三方用户唯一标识 |
| `uk_issuer_subject` | UNIQUE，确保同一第三方用户只能绑定一个自然人 |
| `detail` | 外部身份详情（JSON） |
| `last_used_at` | 最后使用时间 |

**场景**：
- 企业微信扫码登录：员工的企业微信 userid 绑定到 person。
- LDAP 账号登录：LDAP 的 dn 绑定到 person。
- 社交登录：微信/Google OAuth 关联已有账号。

### user_login_log（用户登录日志表）

记录每次登录行为。

| 字段 | 业务含义 |
|------|---------|
| `tenant_id` / `user_id` | 登录者 |
| `login_type` | 密码 / oauth / ldap 等 |
| `login_ip` / `user_agent` | 客户端信息 |
| `login_time` | 时间 |

---

## 2. 权限域（RBAC + 菜单）

落地为**双维 RBAC**：权限检索（scope）+ 菜单可见（menu）。

```
user ──> user_role ──> role ──> role_scope ──> scope ──> resource
                                 └──> role_menu ──> menu
```

### resource（资源表）

| 字段 | 业务含义 |
|------|---------|
| `tenant_id` | 租户 |
| `name` | 资源名称 |
| `indicator` | 资源标识符，如 `urn:ark:iam:admin` |
| `is_default` | 是否默认资源 |
| `access_token_ttl` | 访问该资源的 token 默认有效期 |

### scope（权限范围表）

| 字段 | 业务含义 |
|------|---------|
| `tenant_id` / `resource_id` | 所属租户/资源 |
| `name` | 权限名称，如 `admin:user:read` |
| `description` | 权限描述 |

### role（角色表）

| 字段 | 业务含义 |
|------|---------|
| `tenant_id` / `app_id` | 所属租户/应用 |
| `name` / `code` | 角色名称 / 编码（如 `admin`, `user`）|
| `type` | 角色类型，默认 `User` |
| `is_default` | 是否默认角色 |

### role_scope / user_role（角色-权限、用户-角色关联）

- `role_scope(role_id, scope_id)`：角色拥有的权限集合。
- `user_role(user_id, role_id)`：为用户分配角色。

**场景**：RBAC；默认角色：新加入租户的用户自动获得基础权限；用户拥有多个角色，权限取并集。

### menu（菜单表）+ role_menu（角色菜单关联表）

| menu 字段 | 业务含义 |
|-----------|---------|
| `parent_id` | 父菜单 ID，树形结构 |
| `path` / `component` | 前端路由 / 组件路径 |
| `permission` | 权限标识，如 `iam:user:create` |
| `type` | 菜单类型（目录/菜单/按钮）|
| `hidden` / `external_link` / `keep_alive` | 显示与缓存控制 |
| `status` | `enable` / `disable` |

- `role_menu(role_id, menu_id)`：决定角色可见的菜单项。

**场景**：不同角色登录后看到不同菜单（管理员看全部，普通用户只看部分）；按钮级权限控制。

---

## 3. 组织域

### organization（组织表）

租户下的顶层组织架构，独立于部门树。

| 字段 | 业务含义 |
|------|---------|
| `tenant_id` | 租户 |
| `name` / `description` | 组织名称 / 描述 |
| `custom_data` | 自定义数据（JSON） |
| `is_mfa_required` | 是否要求 MFA |

### organization_role / organization_user / organization_role_user

- `organization_role(organization_id, name, type)`：组织级角色（不依赖全局 `role` 表）。
- `organization_user(organization_id, user_id)`：用户与组织的隶属关系。
- `organization_role_user(organization_id, organization_role_id, user_id)`：组织内为用户分配组织角色。

**场景**：大型企业按"事业部"划分组织，每个事业部有独立的管理员角色；某些组织要求 MFA 认证。

---

## 4. 部门域

### department（部门表）

树形部门结构，`parent_id` 实现层级。

| 字段 | 业务含义 |
|------|---------|
| `parent_id` | 父部门 ID，0 为根部门 |
| `code` | 部门编码，如 `DEPT-IT-001` |
| `sort` | 排序 |
| `leader_user_id` | 部门负责人 |

### user_department（用户部门表）

| 字段 | 业务含义 |
|------|---------|
| `is_primary` | 是否主部门，一个用户有且仅有一个主部门 |

**场景**：组织架构树展示；按部门统计用户；用户可属于多个部门（兼职）。

---

## 5. 应用域（双层模型）

### application（应用表 · product）

代表被 IAM 认证授权的"业务产品"。

| 字段 | 业务含义 |
|------|---------|
| `code` | 应用编码，UNIQUE |
| `name` / `description` | 名称 / 描述 |
| `logo_url` / `homepage_url` | 品牌信息 |
| `type` | `first_party` / `third_party` |
| `visibility` | `public` / `private` |
| `tenant_policy` | 租户策略 JSON：`{allowPersonCreateTenant, allowJoinByInvite}` |
| `is_system` | 是否系统内置 |

**不再在 application 上存放 client 配置**（`secret`/`oidc_client_metadata` 已移除），OIDC 接入项统一下沉到 `application_client`。

### application_client（OIDC 接入端表）

一个 product 可有多个接入端，对应 OIDC Client。

| 字段 | 业务含义 |
|------|---------|
| `app_id` | 所属 product（application.id）|
| `client_id` | OIDC Client ID，UNIQUE |
| `redirect_uris` | 登录回调白名单（JSON）|
| `post_logout_redirect_uris` | 登出回调白名单（JSON）|
| `grant_types` | `["authorization_code","refresh_token"]` 等 |
| `response_types` | `["code"]` |
| `token_endpoint_auth_method` | `client_secret_basic` / `client_secret_post` / `none` |
| `allowed_origins` | CORS 白名单（JSON）|
| `require_pkce` | 是否强制 PKCE |
| `require_auth_time` | 是否需要 auth_time 声明 |
| `default_scopes` | 默认 scope（JSON）|
| `access_token_ttl` / `refresh_token_ttl` | 令牌有效期（秒）|
| `type` / `is_third_party` / `is_system` | 属性标记 |

### application_client_secret（客户端密钥表）

支持**多密钥平滑轮换**。

| 字段 | 业务含义 |
|------|---------|
| `application_client_id` | 所属接入端 |
| `name` | 密钥名称，如 "当前密钥-v3" |
| `value_hash` | SHA-256 哈希，明文仅创建/轮换时返回 |
| `value_prefix` | 密钥前缀，用于识别 |
| `expired_at` / `revoked_at` | 过期 / 吊销 |

### tenant_application（租户应用订阅表）

租户 ↔ 应用可用性显式关联。

| 字段 | 业务含义 |
|------|---------|
| `tenant_id` + `app_id` | UNIQUE，租户开通应用 |
| `granted_scope` | 租户级 scope 授权（JSON）|
| `config` | 租户级应用配置（JSON）|
| `status` | 启停 |

**场景**：按租户开通应用；"0 租户 = 无可用应用"，支持 `allowPersonCreateTenant` 自助建租户。

---

## 6. 认证与安全域

### refresh_token（刷新令牌表）

| 字段 | 业务含义 |
|------|---------|
| `person_id` / `tenant_id` / `user_id` | 三方联合确定用户身份 |
| `application_client_id` | 颁发该 token 的接入端 |
| `session_id` | 关联的 SSO 会话 |
| `token` | token 的 SHA256 哈希，不存明文 |
| `client_type` / `client_ip` / `user_agent` | 客户端信息 |
| `expired_at` / `revoked_at` / `last_rotated_at` | 生命周期 |

**场景**：长会话续期（默认 30d，单次使用轮换）；登出/改密时按 person 批量吊销实现全局登出。

### api_key（API 密钥表 · M2M）

| 字段 | 业务含义 |
|------|---------|
| `tenant_id` | 所属租户 |
| `name` | 密钥名称，如 "生产环境-订单服务" |
| `key_hash` | SHA256 哈希，不存原始密钥 |
| `key_prefix` | 前缀（前 7 位）|
| `scope` | 权限范围（JSON）|
| `expired_at` / `revoked_at` / `last_used_at` | 生命周期 |

**场景**：CI/CD 用 API Key 调用管理 API（`x-api-key` 或 OIDC client_credentials）；泄漏后立即吊销。

### session（会话审计表）

| 字段 | 业务含义 |
|------|---------|
| `person_id` | 会话归属自然人 |
| `session_id` | UNIQUE，与 SSO session 对应 |
| `tenant_id` | 租户 |
| `login_time` / `last_active_at` / `revoked_at` / `status` | 生命周期 |

### audit_log（审计日志表）

| 字段 | 业务含义 |
|------|---------|
| `actor_person_id` / `actor_user_id` | 操作人 |
| `tenant_id` / `client_id` | 上下文 |
| `action` | 动作标识，如 `login`, `tenant.switch`, `application.create` |
| `target_type` / `target_id` | 目标对象 |
| `result` | `success` / `failure` |
| `ip` / `user_agent` / `detail` | 环境与详情 |

---

## 7. 连接器域

### connector（连接器表）

外部身份源配置，驱动 IAM 作为 RP 对接（OIDC/OAuth2，LDAP/SAML 预留）。

| 字段 | 业务含义 |
|------|---------|
| `protocol` | `OIDC` / `LDAP` / `SAML` / `social` |
| `provider` | `google`, `microsoft-entra`, `github` 等 |
| `status` | 启用/停用 |
| `config` | 连接器配置（JSON），如 OIDC discovery URL |
| `claim_mapping` | 声明映射（JSON），外部字段 → IAM 字段 |
| `domain_policy` | 域策略（JSON），限定 email 域名 |
| `allow_auto_create_user` / `allow_account_link` / `sync_profile` | 身份解析策略 |

**场景**：企业微信/Google 扫码登录；LDAP 账号登录；自动建号（person + user + user_identity）。

---

## 8. 基础设施

### system（系统配置表）

租户级 KV 配置。

| 字段 | 业务含义 |
|------|---------|
| `tenant_id` / `key` / `value` | 如 `login_policy`, `password_policy`（JSON 值） |

### domain（域名表）

租户绑定的自有域名。

| 字段 | 业务含义 |
|------|---------|
| `tenant_id` + `domain` | UNIQUE |
| `is_verified` / `verified_at` | DNS 验证状态 |

**场景**：租户用自有域名作为 IAM 登录页；邮箱域名归属判定。

### log（日志表）

通用日志。

| 字段 | 业务含义 |
|------|---------|
| `tenant_id` / `key` | 日志分类 |
| `payload` | JSON 详情 |

---

## 9. 权限判定路径汇总

- **用户维度**：`user → user_role → role → role_scope → scope → resource`
- **菜单维度**：`user → user_role → role → role_menu → menu`
- **租户隔离**：所有租户内表强制携带 `tenant_id`，OIDC 令牌以 `tenant_id` claim 承载。
- **M2M**：`api_key` 归属某 user，等价该用户授权机器用其身份调用。
