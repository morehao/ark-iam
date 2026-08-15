# 组织架构容器化设计（organization = 租户下用户容器）

> 状态：已落地（feat/organization-container 分支）
> 涉及：`organization` / `organization_user` 两张表重构为"租户下用户容器"；删除 `department` / `user_department` / `organization_role` / `organization_role_user` 四张表；tenantadmin 组织树 API 重建；platformadmin 部门管理下线（组织树只读）；平台管理应用编码 `admin` → `platform-admin`；`application_client.client_id` → `code`。项目处于开发期，无兼容性顾虑，直接改 + 重置开发库。

## 1. 背景与目标

### 1.1 现状问题

当前"组织架构域"共有 6 张表，存在四个问题：

| 表 | 问题 |
|---|---|
| `organization`（扁平） | 无层级；`is_mfa_required` / `custom_data` 无消费方（纯占位） |
| `department`（树） | 与 organization 语义重叠（都是"用户分组"），两套模型互不关联 |
| `organization_user` / `user_department` | 成员归属拆在两张表，归属语义分裂 |
| `organization_role` / `organization_role_user` | "组织节点级角色"，全库无消费方，本质是节点级授权，不属于归属模型 |

根本矛盾：**IAM 试图为"组织/部门"预设业务形态，而这恰是各家租户差异最大的地方**（公司、学校、医院、项目组、班级……）。

### 1.2 核心定位（本文档的契约）

> **租户可以有类型（公司/学校/医院……），但无论什么租户，下面一定有组织架构。**
> **IAM 只维护"组织架构"这个通用结构能力，不解释节点是部门还是班级。**

- **IAM 只做结构**：树形层级 + 成员归属（所有组织形态的公共子集）；
- **业务只做语义**：节点类型（部门/项目组/班级）、职位、职级、业务属性全部由业务侧承载；
- **`organization` 的定位一句话**：**租户下用户的容器**（组织节点树 + 成员）。

### 1.3 设计原则

1. **通用 ≠ 万能字段**：核心表只放"结构 + 归属"字段（`custom_data` 已移除，见 §6.2），业务差异走业务侧独立表（`project_group(organization_id, budget, ...)`）。
2. **权威与派生分离**：`parent_id` 是唯一事实来源，`org_path` / `org_depth` 是派生物化字段，服务层事务内同步维护，并提供一次性重建函数兜底。
3. **租户隔离零成本**：沿用 `tenant_id` 冗余列 + `pkg/dbclient` 租户过滤插件，业务侧新表自动获得同样隔离。
4. **高频读、低频写**：路径展示（面包屑/树/子树成员聚合）是高频，节点移动是低频——物化路径换取 O(1) 前缀查询。
5. **枚举一律字符串、布尔一律 `bool`（全系统约束）**：数据库存储的枚举值（状态/类型等）统一字符串常量（如 `status = "active"`），禁止 int 型字典；布尔值统一 Go `bool` 类型（DB `boolean`，默认 `false`），**禁止 `int8` 0/1 表达布尔**——`0` 是 Go 零值，语义模糊（JSON 输出 `0`、前端被迫写 `v === 1` 魔法数字、与"未设置"无法区分）。本条同时**修正** `string-id-pg-automigrate-seed.md` §2 中"布尔/枚举类 int8 用 smallint"的旧约定。

## 2. 数据模型（两张表）

### 2.1 `organization`：组织节点树

```go
const TableNameOrganization = "organization"

type OrganizationEntity struct {
    gormdao.BaseEntity
    TenantID   string          `gorm:"column:tenant_id;type:varchar(36);not null;default:'';comment:租户id" json:"tenantID"`
    ParentID   string          `gorm:"column:parent_id;type:varchar(36);not null;default:'';comment:父节点ID,空为根节点" json:"parentID"`
    OrgPath    string          `gorm:"column:org_path;type:varchar(1024);not null;default:'';comment:祖先链路径,含自身,如 /rootID/midID/nodeID" json:"orgPath"`
    OrgDepth   int             `gorm:"column:org_depth;type:int;not null;default:1;comment:节点深度,根=1" json:"orgDepth"`
    Name       string          `gorm:"column:name;type:varchar(128);not null;default:'';comment:组织名称" json:"name"`
    Code       string          `gorm:"column:code;type:varchar(64);not null;default:'';comment:组织编码(租户内唯一,可空,外部系统同步用)" json:"code"`
    Sort      int    `gorm:"column:sort;type:int;not null;default:0;comment:同级排序" json:"sort"`
    Status    string `gorm:"column:status;type:varchar(32);not null;default:'active';comment:状态(字符串枚举)" json:"status"`
    CreatedBy string `gorm:"column:created_by;type:varchar(36);not null;default:'';comment:创建人id" json:"createdBy"`
    UpdatedBy  string          `gorm:"column:updated_by;type:varchar(36);not null;default:'';comment:更新人id" json:"updatedBy"`
    DeletedBy  string          `gorm:"column:deleted_by;type:varchar(36);not null;default:'';comment:删除人id" json:"deletedBy"`
}
```

**与旧表差异**：新增 `parent_id`（树化）、`org_path` / `org_depth`（物化路径）、`code` / `sort` / `status`（对齐旧 department 的能力）；删除 `is_mfa_required`（MFA 是认证策略，归租户级/system 配置，不属组织架构）与 `custom_data`（无消费方，需要时再加；业务扩展走业务侧关联表，见 §6.2）。

**表级常量与字典**（沿用现有规范，`model/organization.go` 内定义；枚举一律字符串，全系统约束）：

```go
// 节点状态枚举（字符串，禁止 int 字典）
type OrgNodeStatus string
const (
    OrgNodeStatusActive   OrgNodeStatus = "active"   // 启用
    OrgNodeStatusInactive OrgNodeStatus = "inactive" // 停用
)

const MaxOrgDepth = 10 // 深度上限，防病态树
```

### 2.2 `organization_user`：组织关系（用户 ↔ 组织节点，互斥关系类型 + 主归属属性）

```go
const TableNameOrganizationUser = "organization_user"

// 关系类型枚举（字符串，全系统约束）——"用户↔组织节点"之间是多态关系，
// 枚举值互斥纯净：member（归属）与 leader（负责）是不同关系种类
type OrgUserRelationType string
const (
    OrgUserRelationMember OrgUserRelationType = "member" // 归属（成员）
    OrgUserRelationLeader OrgUserRelationType = "leader" // 负责人（独立于归属，不要求同时是成员）
    // admin 等其他关系种类由业务按需扩展常量，无需改表
)

type OrganizationUserEntity struct {
    gormdao.BaseEntity
    TenantID       string `gorm:"column:tenant_id;type:varchar(36);not null;default:'';comment:租户id" json:"tenantID"`
    OrganizationID string `gorm:"column:organization_id;type:varchar(36);not null;default:'';comment:组织节点ID" json:"organizationID"`
    UserID         string `gorm:"column:user_id;type:varchar(36);not null;default:'';comment:用户ID" json:"userID"`
    RelationType   string `gorm:"column:relation_type;type:varchar(32);not null;default:'member';comment:关系类型(字符串枚举)" json:"relationType"`
    IsPrimary      bool   `gorm:"column:is_primary;type:boolean;not null;default:false;comment:是否主归属(仅member关系可置位)" json:"isPrimary"`
    CreatedBy      string `gorm:"column:created_by;type:varchar(36);not null;default:'';comment:创建人id" json:"createdBy"`
    UpdatedBy      string `gorm:"column:updated_by;type:varchar(36);not null;default:'';comment:更新人id" json:"updatedBy"`
    DeletedBy      string `gorm:"column:deleted_by;type:varchar(36);not null;default:'';comment:删除人id" json:"deletedBy"`
}
```

**语义要点**：

| 场景 | 表达 |
|---|---|
| 跨部门 | 一个 user 多行（`relation_type=member`，一个 `organization_id`） |
| 主部门 | `is_primary=true` 一行（仅 `member` 关系可置位）；租户内每用户**至多 1 行**（服务层校验 + 部分唯一索引 `(tenant_id, user_id) WHERE is_primary` 双保险） |
| 负责人 | `relation_type=leader` 一行；与归属是**独立关系**（负责人不必是成员）；一节点可多负责人（正/副职），由业务在展示层定义 |
| 同一用户同一节点多关系 | 多条记录（如 member + leader 各一行），唯一键 `(tenant_id, organization_id, user_id, relation_type)` |
| 查归属 | **单值过滤** `relation_type='member'`（需要时叠加 `is_primary` 过滤主归属），无需关系组 IN |
| 职位 | **不落核心表**——职位是完整业务域（职级/字典/薪酬联动），由各应用自建（如 `project_group` 或业务职位表关联本表），轻量场景由业务侧自行承载 |

> `relation_type` 是**字符串枚举**（互斥关系种类）；`is_primary` 是**布尔值**（Go `bool`，DB `boolean`）——是 member 关系的**属性**而非关系类型（与 `is_leader` 用布尔表达关系类型的错误不同）。换主归属 = 事务内两行 update（旧 `is_primary=false`，目标 `is_primary=true`）。"负责人是否必须同时是成员"属业务规则，核心允许独立，需要强约束的业务在服务层校验。

**索引**（`organization_user`）：

```sql
CREATE UNIQUE INDEX uk_org_user    ON organization_user (tenant_id, organization_id, user_id, relation_type);
-- 主归属唯一：PG / SQLite 均支持部分索引（DB 级兜底，服务层换主事务保证一致）
CREATE UNIQUE INDEX uk_primary_org ON organization_user (tenant_id, user_id) WHERE is_primary;
```

## 3. org_path 物化路径设计

### 3.1 方案对比

| 方案 | 祖先/子树查询 | 插入 | 移动节点 | 存储 | 结论 |
|---|---|---|---|---|---|
| 递归 CTE | 每次递归，深树/高频差（列表页 N+1 灾难） | O(1) | O(1) | 无冗余 | 只适合小树 + 低频查询 |
| **org_path（采纳）** | 前缀匹配，索引可走 | O(1) | 级联改子树 path（单条 UPDATE） | 每行冗余 1 列 | **读多写少场景权衡最优** |
| 闭包表 | 全等值查询，上限最高 | O(深度) 行 | 级联改多行 | N×深度 行 | 极端场景（近静态树 + 极高频聚合）再启用 |
| 嵌套集 | 子树极快 | 重算半棵树 | 重算整棵树 | 2 列 | 动态树不适用 |

**结论**：本项目场景为"租户自管、节点数千级、深度 ≤10、路径展示高频（面包屑/树/成员归属/审计）"，读远多于写；`org_path` 以"移动时级联更新"换"祖先/子树 O(1) 前缀查询"，交换划算。闭包表作为升级路径保留（见 §6.3）。

### 3.2 字段约定

- `org_path`：**含自身的完整祖先链**，`/` 分隔，如 `/rootID/midID/nodeID`；
- `org_depth`：节点深度，根 = 1；
- `parent_id` 为唯一权威，`org_path` / `org_depth` 为派生物化字段。

### 3.3 查询模式（高频）

```sql
-- ① 子树（含自身）：节点 X 的 org_path 为 P_X（'/rootID/.../X'）
WHERE org_path = P_X OR org_path LIKE P_X || '/%'        -- 前缀匹配，走 text_pattern_ops 索引

-- ② 子树（不含自身）
WHERE org_path LIKE P_X || '/%'

-- ③ 面包屑：按 '/' split 取祖先 ID 链，一次 IN 批量查 name，应用层按序重组（零递归）
SELECT id, name FROM organization
WHERE tenant_id = ? AND id IN (...ancestorIDs...)

-- ④ 子树成员聚合（org_path 最大价值）：一次查询拿全子树成员（含子节点归属）
SELECT DISTINCT ou.user_id FROM organization_user ou
JOIN organization o ON ou.organization_id = o.id
WHERE o.tenant_id = ? AND (o.org_path = P_X OR o.org_path LIKE P_X || '/%')
```

### 3.4 写操作（低频，事务内完成）

**创建节点**：`org_path = parent.org_path + '/' + newID`，`org_depth = parent.org_depth + 1`（根节点 `org_path = '/' + newID`，`org_depth = 1`）。

**移动节点 X → 新父 P**（校验通过后，单条 SQL 级联更新整棵子树）：

```sql
-- 前置校验（O(1)，同一事务内）：
--   1) 环路：P.org_path = X.org_path OR P.org_path LIKE X.org_path || '/%'  ⇒ 拒绝（P 是 X 或 X 的后代）
--   2) 深度：P.org_depth + 1 > MaxOrgDepth(10) ⇒ 拒绝

-- 级联更新：
UPDATE organization
SET org_path = replace(org_path, oldPath, newPath),
    org_depth = org_depth - oldDepth + newDepth      -- newDepth = P.org_depth + 1
WHERE org_path = oldPath OR org_path LIKE oldPath || '/%';
-- WHERE 守卫保证 oldPath 出现位置必为前缀且后跟 '/'，replace 不会误伤形似路径
```

**删除节点**：软删除（`deleted_at`），默认拒绝（有子节点 `parent_id = X` 或成员时），`?cascade=1` 显式级联（事务内软删子树 + 解绑成员）。停用节点（`status=0`）可保留在树中，子树查询默认带出、树接口标记停用态。

**环路检测 O(1)**：移动时用新父的 `org_path` 前缀判断，比递归遍历便宜一个量级。

### 3.5 索引与一致性

```sql
CREATE INDEX idx_organization_parent ON organization (tenant_id, parent_id);
CREATE INDEX idx_organization_path  ON organization (org_path text_pattern_ops); -- PG 前缀 LIKE 走索引
-- SQLite（测试库）：普通 btree 索引即支持前缀 LIKE，与 PG 兼容，无需条件编译
```

- **一致性兜底**：提供一次性重建函数（遍历租户树按 `parent_id` 重算 `org_path` / `org_depth`），异常时修复派生物化字段；
- **字段类型**：`org_path varchar(1024)`（36 字符 UUID × 深度 20 留足余量）。

## 4. API 设计

### 4.1 tenantadmin（租户自管为主，`/v1/tenant`）

| 方法 | 路径 | 说明 |
|---|---|---|
| POST | `/v1/tenant/organizations` | 创建节点（body: parentID/name/code/sort/status…） |
| GET | `/v1/tenant/organizations/tree` | 组织树（?name=/keyword= 过滤，含停用态标记） |
| GET | `/v1/tenant/organizations/:organizationID` | 节点详情（含面包屑 ancestors、子节点数、成员数） |
| PUT | `/v1/tenant/organizations/:organizationID` | 全量更新（改 parentID = 移动节点，含环路/深度校验） |
| PATCH | `/v1/tenant/organizations/:organizationID` | 局部更新（如 status 启停用） |
| DELETE | `/v1/tenant/organizations/:organizationID` | 删除（有子/成员默认拒绝，?cascade=1 级联） |
| GET | `/v1/tenant/organizations/:organizationID/users` | 节点关系分页（?relationType=&isPrimary=&userName=） |
| POST | `/v1/tenant/organizations/:organizationID/users` | 添加关系 {userID, relationType, isPrimary}（isPrimary 仅 member 可置位） |
| PUT | `/v1/tenant/organizations/:organizationID/users/:userID` | 更新关系（relationType/isPrimary，含换主事务） |
| DELETE | `/v1/tenant/organizations/:organizationID/users/:userID` | 移除成员 |
| GET | `/v1/tenant/organizations/:organizationID/users/descendants` | 子树成员聚合（去重，跨子节点） |
| GET | `/v1/tenant/users/:userID/organizations` | 用户归属列表（含主组织 + 各节点面包屑） |
| PUT | `/v1/tenant/users/:userID/organizations` | 批量替换用户归属（全量替换集合，走路由规范 PUT 语义） |

### 4.2 platformadmin（只读视角，`/v1/platform`）

| 方法 | 路径 | 说明 |
|---|---|---|
| GET | `/v1/platform/organizations` | 只读分页（?tenantID= 必填，跨租户排查） |
| GET | `/v1/platform/organizations/tree` | 跨租户只读树（?tenantID= 必填） |
| GET | `/v1/platform/users/:userID/organizations` | 用户归属只读查询（原 `/users/:userID/departments` 迁移） |

> 原 platformadmin 的 `PUT /users/:userID/departments`（批量分配部门）下线，归属维护收敛到 tenantadmin；platformadmin 不再写组织数据。

### 4.3 路由注册

按现有规范：tenantadmin 用 `ginserver.NewRouterGroups(engine, "tenant", ...)`，platformadmin 用 `"platform"`；资源名 `organizations` 跨应用复用，由服务标识段区分归属。

## 5. 迁移路径

### 5.1 表结构迁移（存量数据）

```sql
-- ① organization：树化 + 去 MFA
ALTER TABLE organization ADD COLUMN parent_id  varchar(36)  NOT NULL DEFAULT '';
ALTER TABLE organization ADD COLUMN org_path   varchar(1024) NOT NULL DEFAULT '';
ALTER TABLE organization ADD COLUMN org_depth  int          NOT NULL DEFAULT 1;
ALTER TABLE organization ADD COLUMN code       varchar(64)  NOT NULL DEFAULT '';
ALTER TABLE organization ADD COLUMN sort       int          NOT NULL DEFAULT 0;
ALTER TABLE organization ADD COLUMN status     varchar(32)  NOT NULL DEFAULT 'active';
ALTER TABLE organization DROP COLUMN is_mfa_required;
ALTER TABLE organization DROP COLUMN custom_data;   -- 无消费方，需要时再加
UPDATE organization SET org_path = '/' || id WHERE org_path = '';  -- 存量行视为根节点
CREATE INDEX idx_organization_parent ON organization (tenant_id, parent_id);
CREATE INDEX idx_organization_path  ON organization (org_path text_pattern_ops);

-- ② organization_user：关系类型（互斥枚举）+ 主归属（布尔属性，禁止 smallint 0/1）
ALTER TABLE organization_user ADD COLUMN relation_type varchar(32) NOT NULL DEFAULT 'member';
ALTER TABLE organization_user ADD COLUMN is_primary    boolean     NOT NULL DEFAULT false;
CREATE UNIQUE INDEX uk_org_user     ON organization_user (tenant_id, organization_id, user_id, relation_type);
CREATE UNIQUE INDEX uk_primary_org  ON organization_user (tenant_id, user_id) WHERE is_primary;
```

### 5.2 数据迁移（department → organization）

```sql
-- ① 部门行 → 组织节点（保留原 ID，避免成员关联重映射）
INSERT INTO organization (id, tenant_id, parent_id, org_path, org_depth, name, code, sort, status, created_at, updated_at, deleted_at, created_by, updated_by, deleted_by)
SELECT id, tenant_id, parent_id,
       '/' || id, 1, name, code, sort, 'active', created_at, updated_at, deleted_at, created_by, updated_by, deleted_by
FROM department;
-- ② 重建 org_path / org_depth（按 parent_id 从根遍历；根 = parent_id 为空的行）
-- ③ 用户-部门关联 → 组织关系（relation_type 一律 member，is_primary 直接带过来）
INSERT INTO organization_user (id, tenant_id, organization_id, user_id, relation_type, is_primary, created_at, updated_at, deleted_at, created_by, updated_by, deleted_by)
SELECT id, tenant_id, department_id, user_id, 'member', is_primary, created_at, updated_at, deleted_at, created_by, updated_by, deleted_by
FROM user_department;
-- ④ 删除旧表
DROP TABLE user_department;
DROP TABLE department;
DROP TABLE organization_role_user;
DROP TABLE organization_role;
```

> 若担心 organization 与 department 存量 ID 冲突（均为 UUID v7，概率极低），可改为生成新 ID + 维护映射表，代价是重映射成员关联。开发阶段建议直接复用旧 ID。

### 5.3 根组织节点（创建租户即创建，双路径）

**保证每个租户必有同名根组织节点**（`Name = tenant.Name`，`Code = tenant.Code`，`org_path = '/' + id`，`org_depth = 1`，`status = "active"`）：

1. **种子初始化**：`seedRootDepartment` → `seedRootOrganization`，平台种子租户启动时幂等创建（按 `tenant_id + parent_id = ''` 查重）；
2. **运行时创建租户**：`CreateTenantAsOwner` 事务内，创建租户后**同步插入同名根组织节点**（与现有"创建租户自动建根部门"行为一致，`create_tenant_as_owner_test.go` 断言同步迁移为根组织）。

### 5.4 平台管理应用编码调整（`admin` → `platform-admin`）

现状：平台管理（platformadmin）应用编码为 `admin`（`seed.go` `appCodeAdmin = "admin"`），与角色编码 `admin`（管理员）语义撞名，表达有歧义。统一调整为 `platform-admin`（与租户自服务 `tenant-admin` 对称）：

| 编码项 | 现值 | 调整后 |
|---|---|---|
| 应用编码（`application.code`） | `admin` | `platform-admin` |
| 资源标识（`resource.indicator`） | `urn:ark:iam:admin` | `urn:ark:iam:platform-admin` |
| 权限/scope 编码（`scope.name`） | `admin:user:read` 等 12 个 | `platform-admin:user:read` 等 |
| 菜单 permission 字段 | `admin:*` | `platform-admin:*` |
| 角色编码（`role.code`） | `admin` / `user` / `guest` | **保留**（"管理员/普通用户/访客"语义本身清晰） |
| 前端 OIDC client_id | `platform-admin-web` | **不变**（已是 platform-admin 前缀） |

> 本项目处于开发期，**无兼容性顾虑**：直接改常量并重置开发库即可（种子按 `application.code` 幂等查重，编码变更在已初始化库中会视为新应用，重置即消解）。

### 5.5 OAuth 客户端标识字段命名（`application_client.client_id` → `code`）

`application_client.client_id` 全库唯一（`uniqueIndex`），是 OAuth 协议对外的客户端**业务标识**，与 `application.code` / `tenant.code` / `role.code` 同构——按项目命名体系应命名为 `code`；`client_id` 是 OAuth2/OIDC 协议术语（RFC 6749 参数名、token claim、introspection），**只保留在协议边界**：

| 层 | 现状 | 处理 |
|---|---|---|
| 实体/DB | `ApplicationClientEntity.ClientID`（column `client_id`，uniqueIndex） | → `Code`（column `code`） |
| DAO 条件 | `ApplicationClientCond.ClientID` | → `Code` |
| OIDC 协议适配（**唯一映射点**） | `OIDCClient.GetID()` 返回 `clientEntity.ClientID`（`oidcop/client.go`） | → 返回 `clientEntity.Code` |
| 协议查询 | `GetClientByClientID` / `AuthorizeClientIDSecret`（zitadel op 接口签名） | 接口名/参数**保留** `clientID`（协议语境），内部按 `code` 查询 |
| token claim / introspection / `audit_log.client_id` | `client_id` | **保留**（记录的是协议身份，非实体字段） |
| 外键 | `application_client_secret.application_client_id`、`refresh_token.application_client_id` | **不动**（指向内部主键 `id`） |
| API DTO | `dtoapplicationclient` 响应 `clientID` JSON 字段 | → `code`（与 application 详情返回 `code` 的展示一致） |

> 顺带消除一个隐患：改名后，`application_client` 领域内不再同时存在 `client_id`（业务码）与 `application_client_id`（外键）两个易混淆命名。

## 6. 边界与演进

### 6.1 明确不做（v1）

| 事项 | 理由 | 承载方 |
|---|---|---|
| 节点类型（type） | IAM 不该理解"部门/项目组/班级"——各家分类不同 | 业务侧建表（`tenant_node_type`）或业务属性表 |
| 组织节点级角色/授权 | 属授权域，非关系模型 | v2 按 role 维度设计（`org_node_role_user`）；节点管理类关系（如 admin）可先用 `relation_type` 扩展常量承载 |
| 职位/职级 | 完整业务域（职级序列/职称/薪酬联动） | 各应用自建职位表，关联 `organization_user` |
| 节点级 MFA | MFA 是认证策略 | 租户级 `system` 配置 |
| 关系专属业务属性（leader 正副职、admin 管理范围等） | 业务语义，非结构事实 | 业务侧承载；**核心只承诺 `relation_type`（互斥关系种类）+ `is_primary`（唯一结构属性）两个维度**，新增关系类型通常无需加字段；结构性属性若膨胀，走 §6.3 关系属性演进 |

### 6.2 类型与业务扩展的标准姿势

组织节点表**不承载任何业务扩展字段**（`custom_data` 已移除，需要时再加），业务差异一律走业务侧关联表：

1. **业务属性**：业务 app 建表 `project_group(organization_id, budget, deadline, ...)`，关联到组织节点；
2. **类型化查询**：业务侧建索引表 `tenant_node_type(node_id, type)`，按自己的分类体系维护——不同客户可共存互不污染；
3. **何时再加 `custom_data`**：出现"多个业务都只想存少量无结构扩展、不值得各自建表"的真实需求时，再在 `organization` 上加回 JSON 列（届时同步评估 PG JSONB 查询/索引）。

### 6.3 升级路径

**① 闭包表**：将来若出现"全租户人事报表等近静态树 + 极高频跨组织聚合"，再补 `organization_closure(ancestor_id, descendant_id, depth)`，与 `org_path` 共存（写入时双写），无需推翻本文设计。

**② 关系属性演进**：`organization_user` 当前只承诺 `relation_type` + `is_primary` 两个维度（§6.1）。若未来出现"多种关系类型的结构级专属属性"（如 leader 需要"是否代理"、admin 需要"管理范围"）：
- **少数（1~2 个）**：按 `is_primary` 同模式加列即可（边际成本 = 一列 + 一个 DTO 字段，只影响该关系类型的代码）；
- **多数（开始膨胀）**：引入关系属性表 `organization_user_attr(relation_id, attr_key, attr_value)`（或 JSON 列）承载"关系级属性"，`organization_user` 核心结构不变——与闭包表同理，**只做加法、不推翻**。

## 7. 影响面与改动清单

### 7.1 后端

**删除（23 个文件 + 4 张表）**

- `pkg/iam/model`：`department.go`、`user_department.go`、`organization_role.go`、`organization_role_user.go`
- `pkg/iam/dao`：`department.go`、`user_department.go`、`organization_role.go`、`organization_role_user.go`
- `pkg/iam/object`：objtenant 中 department 相关对象
- `apps/platformadmin`：`svctenant/department.go`、`ctrtenant/department.go`、`router` 中 departmentRouter、`dto/dtotenant` department 相关、用户部门分配（`AssignDepartments` / `GetUserDepartmentByUser`）
- `apps/tenantadmin`：`svctenant/organization_role*.go`、`ctrtenant/organization_role*.go`、`dto/dtotenant` organization_role 相关
- `pkg/code`：100130~100153（org role）、100400+（department）、104100+（user_department）错误码

**修改**

- `pkg/iam/model/organization.go`：加 `parent_id/org_path/org_depth/code/sort`，`status` 为字符串枚举（`"active"/"inactive"`）；去 `is_mfa_required` 与 `custom_data`
- `pkg/iam/model/organization_user.go`：重写为**关系表**——`relation_type`（字符串枚举 member/leader，可扩展，**互斥**）+ `is_primary`（布尔 `bool`，仅 member 可置位）
- **全系统布尔字段改造（§1.3 原则 5）**：21 处 `int8` 0/1 布尔 → Go `bool`——`tenant/user/person.is_suspended`、`user.is_owner`、`connector.allow_auto_create_user / allow_account_link / sync_profile / enable_token_storage`、`application.is_system`、`menu.hidden / external_link / keep_alive`、`role.is_default`、`resource.is_default`、`domain.is_verified`、`application_client.require_pkce / require_auth_time / is_third_party / is_system`（另 2 处 `is_mfa_required` / `user_department.is_primary` 随旧表删除）
- `pkg/iam/model/automigrate.go`：实体清单（删 4 留 2）
- `pkg/iam/dao/organization.go`：Cond 增 `ParentID/OrgPath/Status/Code`；新增树方法（GetTreeByTenant / GetDescendants / MoveNode 事务 / 环路与深度校验）
- `pkg/seed/seed.go`：`seedRootDepartment` → `seedRootOrganization`；`appCodeAdmin = "admin"` → `"platform-admin"`（含资源标识、12 个 scope、菜单 permission 前缀，见 §5.4）；菜单（tenantadmin 4 个组织菜单 → 1 个「组织架构」；platformadmin 删部门菜单）
- `pkg/seed/seed_pg_test.go`：删表顺序列表
- `apps/platformadmin/internal/service/svctenant/tenant.go`：`CreateTenantAsOwner` 事务内创建**同名根组织节点**（替代原根部门逻辑）
- `pkg/iam/model/application_client.go` / `pkg/iam/dao/application_client.go`：`client_id` → `code`（见 §5.5）
- `apps/auth/internal/service/oidcop/client.go`：`OIDCClient.GetID()` 返回 `Code`（协议映射点）
- `apps/platformadmin` `svcapplicationclient` / `dtoapplicationclient`：生成函数（`generateClientID` → 生成 code）与 DTO `clientID` JSON 字段 → `code`
- `pkg/seed/seed.go`：OAuth 客户端种子 `clientID` → `code`
- 测试：`create_tenant_as_owner_test.go` 根部门断言 → 根组织断言

**新增/重写**

- `svctenant/organization.go`：Tree / Create / Update（含移动）/ Delete（级联）/ Detail（含面包屑）
- `svctenant/organization_user.go`：AddRelation / RemoveRelation / UpdateRelation / PageList / SubtreeMembers（按 `relation_type` 单值过滤；`is_primary` 仅 `member` 可置位，服务层校验；换主归属 = 事务内两行 update，部分唯一索引兜底）
- 用户归属：`GET/PUT /v1/tenant/users/:userID/organizations`
- platformadmin 只读：`GET /v1/platform/organizations`、`GET /v1/platform/organizations/tree`、`GET /v1/platform/users/:userID/organizations`

### 7.2 前端

- **tenantadmin**：4 个组织页面 → 1 个「组织架构」页（左侧树 + 成员抽屉）；`App.tsx` 菜单/路由/默认落地页；`api/organization.ts`、`types/organization.ts` 重写
- **platformadmin**：删除部门菜单/页面/API；用户详情部门归属改只读组织归属或移除
- **packages/types**：`OrganizationItem` 字段更新（parentID/orgPath/orgDepth/code/sort/status）

### 7.3 文档与测试

- `api-reference.md`：tenant/platform 端点更新
- `system-design.md`：ER 图、表清单（6 张 → 2 张）、应用职责、演进方向（删除"部门/组织与角色联动"旧条目，更新 MFA 条目）
- `glossary.md`：组织/部门词条合并为"组织（用户容器）"
- 测试：删除 org role / relation delete 用例；新增树构建、子树查询、移动环路、级联删除、主组织唯一性、org_path 一致性用例；platformadmin 部门用例迁移为组织用例

## 8. 验收标准

1. `go test ./...` 全绿，`make lint` 通过；
2. tenantadmin 组织树完整可用：建树、移动（环路拒绝）、级联删除、**关系管理（relation_type 增删改、is_primary 仅 member 可置位、主组织唯一由部分唯一索引兜底）**、子树成员聚合；
3. platformadmin 组织树只读、无写接口；
4. 新租户创建自动种**同名**根组织节点（种子与运行时双路径，幂等）；
5. 存量库迁移脚本在 PG 实测通过（`//go:build pg` 测试覆盖）；
6. 平台管理应用编码重命名（§5.4）后 seed 幂等通过，scope/角色/菜单权限引用一致；
7. `application_client.client_id` → `code`（§5.5）后，OIDC 登录/令牌/登出链路（含协议一致性测试）全绿；
8. 全系统布尔字段 `int8 0/1` → `bool`（§1.3 原则 5）改造后测试全绿，JSON 输出 `true/false`（前端不再有 `v === 1` 魔法数字）。
