# 术语表（Glossary）

> Ark IAM 相关术语的统一定义。按主题分组；标注 ✅ 的为**本系统内**术语，其余为通用协议/行业术语。

---

## 一、身份与账号

| 术语 | 英文 | 说明 |
|---|---|---|
| 自然人 ✅ | Person | 跨租户的**全局身份**。用户名/邮箱/手机号全局唯一（可空），密码、全局状态（挂起）在此维护。OIDC `sub` 为 `person:<id>` |
| 租户成员 ✅ | User | 自然人（person）在某个**租户内**的成员记录。租户内姓名/资料/角色、是否拥有者（`is_owner`）、加入时间 |
| 租户 | Tenant | 独立的客户边界（业务主体的隔离单元）。数据与权限按租户隔离；类型分 `customer`（客户租户）/`platform`（平台租户） |
| 租户类型 | Tenant Type | **分类标识**：`customer` = 外部客户/合作方的独立租户；`platform` = 平台自运营租户（种子数据 Default Tenant 即平台租户）。当前仅用于分类展示，不参与数据隔离与权限判定（隔离一律按 `tenant_id`） |
| 租户编码 | Tenant Code | 租户的业务编码，全局唯一、创建后不可修改。由服务端自动生成，规则 `t_<12 位随机小写 hex>`（如 `t_3f7a9c1d2e4b`），见 `pkg/iam/tenant.GenerateCode`；平台租户（种子数据 Default Tenant）为固定值 `t_platform`——同前缀、后缀固定可读，因自动生成的随机段只用小写 hex，两者不会冲突 |
| 租户拥有者 | Tenant Owner | 租户的拥有者成员（注册即成为首个拥有者），拥有租户管理权限 |
| 外部身份 | User Identity | person 在外部身份源（Connector）中的身份映射（issuer + external_subject） |
| 多租户 | Multi-tenant | 一个 person 可同时属于多个租户；登录时需选择租户（或由 `tenant` hint 指定） |
| 挂起 | Suspended | person/user 被停用，禁止登录（`is_suspended`） |
| 租户状态 | Tenant Status | 租户生命周期状态（`tenant.status`）：`active` 正常 / `suspended` 已挂起。仅 active 允许其成员登录、签发与轮换令牌；挂起会撤销该租户成员的 refresh token 与 SSO 会话；禁止挂起操作者自己所在的租户（不可逆自锁） |

## 二、部门与归属

| 术语 | 英文 | 说明 |
|---|---|---|
| 部门 ✅ | Department | 租户内用户归属的容器（树形，可嵌套）。表 `department`，`dept_path`/`dept_depth` 物化祖先链与深度；每租户唯一根部门由建租户时创建 |
| 部门关系 ✅ | Department User | 用户与部门的归属关系（表 `department_user`），`relation_type` 三值：`primary` 行政主部门（每用户至多 1 条）、`secondary` 参与部门（可多条）、`leader` 部门负责人（每部门至多 1 人） |
| 根部门 ✅ | Root Department | 建租户时自动创建的顶级部门节点（`parent_id` 为空），同时是租户管理员的行政主部门 |

## 三、应用与客户端

| 术语 | 英文 | 说明 |
|---|---|---|
| 应用 ✅ | Application | 一个业务系统定义（编码/名称/类型/状态/可见性）。如"平台管理台" |
| OAuth 客户端 ✅ | Application Client | 应用下的 OIDC 接入凭证：client_id、回调白名单、授权类型、令牌 TTL 等 |
| 客户端密钥 ✅ | Client Secret | 机密客户端在令牌端点的认证凭证（库中只存哈希） |
| 第一方应用 | First-party App | 自有应用（`application.type=first_party`） |
| 第三方应用 | Third-party App | 外部接入应用（`application.type=third_party`） |
| 回调地址 | Redirect URI | 授权码回传地址，**必须精确白名单匹配** |

## 四、协议与令牌

| 术语 | 英文 | 说明 |
|---|---|---|
| 单点登录 | SSO | 一次认证，多应用免密通行 |
| 单点登出 | SLO | 一处登出，处处登出（含反向通道通知） |
| 授权服务器 / 身份提供商 | OP / IdP | 认证用户并签发令牌的一方（本系统为 auth 应用 `/oidc`） |
| 依赖方 | RP / Client | 接入认证的业务应用 |
| 资源服务器 | Resource Server | 承载受保护 API 的服务（业务后端） |
| 授权码 | Authorization Code | 授权后经回调回传的一次性凭证，用于换令牌 |
| 访问令牌 | Access Token | 访问 API 的短期凭证（本系统为 RS256 JWT） |
| ID 令牌 | ID Token | 携带用户身份声明的 JWT（本系统 10 分钟有效） |
| 刷新令牌 | Refresh Token | 换取新访问令牌的长期凭证（哈希存储、支持轮换） |
| 声明 | Claim | 令牌/UserInfo 中关于主体的键值信息（`sub`/`iss`/`aud`/`tenant_id` 等） |
| 范围 | Scope | 请求的权限范围（`openid`/`profile`/`email`…） |
| PKCE | Proof Key for Code Exchange | 授权码证明密钥（S256），防授权码拦截 |
| 发现端点 | Discovery | `/.well-known/openid-configuration`，声明 issuer 与全部端点 |
| JWKS | JSON Web Key Set | OP 验签公钥集合（`/oidc/keys`） |
| 反向通道登出（又称背信道登出） | Back-Channel Logout | 认证中心不经浏览器、在服务端直接向 RP 的 `back_channel_logout_uri` 推送 logout_token |
| 登出令牌 | logout_token | SLO 通知令牌（含 `events`、`sid`、`jti`） |
| 会话 ID | sid | SSO 会话标识，用于登出关联与 token 关联 |
| 认证方法引用 | AMR | 认证方法引用（如 `["pwd"]`），还原到 id_token |

## 五、权限模型

| 术语 | 英文 | 说明 |
|---|---|---|
| 角色 | Role | 权限载体（租户内、按应用作用域），类型 User/Machine。**无业务编码**：以名称作为应用内可读标识（同一应用内名称唯一），内置角色以「所属应用 + `source=builtin`」定位 |
| 菜单 | Menu | 前端可访问的菜单/路由（树形，按应用管理） |
| 权限点 | Scope | 细粒度权限标识（隶属于资源） |
| 资源 | Resource | 受保护资源（`indicator` 标识符，可配令牌 TTL） |
| 用户-角色 | User-Role | 用户与角色的多对多关联 |
| 角色-菜单 | Role-Menu | 角色可访问菜单的授权 |
| 角色-权限点 | Role-Scope | 角色拥有的权限点授权 |

## 六、认证通道与凭证

| 术语 | 英文 | 说明 |
|---|---|---|
| 密码登录 | Password Login | 用户名/邮箱/手机号 + 密码（bcrypt） |
| 连接器 | Connector | 外部身份源接入配置（OIDC/OAuth2 驱动，如企业微信、Google） |
| 登录风控 | Login Guard | 失败次数窗口与锁定（默认 5 次/5 分钟/锁 15 分钟） |
| API Key ✅ | API Key | 机器凭证（`x-api-key` 头携带，哈希存储、可过期/吊销/scope） |
| 机器令牌 ✅ | Machine Token | `token_usage=machine` 的令牌（client_credentials / API Key 签发），不依赖浏览器会话 |

## 七、基础设施

| 术语 | 英文 | 说明 |
|---|---|---|
| 认证 Redis ✅ | Auth Redis | 存放 SSO 会话/授权状态/令牌元数据/SLO 队列的 Redis（多应用共享） |
| 中心会话 ✅ | SSO Session | Redis 中的认证态（`iam:oidc:sso_session:*`），对应浏览器 `iam_sso_session` Cookie |
| 会话审计 ✅ | Session Audit | `session` 表记录，会话创建/撤销的审计落库 |
| 登录日志 ✅ | Login Log | `user_login_log` 表，每次密码登录的 IP/UA/时间 |
| 审计日志 ✅ | Audit Log | `audit_log` 表，业务操作审计（动作/目标/结果/详情） |
| 网关聚合 ✅ | Gateway | gateway 应用（:8100）单进程挂载 auth/platformadmin/tenantadmin |
