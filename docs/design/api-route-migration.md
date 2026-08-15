# IAM API 路由改造方案（v1 动作式 → v2 资源式 + 动作式混合）

> 状态：✅ **已执行完毕**（后端三应用 + 前端 api 包 + gocli 模板全部落地，build/test/tsc 通过）
> 前置条件：先评审 [api-routing-convention.md](api-routing-convention.md)（本方案的规范依据）
> 影响面：`/v1` 管理 API 全部约 **134 条**路由（platformadmin 97 / auth 20 / tenantadmin 17），**破坏性变更**（路径与 HTTP 方法均变化）。项目处于开发期、无外部 API 消费者、前端 api 封装集中，是切换的最佳窗口。
>
> **执行记录**（2026-08）：路径绑定统一使用 golib `biz/gcontext/gincontext.BindPathParams`（gin 原生 `ShouldBindUri` 会对"path ID + body 必填字段"结构体提前全量校验，故用免校验映射替代；该函数与 gin 的 uri 绑定共用 `binding.MapFormWithTag`，已随代码迁入 golib，本地开发经 `backend/go.work` replace 指向 golib 源码）；`organization-users` / `organization-role-users` 落地为顶层资源（前端为跨组织管理页）；gocli 模板新增 `toKebabCase`/`pluralize` 模板函数并更新 router/controller/request 模板；三应用各新增 `router_smoke_test.go` 全量注册冒烟测试。**安全加固**：全部 `uri:"xxxID"` 字段统一 `json:"-"`（并移除 form tag），保证 path 为 ID 唯一来源，杜绝 body/query 在 JSON/Query 绑定阶段覆盖 path 值（参数污染/IDOR）；规则已写入规范文档 §4.2 与 AGENTS.md。

---

## 1. 背景

v1 路由约定为动作式（`/v1/{service}/{module}/{operation}`），实际演进中出现两套风格并存（动作式为主 + 少数 REST）、`/user` 模块动作命名失控、HTTP 方法丧失语义（无 PUT/PATCH、删除也用 POST）等问题。详见规范文档 §1 与设计盘点结论。

本次按 **规则化混合**（R1 资源 CRUD 走 REST / R2 业务动作走动作子路径（`/id/action`） / R3 协议端点不动）统一全部路由。

## 2. 迁移原则（不变量）

| 层 | 是否改动 | 说明 |
|---|---|---|
| controller / service / DTO / model / dao | ❌ 不动 | 方法签名与 DTO 字段保持兼容 |
| controller 绑定方式 | ⚠️ 微调 | 原 body/query 中的 ID 字段移入 path 时，controller 内改用 `ctx.Param` / `ctx.ShouldBindUri` 获取并回填 DTO（每处改动极小，不涉及业务逻辑） |
| router 注册 | ✅ 重写 | 全部路由路径 + HTTP 方法 |
| swagger 注解 | ✅ 同步 | `@Router`、`@Param`（body→path/query）约 130+ 处 |
| 前端 `packages/api` | ✅ 只改 URL | **函数名与参数签名保持不变**，仅替换内部 URL 与方法 → 页面组件零改动 |
| 前端页面组件 | ❌ 不动 | 经 api 包隔离 |
| gocli 模板 `router.go.tpl` | ✅ 更新 | 后续生成代码遵循新规范 |
| `/oidc/*`、back-channel logout、docs | ❌ 不动 | 协议路径 |

**关键点**：迁移后前端调用形态几乎不变（如 `getUserDetail(userID)` 仍传 userID，只是内部 URL 从 `GET /user/detail?userID=` 变为 `GET /users/{userID}`）；后端 controller 方法内把 `userID` 从 path 取回填 DTO 即可。

---

## 3. 完整对照表

> 约定：`←` 表示"由旧路由迁移而来"；「参数变化」列描述 ID 类参数的传递方式变化（body/query → path）。

### 3.1 platformadmin（97 条）

#### 3.1.1 tenant（6 条）

| 旧 | 新 | 参数变化 |
|---|---|---|
| `POST /v1/platform/tenant/create` | `POST /v1/platform/tenants` | — |
| `POST /v1/platform/tenant/createAsOwner` | `POST /v1/platform/tenants/createAsOwner` | — |
| `POST /v1/platform/tenant/delete` | `DELETE /v1/platform/tenants/{tenantID}` | tenantID: body → path |
| `POST /v1/platform/tenant/update` | `PUT /v1/platform/tenants/{tenantID}` | tenantID: body → path |
| `GET /v1/platform/tenant/detail` | `GET /v1/platform/tenants/{tenantID}` | tenantID: query → path |
| `POST /v1/platform/tenant/pageList` | `GET /v1/platform/tenants` | body → query（page/pageSize/筛选） |

#### 3.1.2 department（6 条）

| 旧 | 新 | 参数变化 |
|---|---|---|
| `POST /v1/platform/department/create` | `POST /v1/platform/departments` | — |
| `POST /v1/platform/department/delete` | `DELETE /v1/platform/departments/{departmentID}` | body → path |
| `POST /v1/platform/department/update` | `PUT /v1/platform/departments/{departmentID}` | body → path |
| `GET /v1/platform/department/detail` | `GET /v1/platform/departments/{departmentID}` | query → path |
| `POST /v1/platform/department/pageList` | `GET /v1/platform/departments` | body → query |
| `GET /v1/platform/department/tree` | `GET /v1/platform/departments/tree` | — |

#### 3.1.3 system（5 条）

| 旧 | 新 | 参数变化 |
|---|---|---|
| `POST /v1/platform/system/create` | `POST /v1/platform/systems` | — |
| `POST /v1/platform/system/delete` | `DELETE /v1/platform/systems/{systemID}` | body → path |
| `POST /v1/platform/system/update` | `PUT /v1/platform/systems/{systemID}` | body → path |
| `GET /v1/platform/system/detail` | `GET /v1/platform/systems/{systemID}` | query → path |
| `POST /v1/platform/system/pageList` | `GET /v1/platform/systems` | body → query |

#### 3.1.4 log（2 条）

| 旧 | 新 | 参数变化 |
|---|---|---|
| `GET /v1/platform/log/detail` | `GET /v1/platform/logs/{logID}` | query → path |
| `POST /v1/platform/log/pageList` | `GET /v1/platform/logs` | body → query |

#### 3.1.5 apiKey（4 条）

| 旧 | 新 | 参数变化 |
|---|---|---|
| `POST /v1/platform/apiKey/create` | `POST /v1/platform/api-keys` | — |
| `POST /v1/platform/apiKey/pageList` | `GET /v1/platform/api-keys` | body → query |
| `POST /v1/platform/apiKey/revoke` | `POST /v1/platform/api-keys/{apiKeyID}/revoke` | id: body → path |
| `POST /v1/platform/apiKey/delete` | `DELETE /v1/platform/api-keys/{apiKeyID}` | id: body → path |

#### 3.1.6 user（18 条）

| 旧 | 新 | 参数变化 |
|---|---|---|
| `POST /v1/platform/user/create` | `POST /v1/platform/users` | — |
| `POST /v1/platform/user/delete` | `DELETE /v1/platform/users/{userID}` | userID: body → path |
| `POST /v1/platform/user/update` | `PUT /v1/platform/users/{userID}` | userID: body → path |
| `GET /v1/platform/user/detail` | `GET /v1/platform/users/{userID}` | userID: query → path |
| `POST /v1/platform/user/pageList` | `GET /v1/platform/users` | body → query |
| `POST /v1/platform/user/updatePassword` | `POST /v1/platform/users/{userID}/changePassword` | userID: body → path |
| `POST /v1/platform/user/updateStatus` | `PATCH /v1/platform/users/{userID}` | userID: body → path；body 只带 `isSuspended` |
| `POST /v1/platform/user/createUserIdentity` | `POST /v1/platform/users/{userID}/identities` | userID: body → path；tenantID/issuer/identityID 留 body |
| `POST /v1/platform/user/deleteUserIdentity` | `DELETE /v1/platform/users/{userID}/identities/{identityID}` | identityID: body → path；userID 需在 path（controller 由 identityID 反查或前端回传，见 §7 风险） |
| `POST /v1/platform/user/updateUserIdentity` | `PUT /v1/platform/users/{userID}/identities/{identityID}` | identityID/userID: body → path |
| `GET /v1/platform/user/detailUserIdentity` | `GET /v1/platform/users/{userID}/identities/{identityID}` | query → path |
| `POST /v1/platform/user/pageListUserIdentity` | `GET /v1/platform/user-identities`（全局检索，子资源列表另见下条） | body → query |
| `GET /v1/platform/user/getUserIdentityByUser` | `GET /v1/platform/users/{userID}/identities` | userID: query → path |
| `GET /v1/platform/user/detailUserLoginLog` | `GET /v1/platform/login-logs/{loginLogID}` | query → path |
| `POST /v1/platform/user/pageListUserLoginLog` | `GET /v1/platform/login-logs`（全局检索） | body → query |
| `GET /v1/platform/user/getUserLoginLogByUser` | `GET /v1/platform/users/{userID}/login-logs` | userID: query → path |
| `GET /v1/platform/user/getUserDepartmentByUser` | `GET /v1/platform/users/{userID}/departments` | userID: query → path |
| `POST /v1/platform/user/assignDepartments` | `PUT /v1/platform/users/{userID}/departments` | userID: body → path；body 只带 `departmentIDs`（全量替换） |

> 注：`pageListUserIdentity` / `pageListUserLoginLog` 保留为顶层只读检索（管理员跨用户按 issuer/identityID 查身份、查登录日志）；用户视角列表走子资源。若评审确认顶层检索非必需，可裁剪为仅子资源，减少端点。

#### 3.1.7 role（8 条）

| 旧 | 新 | 参数变化 |
|---|---|---|
| `POST /v1/platform/role/create` | `POST /v1/platform/roles` | — |
| `POST /v1/platform/role/delete` | `DELETE /v1/platform/roles/{roleID}` | body → path |
| `POST /v1/platform/role/update` | `PUT /v1/platform/roles/{roleID}` | body → path |
| `GET /v1/platform/role/detail` | `GET /v1/platform/roles/{roleID}` | query → path |
| `POST /v1/platform/role/pageList` | `GET /v1/platform/roles` | body → query |
| `GET /v1/platform/role/users` | `GET /v1/platform/roles/{roleID}/users` | roleID: query → path |
| `POST /v1/platform/role/assignUsers` | `PUT /v1/platform/roles/{roleID}/users` | roleID: body → path；body 只带 `userIDs`（全量替换） |
| `DELETE /v1/platform/role/users/:roleID/:userID` | `DELETE /v1/platform/roles/{roleID}/users/{userID}` | 已走 path，仅路径改名 |

#### 3.1.8 menu（6 条）

| 旧 | 新 | 参数变化 |
|---|---|---|
| `POST /v1/platform/menu/create` | `POST /v1/platform/menus` | — |
| `POST /v1/platform/menu/delete` | `DELETE /v1/platform/menus/{menuID}` | body → path |
| `POST /v1/platform/menu/update` | `PUT /v1/platform/menus/{menuID}` | body → path |
| `GET /v1/platform/menu/detail` | `GET /v1/platform/menus/{menuID}` | query → path |
| `POST /v1/platform/menu/pageList` | `GET /v1/platform/menus` | body → query |
| `GET /v1/platform/menu/tree` | `GET /v1/platform/menus/tree` | — |

#### 3.1.9 scope / resource（各 5 条）

| 旧 | 新 |
|---|---|
| `POST /v1/platform/scope/create` | `POST /v1/platform/scopes` |
| `POST /v1/platform/scope/delete` | `DELETE /v1/platform/scopes/{scopeID}` |
| `POST /v1/platform/scope/update` | `PUT /v1/platform/scopes/{scopeID}` |
| `GET /v1/platform/scope/detail` | `GET /v1/platform/scopes/{scopeID}` |
| `POST /v1/platform/scope/pageList` | `GET /v1/platform/scopes` |
| `POST /v1/platform/resource/create` | `POST /v1/platform/resources` |
| `POST /v1/platform/resource/delete` | `DELETE /v1/platform/resources/{resourceID}` |
| `POST /v1/platform/resource/update` | `PUT /v1/platform/resources/{resourceID}` |
| `GET /v1/platform/resource/detail` | `GET /v1/platform/resources/{resourceID}` |
| `POST /v1/platform/resource/pageList` | `GET /v1/platform/resources` |

#### 3.1.10 关联模块 roleMenu / roleScope / userRole（各 3 条）

| 旧 | 新 | 说明 |
|---|---|---|
| `POST /v1/platform/roleMenu/create` | `POST /v1/platform/roles/{roleID}/menus` | 子资源化 |
| `POST /v1/platform/roleMenu/delete` | `DELETE /v1/platform/roles/{roleID}/menus/{menuID}` | 子资源化 |
| `POST /v1/platform/roleMenu/pageList` | `GET /v1/platform/roles/{roleID}/menus` | 子资源化 |
| `POST /v1/platform/roleScope/create` | `POST /v1/platform/roles/{roleID}/scopes` | 子资源化 |
| `POST /v1/platform/roleScope/delete` | `DELETE /v1/platform/roles/{roleID}/scopes/{scopeID}` | 子资源化 |
| `POST /v1/platform/roleScope/pageList` | `GET /v1/platform/roles/{roleID}/scopes` | 子资源化 |
| `POST /v1/platform/userRole/create` | `POST /v1/platform/users/{userID}/roles` | 子资源化（与 role/users 同一关联） |
| `POST /v1/platform/userRole/delete` | `DELETE /v1/platform/users/{userID}/roles/{roleID}` | 子资源化 |
| `POST /v1/platform/userRole/pageList` | `GET /v1/platform/users/{userID}/roles` | 子资源化 |

#### 3.1.11 applicationClient（8 条）

| 旧 | 新 | 参数变化 |
|---|---|---|
| `POST /v1/platform/applicationClient/create` | `POST /v1/platform/application-clients` | — |
| `POST /v1/platform/applicationClient/delete` | `DELETE /v1/platform/application-clients/{applicationClientID}` | body → path |
| `POST /v1/platform/applicationClient/update` | `PUT /v1/platform/application-clients/{applicationClientID}` | body → path |
| `GET /v1/platform/applicationClient/detail` | `GET /v1/platform/application-clients/{applicationClientID}` | query → path |
| `POST /v1/platform/applicationClient/pageList` | `GET /v1/platform/application-clients` | body → query |
| `GET /v1/platform/applicationClient/secrets` | `GET /v1/platform/application-clients/{applicationClientID}/secrets` | applicationClientID: query → path |
| `POST /v1/platform/applicationClient/secrets` | `POST /v1/platform/application-clients/{applicationClientID}/secrets` | applicationClientID: body → path |
| `DELETE /v1/platform/applicationClient/secrets/:secretID` | `DELETE /v1/platform/application-clients/{applicationClientID}/secrets/{secretID}` | 已走 path，补 applicationClientID 段 |

#### 3.1.12 application / domain / tenantApplication（各 5 条）

| 旧 | 新 |
|---|---|
| `POST /v1/platform/application/create` | `POST /v1/platform/applications` |
| `POST /v1/platform/application/delete` | `DELETE /v1/platform/applications/{appID}` |
| `POST /v1/platform/application/update` | `PUT /v1/platform/applications/{appID}` |
| `GET /v1/platform/application/detail` | `GET /v1/platform/applications/{appID}` |
| `POST /v1/platform/application/pageList` | `GET /v1/platform/applications` |
| `POST /v1/platform/domain/create` | `POST /v1/platform/domains` |
| `POST /v1/platform/domain/delete` | `DELETE /v1/platform/domains/{domainID}` |
| `POST /v1/platform/domain/update` | `PUT /v1/platform/domains/{domainID}` |
| `GET /v1/platform/domain/detail` | `GET /v1/platform/domains/{domainID}` |
| `POST /v1/platform/domain/pageList` | `GET /v1/platform/domains` |
| `POST /v1/platform/tenantApplication/create` | `POST /v1/platform/tenant-applications` |
| `POST /v1/platform/tenantApplication/delete` | `DELETE /v1/platform/tenant-applications/{tenantAppID}` |
| `POST /v1/platform/tenantApplication/update` | `PUT /v1/platform/tenant-applications/{tenantAppID}` |
| `GET /v1/platform/tenantApplication/detail` | `GET /v1/platform/tenant-applications/{tenantAppID}` |
| `POST /v1/platform/tenantApplication/pageList` | `GET /v1/platform/tenant-applications` |

> 注：`domain` 现状 delete 注册在末尾（其他模块 delete 紧随 create），迁移时统一按「create → list → tree/detail → update → delete → 动作」顺序排列，消除注册顺序不一致。

### 3.2 auth（20 条）

| 旧 | 新 | 说明 |
|---|---|---|
| `GET /v1/auth/myTenants` | `GET /v1/auth/me/tenants` | me 资源化（已确认，见 §7） |
| `POST /v1/auth/register` | `POST /v1/auth/register` | 认证动作式保留 |
| `POST /v1/auth/joinTenant` | `POST /v1/auth/joinTenant` | 认证动作式保留 |
| `POST /v1/auth/logout` | `POST /v1/auth/logout` | 认证动作式保留 |
| `POST /v1/auth/logoutAll` | `POST /v1/auth/logoutAll` | 认证动作式保留 |
| `GET /v1/auth/userinfo` | `GET /v1/auth/userinfo` | 保留（OIDC 命名习惯） |
| `GET /v1/auth/person/detail` | `GET /v1/auth/me` | person → me |
| `POST /v1/auth/person/updatePassword` | `POST /v1/auth/me/changePassword` | person → me 动作 |
| `GET /v1/auth/user/sessions` | `GET /v1/auth/me/sessions` | user → me 子资源 |
| `DELETE /v1/auth/user/sessions` | `DELETE /v1/auth/me/sessions` | user → me 子资源 |
| `DELETE /v1/auth/user/sessions/:sessionID` | `DELETE /v1/auth/me/sessions/{sessionID}` | user → me 子资源 |
| `POST /v1/auth/connector/create` | `POST /v1/auth/connectors` | 复数化 |
| `POST /v1/auth/connector/delete` | `DELETE /v1/auth/connectors/{connectorID}` | body → path |
| `POST /v1/auth/connector/update` | `PUT /v1/auth/connectors/{connectorID}` | body → path |
| `GET /v1/auth/connector/detail` | `GET /v1/auth/connectors/{connectorID}` | query → path |
| `POST /v1/auth/connector/pageList` | `GET /v1/auth/connectors` | body → query |
| `POST /v1/auth/connector/getFactoryList` | `GET /v1/auth/connector-factories` | POST → GET；protocol/provider 移入 query |
| `POST /v1/auth/connector/:connectorID/test` | `POST /v1/auth/connectors/{connectorID}/test` | 动作化统一 |
| `POST /v1/auth/connector/:connectorID/authorize` | `POST /v1/auth/connectors/{connectorID}/authorize` | 动作化统一 |
| `GET /v1/auth/connector/callback` | `GET /v1/auth/connectors/callback` | OAuth 回调保留 |

### 3.3 tenantadmin（17 条）

| 旧 | 新 | 说明 |
|---|---|---|
| `POST /v1/tenant/organization/create` | `POST /v1/tenant/organizations` | 复数化 |
| `POST /v1/tenant/organization/delete` | `DELETE /v1/tenant/organizations/{organizationID}` | body → path |
| `POST /v1/tenant/organization/update` | `PUT /v1/tenant/organizations/{organizationID}` | body → path |
| `GET /v1/tenant/organization/detail` | `GET /v1/tenant/organizations/{organizationID}` | query → path |
| `POST /v1/tenant/organization/pageList` | `GET /v1/tenant/organizations` | body → query |
| `POST /v1/tenant/organizationRole/create` | `POST /v1/tenant/organization-roles` | 复数化 |
| `POST /v1/tenant/organizationRole/delete` | `DELETE /v1/tenant/organization-roles/{organizationRoleID}` | body → path |
| `POST /v1/tenant/organizationRole/update` | `PUT /v1/tenant/organization-roles/{organizationRoleID}` | body → path |
| `GET /v1/tenant/organizationRole/detail` | `GET /v1/tenant/organization-roles/{organizationRoleID}` | query → path |
| `POST /v1/tenant/organizationRole/pageList` | `GET /v1/tenant/organization-roles` | body → query |
| `POST /v1/tenant/organizationUser/create` | `POST /v1/tenant/organization-users` | **顶层资源化**（前端为跨组织管理页，非组织子资源视图）；organizationID/userID 留 body |
| `POST /v1/tenant/organizationUser/delete` | `DELETE /v1/tenant/organization-users/{organizationID}/{userID}` | 顶层资源；双 ID 进 path（复合键） |
| `POST /v1/tenant/organizationUser/pageList` | `GET /v1/tenant/organization-users` | 顶层资源；query 过滤（organizationID/userID 可选） |
| `POST /v1/tenant/organizationRoleUser/create` | `POST /v1/tenant/organization-role-users` | 顶层资源化；organizationID/organizationRoleID/userID 留 body |
| `POST /v1/tenant/organizationRoleUser/delete` | `DELETE /v1/tenant/organization-role-users/{organizationRoleID}/{userID}` | 顶层资源；复合键进 path |
| `POST /v1/tenant/organizationRoleUser/pageList` | `GET /v1/tenant/organization-role-users` | 顶层资源；query 过滤 |
| `GET /v1/tenant/myMenu/tree` | `GET /v1/tenant/menus/tree` | 正名（service 段已区分归属） |

---

## 4. 配套变更

| 项 | 内容 | 归属 |
|---|---|---|
| AGENTS.md | 「API 路由规范」章节替换为 v2 摘要，示例代码同步 | 文档 |
| iam-design.md | §4.2 路由前缀描述更新 | 文档 |
| docs/design/README.md | 索引补充两篇新文档 | 文档 |
| swagger 注解 | 各 controller `@Router` 同步新路径；`@Param` body/query→path | 后端 |
| gocli 模板 | `router.go.tpl` 更新为资源式注册模板（生成 create/update/delete/detail/pageList 时输出 REST 路径） | 工具链 |
| 路由注册顺序 | 各 router 文件统一「create → list → tree/detail → update → delete → 动作/子资源」 | 后端 |
| 前端 `packages/api` | `resources/platform.ts`、`resources/auth.ts`、tenant-admin-web `api/*.ts` 仅改 URL 与方法 | 前端 |
| 测试 | router 相关测试（如 auth `assert_test.go`、oidc 测试中的路由断言）同步 | 后端 |

## 5. 执行计划

| 阶段 | 内容 | 产出/验收 |
|---|---|---|
| P0 评审 | 评审本方案与规范文档 | 评审通过、定稿 |
| P1 后端 platformadmin | 重写 15 个 router 文件 + 同步 swagger 注解 + controller 绑定微调 | `make build APP=platformadmin`、`make test APP=platformadmin`、`make swag APP=platformadmin` |
| P2 后端 auth | 重写 4 个 router 文件 + 同步注解 + controller 微调 | 同上（auth） |
| P3 后端 tenantadmin | 重写 3 个 router 文件 + 同步注解 + controller 微调 | 同上（tenantadmin） |
| P4 前端 | `packages/api` + 各 web 应用 api 文件仅改 URL/方法 | 三个 web 应用编译通过、页面功能回归 |
| P5 工具链与文档 | gocli 模板更新；AGENTS.md / iam-design.md 同步（本文档已含目标内容，执行时核对） | 模板生成代码为资源式 |
| P6 回归 | gateway 聚合启动（:8100），全接口冒烟 | 路由无冲突、swagger 文档完整 |

> 每个 app 一个 PR，独立评审，避免跨应用大 PR。

## 6. 验证方式

1. `make build APP=<app>` + `go vet ./...` + `make lint`。
2. `make test APP=gateway`（含 router 测试，确认无重复路由 panic）。
3. `make swag APP=<app>` 生成后 diff 文档，核对路径/方法/参数与对照表一致。
4. 启动 gateway（:8100），curl 冒烟：`GET /v1/platform/users?page=1&pageSize=10`、`POST /v1/platform/users`、`PUT /v1/platform/users/{id}`、`DELETE /v1/platform/users/{id}`、`PATCH /v1/platform/users/{id}`、`GET /v1/platform/users/{id}/identities`、`POST /v1/platform/api-keys/{id}/revoke` 等。
5. 前端联调：登录 → 用户/角色/部门/应用/API Key/菜单各页面操作回归；auth 侧登出/会话管理/改密回归。

## 7. 风险与注意事项

> 评审结论（2026-XX 确认）：① 前端回传 userID；② 接受全量替换语义；③ `myTenants` me 化为 `/v1/auth/me/tenants`。

1. **`DELETE /users/{userID}/identities/{identityID}` 需要 userID 进 path**：现状删除身份只传 `userIdentityID`。✅ **已确认：前端回传 userID**（前端删除身份时有 user 上下文；controller 内 `ctx.ShouldBindUri` 同时取 userID 与 identityID，DTO 增加 uri 绑定 tag）。
2. **`PUT /roles/{roleID}/users`、`PUT /users/{userID}/departments` 为全量替换语义**：✅ **已确认：接受"设置为该集合"语义**（授权/分配场景即全量替换；前端调用处由"追加式"改为"提交完整集合"，页面交互不变）。
3. **`myTenants` me 化**：✅ **已确认：`GET /v1/auth/myTenants` → `GET /v1/auth/me/tenants`**（与 person/sessions 统一到 me 资源下；`userinfo` 因对应 `/oidc/userinfo` 协议端点保留原名）。前端仅改 `getMyTenants` 的 URL 一行，函数名不变。
4. **PATCH 语义首次引入**：`PATCH /users/{userID}` 只更新 `isSuspended`；gin 侧用 `ctx.ShouldBindJSON` 接收部分字段即可（DTO 复用 `UserStatusUpdateReq` 去掉 userID）。
5. **`pageList` 从 POST body 改为 GET query**：筛选参数需 `form` tag 支持（现有 DTO 多数已带 `form`，`PageQuery` 需确认含 `form:"page"`/`form:"pageSize"`；不足处补 tag，属 DTO 兼容性微调）。
6. **破坏性变更**：新旧路径不共存（开发期不提供兼容层）；前端 api 包与后端 PR 需同批次合入，避免中间态（前端 URL 已改而后端未上线）。
7. **`/oidc/*`、back-channel logout、swagger docs、`/v1/auth/userinfo` 等不在改造范围**，代码不动。
8. **gocli 模板更新后**，历史已生成代码不回改；模板只影响后续生成。
9. **集合层级 ≤ 3**（`v1/service/collection/{id}/subcollection/{id}` 为上限），后续建模不得出现 4 层集合路径。

---

## 8. 工作量预估

| 项 | 预估 |
|---|---|
| 后端 router + swagger + controller 微调 | 约 20 个文件、130+ 处注解，机械替换为主 |
| 前端 api 包 | 4 个文件（platform.ts / auth.ts / tenant organization.ts / menu.ts 等），仅 URL |
| 测试与联调 | gateway 冒烟 + 三端页面回归 |
