# 术语表（Glossary）

> Ark IAM 相关术语的统一定义。按主题分组；标注 ✅ 的为**本系统内**术语，其余为通用协议/行业术语。

---

## 一、身份与账号

| 术语 | 英文 | 说明 |
|---|---|---|
| 自然人 ✅ | Person | 跨租户的**全局身份**。用户名/邮箱/手机号全局唯一（可空），密码、全局状态（挂起）在此维护。OIDC `sub` 为 `person:<id>` |
| 租户成员 ✅ | User | 自然人（person）在某个**租户内**的成员记录。租户内姓名/资料/角色、是否拥有者（`is_owner`）、加入时间 |
| 租户 | Tenant | 独立的客户/组织边界。数据与权限按租户隔离；类型分 `customer`（客户）/`platform`（平台） |
| 租户拥有者 | Tenant Owner | 租户的拥有者成员（注册即成为首个拥有者），拥有租户管理权限 |
| 外部身份 | User Identity | person 在外部身份源（Connector）中的身份映射（issuer + external_subject） |
| 多租户 | Multi-tenant | 一个 person 可同时属于多个租户；登录时需选择租户（或由 `tenant` hint 指定） |
| 挂起 | Suspended | person/user 被停用，禁止登录（`is_suspended`） |

## 二、应用与客户端

| 术语 | 英文 | 说明 |
|---|---|---|
| 应用 ✅ | Application | 一个业务系统定义（编码/名称/类型/状态/可见性）。如"平台管理台" |
| OAuth 客户端 ✅ | Application Client | 应用下的 OIDC 接入凭证：client_id、回调白名单、授权类型、令牌 TTL 等 |
| 客户端密钥 ✅ | Client Secret | 机密客户端在令牌端点的认证凭证（库中只存哈希） |
| 第一方应用 | First-party App | 自有应用（`application.type=first_party`） |
| 第三方应用 | Third-party App | 外部接入应用（`application.type=third_party`） |
| 回调地址 | Redirect URI | 授权码回传地址，**必须精确白名单匹配** |

## 三、协议与令牌

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

## 四、权限模型

| 术语 | 英文 | 说明 |
|---|---|---|
| 角色 | Role | 权限载体（租户内、按应用作用域），类型 User/Machine |
| 菜单 | Menu | 前端可访问的菜单/路由（树形，按应用管理） |
| 权限点 | Scope | 细粒度权限标识（隶属于资源） |
| 资源 | Resource | 受保护资源（`indicator` 标识符，可配令牌 TTL） |
| 用户-角色 | User-Role | 用户与角色的多对多关联 |
| 角色-菜单 | Role-Menu | 角色可访问菜单的授权 |
| 角色-权限点 | Role-Scope | 角色拥有的权限点授权 |

## 五、认证通道与凭证

| 术语 | 英文 | 说明 |
|---|---|---|
| 密码登录 | Password Login | 用户名/邮箱/手机号 + 密码（bcrypt） |
| 连接器 | Connector | 外部身份源接入配置（OIDC/OAuth2 驱动，如企业微信、Google） |
| 登录风控 | Login Guard | 失败次数窗口与锁定（默认 5 次/5 分钟/锁 15 分钟） |
| API Key ✅ | API Key | 机器凭证（`x-api-key` 头携带，哈希存储、可过期/吊销/scope） |
| 机器令牌 ✅ | Machine Token | `token_usage=machine` 的令牌（client_credentials / API Key 签发），不依赖浏览器会话 |

## 六、基础设施

| 术语 | 英文 | 说明 |
|---|---|---|
| 认证 Redis ✅ | Auth Redis | 存放 SSO 会话/授权状态/令牌元数据/SLO 队列的 Redis（多应用共享） |
| 中心会话 ✅ | SSO Session | Redis 中的认证态（`iam:oidc:sso_session:*`），对应浏览器 `iam_sso_session` Cookie |
| 会话审计 ✅ | Session Audit | `session` 表记录，会话创建/撤销的审计落库 |
| 登录日志 ✅ | Login Log | `user_login_log` 表，每次密码登录的 IP/UA/时间 |
| 审计日志 ✅ | Audit Log | `audit_log` 表，业务操作审计（动作/目标/结果/详情） |
| 网关聚合 ✅ | Gateway | gateway 应用（:8100）单进程挂载 auth/platformadmin/tenantadmin |
