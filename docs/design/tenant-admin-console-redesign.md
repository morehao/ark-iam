# 管理台页面功能重规划（租户自管理 / 平台管理职责划分）

> 状态：已实施（阶段一后端 + 阶段二租户端前端 + 阶段三平台端下线均完成；`go test ./...` 全绿、`golangci-lint` 0 issues、两个前端应用构建通过）
> 涉及：`tenant-admin-web` 页面从 1 个扩展为 3 个（部门架构 / 用户管理 / 角色管理）；`platform-admin-web` 用户/角色页面调整为「平台排查视角」；`tenantadmin` 后端新增用户 / 角色接口并富化部门成员信息；种子菜单补充。

## 1. 背景与目标

### 1.1 现状问题

| 应用 | 现状 | 问题 |
|---|---|---|
| 租户自管理（tenant-admin-web） | 仅 1 个「部门架构」页（左侧树 + 右侧节点操作与成员） | 页面数严重不足，用户期望的「用户管理 / 角色管理（含授权）」缺失；成员列表只展示 `DepartmentUser` 关系字段（`userID / userName / relationType / isPrimary`），**没有租户用户基础信息**（头像 / 用户名 / 邮箱 / 手机 / 状态）——后端 `department_user` PageList 也未返回这些字段；添加成员需手输 `userID`，无用户选择器 |
| 平台管理（platform-admin-web） | 14 个静态菜单页面 | 用户管理（跨租户目录）、角色管理（实为按当前租户隔离）、菜单/权限域/资源（权限字典）与租户自管理职责重叠，平台端不聚焦 |

### 1.2 职责划分原则（本文档的契约）

> **租户自管理 = 租户内的「部门 + 人 + 权限」自服务**：部门架构（部门）、租户用户目录、租户角色与授权，全部收敛到 tenant-admin。
> **平台管理 = 平台层（跨租户）的管理与排查**：租户生命周期、应用与接入、平台级权限字典（菜单/资源/权限域）、安全运维；对租户内数据只保留「只读排查视角」。

## 2. 租户自管理页面规划

### 2.1 菜单结构

种子菜单（`appCodeTenantAdmin`）+ 前端组件白名单同步：

| 菜单 | 路径 | 图标 | 排序 |
|---|---|---|---|
| 部门架构 | `/department` | apartment | 1（已有） |
| 用户管理 | `/user` | user | 2（新增） |
| 角色管理 | `/role` | role | 3（新增） |

前端 `App.tsx`：`COMPONENT_MAP` 增加 `/user`、`/role`；`ICON_MAP` 增加 `user`、`role`；静态 fallback 菜单同步 3 项；默认落地页仍为 `/department`。

### 2.2 部门架构（部门管理）

**布局重构**：左侧部门树卡片（**收窄至 260**，选中节点高亮，避免占太宽）+ 右侧内容区改用 **Tabs**：

- **Tab1 节点信息**：
  - 顶部：面包屑（`deptPath` 转名称链）、名称、编码、排序、状态（启用/停用 Tag）、子节点数、成员数；
  - 操作：新建子部门、编辑、移动（改父部门，走 `PUT` 全量更新）、启停用、删除（有子节点/成员默认拒绝，`?cascade=1` 级联）。
- **Tab2 成员管理**：
  - 表格字段：**头像、姓名、用户名、邮箱、手机、状态（挂起/正常）、关系（成员/负责人）、主归属、操作（改关系、设主归属、移除）**——用户基础信息来自租户用户目录（person 表富化），不再只有关系字段；
  - 支持按关系类型 / 关键词（姓名/用户名/邮箱/手机）筛选 + 分页；
  - 「添加成员」改为**用户选择器**：弹出抽屉，从 `/tenant/users` 按关键词搜索选择（可多选），默认关系 `member`，首个可勾选为主归属；禁止手输 `userID`。

### 2.3 用户管理

- **列表**：头像、姓名、用户名、邮箱、手机、状态（挂起 Tag）、主部门、角色数、创建时间、更新时间；关键词（姓名/用户名/邮箱/手机）+ 状态筛选 + 分页。
- **创建用户**：租户管理员在租户内新建账号（**姓名 / 部门(必填下拉) / 邮箱 / 手机 / 状态**），`POST /tenant/users`；**初始密码由服务端生成并在创建响应中一次性回显**（前端不填密码，该成员首次登录强制改密）。**姓名即自然人信息**：无匹配自然人时按姓名创建 person；部门必选，创建同时建立部门归属（`departmentIDs`，首个为主部门）。用户名不作为创建属性。
- **行内操作**：
  - 详情 Drawer：基础信息 + **部门归属管理**（勾选部门，首个为主归属，`GET/PUT /users/:userID/departments`，接口已有）+ **角色列表**（展示已分配角色）；
  - **分配角色**：Drawer 列出租户角色（搜索），勾选后 `PUT /users/:userID/roles` 全量替换——**用户侧入口**（对应决策：用户列表的操作负责分配角色）；
  - 编辑基础信息、挂起/恢复（PATCH）、重置密码（`POST /users/:userID/reset-password`）。

### 2.4 角色管理（含角色授权）

- **角色从属于应用**：租户内角色归属到租户订阅的**启用**应用（`appID` 必选，应用选项来自 `GET /tenant/apps`，**含系统内置应用**如管理后台），不同应用角色相互独立、编码应用内唯一。
- **列表**：名称、所属应用、编码、描述、类型、成员数、菜单数、创建时间、更新时间；**应用过滤**（下拉）+ 关键词筛选 + 分页。
- **CRUD**：创建（选所属应用）/ 编辑 / 删除（租户内角色，名称必填、编码应用内唯一）。
- **授权**：行内「菜单权限」操作——Drawer 内以**树形菜单**勾选该角色可访问的**所属应用的菜单**，`GET/PUT /roles/:roleID/menus` 全量替换——**角色侧入口**（对应决策：角色列表的操作负责分配菜单）。
- 角色成员关系由用户侧（2.3 分配角色）维护，角色列表仅展示成员数；如需成员明细可后续加 `GET /roles/:roleID/users` 只读列表。

### 2.5 后续扩展（「等等」预留）

- 登录日志（租户视角，只读，复用 `user_login_log` / `audit_log`）；
- 我的应用（租户已开通应用列表）；
- 个人中心（个人信息 / 修改密码 / 我的部门 / 我的角色）。

## 3. 平台管理页面规划（聚焦平台层）

### 3.1 保留（平台层职责）

| 分组 | 页面 |
|---|---|
| 仪表盘 | 仪表盘 |
| 部门与租户 | 租户管理、租户应用（开通关系） |
| 应用与接入 | 应用管理、OAuth 客户端、域名管理 |
| 权限基础（平台级权限字典） | 菜单管理、权限域、资源 |
| 安全与运维 | API Key、系统配置、审计日志 |

### 3.2 平台端不设用户 / 角色页面（已执行）

**决策**：用户与角色**按租户归属**，其页面归属由授权边界决定——平台端不提供任何用户/角色页面，相关接口一并删除。

| 事项 | 结论 |
|---|---|
| 平台用户管理 | **整页下线**（非降级）。跨租户用户目录/详情/挂起/重置密码全部删除：这些操作的真实授权边界是租户，平台没有"管理别人租户成员"的正当场景 |
| 平台角色管理 | **整页下线**。原只读视角的 `GET /roles/:roleID/users` 不校验租户、不过滤租户内成员，属跨租户数据泄漏，删除即修复 |
| 第三方身份 / 登录日志 | **迁移到租户端**：`GET/POST/DELETE /v1/tenant/users/:userID/identities`、`GET /v1/tenant/users/:userID/login-logs`，作为用户详情 Drawer 的子 Tab |
| 租户管理员指派 | 不提供。`PUT /v1/platform/users/:userID/owner` 删除；`is_owner` 仅作展示用途，不参与鉴权，删除不产生权限真空 |
| 部门架构 | 平台不提供写入与只读页；部门归属由租户端维护 |

**为什么不保留"平台排查视角"**：排查能力若不在租户边界内，就必须跨租户读取与写入，等于把租户授权边界开在平台侧；而平台控制台本身是"平台租户"的一个租户控制台实例（平台租户同时订阅 `admin` 与 `tenant-admin` 应用），平台运维账号要排查某个租户，正确姿势是切换到该租户的租户控制台，而不是在平台侧另开一套跨租户页面。

## 4. 后端 API 支撑（tenantadmin）

### 4.1 部门成员信息富化

`GET /v1/tenant/departments/:departmentID/users` 返回项从关系字段扩展为**关系 + 租户用户基础信息**（对齐 `/v1/tenant/users` 的 `UserPageListItem` 模式）：`userID / userName(姓名) / username / primaryEmail / primaryPhone / avatar / isSuspended / relationType / isPrimary / joinedAt`。实现：批量加载 person 信息（复用 `loadPersonMap`），消除现有按行 `GetByID` 的 N+1。

### 4.2 用户

| 方法 | 路径 | 说明 |
|---|---|---|
| POST | `/v1/tenant/users` | 创建租户用户（姓名/部门(departmentIDs)/邮箱/手机/状态；**不含密码**——服务端生成一次性临时密码随响应返回，见 `tenant-admin-provisioning-design-20260912.md` D3/D7）；**person find-or-create**（§4.4：姓名即自然人信息，未命中则按姓名创建）；部门归属同事务建立 |
| GET | `/v1/tenant/users` | 目录分页（keyword 过滤：姓名/用户名/邮箱/手机；含主部门/角色数聚合） |
| GET | `/v1/tenant/users/:userID` | 详情：基础信息 + 部门归属 + 角色列表 |
| PATCH | `/v1/tenant/users/:userID` | 局部更新（状态 / 基础信息） |
| POST | `/v1/tenant/users/:userID/reset-password` | 重置密码（动作子路径，R2） |
| GET | `/v1/tenant/users/:userID/roles` | 用户已分配角色（父资源子集合视角） |
| PUT | `/v1/tenant/users/:userID/roles` | 全量替换用户角色（批量授权） |
| GET | `/v1/tenant/users/:userID/identities` | 用户已绑定的第三方身份列表（自平台端迁入；身份按 person 归属，租户可见性借道"该 person 在本租户有 user"判定） |
| POST | `/v1/tenant/users/:userID/identities` | 绑定第三方身份（租户取自登录上下文，**不接受请求体传 tenantID**） |
| DELETE | `/v1/tenant/users/:userID/identities/:identityID` | 解绑第三方身份 |
| GET | `/v1/tenant/users/:userID/login-logs` | 用户登录日志（只读，租户 + 用户双重过滤） |
| GET / PUT | `/v1/tenant/users/:userID/departments` | 用户部门归属（已有，不动） |

### 4.3 角色（租户隔离，`tenant_id` 取当前登录租户）

| 方法 | 路径 | 说明 |
|---|---|---|
| GET | `/v1/tenant/apps` | 租户订阅的启用应用列表（角色归属 / 菜单授权的应用选项；含系统内置应用；控制台侧边栏菜单另按「非系统应用」口径，见 `loadConsoleApps`） |
| POST | `/v1/tenant/roles` | 创建角色（**appID 必选**，角色从属于应用；name/code/description/type，编码应用内唯一） |
| GET | `/v1/tenant/roles` | 分页列表（?appID=&keyword=），带成员数 / 菜单数聚合 + 所属应用名 |
| GET | `/v1/tenant/roles/:roleID` | 详情 |
| PUT | `/v1/tenant/roles/:roleID` | 全量更新 |
| DELETE | `/v1/tenant/roles/:roleID` | 删除（级联清理成员/菜单关联） |
| GET | `/v1/tenant/roles/:roleID/menus` | 角色已授权菜单（**所属应用的菜单树** + 已授权ID，供勾选回显；无应用归属的种子角色回退全控制台菜单） |
| PUT | `/v1/tenant/roles/:roleID/menus` | 全量替换角色菜单（校验菜单属于角色所属应用） |

> 说明：角色-用户关系仅由用户侧维护（4.2 的 `PUT /users/:userID/roles`），角色侧不提供写接口；角色列表的成员数由 `user_role` 聚合查询得到。

### 4.4 创建租户用户时的自然人（person）处理

**核心规则**：`user.person_id` 必须指向存在的 person。创建租户用户时**默认以姓名作为自然人信息**，若没有对应自然人则先创建 person（与 user 同事务），即 find-or-create。

```mermaid
flowchart TD
    A[POST /v1/tenant/users] --> B{提供 personID?}
    B -- 是 --> C[校验 person 存在<br/>且未在本租户已有 user]
    C --> C1[关联该 person]
    B -- 否 --> D{按 primaryEmail /<br/>primaryPhone 查已有 person}
    D -- 命中 --> D1[关联已有 person<br/>已有全局身份加入本租户]
    D -- 未命中 --> E[事务内创建 person<br/>Name=姓名, 含 bcrypt 密码哈希]
    E --> E1[关联新 person]
```

细节约定：

1. **指定 personID**：校验 person 存在；同一 person 在本租户只能有一条 `user`（服务层校验 `tenant_id + person_id` 唯一），已存在则拒绝并提示「该用户已在本租户内」；
2. **按全局标识 find-or-create**：`primaryEmail / primaryPhone`（及可选 username）为 person 表全局唯一标识（空值存 NULL）。请求未带 personID 时，按提供的标识逐个查 person：命中 → 直接关联（视为「已有自然人加入本租户」）；未命中 → 同事务内先创建 person（`PersonDao.WithTx(tx).Insert`，**Name 取姓名**，密码用 `gcrypto.GeneratePasswordHash` 生成 bcrypt 哈希），再关联；
3. **与平台端语义差异**：平台端 `svcuser.Create` 对标识冲突直接报 `AlreadyExists`（创建「新 person」语义）；租户端为 find-or-create（加入已有全局身份语义），显式提供标识即视为意图关联；
4. **无 personID 且无任何登录标识**（email/phone/password 均空）：**仍创建 person（仅含姓名）并关联**——person 始终存在，后续可通过重置密码等补充登录能力（不再创建无自然人关联的 user）；
5. **部门归属同事务建立**：`departmentIDs` 提供时校验均属于本租户，创建 user 后插入 `member` 关系（首个为主部门）；
6. **实现复用**：person 创建逻辑对齐平台端 `svcuser.Create`（person 实体字段 Username/PrimaryEmail/PrimaryPhone 用 `model.StrPtr` 存 NULL、Profile/CustomData 初始 `'{}'`、CreatedBy 取当前操作人）。

## 5. 前端改动清单

### 5.1 tenant-admin-web

- `App.tsx`：`ICON_MAP` / `COMPONENT_MAP` / `STATIC_MENU_TREE` 增加用户管理、角色管理；
- `pages/department/index.tsx`：布局重构为「左树 + 右 Tabs（节点信息 / 成员管理）」；成员表格展示完整用户信息；添加成员改用户选择器抽屉；
- `pages/user/index.tsx`（新增）：用户列表 + 创建 + 详情 Drawer（基础信息 / 部门关系 / 角色 / 第三方身份 / 登录日志）+ 挂起 / 重置密码；
- `components/UserIdentityTab.tsx`、`components/UserLoginLogTab.tsx`（新增）：用户详情 Drawer 的子 Tab——身份列表 + 绑定/解绑、登录日志只读表（自平台端迁入）；
- `pages/role/index.tsx`（新增）：角色 CRUD + 菜单权限授权 Drawer（树形菜单勾选）；
- `api/user.ts` 扩展（创建 / 详情 / PATCH / 重置密码 / 角色分配 / 身份增删查 / 登录日志），新增 `api/role.ts`（CRUD + 菜单授权）；
- `pages/department` 成员选择器复用 `getTenantUserPageList`。

### 5.2 platform-admin-web（已执行：整页删除）

- **删除** `pages/user/index.tsx`、`pages/user/Detail.tsx`、`pages/role/index.tsx`；
- `App.tsx`：删除上述导入、`COMPONENT_MAP` / `ICON_MAP` 条目（`user` / `role` / `team`）与 `/user`、`/role`、`/user/:id` 静态路由；
- `pages/dashboard/index.tsx`：统计卡从「用户总数 / 角色总数 / 应用总数 / 租户总数」改为「应用总数 / 租户总数 / API 密钥 / 审计日志」，「平台能力」文案改为平台层职责（多租户治理 / 应用接入 / 密钥监督 / 平台治理）。

### 5.3 packages

- `packages/types`：新增/扩展租户域类型（`TenantUserItem`（含 person 基础信息）、`TenantUserDetail`（含部门归属+角色）、`TenantRoleItem`、`TenantUserIdentityItem`、`TenantUserLoginLogItem` 等），与 `dtotenant` 对齐；**删除**平台域 `UserItem` / `UserCreateReq` / `UserUpdateReq` / `UserStatusUpdateReq` / `UserPasswordUpdateReq` / `UserIdentityItem` / `UserIdentityCreateReq` / `UserLoginLogItem` / `RoleItem` / `RoleCreateReq` / `RoleUpdateReq` / `RoleUserItem`（`AdminLevel` 保留，租户域 `TenantRoleItem` 复用）；
- `packages/api`：`resources/platform.ts` 删除用户 / 角色两段共 10 个函数与对应类型导入；租户端接口走 tenant-admin-web 自身 `api/` 目录。

## 6. 种子数据调整（pkg/seed）

- `seedMenus` 的 `appCodeTenantAdmin` 段含用户/角色菜单（编码加 `tenant-` 前缀，避免与平台菜单 code 撞名）：

```
{appCode: appCodeTenantAdmin, name: "用户管理", code: "tenant-user", path: "/user", icon: "user", sort: 2, component: "pages/user"}
{appCode: appCodeTenantAdmin, name: "角色管理", code: "tenant-role", path: "/role", icon: "role", sort: 3, component: "pages/role"}
```

- `seedMenus` 的 `appCodeAdmin` 段**删除** `身份中心`（`grp-identity`）目录及其下 `用户管理`（`user`）、`角色管理`（`role`）两条菜单，`平台管理` 分组 sort 由 5 调整为 4；
- `seedRoleMenus` 的 `admin` 角色菜单授权删除 `user` / `role` 两项（14 → 12 条；`tenant_admin` 的 4 条不变）；
- 种子幂等（按 `app_id + code` 查重）。注意：平台菜单是**幂等补种**而非删旧，已有库中的 `grp-identity` / `user` / `role` 行不会被自动清理，需要重建库或手工删除（本次交付允许重建 docker 数据库）。

## 7. 实施阶段与验收标准

### 7.1 实施阶段

1. **阶段一（后端支撑）**：`department_user` 成员信息富化；tenantadmin 用户接口（创建/详情/PATCH/重置密码/角色分配/第三方身份/登录日志）+ 角色接口（CRUD/菜单授权）；种子菜单补充；单测（复用 `testutil.SetupSQLite`）。
2. **阶段二（租户端前端）**：部门页重构（Tabs + 完整成员信息 + 用户选择器）；用户管理页（含第三方身份 / 登录日志 Tab）；角色管理页（含菜单授权）；`App.tsx` 路由/菜单/白名单。
3. **阶段三（平台端下线 + 打磨）**：执行平台端下线清单（见 §8）——用户/角色**整页下线**、删除对应接口与孤儿代码；关键词筛选、分页、空态、加载态细节；README/文档同步。

### 7.2 验收标准

1. 租户自管理侧边栏出现「部门管理 / 用户管理 / 角色管理 / API密钥」四项，动态菜单与静态 fallback 一致，路由均可进入（无 404）；
2. 部门架构成员表格展示完整用户基础信息（头像/用户名/邮箱/手机/状态），添加成员可从用户目录搜索选择，不再手输 ID；
3. 用户管理支持创建、挂起/恢复、重置密码、部门归属管理、角色分配（用户侧入口，全量替换生效）、第三方身份绑定/解绑、登录日志查看；
3.1 创建租户用户时 person find-or-create 正确：指定 personID 直接关联并校验租户内唯一；按 username/email/phone 命中已有 person 则关联、未命中则同事务先创建 person；无任何标识则创建无自然人关联的租户内用户；
4. 角色管理支持 CRUD 与菜单权限授权（角色侧入口，树形勾选回显正确、全量替换生效）；
5. 平台管理**不存在**用户 / 角色页面与接口：`/v1/platform/users*`、`/v1/platform/roles*` 全部 404，侧边栏无「身份中心」分组；平台端与租户端身份/登录日志接口跨租户访问一律按「用户不存在」拒绝；
6. `go test ./...` 全绿、`make lint` 通过；platform-admin-web / tenant-admin-web 构建通过，页面可用。

## 8. 平台端下线清单（页面 / 接口 / 代码）

> 判定标准：页面归属由**授权边界**决定——租户所有的数据只在租户控制台出现；平台端只保留跨租户的"监督/干预"类能力（租户状态、租户应用、API 密钥只读监督）。**用户与角色整页下线**（非降级，§3.2），其余页面保留；**数据表 / 实体保留**（`role_menu` / `user_role` 由 tenantadmin 写入，`role_scope` 数据保留供权限引擎后续消费）。

### 8.1 前端 API 函数删除（packages/api/resources/platform.ts）

| 函数 | 原因 |
|---|---|
| `getRoleDetail` | 无页面消费（角色详情 Drawer 用列表数据） |
| `getTenantDetail` / `getTenantApplicationDetail` | 无页面消费（页面只用列表 + create/update/delete） |
| `getMenuPageList` / `getMenuDetail` | 无页面消费（菜单页用 tree + create/update/delete） |
| `getConnectorPageList` / `getConnectorDetail` / `createConnector` / `updateConnector` / `deleteConnector` / `getConnectorFactoryList` | 前端无任何页面消费；后端 `/auth/connectors` 属认证域保留，前端函数先删，连接器管理页后续按需重建 |
| `createUser` / `updateUser` / `deleteUser` | 用户页下掉新建/编辑/删除后无消费（创建收敛到 `POST /tenant/users`） |
| `createRole` / `updateRole` / `deleteRole` | 角色页下掉 CRUD 后无消费（角色管理收敛到 tenantadmin） |
| `assignRoleUsers` / `removeRoleUser` | 角色页下掉成员管理（写）后无消费（成员分配收敛到租户端用户页） |
| `getUserPageList` / `getUserDetail` / `updateUserStatus` / `updateUserPassword` | 平台用户页整页下线后无消费；租户端对应函数见 `tenant-admin-web/src/api/user.ts` |
| `getUserIdentityByUser` / `createUserIdentity` / `deleteUserIdentity` / `getUserLoginLogByUser` | 能力迁移到租户端：`getTenantUserIdentities` / `createTenantUserIdentity` / `deleteTenantUserIdentity` / `getTenantUserLoginLogs` |
| `getRolePageList` / `getRoleUsers` | 平台角色页整页下线后无消费（其中 `GET /roles/:roleID/users` 原本缺租户校验，属跨租户泄漏，删除即修复） |

保留：`getApplicationPageList` / `getTenantPageList` / `getApiKeySupervisionPageList` / `getAuditLogPageList`（仪表盘计数）及其余应用、OAuth 客户端、租户、租户应用、API 密钥监督、菜单、域名、审计日志函数；另新增 `getTenantUserIdentities` 等 4 个函数于租户端 `api/user.ts`（不放在 `packages/api`）。

### 8.2 前端页面删除（platform-admin-web）

| 页面 | 处理 |
|---|---|
| `pages/user/index.tsx` | **删除**（列表 / 挂起恢复 / 重置密码） |
| `pages/user/Detail.tsx` | **删除**（基本信息 / 第三方身份 / 登录日志） |
| `pages/role/index.tsx` | **删除**（只读列表 + 成员抽屉） |
| `App.tsx` | 删除导入、`COMPONENT_MAP`（`/user`、`/role`、`/user/:id`）、`ICON_MAP`（`user` / `role` / `team`）、静态详情路由 |
| `pages/dashboard/index.tsx` | 统计卡改为应用 / 租户 / API 密钥 / 审计日志；「平台能力」文案改为平台层职责 |
| 其余页面 | 保留不动 |

### 8.3 后端路由删除（platformadmin）

| 路由文件 | 删除 | 保留 |
|---|---|---|
| `router/user.go` | **整文件删除**：`GET /users`、`GET/PATCH /users/:userID`、`PUT /users/:userID/owner`、`POST /users/:userID/changePassword`、identities 子资源 list/create/delete、`GET /users/:userID/login-logs` | 无（能力已在 `/v1/tenant/users/*` 重建） |
| `router/permission.go` | `roleRouter` 整组：`GET /roles`、`GET /roles/:roleID`、`GET /roles/:roleID/users` | `menuRouter`（菜单字典）不动 |
| `router/router.go` | `userRouter(groups)`、`roleRouter(groups)` 注册 | 其余不动 |

> 说明：`PUT /v1/platform/users/:userID/owner` 一并删除（平台侧不代管租户管理员指派）；`is_owner` 仅用于展示，不参与鉴权，删除不产生权限真空。

### 8.4 后端孤儿代码清理（已执行）

- **controller**：`ctruser/`（`user.go`）整目录删；`ctrpermission/role.go` 删；
- **service**：`svcuser/`（`user.go` + `user_identity.go` 及全部 `user_*_test.go`）整目录删；`svcperson/`（`person.go` + `person_identity_test.go`）整目录删（身份读写迁到 `svctenant/user_identity.go`）；`svcpermission/role.go` 删，`svcpermission/tenant_scope_test.go` 裁剪掉两个角色用例只留菜单用例；
- **dto**：`dtouser/` 整目录删；`dtopermission` 删除 `RoleDetailReq` / `RolePageListReq` / `RoleMenuCreateReq` / `RoleMenuDeleteReq` / `RoleMenuPageListReq` / `RoleDetailResp` / `RolePageListItem` / `RolePageListResp`；
- **路由**：`userRouter` / `roleRouter` 注册移除；
- **swagger**：`make swag APP=platformadmin` 重新生成，`dtouser` / `dtopermission.Role*` 定义已消失；
- **错误码**：`pkg/code/user.go` 的 `UserIdentity*` / `UserLoginLog*` 段**保留**——租户端迁移后的接口继续使用该段，无需改动。

### 8.5 保留（数据 / 实体 / 接口）

- 实体与表：`role_menu`（tenantadmin 角色菜单授权写入）、`user_role`（tenantadmin 用户角色分配写入）、`role_scope`（数据保留，权限引擎后续消费）、`department` / `department_user`（tenantadmin 维护）、`user_identity` / `user_login_log`（改为租户端接口读写）；
- 接口：租户端 `/v1/tenant/users/*`（含身份与登录日志子资源）与 `/v1/tenant/roles/*`；平台端租户、租户应用、应用、OAuth 客户端、API Key 监督、菜单、域名、审计日志。
