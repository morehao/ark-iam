# 角色表两维度调整方案：内置/自定义 × 管理员/普通成员

> ⚠️ **本方案已废弃（superseded）**。下文的核心设计——「业务权限 scope（`resource`/`scope`/`role_scope`）+ 授权驱动推导 `admin_level`」——已在后续调整中**彻底移除**：业务权限点不再由 IAM 承载，`admin_level` 改为角色的**显式能力标签**（由租户管理员直接勾选 none/basic/super），权限职责回归业务应用，IAM/auth 仅做身份。若需追溯当时的拆解依据，保留本文作为历史记录；**勿再按本方案的 scope 授权驱动落地**。
>
> 说明：本文在此前的会话中已记录了「内置/自定义 × 管理员/普通成员」两维度的落地（`role.source`/`role.admin_level` 字段保留、`role.type` 移除），其中 `source` 与 `admin_level` 两项仍有效；**仅「由 scope 推导 admin_level」的授权驱动部分已废弃**。

> 关联需求：角色表需要体现「内置角色/自定义角色」与「管理员角色/普通成员角色」两个维度。
> 状态：设计方案（v2，已按反馈修订）。约定：**合理性优先，不要求最小改动，允许破坏性调整。**

---

## 零、先理解两个维度，再谈改表

这两个诉求**不是同一层面的问题**，必须分开看待，否则会把模型做歪：

| 维度 | 本质 | 它回答的问题 |
|------|------|-------------|
| **内置 / 自定义** | **数据来源**（谁创建的、可否删改） | 「这个角色账本上记的这笔，是系统送的还是用户自己建的？」 |
| **管理员 / 普通成员** | **能力 / 授权结果**（能做什么） | 「这个用户在当前租户里，权限够不够做系统管理？」 |

第一维是「**账目属性**」，改表时加一个字段就行；第二维是「**能力**」，它**不该由一行数据拍死**，而应该由「用户→角色→授权」推导出来——这就是「授权驱动」。下面专门解释。

---

## 一、授权驱动到底是什么（重点）

先看你系统里已有的数据，它其实**已经是一个 RBAC**：

```
用户 ──┬── role_scope ──▶ scope（权限点）────▶ 资源(如果能在系统级别写/读)
       ├── user ──▶ role ──┬── role_menu ──▶ 菜单（能看哪些页面）
       └── role_scope      └── role_scope ──▶ scope ──▶ OIDC token 里的 scope（真正被执行）
```

seed 里已经写死了这个结构：

```
角色 admin（seed 内置唯一角色）：
  role_scope ⊳ platform-admin:user:write, platform-admin:role:write,
              platform-admin:application:write, platform-admin:resource:write ...   ← 全部写权限
```

**「管理员」的本质是什么？不是它名字叫 admin，而是它的 `role_scope` 里躺着 `:write` 权限点。** 换句话说：**admin 之所以能管系统，是因为它拿到了管理系统的授权，而不是因为它贴了个「管理员」标签。**

「授权驱动」就是指：**判定用户有没有某种能力，一律去查「授权记录」(`role_scope`/`role_menu`)，而不是查角色上的一个字段/标签。**

### 为什么不能靠 `type='Admin'` 这种字段来判定？

假设你在 `role` 上加了 `type`，`admin` 角色 `type='Admin'`，于是代码写 `if role.type == 'Admin'` 才允许系统管理。会出现三个问题：

1. **自定义角色无法成为管理员。** 今天你想加一个「运维」角色，它也要能做系统管理，但 `type` 只有 Admin/User 两档，装不下，要么硬塞 Admin 但语义错误、要么再加枚举。而如果靠授权——只要给「运维」授权 `platform-admin:*:write`，它自动就是管理员，**不用改任何枚举**。
2. **标签与授权不一致会撕裂。** 一旦出现 `type='Admin'` 但 `role_scope` 里没有 `:write`（数据被改、被误配），代码却放行了 → 越权；反过来 `type='User'` 却有 `:write` → 该有的权限被 `type` 挡住。**授权记录是唯一事实来源，标签不是。**
3. **OIDC token 是按 scope 发的。** 最终真正生效的是 token 里的 scope（前端据此渲染权限）。如果你的判定逻辑靠 `type` 而 token 靠 scope，两边会打架。

> 一句话总结授权驱动：**「能不能」是一个求值结果（查授权记录算出来的），不是一个存着的标签。** 存标签只是给人看的（展示），不能当权限判定的依据。
>
> 顺带澄清你可能的担心：授权驱动**不等于**不能展示「管理员」标签。我们仍然可以在列表里显示「此角色是管理员（因为它拿到了系统管理授权）」，只是这个「管理员」是**算出来/推导出来**的，不是写死存的。两者可兼得。

---

## 二、既然可以破坏性调整，推荐怎么落地

在「合理性优先、可破坏性调整」前提下，推荐做**三项正交改造**，把两维都建成干净的第一类模型：

### 2.1 维度一「内置 / 自定义」→ `role.source` 真实字段

这一维确实是账目属性，加字段合理。

```go
const TableNameRole = "role"

type RoleSource string
const (
    RoleSourceBuiltin RoleSource = "builtin" // 系统内置（种子数据），禁删、禁改核心字段
    RoleSourceCustom   RoleSource = "custom" // 租户自定义
)

type RoleEntity struct {
    gormdao.BaseEntity
    TenantID    string `json:"tenantID"`
    AppID       string `json:"appID"`
    Name        string `json:"name"`
    Code        string `json:"code"`
    Description string `json:"description"`
    Source      string `gorm:"column:source; type:varchar(16); not null; default:'custom';" json:"source"` // builtin/custom
    AdminLevel  string `gorm:"column:admin_level; type:varchar(16); not null; default:'none';" json:"adminLevel"` // 系统管理等级，见 2.2
    // ...
}
```

> 语义边界：`source`＝来源/可维护性；`admin_level`＝系统管理等级。两者正交，互不替代。
> 注：早年 `role.is_default`（是否默认授予新成员）已废弃删除 —— 新成员的角色分配由管理员显式选择，不再默认自动授予，故角色表不再有该字段（resource 表的 `is_default` 表示"默认资源"，语义不同、保留）。

**删除/编辑保护**（内置角色）：
- `Delete`：`source==builtin` 拒绝（尤其禁止删掉内置 admin，防止管理能力失控且 seed 幂等不自动重建）。
- `Update`：内置角色禁改 `code/source/admin_level`（名称/描述可改）。

### 2.2 维度二「管理员 / 普通成员」→ `role.admin_level`（授权驱动下的显式断言）

这里的关键是把「能力边界」作为一个**系统级语义**显式建模，但仍**由授权驱动判定**，而不是 `type` 字符串硬编码。

- `role.admin_level`：系统管理**等级**（`none`/`basic`/`super`，单调门槛），作为展示/约束的便捷索引；权限放行一律走 `role_scope` 判定。
- 判定方法（授权驱动）：**允许进入系统管理 = 该角色任一 scope 命中了「系统级写权限」**，例如 `platform-admin:*:write` / `tenant:*:write`。
- 实现：
  ```go
  // 该用户是否具备系统管理能力
  func HasSystemAdminCapability(ctx, tenantID, userID) (bool, error) {
      // 1. user_role → roleIDs
      // 2. role_scope JOIN scope → scope 名集合
      // 3. 存在任一 resource:xxx:write（系统级）=> true
  }
  ```
- `admin_level` 是**授权驱动的投影**（`>= basic` == 具备系统管理能力），用于列表展示/快速筛选；`HasSystemAdminCapability` 才是权限放行的真判定，与 token scope 完全对齐，避免标签与授权漂移。

> 它等价于：`role.admin_level >= basic == (role拥有系统级写scope)`。我们把推导结果物化成一个可索引字段，便于展示和快速筛选，但**它永远是 scope 授权的投影，不独立定义权限**。

### 2.3 破坏性调整：`role.type` 已移除

原 `type`（Admin/User/Guest）仅做展示标签，后端从未用它判定权限。为避免两套语义（`type` vs `role_scope`）并存造成漂移，**已彻底移除 `role.type` 列**：

- 展示以 `source`（来源）+ `admin_level`（算出来的系统管理等级）+ `role_scope`（真实授权）为准。
- 前端角色列表/详情不再渲染「类型」列或表单字段。
- 内置角色保护不再比较 `type`，仅锁定 `code`。
- 破坏性调整一步到位，不留易混淆字段。

### 2.4 迁移注意（存量库）

- 新增列默认 `custom`，会误标存量内置角色。需存量回填：`role.source='builtin' where code='admin'`（按实际内置 code 集，现仅 admin）。
- `admin_level` 回填：用已有的 `role_scope` 推导（admin→super），或按 code 显式指定。
- 删 `type` 列：PG 用 `DROP COLUMN`（AutoMigrate 不自动删列，需显式迁移 + seed 兼容）。

---

## 三、需要同步调整的功能清单

### 后端
| 文件 | 改动 |
|------|------|
| `pkg/iam/model/role.go` | 加 `Source`、`AdminLevel`（+`SysAdminLevel` 常量）；删除 `Type` |
| `pkg/iam/dao/role.go` `RoleCond` | 加 `Source`/`AdminLevel`/`AdminLevelAtLeast` 查询条件 |
| `pkg/seed/seed.go` `seedRoles` | 内置角色设 `source=builtin`、`admin_level`（admin→super）；加存量回填 |
| `svctenant/role.go`（Create/Delete/Update） | 创建固定 `custom`+`admin_level=none`；删除/改键防内置；改权限时重算 `admin_level` |
| `svctenant/user.go` `UpdateRoles` / `UpdateMenus` | 改授权后同步 `admin_level`；防把最后一个内置 admin 拥有者移除 |
| 新增 `HasSystemAdminCapability` | 授权驱动的管理员判定接口（供中间件/`me`） |
| platformadmin/tenantadmin DTO+service | 透出 `source`/`adminLevel`；创建忽略客户端传 `adminLevel`（由授权推导） |

### 前端
| 文件 | 改动 |
|------|------|
| `tenant-admin-web/pages/role/index.tsx` | 表单：内置角色禁编辑/删除；列表/详情展示「来源 + 管理员等级 + 权限」 |
| `platform-admin-web/pages/role/index.tsx` | 展示 `source`/`adminLevel`；内置只读标注 |
| `packages/types` | `RoleItem`/`TenantRoleItem` 补 `source`/`adminLevel` |
| `packages/api` | 透传字段 |

### 测试
- 内置角色删除被拒、改 `code`（核心键）被拒、改 name/desc 放行。
- 自定义角色默认 `custom`+`admin_level=none`；授权 `:write` 后 `admin_level` 自动变为 >=basic。
- `HasSystemAdminCapability`：admin→true、无管理 scope→false、授 write→true。
- 存量回填与删 `type` 列表迁移。

---

## 四、为什么这是「授权驱动」而不是「改挂了」

- **唯一事实来源**是 `role_scope`（/token scope），`admin_level`、前端标签都只是它的**投影/便捷索引**，权限放行永远回到授权记录。
- **加角色不用改枚举**：任何自定义角色只要授予系统级写 scope，就自动成为管理员。产品上要加「运维」这类新角色零成本。
- **消除双源漂移**：废弃 `type`，展示与判定统一指向 `role_scope` + `admin_level` 投影。

---

## 五、菜单可见性改造（按角色驱动 + 隐藏未授权）

### 5.1 需求澄清

用户侧的需求是**「有些菜单需要默认某个用户/某个角色默认看到，方便操作——例如管理员要做用户管理操作，就需要看到相关内容菜单」**，并在此基础上追问「**有些菜单就不应该给普通成员可见**」。相关澄清结论：

- **菜单可见性按「可见等级」（visibility）三级建模**：`public`（所有人）/ `member`（任意普通成员）/ `admin`（仅管理员）——这就是"未授权就隐藏"与"管理员专有硬隔离"的统一表达。
- **不再**以"仅靠 `role_menu` 勾选"作为唯一可见依据：管理专有（`admin`）菜单对普通成员**硬隔离**，即使被误配了授权也仍不可见。
- 不做用户级单独配菜单的 `user_menu` 表（现实中"给某用户单独配菜单不经过角色"极少见，通常通过给用户配一个角色解决）。

### 5.1.1 可见等级建模（本方案定案）

新增 `menu.visibility` 独立字段（与 `menu.type` 节点类型解耦），采用**单调门槛枚举**：

```go
// pkg/iam/model/menu.go ——菜单可见性门槛（单调：等级越高可见范围越窄）
type MenuVisibility string
const (
    MenuVisibilityPublic MenuVisibility = "public" // 所有人可见（无门槛）
    MenuVisibilityMember MenuVisibility = "member" // 任意租户成员可见（登录即可）
    MenuVisibilityAdmin  MenuVisibility = "admin"  // 仅管理员角色可见（硬隔离）
)
```

- **为何用单调门槛而非组合枚举**：用户提出的 `admin_normal` 这类组合在语义上会"塌缩"（普通成员与管理员都可见 = 无门槛 = `public`），组合枚举会随维度增加而指数膨胀；单调门槛加档位只加常量，判定主干不变。
- **判定规则（线性比较，无 if 分支堆叠）**：
  ```go
  // 当前用户可达档位
  userLevel := resolveUserMenuLevel(ctx, user) // public < member < admin
  visible := menu.Visibility <= userLevel
  ```
  （实现用数值映射或字符串偏序比较。）
- **`admin` 是硬隔离**：不看 `role_menu`，只判断"当前用户是否管理员"（依赖 `admin_level`/`HasSystemAdminCapability`，见 §2.2）；故即使误给普通角色授权了 `admin` 菜单，普通成员仍不可见。

### 5.1.2 扩展性预留（权限表，本轮不建）

**单调门槛枚举的下半场**：对"加档位"（仅超管/仅 owner/灰度渐进）扩展充分——加常量 + 更新 `userLevel` 即可。但对"多主体并集/特批"（给某角色/某用户/某组织单独可见、灰度白名单）单调枚举不够，需**预留 `menu_authorization`（菜单-主体授权）关联表**：

```go
// 预留：menu_authorization —— 菜单对特定主体可见（多主体横向扩展，本轮不建表）
type MenuAuthorizationEntity struct { // 设计预留
    MenuID     string  // 目标菜单
    TargetType string  // everyone | admin | role | user | org | ...
    TargetID   string  // 目标ID
    // TenantID / 审计字段
}
```

- 判定合并：`可见 = 达标门槛(visibility) || 命中特批授权(menu_authorization)`。
- 这样"水平多主体"的未来扩展 **永不改 struct、永不改判定主干**，加行即可。
- 本轮仅将本设计记录在案，不落地建表，避免过度设计。

---

### 5.2 现状（需要改变的地方）

当前侧边栏菜单**不做任何权限过滤**：
- 后端 `buildTenantMenuTree` → `buildAppMenuTree`（`svctenant/menu.go`）只按「租户订阅应用 + `status=enable`」平铺全部菜单，**不查 `role_menu`、不查 `visibility`，也不区分当前用户/角色**。
- 前端 `tenant-admin-web/App.tsx` 还有一份 `STATIC_MENU_TREE`（组织架构/用户管理/角色管理）硬编码，与后端返回重叠。
- 于是管理员与普通成员看到的侧边栏相同，普通成员也能看到用户/角色管理等管理菜单 —— 与本需求相悖。

好消息： seed 里内置唯一角色 admin 已配全量管理菜单；将来如新增差异化角色，只需在 `role_menu`/`role_scope` 上按角色授权即可，**扩展三级可见性时有现成数据可映射**。

### 5.3 改造点

**后端（核心）**
1. `pkg/iam/model/menu.go`：新增 `Visibility` 字段 + `MenuVisibility` 常量（§5.1.1）。
2. `pkg/iam/dao/menu.go`（`MenuCond`）：新增 `Visibility`/按可见等级过滤条件。
3. `pkg/seed/seed.go`：`seedMenu` 结构加 `visibility`，按本表映射各菜单等级（管理后台管理类菜单 → `admin`；租户侧给普通成员用的 → `public`/`member`；业务自定义 → `role_custom` 待扩展）。
4. `svctenant/menu.go`：
   - 新增`buildMyMenuTree(ctx)`：判定当前用户可达 `userLevel`（`public<member<admin`）→ 过滤 `buildAppMenuTree` 的树（`visibility <= userLevel`），含父子收敛（父已达标则保留，避免子可见父不可见）。
   - 管理后台菜单（系统应用）按"是否具备平台管理能力"过滤（`admin` 档）。
   - `Tree` 接口（`GET /tenant/menus/tree`）改为返回按当前用户可见等级过滤后的菜单。
5. （前置依赖）`admin` 档判定需"当前用户是否管理员"：接入 `admin_level`/`HasSystemAdminCapability`（见 §4），当前未接回时以种子 `IsSystem` 应用的平台角色能力近似。

**前端**
1. `tenant-admin-web/App.tsx`：移除 `STATIC_MENU_TREE` 硬编码，完全以 `getMyMenuTree()` 返回为准（后端已按可见等级过滤）。保留对空结果/无管理能力的兜底（如登录后跳转到唯一可见页）。
2. （同步提醒）既然侧边栏按可见等级过滤，**路由守卫**也应配套：未授权用户不应能通过直接输入 URL 访问管理页面；建议前端按菜单树生成可访问路由白名单，或后端接口层面也已靠 scope 校验（依赖后续 `role_scope` 接回）。

**管理员入口保护（关联 §5.4）**
- 若要求"只有管理员角色能做系统管理"，则受限接口需后端以 scope 判定（`HasSystemAdminCapability`）兜底，**不能只依赖前端隐藏菜单**（前端隐藏只是 UX，不是安全边界）。

### 5.4 管理员能力与菜单的关系（回到授权驱动）

- 菜单可见（`role_menu`）= **"看得到界面"**。
- 操作许可（`role_scope`）= **"能执行操作"**。
- 一个管理员要"能进用户管理页做用户管理"，需要同时满足两者。当前 `role_menu` 链路可立即接上（本 §5）；`role_scope` 链路（操作鉴权）是后续把断开处（`ValidateJWTProfileScopes` / 新增 `HasSystemAdminCapability`）接回，见 §四表。

---

## 六、遗留 / 可选项

- `is_system` 的「系统级写 scope」如何界定（按 scope 名前缀 `*:write` 还是按 resource 标记）——建议在 `resource` 上加 `is_system` 标记，最干净。
- 菜单可见性：`role_custom`（业务自定义菜单按 `role_menu` 勾选可见）这一档是否要在第一版就引入，取决于是否需要"租户可为普通角色单独配菜单可见"（见 §5.1.2）。
- 菜单可见性的 `admin` 档依赖"用户是否管理员"判定，需与 `role_scope` 接回（§4）协同落地，否则只能以系统应用角色近似。

---

## 七、一期落地状态（本会话已实现）

> 在「自底向上依次推进」下，以下已落地并编译通过 + 相关单测通过；前端环境因沙箱限制编译，改动为低风险字段透出与展示。

### 7.1 菜单可见性（角色驱动 + 三级门槛）
- `pkg/iam/model/menu.go`：新增 `Visibility` 字段、`MenuVisibility` 常量、`VisibilityRank`。
- `pkg/iam/dao/menu.go`：`MenuCond` 新增 `Visibility` 过滤。
- `pkg/seed/seed.go`：`seedMenu.visibility` 映射（管理后台管理菜单 `admin`、工作台 `member`、租户侧组织架构 `public`、租户侧用户/角色管理 `admin`）＋ 存量幂等回填。
- `svctenant/menu.go`：新增 `buildMyMenuTree`（按 `visibility <= userLevel` 剪枝，父子收敛）、`pruneMenuTree`、`resolveUserMenuLevel`、`HasSystemAdminCapability`（授权驱动，读 `role_scope`）。`Tree` 接口改走 `buildMyMenuTree`。
- 前端 `tenant-admin-web/App.tsx`：静态 fallback 收敛为仅含公共菜单（移除管理菜单），避免后端故障时暴露管理入口；types 补 `visibility`。

### 7.2 角色两维度 + 内置角色保护
- `pkg/iam/model/role.go`：新增 `Source`(builtin/custom)、`AdminLevel`(none/basic/super + `SysAdminLevel` 常量) 字段 + `RoleSource` 常量。
- `pkg/iam/model/role.go`：新增授权驱动的投影推导函数 `DeriveAdminLevelFromScopeNames`（任一管理类 `:write` → `super`；仅管理类 `:read` → `basic`；否则 `none`；`me:` 个人中心不计）。`HasSystemAdminCapability` 与其共用同一推导，保证标签与授权同源。
- `pkg/iam/dao/role.go`：`RoleCond` 新增 `Source`/`AdminLevel`/`AdminLevelAtLeast` 过滤。
- `pkg/seed/seed.go`：内置角色写 `source=builtin`、admin 设 `admin_level=super`；存量幂等回填；并在 seedRoleScopes 之后调用 `syncRoleAdminLevels` 按 scope 重算各角色 `admin_level`（授权驱动投影同步）。
- `svctenant/role.go`：Create 固定 `source=custom`+`admin_level=none`；Delete 拒绝内置角色；Update 内置禁改 code；`HasSystemAdminCapability` 改按 `admin_level>=basic` 识别系统管理角色。
- DTO（tenantadmin/platformadmin + `objpermission.RoleBaseInfo`）透出 `source`/`adminLevel`。
- 新增错误码 `RoleDeleteBuiltinForbiddenError`/`RoleUpdateBuiltinForbiddenError`。
- 前端 types 补 `source`/`adminLevel`；tenant 角色页显示来源标签 + 内置角色禁用编辑/删除；platform 详情显示来源/系统管理等级。

### 7.3 新增测试
- `svctenant/role_visibility_test.go`：内置删除拒绝、内置改核心字段拒绝、`HasSystemAdminCapability`（admin 有 / me-only 无）、`pruneMenuTree` 三级剪枝。
- `svctenant/role_visibility_test.go`：`UpdateRoles` 最后一个内置管理员保护（唯一持有者移除被拒 / 换新系统角色放行 / 有其他持有者放行）。
- `svctenant/role_visibility_test.go`：`DeriveAdminLevelFromScopeNames` 推导规则（me-only→none / 管理只读→basic / 管理写→super）。
- `pkg/seed/seed_test.go`：断言 seed 内置唯一角色 admin 的 admin_level 与 scope 推导结果一致（=super）；断言各表行数（role/role_menu/role_scope 仅 admin 一份）。

### 7.4 最后一个内置管理员持有者保护（遗留项已实现）
- `svctenant/user.go`：`UpdateRoles` 前置校验——若目标用户当前持有内置系统管理角色（`source=builtin && admin_level>=basic`），且新角色列表不含任何内置系统管理角色，且当前租户内除目标用户外无其他仍持有内置系统管理角色的用户，则拒绝（`UserRoleRemoveLastAdminForbiddenError`），防止平台系统管理能力永久锁死。
- 新增错误码 `UserRoleRemoveLastAdminForbiddenError`。
- 说明：当前 tenantadmin 无用户删除接口（`UserSvc` 仅 PageList/Create/Detail/Update/ResetPassword/ListRoles/UpdateRoles），故只有 `UpdateRoles` 是「解绑系统管理能力」的唯一入口，保护已覆盖。若后续新增用户删除/停用接口，应复用同一判定复用 `hasOtherSystemAdminHolder`。

### 7.5 派生化：`admin_level` 按 scope 自动推导（遗留项已实现）
- 单一推导源：`model.DeriveAdminLevelFromScopeNames`（`me:` 忽略；`resource:name:write` → `super`；`resource:name:read` → `basic`；否则 `none`）。权限判定 `HasSystemAdminCapability`（= 用户聚合等级 ≥ basic）与展示投影共用同一推导，杜绝标签与授权漂移。
- 用户级判定：`ResolveUserAdminLevel(ctx)` 聚合并推导当前用户等级；`HasSystemAdminCapability(ctx)` 基于它返回布尔。
- 角色投影同步：`svctenant.SyncRoleAdminLevel(ctx, role)` 供**角色 scope 授权变更后**调用（重算并回写 role.admin_level）。当前 `role_scope` 尚无运行时 CRUD 接口，同步入口先落地；后续新增「角色授权 scope」界面时，`PUT /roles/{roleID}/scopes` 全量替换后调用它即可。
- seed 已接入：`seedRoleScopes` 后调用 `syncRoleAdminLevels` 按 scope 重写各内置角色 admin_level，持续自洽。
- 一致性说明：现 seed 内置唯一角色 admin 持 `platform-admin:*:write` 等写 scope → 推导为 `super`，与授权一致。将来新增内置角色时，`admin_level` 一律由 scope 推导，不写死标签（如某角色仅持 `platform-admin:*:read` 只读权限 → 自然推导为 `basic`，与 token scope 授权一致）。
