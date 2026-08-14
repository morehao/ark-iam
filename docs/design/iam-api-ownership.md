# IAM 后端接口归属规划（API Ownership）

> 对应任务：规划 `backend/apps` 下除 `gateway`（聚合应用）外各应用的接口归属，
> 检查实现问题（缺失 / 多余接口、契约不一致、实现 bug），为前端各应用完善提供依据。
> 本文件为规划与审计结论，代码层面的修复见 `git log` 对应提交。

## 1. 应用边界与接口归属

| 应用 | 职责（设计文档 §6.2） | 路由前缀 | 端口 |
|------|----------------------|----------|------|
| `auth` | 认证网关：登录/注册/token/OIDC Provider、个人中心、会话、connector | `/v1/iam/auth/*`、`/v1/iam/person/*`、`/v1/iam/user/sessions`、`/v1/iam/connector/*`、`/oidc/*` | 8081 |
| `platformadmin` | 平台管理：user/role/menu/tenant/department/system/application/apiKey/domain/log 等 | `/v1/iam/{user,role,menu,scope,resource,tenant,department,system,log,application,applicationClient,apiKey,domain,tenantApplication}/*` | 8082 |
| `tenantadmin` | 租户自服务：organization + orgRole/orgUser/orgRoleUser | `/v1/iam/organization*` | 8083 |
| `gateway` | 聚合应用：单进程挂载上述三者全部路由（不新增业务接口） | - | 8100 |

### 归属原则（与设计文档 §4.1 一致）

1. **共享 `/v1/iam/{module}/{operation}` 前缀**，`{module}` 为业务模块，`{operation}` 为操作（create/update/detail/pageList/delete…），不按应用分段。
2. 认证相关（注册/加入租户/登出/我的租户/用户信息/个人中心/会话/connector）归属 `auth`；
   平台管理各模块归属 `platformadmin`；组织与组织角色归属 `tenantadmin`。
3. **前端应用映射**：
   - `login-web`（:3000）→ `auth` 的 OIDC 凭证登录（`/oidc/login`、`/oidc/login/selectTenant`）。
   - `platform-admin-web`（:3001）→ `platformadmin` 全部模块 + `auth` 的个人中心/会话/租户切换。
   - `tenant-admin-web`（:3003）→ `tenantadmin` 全部模块 + `auth` 的个人中心/会话/租户切换。

## 2. 接口清单（现状全量）

### 2.1 auth 应用

#### 业务接口（`/v1/iam`）

| 方法 | 路径 | 请求 | 响应 |
|------|------|------|------|
| GET | /auth/myTenants | - | `{list:[{tenantID,name}]}` |
| POST | /auth/register | tenantID/username/primaryEmail/primaryPhone/password/name | `{userID}` |
| POST | /auth/joinTenant | tenantID | `{userID}` |
| POST | /auth/logout | refreshToken | - |
| POST | /auth/logoutAll | refreshToken | - |
| GET | /auth/userinfo | - | `{personInfo,userInfo}` |
| GET | /person/detail | - | PersonDetailResp |
| POST | /person/updatePassword | oldPassword/newPassword | - |
| GET | /user/sessions | page/pageSize | `{list,total}` |
| DELETE | /user/sessions | - | - |
| DELETE | /user/sessions/:sessionId | - | - |
| POST | /connector/create | ConnectorBaseInfo | `{connectorId}` |
| POST | /connector/delete | connectorId | - |
| POST | /connector/update | connectorId + ConnectorBaseInfo | - |
| GET | /connector/detail | connectorId | ConnectorDetailResp |
| POST | /connector/pageList | page/pageSize/tenantId/protocol/provider/status/name/displayName | `{list,total}` |
| POST | /connector/getFactoryList | protocol/provider | `{list}` |
| POST | /connector/:connectorId/test | - | `{success,message}` |
| POST | /connector/:connectorId/authorize | redirectUri/state/loginHint/responseMode | `{authorizationUrl}` |
| GET | /connector/callback | connectorId/code/state | `{accessToken,refreshToken}` |

#### OIDC 协议接口（`/oidc`）

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | /oidc/login | 凭证登录（login-web 专用） |
| POST | /oidc/login/selectTenant | 多租户选择 |
| GET | /oidc/sso-login | SSO 会话放行 |
| GET | /oidc/logged-out | 登出回跳 |
| * | /oidc/authorize、/oidc/oauth/token、/oidc/userinfo、/.well-known/* 等 | zitadel oidc Provider 标准端点 |

### 2.2 platformadmin 应用（`/v1/iam`）

| 模块 | 接口（create/delete/update/detail/pageList + 特有） |
|------|------|
| tenant | create/createAsOwner/delete/update/detail/pageList |
| department | create/delete/update/detail/pageList/tree |
| system | create/delete/update/detail/pageList |
| log | detail/pageList |
| apiKey | create/pageList/revoke/delete |
| user | create/delete/update/detail/pageList/updatePassword/updateStatus + createUserIdentity/deleteUserIdentity/updateUserIdentity/detailUserIdentity/pageListUserIdentity/getUserIdentityByUser + detailUserLoginLog/pageListUserLoginLog/getUserLoginLogByUser + getUserDepartmentByUser/assignDepartments |
| role | create/delete/update/detail/pageList + users(GET)/assignUsers(POST)/users/:roleId/:userId(DELETE) |
| menu | create/delete/update/detail/pageList/tree |
| scope | create/delete/update/detail/pageList |
| resource | create/delete/update/detail/pageList |
| roleMenu | create/delete/pageList |
| roleScope | create/delete/pageList |
| userRole | create/delete/pageList |
| applicationClient | create/delete/update/detail/pageList + secrets(GET)/secrets(POST)/secrets/:secretId(DELETE) |
| application | create/delete/update/detail/pageList |
| domain | create/update/detail/pageList/delete |
| tenantApplication | create/delete/update/detail/pageList |

### 2.3 tenantadmin 应用（`/v1/iam`）

| 模块 | 接口 |
|------|------|
| organization | create/delete/update/detail/pageList |
| organizationRole | create/delete/update/detail/pageList |
| organizationUser | create/delete/pageList |
| organizationRoleUser | create/delete/pageList |

## 3. 审计发现的问题与处置

### 3.1 契约不一致（前端 ↔ 后端，已修）

| # | 问题 | 处置 |
|---|------|------|
| 1 | 前端 `oauthClient.ts` 全量使用 `oauthClientId`，后端 DTO 为 `applicationClientId`（create/update/delete/detail/secrets），导致 OAuth 客户端 CRUD 与密钥管理全部不可用；删除密钥路径前端为 `/oauthClient/secrets/:id`，后端为 `/applicationClient/secrets/:id` | 前端统一改为 `applicationClientId` + 正确路径 |
| 2 | 前端 `department.ts` 调用 `/department/list`，后端无此路由（有 `pageList`/`tree`） | 前端改用 `/department/tree` |
| 3 | 前端 `application.ts`/`role.ts` 类型用 `id`，后端返回 `appId`/`roleID`；表格 ID 列永远为空 | 前端类型与字段对齐 |
| 4 | 前端 `tenant.ts` 创建/编辑传 `type`（customer/platform），后端 `TenantBaseInfo` 缺 `type` 字段，静默丢弃 | 后端 `objtenant.TenantBaseInfo` 增加 `Type`，create/update/detail/pageList 全链路透传 |
| 5 | 前端 `user.ts` 期望 `id/username/email/phone`，后端用户详情/列表只返回 `userID/name/avatar/...`（person 信息未关联），用户名/邮箱/手机号为空 | 后端 user 服务关联 person 信息（见 3.2-#2） |
| 6 | `dtoapplicationclient.DetailReq` 只有 `json` tag 无 `form` tag，查询绑定靠字段名大小写不敏感兜底，规范上应显式声明 | 补 `form` tag |
| 7 | 角色 users 系列接口用 `roleId`（小写 i），其余角色接口用 `roleID`，命名不一致 | 保持后端现状，前端按后端实际字段调用，并在文档标注 |

### 3.2 后端实现问题（缺失/不完整，已修/待修）

| # | 问题 | 影响 | 处置 |
|---|------|------|------|
| 1 | `svcuser.UpdatePassword` 只更新 `updated_by`，未真正修改密码 | 管理员改密无效 | 改为更新 person 密码（校验原密码逻辑由前端页面承担） |
| 2 | `svcuser.Create` 不创建 person、不落用户名/邮箱/手机号/密码，创建出的"用户"无法登录 | 用户管理无法真正开通账号 | Create 支持两种模式：指定 `personID` 关联已有自然人；或提供 username+password 时自动创建 person（复用 auth 注册逻辑） |
| 3 | `svcuser.Detail`/`PageList` 不返回 person 的 username/email/phone/avatar | 用户列表/详情信息缺失 | 关联查询 person 并填充 `UserBaseInfo` |
| 4 | role/menu/resource/scope 的 Create 未从认证上下文注入当前租户，前端不传 `tenantID` 时创建到 0 租户，随后不可见 | 平台管理建角色/菜单等"创建成功但看不到" | 服务端注入 `gincontext.GetTenantID(ctx)` |
| 5 | `user_identity` 相关接口存在但无前端页面 | 用户详情页第三方身份管理未覆盖 | 前端用户详情页补齐身份 Tab |

### 3.3 建议的前端最小接口集（页面 → 接口）

| 前端页面 | 依赖接口 | 现状 |
|----------|----------|------|
| 登录门户 login-web | /oidc/login、/oidc/login/selectTenant | ✅ 已实现 |
| 仪表盘 | 各模块 pageList（统计） | 页面已有，统计待完善 |
| 用户管理 | user 全套 + identity + loginLog + department 关联 + role users | 部分 |
| 角色管理 | role 全套 + role/users + role/assignUsers + role/users/:roleId/:userId | 部分 |
| 部门管理 | department/tree + create/update/delete | 部分 |
| 应用管理 | application 全套 | 部分 |
| OAuth 客户端 | applicationClient 全套 + secrets | 部分（契约已修） |
| 租户管理 | tenant 全套 | ✅ |
| 租户应用 | tenantApplication 全套 | ✅ |
| API Key | apiKey/create/pageList/revoke/delete | ❌ 前端缺失 |
| 菜单管理 | menu/tree + create/update/delete | ❌ 前端缺失 |
| 权限域/资源 | scope/resource 全套 | ❌ 前端缺失 |
| 域名管理 | domain 全套 | ❌ 前端缺失 |
| 系统配置 | system 全套 | ❌ 前端缺失 |
| 审计日志 | log/pageList | ❌ 前端缺失 |
| 个人中心/会话 | person/detail、person/updatePassword、user/sessions | ✅ 已实现 |
| 租户切换 | auth/myTenants | ✅ 已实现 |
| 组织管理（租户自服务） | organization 全套 | ✅ |
| 组织角色（租户自服务） | organizationRole 全套 | ✅ |
| 组织用户（租户自服务） | organizationUser 全套 | ✅ |
| 组织角色用户（租户自服务） | organizationRoleUser 全套 | ✅ |

## 4. 后续

前端完善按上述"最小接口集"推进；后端修复按 3.1 / 3.2 执行；`gateway` 不新增任何业务接口。
