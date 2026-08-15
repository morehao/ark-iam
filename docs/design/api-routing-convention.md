# IAM API 路由规范（v2：资源式 + 动作式混合）

> 状态：✅ 规范定稿并已随代码改造落地（执行记录见 [api-route-migration.md](api-route-migration.md)）
> 适用范围：`auth` / `platformadmin` / `tenantadmin` 三应用的全部 `/v1` 管理 API
> 不在范围内：`/oidc/*` 标准协议端点、back-channel logout、swagger docs（见 §9）
> 本文档取代：AGENTS.md「API 路由规范」章节、iam-design.md §4.2 中的旧（v1 动作式）路由描述

---

## 1. 背景与目标

v1 时代路由约定为 `/{version}/{service}/{module}/{operation}` 的动作式（RPC）风格，演进中出现三类问题：

1. **两套风格并存**：约 85% 路由为动作式（`POST /v1/platform/user/delete`），少数路由为 REST 资源式（`DELETE /v1/auth/user/sessions/{sessionID}`、`GET/POST /v1/platform/applicationClient/secrets`），方法语义分裂（同是删除，POST 与 DELETE 混用）。
2. **动作命名失控**：`/user` 模块 18 条路由拍平，动词与实体顺序不一（`createUserIdentity` vs `getUserIdentityByUser` vs `detailUserIdentity`，语义重叠）。
3. **HTTP 方法丧失语义**：全项目无 PUT/PATCH，创建/更新/删除全 POST。

本规范采用**规则化混合**（REST 资源式为主 + 显式动作式补充），与主流 API 设计一致（Google AIP、GitHub、Stripe、Keycloak 管理 API 均为该模式）。目标是让每条路由的形态由规则唯一确定，不再依赖个人发挥。

---

## 2. 设计原则：规则化混合（R1 / R2 / R3）

**R1 — 资源 CRUD → REST 资源式**

资源（名词）的增删改查、列表、详情、树，一律用「集合 + 方法 + ID」表达：

```
GET/POST    /v1/{service}/{resource}
GET/PUT/DELETE/PATCH /v1/{service}/{resource}/{id}
```

**R2 — 业务动作 → 显式动作**

状态流转、触发副作用、认证会话类操作，用动作子路径（Stripe/GitHub 风格），或认证域动作式专用段：

```
POST /v1/{service}/{resource}/{id}/action      # 单体动作
POST /v1/{service}/{resource}/action           # 集合动作（如创建类动作）
POST /v1/auth/{action}                         # 认证域动作（register/logout 等）
```

> 说明：原设计采用 AIP 的 `:action` 冒号后缀（`/resource/{id}:action`），但 **Gin 路由不支持**——`tree.go` 的 `findWildcard` 对同一路径段内的第二个 `:` 直接 panic（"only one wildcard per path segment is allowed"）。故落地为**斜杠式动作子路径**，语义等价（均为显式动作，非资源 CRUD）。

**R3 — 标准协议 → 专用前缀**

OIDC 协议端点、back-channel logout、swagger docs 使用专用前缀，不进入业务路由规范：

```
/oidc/*、/oidc/bc-logout/*、/{appName}/*（dev 环境 docs）
```

**新接口判定流程**（新增路由时必须按序判断，评审以此为准）：

```
1. 是标准协议端点？              → R3：/oidc/* 等专用前缀
2. 是某资源的增删改查/列表/详情/树？ → R1：REST 资源式
3. 是对单个资源的业务动作？        → R2：POST /资源/{id}/动作
4. 是认证/会话类动作？            → R2：/v1/auth 动作式专用段
```

---

## 3. 路径骨架

```
/v1/{service}/{resource}[/{id}[/{sub-resource}[/{id}]]]     # R1 资源式
/v1/{service}/{resource}(/{id})/{action}                    # R2 动作式（斜杠式动作子路径）
```

- **service（服务标识段）**：`auth`（auth 应用）、`platform`（platformadmin）、`tenant`（tenantadmin）。模块可跨应用同名，由 service 段区分归属。
- **resource（资源名）**：复数 + kebab-case（见 §4）。
- **id（路径参数）**：`{userID}` 等，命名规则见 §4.2。
- **层级限制**：集合层级 ≤ 3（即路径段数 ≤ 6：`v1 / service / collection / {id} / subcollection / {id}`）。子资源不得超过 2 层集合。这是对旧「5 层限制」的替换——旧限制导致身份/日志/部门被拍平进 `/user` 前缀，本规范以「集合层级上限」替代「路径段数上限」，兼顾可读性与深度。

**示例**：

```
GET    /v1/platform/users                          # 用户分页列表
GET    /v1/platform/users/{userID}                 # 用户详情
POST   /v1/platform/users                          # 创建用户
PUT    /v1/platform/users/{userID}                 # 更新用户
DELETE /v1/platform/users/{userID}                 # 删除用户
PATCH  /v1/platform/users/{userID}                 # 局部更新（如状态）
GET    /v1/platform/users/{userID}/identities      # 用户的身份列表（子资源）
POST   /v1/platform/api-keys/{apiKeyID}/revoke     # 动作
GET    /v1/auth/me                                 # 当前用户（me 资源）
POST   /v1/auth/register                           # 认证动作（动作式保留）
```

---

## 4. 资源命名

### 4.1 资源名：复数 + kebab-case

旧模块名（驼峰/单数）与新资源名对照：

| 旧模块名 | 新资源名 | 说明 |
|---|---|---|
| `user` | `users` | 用户 |
| `role` | `roles` | 角色 |
| `menu` | `menus` | 菜单 |
| `department` | `departments` | 部门 |
| `tenant` | `tenants` | 租户 |
| `system` | `systems` | 系统配置 |
| `log` | `logs` | 审计日志（只读） |
| `application` | `applications` | 应用 |
| `applicationClient` | `application-clients` | OAuth 客户端 |
| `tenantApplication` | `tenant-applications` | 租户应用 |
| `apiKey` | `api-keys` | API Key |
| `domain` | `domains` | 域名 |
| `scope` | `scopes` | 权限域 |
| `resource` | `resources` | 资源 |
| `connector` | `connectors` | 连接器（auth） |
| `organization` | `organizations` | 组织（tenantadmin） |
| `organizationRole` | `organization-roles` | 组织角色（tenantadmin） |
| `organizationUser` | `organization-users`（顶层，复合键 `{organizationID}/{userID}`） | 组织成员关联（跨组织管理场景，见 §6.2 注） |
| `organizationRoleUser` | `organization-role-users`（顶层，复合键 `{organizationRoleID}/{userID}`） | 组织角色成员关联（同上） |
| `myMenu` | `menus`（`/v1/tenant/menus/tree`） | 租户菜单（正名，service 段已区分归属） |
| `userIdentity` | `users/{userID}/identities` + 顶层 `user-identities` | 用户身份 |
| `userLoginLog` | `users/{userID}/login-logs` + 顶层 `login-logs` | 登录日志 |
| `userDepartment` | `users/{userID}/departments` | 用户-部门关联 |
| `roleMenu` | `roles/{roleID}/menus` | 角色-菜单关联 |
| `roleScope` | `roles/{roleID}/scopes` | 角色-权限域关联 |
| `userRole` | `users/{userID}/roles` | 用户-角色关联（与 `roles/{roleID}/users` 同一关联两端） |
| `person` | `me`（`/v1/auth/me`） | 当前登录人 |
| `user/sessions` | `me/sessions` | 当前登录人会话 |

### 4.2 ID 路径参数命名

- 一律 `{xxxID}` 全大写 ID，与 DTO Go 字段一致（沿用 AGENTS.md 规则）：`{userID}`、`{roleID}`、`{menuID}`、`{tenantID}`、`{appID}`、`{applicationClientID}`、`{tenantAppID}`、`{apiKeyID}`、`{domainID}`、`{systemID}`、`{logID}`、`{connectorID}`、`{secretID}`、`{identityID}`、`{loginLogID}`、`{departmentID}`、`{scopeID}`、`{resourceID}`、`{organizationID}`、`{organizationRoleID}`。
- 关联子资源 ID 用主键名（`{identityID}`、`{secretID}`），不使用父 ID 拼装。
- **swagger 注解路径参数必须与路由 path 参数完全同步**（`@Param xxxID path ...`）。
- **path 是 ID 的唯一来源**：凡带 `uri:"xxxID"` 的字段，DTO tag 一律 `json:"-" uri:"xxxID" binding:"required"`，禁止再挂 `json:"xxxID"`/`form:"xxxID"`——否则 body/query 可在 JSON/Query 绑定阶段覆盖 path 值（参数污染/IDOR 风险）。绑定顺序为 `gincontext.BindPathParams`（免校验映射）→ `ShouldBindJSON/ShouldBindQuery`（全量校验），最终校验仍能覆盖 path 字段。

### 4.3 命名细则

- 资源一律复数；`me` 例外（固定单数，表示当前认证主体）。
- 复合资源用 kebab-case 连字符（`application-clients`），禁止驼峰（`applicationClient`）。
- 只读资源（`logs`、`connector-factories`）不提供写方法，但仍用复数名词表达。

---

## 5. HTTP 方法与语义

| 方法 | 语义 | 使用场景 | 幂等 |
|---|---|---|---|
| `GET` | 只读查询 | 列表、详情、树、me、OAuth 回调 | ✅ |
| `POST` | 创建 / 触发动作 | 创建资源；动作子路径（`/id/action`） | ❌ |
| `PUT` | 全量更新 | 表单式更新（沿用现状 update DTO 语义）；批量授权（全量替换集合） | ✅ |
| `PATCH` | 局部更新 | 单字段/部分字段更新（如 `status`） | ✅ |
| `DELETE` | 删除 | 删除资源、解除关联 | ✅ |

约定：

- **删除一律用 `DELETE`**，禁止 `POST /xxx/delete`。
- **更新一律用 `PUT`**（现状 `update` DTO 即"提交表单更新"语义），**禁止 `POST /xxx/update`**。
- **局部字段更新用 `PATCH`**（如 `PATCH /users/{userID}` 更新 `isSuspended`）。
- **查询一律 `GET`**，禁止 `POST /xxx/pageList`（含分页列表）。
- 响应统一走 `gincontext.Success/Fail`（HTTP 200 + 业务 code）。REST 语义上的 201/204 等状态码作为可选演进项，本规范不强制（与现有网关/前端拦截逻辑兼容优先）。

---

## 6. 子资源与关联建模

### 6.1 从属资源 → 子资源

生命周期/语义上从属于父资源的，建模为子资源：

```
GET    /v1/platform/users/{userID}/identities
POST   /v1/platform/users/{userID}/identities
GET    /v1/platform/users/{userID}/identities/{identityID}
PUT    /v1/platform/users/{userID}/identities/{identityID}
DELETE /v1/platform/users/{userID}/identities/{identityID}

GET    /v1/platform/application-clients/{applicationClientID}/secrets
POST   /v1/platform/application-clients/{applicationClientID}/secrets
DELETE /v1/platform/application-clients/{applicationClientID}/secrets/{secretID}
```

### 6.2 多对多关联 → 父资源子集合 + 两端视角

```
/v1/platform/roles/{roleID}/users          ← 角色的用户（GET 列表 / PUT 全量替换 / DELETE {userID} 移除）
/v1/platform/users/{userID}/roles          ← 用户的角色（GET 列表 / POST 分配 / DELETE 移除）
```

- 同一关联只允许两端视角，禁止重复端点（如现状 `role/assignUsers` 与 `userRole/create` 表达同一操作）。
- **批量授权 = 集合全量替换**：`PUT /roles/{roleID}/users`（body 带 `userIDs: []`），幂等、语义即"设置为该集合"。
- 组织侧同理。**例外**：`organization-users` / `organization-role-users` 为**顶层资源**（跨组织管理页按 org/user/role 过滤全量查询，复合键进 path），不建模为组织子资源：

```
GET/POST  /v1/tenant/organization-users
DELETE    /v1/tenant/organization-users/{organizationID}/{userID}
GET/POST  /v1/tenant/organization-role-users
DELETE    /v1/tenant/organization-role-users/{organizationRoleID}/{userID}
```

> 说明：organizationRole 本应从属于 organization（`/organizations/{id}/roles`），但会突破集合层级 ≤ 3 的限制（3 层集合 + 子集 = 7 段），故 organization-roles 保持顶层资源，organizationID 在 body 中冗余携带。

### 6.3 全局检索 → 顶层只读资源

子资源之外的"跨父资源检索"场景，保留为顶层只读资源：

```
GET /v1/platform/user-identities    # 全局身份检索（按 issuer/identityID 过滤，替代 pageListUserIdentity）
GET /v1/platform/login-logs         # 全局登录日志检索（替代 pageListUserLoginLog）
GET /v1/platform/login-logs/{loginLogID}
```

用户视角列表仍走子资源（`GET /users/{userID}/identities`、`GET /users/{userID}/login-logs`）。

---

## 7. 动作（R2）

### 7.1 动作子路径（原 `:action` 设计）

对单个资源的业务动作：

```
POST /v1/auth/connectors/{connectorID}/test
POST /v1/auth/connectors/{connectorID}/authorize
POST /v1/platform/api-keys/{apiKeyID}/revoke
POST /v1/auth/me/changePassword
POST /v1/platform/users/{userID}/changePassword
```

集合动作（创建时带特殊语义）：

```
POST /v1/platform/tenants/createAsOwner
```

### 7.2 认证域动作（专用段）

认证动作保持动作式，不强行资源化（与 Keycloak `/protocol/openid-connect/*`、Auth0 `/oauth/*` 等主流一致）：

```
POST /v1/auth/register
POST /v1/auth/joinTenant
POST /v1/auth/logout
POST /v1/auth/logoutAll
GET  /v1/auth/userinfo          # 与 /oidc/userinfo 协议端点对应的业务封装，保留命名
```

### 7.3 查询变体

树、工厂目录等"查询变体"允许保留为子路径（只读）：

```
GET /v1/platform/menus/tree
GET /v1/platform/departments/tree
GET /v1/tenant/menus/tree
GET /v1/auth/connector-factories        # 工厂目录（原 getFactoryList，POST→GET，query 过滤）
```

---

## 8. 查询与分页

- 列表一律 `GET` + query 参数：`page`、`pageSize`（沿用 `gobject.PageQuery`），筛选字段沿用现有 DTO JSON tag（如 `tenantID`、`name`、`username`、`issuer`）。
- 排序、时间范围等后续扩展追加 query 参数；筛选条件过复杂时（本阶段不引入），可扩展 `POST /v1/{service}/{resource}:search`（AIP-136 命名）。
- `GET /tree` 返回树形结构，不走分页。

---

## 9. 认证域与协议端点边界

| 面 | 路径 | 规范 |
|---|---|---|
| 认证/注册/SSO 动作 | `/v1/auth/register`、`/joinTenant`、`/logout`、`/logoutAll` | 动作式保留（§7.2） |
| 当前用户资源 | `/v1/auth/me`、`/v1/auth/me/tenants`、`/v1/auth/me/sessions`、`/v1/auth/me/changePassword` | R1 资源式（me 固定单数） |
| 业务封装 | `/v1/auth/userinfo` | 保留现状（OIDC 命名习惯） |
| 当前用户子资源 | `/v1/auth/me/tenants`（原 `myTenants`） | me 化（已确认）；`userinfo` 因对应 `/oidc/userinfo` 协议端点保留原名 |
| OIDC 协议端点 | `/oidc/*`（含 `.well-known/*`、`/oidc/login*`） | 不随应用分段，不进本规范 |
| back-channel logout | `/oidc/bc-logout/*` | 协议路径，不动 |
| swagger docs | `/{appName}/*`（dev） | 不动 |

---

## 10. 与其他规范的衔接

- **ID 全大写**：路径参数、JSON tag、Go 字段一律 `xxxID`（AGENTS.md），禁止小写 d。
- **swagger 注解**：`@Router` 与路由注册完全一致；path 参数用 `@Param xxxID path true "..."`。
- **DTO 命名**：`<业务名词><动词>Req/Resp` 不变（路由改造不涉及 DTO 重命名）。
- **controller/service 签名**：路由改造只改注册行与绑定方式（body→path/query），方法签名与 DTO 字段保持兼容（ID 字段从 body/query 移入 path 时，controller 内改用 `ctx.Param`/`ctx.ShouldBindUri` 获取并回填到 DTO）。
- **gocli 模板**：`router.go.tpl` 同步更新为资源式注册模板，后续生成代码自动遵循本规范。
- **错误码**：不随路由调整，按领域分段不变。

---

## 11. 附录：端点形态速查（示例）

```
# platformadmin（R1 资源式）
POST   /v1/platform/users
GET    /v1/platform/users?page=&pageSize=&name=
GET    /v1/platform/users/{userID}
PUT    /v1/platform/users/{userID}
PATCH  /v1/platform/users/{userID}
DELETE /v1/platform/users/{userID}
POST   /v1/platform/users/{userID}/changePassword
GET    /v1/platform/users/{userID}/identities
GET    /v1/platform/users/{userID}/login-logs
GET    /v1/platform/users/{userID}/departments
PUT    /v1/platform/users/{userID}/departments
GET    /v1/platform/roles/{roleID}/users
PUT    /v1/platform/roles/{roleID}/users
DELETE /v1/platform/roles/{roleID}/users/{userID}

# auth（认证动作式 + me 资源）
POST   /v1/auth/register
POST   /v1/auth/logout
GET    /v1/auth/me
GET    /v1/auth/me/tenants
GET    /v1/auth/me/sessions
DELETE /v1/auth/me/sessions/{sessionID}
POST   /v1/auth/connectors/{connectorID}/test

# tenantadmin
POST   /v1/tenant/organizations
GET    /v1/tenant/organization-users
DELETE /v1/tenant/organization-users/{organizationID}/{userID}
GET    /v1/tenant/menus/tree
```

完整旧→新对照见 [api-route-migration.md](api-route-migration.md)。
