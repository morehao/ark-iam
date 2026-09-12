# 组织架构容器命名回归 department（organization → department）

> 状态：**已落地**（2026-09-12 实施完成；实施结果与偏差见 §10）
> 决策日期：2026-09-12
> 采纳选项：D1 ✅、D2 ✅、D3 ✅、D4 ✖（本次不做）、D5 ✅
> 取代/修订：`organization-container-redesign.md` 的**命名层**（该文 §1.2 的定位表述、§2 表/字段名、§4 API 路径、§5.3、§7 影响面清单，一律按本文执行）。
> **结构层设计不变**：`parent_id` 树 + 物化路径 + 关系表 `relation_type(primary/secondary/leader)` + 租户隔离 + 高频读物化路径，全部保留。
> 影响一句话：2 张表、10 条 API、13 个错误码常量、1 个种子菜单、16 个文件重命名；后端 38 文件 807 处、前端 9 文件 131 处、文档约 146 处。
>
> 字段下线（后续调整）：`department.code` 已**彻底下线**，部门仅有名称（无业务编码）；本文出现的 `name`/`code` 字段并列表述按此阅读，见 [`department-code-retirement.md`](./department-code-retirement.md)。

---

## 0. 决策

### 0.1 结论

租户内层级容器（树节点）统一命名 **`department`**，`organization` 作为该容器名**全链路下线**：不留别名、不加兼容层、不双写。

依据（§1 有出处）：业界的 `organization` 指**租户/客户级容器**，而在本项目里那个槽位已被 `tenant` 占据；`department` 与本仓库既有关系语义（`primary` 行政主部门 / `secondary` 参与部门 / `leader` 部门负责人）自洽，且不与 `tenant` 争词义。

### 0.2 明确接受的两个代价（评审时请确认）

| 代价 | 说明 | 缓解 |
|---|---|---|
| 命名带业务语义 | "学校 / 班级 / 项目组"类租户不再被命名中性覆盖 | 树结构与 `name`/`code` 本身不限制节点含义；若真出现多形态租户，按 §3-D4 追加类型维度（本次不做） |
| 与 SCIM 习惯相反 | SCIM 把 `department` 定义为**用户属性**，容器是 `Group` | 暂不做 SCIM 对接；将来映射为「我们的 `department` ↔ 对方 `Group`，对方 `department` 属性 ↔ 我们的一个用户属性」 |

### 0.3 范围边界

**做**：表名/列名/索引名、Go 实体·DAO·Object·DTO·Service·Controller·Router、日志前缀、错误码常量名与中文文案（**编号不动**）、种子（根部门 + 菜单 + ProvisionMenuCodes + retiredMenus 登记）、全部测试、前端类型/API/页面/路由/文案、Swagger 重新生成、living docs、`AGENTS.md`。

**不做**：结构设计变更；错误码编号变更（前端错误码表无需改）；`relation_type` 枚举值（仍是 `primary`/`secondary`/`leader`）；API 版本（仍 `/v1`，无兼容层，前后端必须同批发布）；`docs/superpowers/` 历史 spec/plan（保留原文作为历史记录）。

---

## 1. 决策依据：业界命名对标

| 项目 | 租户内层级容器 | `organization` 指什么 | `department` 指什么 |
|---|---|---|---|
| [SCIM RFC 7643](https://datatracker.ietf.org/doc/html/rfc7643) | **Group**（明确支持嵌套组） | Enterprise User **属性**（公司名，§4.3） | Enterprise User **属性**（§4.3） |
| AD / LDAP | **OU**（organizationalUnit） | 域/组织概念 | 用户属性 |
| [Google Workspace](https://knowledge.workspace.google.com/admin/users/advanced/how-the-organizational-structure-works) | **OU**（层级树）+ Groups | — | 用户属性 |
| [Auth0](https://auth0.com/docs/get-started/auth0-overview/create-tenants/multi-tenant-apps-best-practices) | Organizations | **客户/租户容器**（官方称 Organization Tenant） | — |
| [Keycloak 26](https://www.keycloak.org/2026/04/org-groups) | Realm **Groups**；组织内的层级另有 Organization Groups | **B2B/伙伴级容器** | — |
| [Logto](https://docs.logto.io/organizations) | Organizations | 官方原话：organization = multi-tenancy | — |
| [Casdoor](https://github.com/casdoor/casdoor) | Organizations | **独立租户**（独立用户与品牌） | — |
| [ZITADEL](https://zitadel.com/docs/guides/manage/console/projects-overview) | Organization / Granted Organization | **组织级容器** | — |
| [FusionAuth](https://fusionauth.io/docs/extend/examples/modeling-hierarchies) | Tenant → **Groups**（可嵌套）/ Entity | — | — |
| [midPoint](https://docs.evolveum.com/midpoint/reference/master/org/organizational-structure/) | **Org**（`OrgType`，自称 "a universal grouping mechanism"，覆盖 organizations / divisions / **departments** / sections / workgroups / projects / teams） | — | Org 的**一种** |

三条可执行的结论：

1. **`organization` 在主流 IAM 里几乎专指租户/客户级容器**（Auth0、Keycloak、Logto、Casdoor、ZITADEL、AWS Organizations），在 SCIM 里是"公司名"用户属性。本项目 `tenant` 已占该槽位，继续叫 `organization` 是**系统性撞义**：对外文档会写成"我们的 organization ↔ 对方的 tenant，对方的 organization ↔ 我们的 tenant"。
2. **`department` 在标准里是用户属性/组织子类**，作容器名不标准——但它不与 `tenant` 撞义，且与本仓库现有的部门化关系语义一致，是"最不坏"的业务化命名。
3. **midPoint 是本项目最贴近的同构参考**：它的 `Org` 明确"只做分组机制、不解释节点是部门还是项目组"，且它的 relation 机制（"default relation is usually interpreted as 'belongs to' … may be used to distinguish organizational unit managers from other employees"）与本仓库 `primary`/`secondary`/`leader` 一一对应。若将来想回到中性命名，参考对象是 `org` / `org_unit`，不是 `organization`。

---

## 2. 命名规格（单一事实源）

### 2.1 表与列

| 旧 | 新 | 说明 |
|---|---|---|
| `TableNameOrganization` = `"organization"` | `TableNameDepartment` = `"department"` | 部门树 |
| `TableNameOrganizationUser` = `"organization_user"` | `TableNameDepartmentUser` = `"department_user"` | 部门关系（用户↔部门节点） |
| `organization_user.organization_id` | `department_user.department_id` | 关系外键列 |
| `organization.org_path` | `department.dept_path` | §3-D1（推荐纳入，否则词根半 org 半 dept） |
| `organization.org_depth` | `department.dept_depth` | §3-D1 |
| 索引 `uk_org_user` / `idx_organization_parent` / `idx_organization_path` | `uk_dept_user` / `idx_department_parent` / `idx_department_path` | ⚠️ **实测：这三个索引只存在于 `organization-container-redesign.md` §3.5 的 SQL，代码里没有任何 gorm tag 或 `EnsurePartialUniqueIndexes` 条目**，即当前实现库中并未创建。见 §3-D5 |

### 2.2 Go 符号（按族归类；全量清单见附录 A）

**model**（`pkg/iam/model`）
`OrganizationEntity`→`DepartmentEntity`、`OrganizationEntityList`→`DepartmentEntityList`、`OrganizationUserEntity`→`DepartmentUserEntity`、`OrganizationUserEntityList`→`DepartmentUserEntityList`、
`OrgPath`→`DeptPath`、`OrgDepth`→`DeptDepth`、`MaxOrgDepth`→`MaxDeptDepth`、
`OrgNodeStatus`→`DeptNodeStatus`、`OrgNodeStatusActive/Inactive`→`DeptNodeStatusActive/Inactive`（值 `"active"`/`"inactive"` 不变）、
`OrgUserRelationType`→`DeptUserRelationType`、`OrgUserRelationPrimary/Secondary/Leader`→`DeptUserRelationPrimary/Secondary/Leader`（**枚举值不变**）

**dao**（`pkg/iam/dao`）
`OrganizationDao`→`DepartmentDao`、`NewOrganizationDao`→`NewDepartmentDao`、`OrganizationCond`→`DepartmentCond`、
`OrganizationUserDao`→`DepartmentUserDao`、`NewOrganizationUserDao`→`NewDepartmentUserDao`、`OrganizationUserCond`→`DepartmentUserCond`

**object / 共享包**
`objtenant.OrganizationBaseInfo`→`DepartmentBaseInfo`；
`tenant.CreateWithRootOrgReq`/`CreateWithRootOrg`→`CreateWithRootDeptReq`/`CreateWithRootDept`（`provision.go` 的 `RootOrg` 字段、局部变量 `rootOrg`→`rootDept`；`apps/auth/internal/service/svcoidc/register.go` 调用点同步）；
`user.ErrMultiplePrimaryOrg`→`ErrMultiplePrimaryDept`、`user.ErrOrgLeaderConflict`→`ErrDeptLeaderConflict`

**tenantadmin（package `svctenant` / `ctrtenant` / `dtotenant` / `router`）**
`OrganizationSvc`→`DepartmentSvc`、`organizationSvc`→`departmentSvc`、`NewOrganizationSvc`→`NewDepartmentSvc`、
`OrganizationUserSvc`→`DepartmentUserSvc`、`organizationUserSvc`→`departmentUserSvc`、
`OrganizationCtr`→`DepartmentCtr`、`NewOrganizationCtr`→`NewDepartmentCtr`、`OrganizationUserCtr`→`DepartmentUserCtr`、`NewOrganizationUserCtr`→`NewDepartmentUserCtr`、
`organizationRouter`→`departmentRouter`、
`organizationVisibleToTenant`→`departmentVisibleToTenant`、`loadUserOrganizations`→`loadUserDepartments`、
日志前缀 `[svcorganization.*]`→`[svcdepartment.*]`、`[svcorganizationuser.*]`→`[svcdepartmentuser.*]`

**DTO 类型名**（`dto/dtotenant`）
`OrganizationCreateReq/Resp`、`OrganizationUpdateReq`、`OrganizationStatusReq`、`OrganizationDeleteReq`、`OrganizationTreeReq/Resp`、`OrganizationTreeItem`、`OrganizationChildrenReq/Resp`、`OrganizationChildItem` → 对应 `Department*`；
`OrganizationUserCreateReq/Delete …`、`OrganizationUserPageListReq/Resp`、`OrganizationUserPageListItem`、`OrganizationUserUpdateReq` → `DepartmentUser*`；
`UserOrganizationItem`→`UserDepartmentItem`、`UserOrganizationRequiredError`→`UserDepartmentRequiredError`（错误码）

**seed**（`pkg/seed`）
`seedRootOrganization`→`seedRootDepartment`、`seedAdminUserOrganization`→`seedAdminUserDepartment`（"顶级部门 / 根部门"）；`ProvisionMenuCodes` 内 `"organization"`→`"department"`

### 2.3 API 路由（tenantadmin，前缀 `/v1/tenant`）

| 方法 | 旧 | 新 |
|---|---|---|
| POST | `/organizations` | `/departments` |
| GET | `/organizations/tree` | `/departments/tree` |
| GET | `/organizations/{organizationID}/children` | `/departments/{departmentID}/children` |
| PUT | `/organizations/{organizationID}` | `/departments/{departmentID}` |
| PATCH | `/organizations/{organizationID}` | `/departments/{departmentID}` |
| DELETE | `/organizations/{organizationID}` | `/departments/{departmentID}` |
| GET | `/organizations/{organizationID}/users` | `/departments/{departmentID}/users` |
| POST | `/organizations/{organizationID}/users` | `/departments/{departmentID}/users` |
| PUT | `/organizations/{organizationID}/users/{userID}` | `/departments/{departmentID}/users/{userID}` |
| DELETE | `/organizations/{organizationID}/users/{userID}` | `/departments/{departmentID}/users/{userID}` |

- path 参数一律 `:departmentID` / `{departmentID}`，DTO tag 保持 `json:"-" uri:"departmentID"`（path 是 ID 唯一来源）。
- swagger 注解 `@Router /v1/tenant/organizations/...` 同步为 `/v1/tenant/departments/...`，`@Param ... path string true "部门ID"`。
- ⚠️ `organization-container-redesign.md` §4.1 里的 `GET/PUT /v1/tenant/users/:userID/organizations` **当前代码已不存在**（用户归属并入 `POST /users` 创建与 `PATCH /users/{userID}` 局部更新、详情返回 `organizations`）。本次文档同步删除这两行，不新增 `/users/{userID}/departments` 子路由。

### 2.4 错误码（**编号不变**，仅常量名 + 中文文案）

| 编号 | 旧常量 | 新常量 | 旧文案 → 新文案 |
|---|---|---|---|
| 100120 | `OrganizationCreateError` | `DepartmentCreateError` | 创建组织失败 → 创建部门失败 |
| 100121 | `OrganizationDeleteError` | `DepartmentDeleteError` | 删除组织失败 → 删除部门失败 |
| 100122 | `OrganizationUpdateError` | `DepartmentUpdateError` | 修改组织失败 → 修改部门失败 |
| 100123 | `OrganizationGetDetailError` | `DepartmentGetDetailError` | 查看组织详情失败 → 查看部门详情失败 |
| 100124 | `OrganizationGetPageListError` | `DepartmentGetPageListError` | 查看组织列表失败 → 查看部门列表失败 |
| 100125 | `OrganizationNotExistError` | `DepartmentNotExistError` | 组织不存在 → 部门不存在 |
| 100140 | `OrganizationUserCreateError` | `DepartmentUserCreateError` | 创建组织用户失败 → 创建部门成员失败 |
| 100141 | `OrganizationUserDeleteError` | `DepartmentUserDeleteError` | 删除组织用户失败 → 删除部门成员失败 |
| 100142 | `OrganizationUserGetPageListError` | `DepartmentUserGetPageListError` | 查看组织用户列表失败 → 查看部门成员列表失败 |
| 100143 | `OrganizationUserNotExistError` | `DepartmentUserNotExistError` | 组织用户不存在 → 部门成员不存在 |
| 100144 | `OrganizationUserUpdateError` | `DepartmentUserUpdateError` | 修改组织用户失败 → 修改部门成员失败 |
| 100145 | `OrganizationUserLeaderConflictError` | `DepartmentUserLeaderConflictError` | 该部门已有负责人（文案已正确，仅改名） |
| 100518 | `UserOrganizationRequiredError` | `UserDepartmentRequiredError` | 用户必须从属于至少一个部门（文案已正确，仅改名） |

### 2.5 DTO / JSON 字段

| 旧 JSON | 新 JSON | 出现处 |
|---|---|---|
| `organizationID` | `departmentID` | 部门请求/响应、用户筛选、成员关系条目 |
| `organizationIDs` | `departmentIDs` | 用户/服务账号创建（primary，仅 1 个） |
| `secondaryOrgIDs` | `secondaryDepartmentIDs` | 用户/服务账号创建与更新 |
| `leaderOrgIDs` | `leaderDepartmentIDs` | 用户创建与更新 |
| `primaryOrgID` / `primaryOrgName` | `primaryDepartmentID` / `primaryDepartmentName` | 用户/服务账号列表、详情、更新 |
| `organizations` | `departments` | 用户/服务账号详情归属列表 |
| `organizationName` | `departmentName` | `UserDepartmentItem` |

顺带统一缩写：不再混用 `Org` 与 `Organization`（现 `OrganizationID` 与 `PrimaryOrgID` 并存）。前端 `packages/types` 镜像字段同步改名。

### 2.6 种子与菜单

- `seed.go` 菜单定义：`name: "组织管理"` → `"部门管理"`；`code: "organization"` → `"department"`；`path: "/organization"` → `"/department"`；`component: "pages/organization"` → `"pages/department"`（`icon: "apartment"` 不变）。
- `seedRoleMenus` 里 `admin` 角色的菜单 code 列表：`"organization"` → `"department"`。
- `pkg/iam/tenant/provision.go`：`ProvisionMenuCodes = {"organization", …}` → `{"department", …}`。
- **`pkg/seed/retired_menu.go` 必须追加登记**（否则存量库残留指向 `/organization` 的死链菜单 + 其 role_menu 授权）：
  ```go
  // 租户端「组织管理」页改名为「部门管理」：菜单 code/path 变更，
  // 登记旧 code 以清理存量库残留（seedMenus 只 upsert 不下线）。
  {appCode: appCodeTenantAdmin, menuCode: "organization"},
  ```
- 根节点语义文案：`seedRootOrganization` 相关注释「顶级部门（根组织节点）」统一为「顶级部门（根部门节点）」。
- 测试断言同步：`seed_test.go`（`assertCount("organization", 1)` → `"department"`；`wantMenuCodes`）、`seed_pg_test.go`（删表顺序列表 `organization_user, organization` → `department_user, department`）。

### 2.7 前端（`frontend/`）

| 旧 | 新 |
|---|---|
| `packages/types/src/organization.ts` | `packages/types/src/department.ts` |
| `packages/types/src/index.ts` 的 `export * from './organization'` | `export * from './department'` |
| `apps/tenant-admin-web/src/api/organization.ts` | `apps/tenant-admin-web/src/api/department.ts` |
| `apps/tenant-admin-web/src/pages/organization/index.tsx` | `apps/tenant-admin-web/src/pages/department/index.tsx` |
| `App.tsx`：`import OrganizationList from './pages/organization'`、`'/organization': OrganizationList` | `import DepartmentList from './pages/department'`、`'/department': DepartmentList` |
| 类型 `OrganizationItem` / `OrganizationChildItem` / `OrganizationUserItem` / `UserOrganizationItem` / `OrganizationTreeResp` / `OrganizationChildrenResp` | `DepartmentItem` / `DepartmentChildItem` / `DepartmentUserItem` / `UserDepartmentItem` / `DepartmentTreeResp` / `DepartmentChildrenResp` |
| API 函数 `getOrganizationTree` / `getOrganizationChildren` / `createOrganization` / `updateOrganization` / `updateOrganizationStatus` / `deleteOrganization` / `getOrganizationUserPage` / `createOrganizationUser` / `updateOrganizationUser` / `deleteOrganizationUser` | 同名 `Department*`；请求路径（axios `baseURL` 已含 `/v1`）`'/tenant/organizations'` → `'/tenant/departments'`、`` `/tenant/organizations/${organizationID}/children` `` → `` `/tenant/departments/${departmentID}/children` `` |
| `packages/types/src/tenant.ts`：`organizationIDs`/`secondaryOrgIDs`/`primaryOrgID`/`primaryOrgName`/`organizations`/`TenantMachineUserOrganization` | 按 §2.5 改名（`TenantMachineUserDepartment`） |
| `pages/user/index.tsx`、`api/user.ts`、`api/machineUser.ts` 的字段与请求体 | 按 §2.5 改名；**列名/表单文案「主部门 / 参与部门 / 所属部门」本来就正确，保留** |

### 2.8 文件重命名清单（16 个）

后端 13：`pkg/iam/model/organization.go`→`department.go`、`pkg/iam/model/organization_user.go`→`department_user.go`、`pkg/iam/dao/organization.go`→`department.go`、`pkg/iam/dao/organization_user.go`→`department_user.go`、`pkg/iam/object/objtenant/organization.go`→`department.go`、`service/svctenant/organization.go`→`department.go`、`service/svctenant/organization_user.go`→`department_user.go`、`service/svctenant/organization_test.go`→`department_test.go`、`controller/ctrtenant/organization.go`→`department.go`、`controller/ctrtenant/organization_user.go`→`department_user.go`、`router/organization.go`→`department.go`、`dto/dtotenant/organization_request.go`→`department_request.go`、`dto/dtotenant/organization_response.go`→`department_response.go`。

前端 3：`packages/types/src/organization.ts`→`department.ts`、`apps/tenant-admin-web/src/api/organization.ts`→`department.ts`、`apps/tenant-admin-web/src/pages/organization/index.tsx`→`pages/department/index.tsx`（默认导出 `OrganizationPage`→`DepartmentPage`）。

---

## 3. 待评审的可选项（D1–D5）

| 项 | 内容 | 建议 | 影响 |
|---|---|---|---|
| **D1** | `org_path`/`org_depth`/`MaxOrgDepth`/`OrgNodeStatus` 是否一并改 `dept_path`/`dept_depth`/`MaxDeptDepth`/`DeptNodeStatus` | **纳入** | 列 2 个 + 符号 5 个；不纳入则词根半 org 半 dept，`department` 表里留 `org_path` 更别扭 |
| **D2** | 创建/更新入参由 `departmentIDs []string`（primary，数组但至多 1 个）改为 `primaryDepartmentID string` + `secondaryDepartmentIDs` + `leaderDepartmentIDs` | **纳入**（可拆为独立变更） | 更贴合"至多 1 个"的语义、省掉服务层数组长度校验；但属 payload 形状变更，非纯改名 |
| **D3** | 关系枚举类型名 `OrgUserRelationType`→`DeptUserRelationType`（枚举值不变） | **纳入** | 符号 4 个，无 DB 变化 |
| **D4** | 是否给 `department` 加类型维度（`department`/`project`/`class`） | **本次不做** | 与 `organization-container-redesign.md` §6.2（类型推给业务侧）一致；若未来确认要中立支持多形态租户，再按该节方案追加 |
| **D5** | 补上缺失的索引与唯一约束：`department_user` 唯一索引 `uk_dept_user(tenant_id, department_id, user_id, relation_type)`、`department` 索引 `idx_department_parent(tenant_id, parent_id)` 与 `idx_department_path(dept_path)` | **纳入**（顺手修一个已存在的缺口） | 现状：`primary` 每用户至多 1 行、`leader` 每部门至多 1 人**全靠服务层校验**，DB 无唯一约束，并发下有竞态窗口；树按 `parent_id`/`dept_path` 查询也无索引。实现方式：GORM tag（`uniqueIndex` / `index`）即可被 AutoMigrate 创建，无需手写 SQL |

---

## 4. 影响面量化（当前主仓实测，不含 `.tmp/`、`.worktrees/`）

| 面 | 数量 |
|---|---|
| 后端 Go | 38 文件 / 807 处（`organization` 词根命中） |
| 前端 TS/TSX | 9 文件 / 131 处 |
| Swagger 生成物 | `apps/tenantadmin/docs/tenantadmin_docs.go` 57 处（**重新生成，不手改**） |
| 文档 | living 11 文件约 146 处：`organization-container-redesign.md` 81、`api-reference.md` 15、`tenant-admin-console-redesign.md` 13、`tenant-admin-provisioning-design-20260912.md` 12、`system-design.md` 10、`AGENTS.md` 10、`string-id-pg-automigrate-seed.md` 1、`README.md` 1、`README.zh.md` 1、`frontend/DESIGN.md` 1、`frontend/README.md` 1 |
| 表 / 列 / 索引 | 2 表 / 1 列（+2 列若采纳 D1）/ 3 索引（**当前代码未创建，见 D5**） |
| API | 10 条路径 + swagger 注解 |
| 错误码 | 13 个常量名 + 13 条文案（编号不变） |
| 文件重命名 | 16 |
| 跨应用测试引用 | `pkg/seed/{seed_test,seed_pg_test}.go`、`apps/auth`（`register_test.go`、`auth_integration_test.go`）、`apps/platformadmin`（`svctenant/tenant_test.go`）、`apps/tenantadmin`（`testutil/dbtest.go` 等） |
| 死代码顺带清理 | `apps/auth/internal/dto/dtoauth/request.go` 的 `AssignDepartmentsReq`（零引用，属上一轮重构残留），本次删除 |

---

## 5. 数据库处置

按 `AGENTS.md`「按新项目处理」约定：

- **主路径 = 重置开发/测试库**：删库重建后 `AutoMigrate` + `Seed` 产出的结构即目标结构（`docs/design/run-and-deploy.md` §2.3）。
- **不写迁移脚本、不在启动流程加兼容分支**（不加 `Migrator()` 判定、不加回填）。旧库残留 `organization*` 表属预期，不被读写。
- **确需保全数据时**（例如某开发库有想留的联调数据），按部署文档口径执行附录 B 的一次性 SQL，**由人工在升级前运行**，不进代码。

---

## 6. 执行顺序（每批结束即可编译 + 测试）

| 批次 | 内容 | 验收 |
|---|---|---|
| P1 | `pkg/iam/model`（表名/列名/索引名 + D1/D3/D5 符号与 gorm tag）→ `pkg/iam/model/automigrate.go` 实体清单 | `cd backend && go build ./...` |
| P2 | `pkg/iam/dao` → `pkg/iam/object/objtenant` → `pkg/iam/user`（`Err*`/关系写入）→ `pkg/iam/tenant`（`CreateWithRootDept`/`provision.go`） | `go build ./...` |
| P3 | `pkg/code`（§2.4 全部常量名 + 文案） | `go build ./...` |
| P4 | `apps/tenantadmin`：`dto/dtotenant` → `service/svctenant` → `controller/ctrtenant` → `router`（+ `testutil`） | `make test APP=tenantadmin` |
| P5 | `pkg/seed`（根部门/菜单/provision codes/**retiredMenus 登记**）+ seed 测试 | `make test APP=platformadmin`、`go test ./pkg/seed/...` |
| P6 | 其余测试引用：`apps/auth`、`apps/platformadmin`、跨包测试名与断言 | `cd backend && go test ./...` |
| P7 | 前端：`packages/types` → `api/*` → `pages/department` → `App.tsx` → `tenant.ts`/`user.ts`/`machineUser.ts` 字段 | `cd frontend && pnpm typecheck && pnpm build:tenant` |
| P8 | `make swag APP=tenantadmin` 重新生成；living docs 与 `AGENTS.md` 更新（§7）；`organization-container-redesign.md` 顶部加「命名层已被本文取代」注记 + 内部可执行内容改 department 口径 | 文档口径无 `organization` 残留（允许残留位见 §7.1）；`make lint` 通过 |

提交切分建议：P1–P3 一个 commit（领域与错误码）、P4–P6 一个（tenantadmin 全链路）、P7 一个（前端）、P8 一个（文档 + swagger）。

---

## 7. 验收标准

1. **零残留**：受管路径下 `organization` / `Organization` / `OrganizationID` / `organizationID` / `organizationIDs` / `primaryOrgID` / `primaryOrgName` / `secondaryOrgIDs` / `leaderOrgIDs` / `org_path` / `org_depth` / `OrgPath` / `OrgDepth` / `primary_org` 全部 0 命中：
   ```bash
   grep -rn "rganization\|org_path\|org_depth\|OrgPath\|OrgDepth\|primaryOrgID" \
     backend/apps backend/pkg frontend/packages frontend/apps docs/design AGENTS.md README.md README.zh.md
   ```
   （允许残留的位置仅三处：① `docs/superpowers/**` 历史 spec/plan；② 本文 `department-container-rename.md` 的对照表；③ `docs/design/organization-container-redesign.md` —— 该文作为"当时决策"的历史记录保留文件名与标题，仅在其顶部加「命名层已被本文取代」注记，其内部可执行内容（§2 表/字段、§4 API、§5.3、§7 清单）改按 department 口径书写。）
2. `cd backend && go build ./... && go test ./...` 全绿；`make lint` 通过。
3. 前端 `pnpm typecheck` + `pnpm build:tenant` 通过，租户控制台「部门管理」页可建树/移动/级联删除/维护成员。
4. Swagger 重新生成，`/v1/tenant/departments/*` 10 条齐全，无 `/organizations` 残留。
5. **存量库启动无幽灵菜单**：`retiredMenus` 生效后「组织管理」菜单及其 `role_menu` 授权被清理。
6. 新建租户自动创建**同名根部门**（种子 + 运行时双路径幂等）；内置管理员归属根部门（`primary`）。
7. 业务回归用例全绿：树构建、移动环路拒绝、深度上限、级联删除、主部门唯一、负责人唯一、子树成员聚合、用户/服务账号创建与更新归属、按部门筛选用户。

---

## 8. 附录 A：符号全量表（执行清单）

**含 `organization` 词根的标识符**（按出现频次降序，实测）：
`OrganizationID`、`OrganizationEntity`、`organizationID`、`OrganizationUserEntity`、`OrganizationBaseInfo`、`organizations`、`OrganizationIDs`、`organization`、`NewOrganizationUserDao`、`NewOrganizationDao`、`organizationSvc`、`OrganizationCreateReq`、`OrganizationUserCond`、`OrganizationNotExistError`、`svcorganization`、`OrganizationUserCreateReq`、`organizationCtr`、`organizationUserSvc`、`OrganizationUpdateError`、`OrganizationCond`、`organizationUserCtr`、`svcorganizationuser`、`UserOrganizationItem`、`OrganizationTreeItem`、`Organizations`、`organizationIDs`、`OrganizationChildrenReq`、`OrganizationUpdateReq`、`OrganizationDeleteError`、`OrganizationCreateError`、`organizationVisibleToTenant`、`OrganizationUserUpdateReq`、`OrganizationChildrenResp`、`UserOrganizationRequiredError`、`OrganizationUserUpdateError`、`OrganizationUserPageListResp`、`OrganizationUserPageListItem`、`OrganizationUserCreateResp`、`OrganizationUserCreateError`、`OrganizationTreeResp`、`OrganizationStatusReq`、`OrganizationCreateResp`、`OrganizationChildItem`、`OrganizationGetPageListError`、`OrganizationDeleteReq`、`OrganizationUserPageListReq`、`OrganizationUserLeaderConflictError`、`OrganizationUserDeleteError`、`OrganizationTreeReq`、`loadUserOrganizations`、`seedRootOrganization`、`OrganizationSvc`、`OrganizationUserSvc`、`OrganizationUserNotExistError`、`OrganizationUserEntityList`、`OrganizationUserDeleteReq`、`OrganizationUserDao`、`OrganizationEntityList`、`OrganizationDao`、`OrganizationGetDetailError`、`OrganizationName`、`OrganizationCtr`、`OrganizationUserCtr`、`OrganizationUserGetPageListError`、`organizationRouter`、`organization_user`、`organization_id`、`ProvisionMenuCodes` 内的 `"organization"`、`"pages/organization"`、`"/organization"`

**`Org` 前缀 / 树字段**：
`OrgPath`、`OrgDepth`、`MaxOrgDepth`、`OrgNodeStatus`、`OrgNodeStatusActive`、`OrgNodeStatusInactive`、`OrgUserRelationType`、`OrgUserRelationPrimary`、`OrgUserRelationSecondary`、`OrgUserRelationLeader`、`org_path`、`org_depth`、`org_id`(日志/键名)、`ErrMultiplePrimaryOrg`、`ErrOrgLeaderConflict`、`CreateWithRootOrg`

**测试函数名**（示例，全量按前缀替换）：
`TestOrganizationCreateRootAndChildPaths`、`TestOrganizationMoveCascadesPathAndRejectsCycle`、`TestOrganizationDeleteRejectsWithChildrenAndCascade`、`TestOrganizationUserMemberSingletonAndValidTypes`、`TestUserCreateRequiresOrganization`、`TestUserCreateWithOrganizations`、`TestUserDetailWithOrganizationsAndRoles`、`TestUserUpdateOrganizations`、`TestUserPageListOrganizationFilter` → 对应 `Department*`

---

## 9. 附录 B：一次性 SQL（**仅救急/保全数据**，不进代码、不进启动流程）

```sql
-- 表
ALTER TABLE organization      RENAME TO department;
ALTER TABLE organization_user RENAME TO department_user;

-- 列
ALTER TABLE department_user RENAME COLUMN organization_id TO department_id;
ALTER TABLE department      RENAME COLUMN org_path        TO dept_path;   -- 采纳 D1 时
ALTER TABLE department      RENAME COLUMN org_depth       TO dept_depth;  -- 采纳 D1 时

-- 索引（⚠️ 仅当历史库确实按 organization-container-redesign.md §3.5 的 SQL 手工建过索引时才需要；
--        若索引不存在会报错，跳过即可 —— 采纳 D5 后由 GORM tag + AutoMigrate 直接创建新索引）
ALTER INDEX uk_org_user            RENAME TO uk_dept_user;
ALTER INDEX idx_organization_parent RENAME TO idx_department_parent;
ALTER INDEX idx_organization_path   RENAME TO idx_department_path;

-- 菜单（或交给启动时的 retiredMenus 自动清理，二选一，勿重复操作）
UPDATE menu SET code = 'department', path = '/department',
                component = 'pages/department', name = '部门管理'
WHERE code = 'organization';
```

> 注：`retiredMenus` 会把旧 code 的菜单**软删除**；若要保留同一行菜单（保持既有 role_menu 授权 ID 不变），就用上面的 `UPDATE`，并**不要**在 `retiredMenus` 中登记 `organization`。二选一需在实施时明确（**默认走 retiredMenus**，与项目既有约定一致）。

---

## 10. 实施记录（2026-09-12）

### 10.1 与本文的偏差

| 项 | 计划 | 实际 |
|---|---|---|
| D2 单值主部门 | 仅改字段名 | 顺带**删除了 `ErrMultiplePrimaryDept` 哨兵与 `len(...) > 1` 校验**——多主部门在类型层面已不可表达，保留该校验即为死代码；`pkg/iam/user/user_test.go` 的 `TestCreate_MultiplePrimaryOrgRejected` 改写为 `TestCreate_PrimaryDepartmentSingleRow`（断言 primary 关系恒为 0 或 1 行） |
| §9 菜单处置 | retiredMenus / 一次性 SQL 二选一 | 走 **retiredMenus**：`retired_menu.go` 登记 `{appCode: appCodeTenantAdmin, menuCode: "organization"}`，**未**使用 §9 的 `UPDATE` |
| §7.1 允许残留 | 三处 | 追加：`backend/log/**`（历史运行日志）、`.worktrees/**`、`.pnpm-store/**`、`.superpowers/sdd/**`（历史评审 diff）、`pkg/seed/retired_menu.go` 的旧 code 登记、`login-web` 登录页「组织/租户」（租户级语义，非部门容器） |
| D5 索引 | GORM tag + AutoMigrate | `idx_department_parent`(tenant_id,parent_id)、`idx_department_path`(dept_path) 走 gorm tag；`uk_dept_user` 因需**部分唯一索引**（`WHERE deleted_at IS NULL`，否则软删除后无法重建同一关系）写在 `pkg/iam/model/automigrate.go` 的 `partialUniqueIndexes` |
| 清理 | — | 顺带删除 `apps/auth/internal/dto/dtoauth/request.go` 中零引用的 `AssignDepartmentsReq`（唯一残留的 organization 词根 DTO） |

### 10.2 验证结果

| 验证 | 命令 | 结果 |
|---|---|---|
| 编译（含测试） | `go vet ./...` × 5 模块（auth/gateway/platformadmin/tenantadmin/pkg） | 通过 |
| 单元测试 | `go test ./...` × 5 模块 | 全绿 |
| 前端 | `pnpm typecheck`；`pnpm build:tenant` | 通过（3240 modules，dist 产出正常） |
| Swagger | `make swag APP=tenantadmin` | 重新生成，`tenantadmin_docs.go` 0 处 `organization` |
| 零残留 | §7.1 的 grep（受管路径） | 仅剩上述「允许残留」位 |
| e2e 文案 | `e2e/` 4 个文件的 `组织管理` 断言 | 改为 `部门管理`（菜单名变更，否则 e2e 必挂） |

### 10.3 受影响的自然语言口径

`organization` → `department` 的替换同时带动中文口径：`组织架构` → `部门架构`、`组织管理` → `部门管理`、`组织成员` → `部门成员`（错误码中文文案 5 条已同步：创建/删除/查看/不存在/修改部门成员），`根组织` → `根部门`。`docs/design/glossary.md` 新增「二、部门与归属」章节（部门 / 部门关系 / 根部门），并把租户定义里的「客户/组织边界」改为「客户边界」以免与新容器词义混淆。

### 10.4 二次扫尾：`orgXXX` / `xxxOrg` 类残留

首轮替换按"整词/整段"匹配，漏掉了**复合标识符里的 `Org` 片段**（`\borganization\b` 之类匹配不到 `setOrgTree`、`newOrgGinCtx`）。二次扫尾改用**词元清单法**（对全部 tracked 文件抽出含 `org` 的标识符再人工判定），共修 20 处：

| 类别 | 修前 → 修后 |
|---|---|
| Go 变量/方法 | `newOrgGinCtx`→`newDeptGinCtx`（4 个测试文件）、`ensureOrgLeaderUnique`→`ensureDeptLeaderUnique`（`svctenant` 内部**同名第二份**实现，含日志前缀）、`childOrgIDSet`→`childDeptIDSet`、`subOrgIDs`→`subDeptIDs`、`oldOrgs`→`oldDepts`、`primaryOrgOfUser`→`primaryDeptOfUser`、`userOrgMap`→`userDeptMap` |
| Go 测试名 | `TestUserCreateWithLeaderOrgs`→`…LeaderDepts`、`TestMachineUserOrgLifecycleAndGuards`→`…DeptLifecycleAndGuards`、`TestCreate_NewPersonWithOrgRelations`→`…WithDeptRelations`、`TestCreate_LeaderSameOrgAllowedForSingleLeader`→`…SameDeptAllowed…` |
| 前端 | `TenantUserOrgUpdate`→`TenantUserDepartmentUpdate`（types + api）、`setOrgTree`/`setOrgBefore`/`setOrgList`→`setDept*`、**CSS 选择器与 DOM id 不一致的实伤**：选择器已改 `#dept-tree-card` 而 `Card` 的 `id` 仍为 `org-tree-card`（部门树样式会失效），二者已统一 |
| 文档 | `tenant-admin-console-redesign.md`：`orgPath`→`deptPath`；`role-dimension-redesign.md`：TargetType 枚举 `org`→`dept`；`self-registration-redesign.md`：`OrgNodeID`→`DeptNodeID`、`registerOrg`→`createTenant`；`tenant-admin-provisioning-design-20260912.md`：`CreateWithRootOrg(Req)`→`CreateWithRootDept(Req)`、`RootOrg`→`RootDept`、`ErrOrgLeaderConflict`→`ErrDeptLeaderConflict`，并同步 D2 口径（`PrimaryDepartmentID string`、删除 `ErrMultiplePrimaryOrg` 边界说明、覆盖规则与调用示例）；`README.md`/`README.zh.md`：`(department/orgRole)`→`(department/user/role)` |

**判定为误报/故意保留**（不再处理）：`golang.org`/`httpbingo.org`/`mermaid.js.org`/`owasp.org`/`keycloak.org`/`cwe.mitre.org` 等域名、`forged` 与 `Connector*Error`（子串巧合）、随机密钥串、`pkg/seed/retired_menu.go` 的旧 code 登记、`seed_test.go` 的 `grp-org` 历史断言、`zitadel register/org` 与 `DisallowPublicOrgRegistration`（外部系统专有名词）。

扫尾后复验：`gofmt` 干净、5 模块 `go vet` + `go test` 全绿、前端 `pnpm typecheck` + `pnpm build:tenant` 通过。

> 可复用的检查命令（词元清单法，比整词 grep 更适合改名收尾）：
> ```bash
> git ls-files -z '*.go' '*.ts' '*.tsx' '*.sql' '*.yaml' | xargs -0 \
>   perl -ne 'while (/([A-Za-z_][A-Za-z0-9_]*)/g) { print "$1\n" if $1 =~ /org/i }' \
>   | sort | uniq -c | sort -rn
> ```
