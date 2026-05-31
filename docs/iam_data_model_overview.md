# IAM 数据模型与业务场景概览

## 1. 租户与用户域

### tenant（租户表）

| 字段 | 业务含义 |
|------|---------|
| `name` | 租户名称，如 "Acme 公司" |
| `type` | `customer`-客户租户, `platform`-平台租户 |
| `is_suspended` | 租户挂起后，该租户下所有用户无法登录 |

**场景**：SaaS 多租户隔离。每个租户有独立的组织架构、用户、应用、权限配置。`platform` 类型租户为平台运营方，可管理所有客户租户。

### person（自然人表）

全局唯一身份，跨租户共享。一个自然人可以加入多个租户。

| 字段 | 业务含义 |
|------|---------|
| `username` | 全局唯一用户名，UNIQUE |
| `primary_email` | 全局唯一邮箱，用于登录和找回密码 |
| `primary_phone` | 全局唯一手机号 |
| `password_encrypted` | 加密后的密码 |
| `password_method` | 加密方式（bcrypt / argon2 等） |
| `is_suspended` | 自然人被挂起后，所有租户下的用户均无法登录 |
| `last_sign_in_at` | 最后登录时间 |
| `profile` | 扩展配置（JSON） |
| `custom_data` | 自定义数据（JSON，业务方自行使用） |

**场景**：
- 用户使用邮箱/手机号/用户名 + 密码登录 IAM
- 一个人既是 A 公司的员工，又是 B 公司的外包人员，一个 person 对应多个 user

### user（租户用户表）

自然人在某个租户下的身份。

| 字段 | 业务含义 |
|------|---------|
| `tenant_id` + `person_id` | UNIQUE，一个自然人在一个租户下只有一个用户 |
| `name` | 租户内姓名，可与 person 层不同 |
| `avatar` | 租户内头像 |
| `is_owner` | 租户拥有者，拥有该租户全部权限 |
| `joined_at` | 加入租户时间 |

**场景**：
- 邀请用户加入租户时创建 user 记录
- 用户切换租户时，本质是切换 user 上下文

### user_identity（自然人外部身份表）

绑定第三方身份（社交账号、企业微信、LDAP 等）。

| 字段 | 业务含义 |
|------|---------|
| `connector_id` | 关联的连接器 |
| `provider` | 身份提供商，如 `wechat`, `dingtalk`, `ldap` |
| `issuer` | 身份签发方 URL |
| `external_subject` | 第三方用户唯一标识 |
| `uk_issuer_subject` | UNIQUE，确保同一第三方用户只能绑定一个自然人 |

**场景**：
- 企业微信扫码登录：员工的企业微信 userid 绑定到 person
- LDAP 账号登录：LDAP 的 dn 绑定到 person
- 社交登录：微信/Google OAuth 关联已有账号

### user_login_log（用户登录日志表）

记录每次登录行为，用于审计和安全分析。

| 字段 | 业务含义 |
|------|---------|
| `login_type` | 登录类型，如 `password`, `sms`, `oauth`, `ldap` |
| `login_ip` | 登录 IP |
| `user_agent` | 客户端信息 |

---

## 2. 组织域

### organization（组织表）

租户下的顶层组织架构，独立于部门树。每个组织可要求 MFA。

### organization_role（组织角色表）

组织级角色，与全局 `role` 表独立。用于在组织内定义管理职能。

### organization_user（组织用户表）

用户与组织的隶属关系。

### organization_role_user（组织角色用户表）

在组织内为用户分配组织角色。

**场景**：
- 大型企业按"事业部"划分组织，每个事业部有独立的管理员角色
- 某些组织要求 MFA 认证（财务部等敏感部门）

---

## 3. 部门域

### department（部门表）

树形部门结构，`parent_id` 实现层级关系。

| 字段 | 业务含义 |
|------|---------|
| `parent_id` | 父部门 ID，0 为根部门 |
| `code` | 部门编码，如 `DEPT-IT-001` |
| `sort` | 排序，同级部门排序 |
| `leader_user_id` | 部门负责人 |

### user_department（用户部门表）

| 字段 | 业务含义 |
|------|---------|
| `is_primary` | 是否主部门，一个用户有且仅有一个主部门 |

**场景**：
- 组织架构树展示（树形递归）
- 按部门统计用户
- 用户可属于多个部门（兼职），主部门用于默认归属

---

## 4. 应用域

### 核心表：application（应用表）

每个租户下可注册多个"应用"（类似 OAuth2 Client），代表需要接入 IAM 认证授权的外部系统。

| 字段 | 业务含义 |
|------|---------|
| `name` | 应用名称 |
| `secret` | 应用密钥，用于 client_secret 认证 |
| `type` | 应用类型：`web` / `spa` / `native` / `machine` |
| `description` | 应用描述 |
| `oidc_client_metadata` | OIDC 客户端配置（JSON）：redirect_uris, grant_types, response_types, token_endpoint_auth_method 等 |
| `custom_client_metadata` | 自定义扩展配置（JSON） |
| `is_third_party` | 是否第三方应用，用于区分自建/外部应用 |

### application_secret（应用密钥表）

一个应用可拥有多个密钥，支持命名管理和过期策略。

| 字段 | 业务含义 |
|------|---------|
| `name` | 密钥名称，如 "当前密钥-v3", "预发布密钥-v4" |
| `value` | 密钥值 |
| `expired_at` | 过期时间，过期后认证失败 |

### application_role（应用角色关联表）

将平台角色授予应用，决定应用作为主体访问 API 时的权限范围。

### 真实场景示例

#### 场景 1：企业内部管理系统（自建 Web 应用）

| 属性 | 值 |
|------|-----|
| `name` | "Acme 内部 OA 系统" |
| `type` | `web` |
| `is_third_party` | `false` |
| `oidc_client_metadata` | `{"redirect_uris": ["https://oa.acme.com/callback"], "grant_types": ["authorization_code", "refresh_token"]}` |

**流程**：
1. 员工打开 OA 系统 → 跳转到 IAM 登录页
2. IAM 识别到来自 OA 系统的授权请求
3. 员工输入账号密码，IAM 认证后颁发 id_token + access_token
4. `application_role` 关联的角色决定 OA 系统能调用哪些 API

#### 场景 2：第三方 SaaS 集成

| 属性 | 值 |
|------|-----|
| `name` | "Slack 工作空间集成" |
| `type` | `web` |
| `is_third_party` | `true` |
| `custom_client_metadata` | `{"homepage_url": "https://slack.com", "privacy_policy_url": "https://slack.com/privacy"}` |

**区别**：
- 第三方应用只能访问被明确授权的 scope
- 用户授权时展示详细权限确认页
- redirect_uris 需人工审核
- token 有效期更短

#### 场景 3：移动 App

| 属性 | 值 |
|------|-----|
| `name` | "Acme 移动办公 App" |
| `type` | `native` |
| `oidc_client_metadata` | `{"redirect_uris": ["com.acme.oa://callback", "https://acme.com/app-callback"], "grant_types": ["authorization_code", "refresh_token"]}` |

**特点**：`type=native`，无 client_secret，强制使用 PKCE 流程。

#### 场景 4：M2M 服务间调用

| 属性 | 值 |
|------|-----|
| `name` | "订单服务 (order-svc)" |
| `type` | `machine` |
| `grant_types` | `["client_credentials"]` |

**流程**：
1. `order-svc` 使用 `client_id` + `client_secret` 获取 token
2. IAM 通过 `application_role` 查出该应用关联的角色和 scope
3. 调用其他服务 API 时传递 token 进行鉴权

#### 场景 5：多密钥轮换

```sql
-- 一个应用可同时存在多个密钥，用于平滑轮换
INSERT INTO application_secret (application_id, name, value, expired_at) VALUES
(1, '当前密钥-v3', 'sk-xxx', '2026-06-01'),       -- 当前在用
(1, '旧密钥-v2',  'sk-yyy', '2025-12-01'),         -- 已过期，过渡期保留
(1, '预发布-v4',  'sk-zzz', '2027-01-01');          -- 预生成，到期前切换
```

#### 场景 6：SPA 前端应用

| 属性 | 值 |
|------|-----|
| `type` | `spa` |
| `token_endpoint_auth_method` | `none` |
| `oidc_client_metadata` | `{"redirect_uris": ["https://admin.acme.com/callback"]}` |

**特点**：无 client_secret，强制 PKCE，推荐使用 BFF 模式或 httpOnly cookie 存储 token。

### 权限模型关系

```
application ──→ application_role ──→ role ──→ role_scope ──→ scope ──→ resource
                         │
                   user_role ←── user
```

权限判断路径：
- **用户维度**：`user → user_role → role → role_scope → scope → resource`
- **应用维度**：`application → application_role → role → role_scope → scope → resource`
- 应用可理解为"服务账号"，通过角色获得访问特定 API 的权限

---

## 5. 权限域

### resource（资源表）

代表受保护的 API 资源。

| 字段 | 业务含义 |
|------|---------|
| `indicator` | 资源标识符，如 `https://api.acme.com/orders` |
| `is_default` | 是否默认资源 |
| `access_token_ttl` | 访问该资源的 token 默认有效期 |

### scope（权限范围表）

资源下的具体操作权限。

| 字段 | 业务含义 |
|------|---------|
| `resource_id` | 所属资源 |
| `name` | 权限名称，如 `orders:read`, `orders:write` |
| `description` | 权限描述 |

### role（角色表）

| 字段 | 业务含义 |
|------|---------|
| `name` | 角色名称，如 "管理员", "普通用户" |
| `code` | 角色编码，如 `admin`, `user` |
| `type` | 角色类型，默认 `User` |
| `is_default` | 是否默认角色，新用户自动赋予 |

### role_scope（角色权限关联表）

角色拥有的权限集合。

### user_role（用户角色关联表）

为用户分配角色。

**场景**：
- RBAC 权限模型
- 默认角色：新加入租户的用户自动获得基础权限
- 用户拥有多个角色，权限取并集

---

## 6. 菜单域

### menu（菜单表）

前端菜单树，与 RBAC 权限联动。

| 字段 | 业务含义 |
|------|---------|
| `parent_id` | 父菜单 ID，树形结构 |
| `code` | 菜单编码，如 `user_management` |
| `path` | 前端路由路径 |
| `component` | 前端组件路径 |
| `icon` | 菜单图标 |
| `sort` | 排序 |
| `type` | 菜单类型（目录/菜单/按钮） |
| `permission` | 权限标识，如 `iam:user:create` |
| `hidden` | 是否隐藏（用于不在菜单栏展示但可访问的页面） |
| `external_link` | 是否外链 |
| `keep_alive` | 是否缓存页面 |
| `status` | `enable` / `disable` |

### role_menu（角色菜单关联表）

决定角色可见的菜单项。

**场景**：
- 不同角色登录后看到不同的菜单（管理员看全部，普通用户只看部分）
- 按钮级权限控制（通过 `permission` + `type=button`）

---

## 7. 连接器域

### connector（连接器表）

外部身份源配置，支持多种协议。

| 字段 | 业务含义 |
|------|---------|
| `protocol` | 协议类型：`OIDC`, `LDAP`, `SAML`, `social` 等 |
| `provider` | 提供商：`azure-ad`, `wechat-work`, `dingtalk`, `google` |
| `status` | 启用/停用 |
| `config` | 连接器配置（JSON），如 LDAP 的 host/port/bindDN，OIDC 的 discovery URL |
| `claim_mapping` | 声明映射（JSON），将外部身份源的字段映射到 IAM 字段 |
| `domain_policy` | 域策略（JSON），限制哪些域名下的邮箱允许登录 |
| `allow_auto_create_user` | 首次登录时是否自动创建用户 |
| `allow_account_link` | 是否允许自动关联已有账号 |
| `sync_profile` | 是否同步头像等资料 |

**场景**：
- 企业微信扫码登录：配置 `provider=wechat-work` 的连接器
- LDAP 账号登录：配置 `protocol=LDAP` 的连接器，映射 `cn` → `name`, `mail` → `email`
- 自动创建用户：外部身份首次登录时自动创建 person + user，免管理员手动录入

---

## 8. 认证与安全域

### refresh_token（刷新令牌表）

管理用户会话，支持 token 刷新和吊销。

| 字段 | 业务含义 |
|------|---------|
| `person_id` / `tenant_id` / `user_id` | 三方联合确定用户身份 |
| `application_id` | 颁发该 token 的应用 |
| `session_id` | 会话 ID，UNIQUE，用于会话管理和吊销 |
| `token` | token 的 SHA256 哈希，不存储明文 |
| `client_type` | 客户端类型 |
| `client_ip` | 客户端 IP |
| `expired_at` | 过期时间 |
| `revoked_at` | 撤销时间，用于主动登出 |

**场景**：
- 用户登录后长期保持会话（无需频繁输入密码）
- 用户登出时吊销 refresh_token，使该会话失效
- 密码修改后吊销所有会话
- 多设备登录管理（每个设备一个 session_id）

### api_key（API 密钥表）

用于服务端调用的持久化凭证。

| 字段 | 业务含义 |
|------|---------|
| `name` | 密钥名称，如 "生产环境-订单服务" |
| `key_hash` | SHA256 哈希，不存储原始密钥 |
| `key_prefix` | 前缀（前 7 位），用于识别和日志，如 `ak_prod_` |
| `scope` | 权限范围（JSON），精细控制该密钥能访问的资源 |
| `expired_at` | 过期时间 |
| `revoked_at` | 吊销时间 |
| `last_used_at` | 最后使用时间，用于检测未使用的密钥 |

**场景**：
- CI/CD 系统使用 API Key 调用 IAM 管理 API
- 第三方集成商使用 API Key 访问开放 API
- 密钥泄漏后立即吊销，不影响其他密钥

---

## 9. 基础设施

### system（系统配置表）

租户级别的 KV 配置存储。

| 字段 | 业务含义 |
|------|---------|
| `key` | 配置键，如 `login_policy`, `password_policy` |
| `value` | JSON 配置值 |

**场景**：
- 密码策略配置（最小长度、特殊字符要求）
- 登录安全策略（失败次数限制、验证码阈值）

### domain（域名表）

租户绑定的域名。

| 字段 | 业务含义 |
|------|---------|
| `domain` | 域名，如 `acme.com` |
| `is_verified` | 是否已通过 DNS 验证 |
| `verified_at` | 验证时间 |

**场景**：
- 租户使用自有域名作为 IAM 登录页（品牌定制）
- 邮箱域名验证：`xxxx@acme.com` 自动归属到 Acme 租户

### log（日志表）

通用审计日志。

| 字段 | 业务含义 |
|------|---------|
| `key` | 日志分类，如 `user.login`, `application.create` |
| `payload` | JSON 格式的日志详情 |
| `created_at` | 日志时间 |

**场景**：
- 操作审计：谁在什么时间做了什么操作
- 安全事件记录
