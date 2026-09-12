# 种子权威矩阵收窄：reconcile 的准入判据

> 状态：**已落地**（2026-09-12）
> 决策日期：2026-09-12
> 上游文档：[seed-initialization-redesign-20260912.md](seed-initialization-redesign-20260912.md)（本文修订其「产品身份 vs 部署数据」判据，不推翻其单一写者模型）
> 一句话结论：上一版把「应用名/描述、客户端名、菜单的结构与展示字段」都判成"产品身份"归种子收敛，导致控制台整页不可编辑；把 reconcile 的准入判据收窄成**「被控制台改写后会导致种子定位失效或鉴权被绕过」**（定位键 + 安全不变式），其余一律归运维，即可在保持单一写者的同时让页面完全可改、重启不回写。
> ⚠️ **"定位键"一类已被取代**（2026-09-12 同日后继改造）：种子认行改由不可见的**种子身份键 `seed_key`** 承担，菜单的 `code` 与归属应用、**自建**应用/客户端的 `code` 因此也可改（内置应用与内置客户端的 `code` 仍只读：控制台菜单入口与网关 aud 边界按它取值）；reconcile 只剩安全不变式。见 [seed-identity-key-20260912.md](seed-identity-key-20260912.md)。本文其余结论（单一写者、拒绝要显式、值匹配迁移）仍然有效。

---

## 背景与现状

### 诉求

期望的场景是三步：**启动播种 → 控制台可改 → 重启后控制台的改动不受影响**。

改造前的表现：控制台**完全不能改**——菜单管理页对内置应用整棵树只读（9 个字段全置灰 + 不可新增/删除），应用管理页内置应用的名称/描述置灰，OAuth 客户端页内置客户端名称置灰，租户页平台租户挂起开关禁用。

### 这不是 bug，是上一版的有意取舍

`pkg/model/seed_authority.go` 的三种语义里，`reconcile` = 「种子唯一写者」，控制台必须拒写，否则就是"运维改完、重启被收回"的双写者（上游文档 P1）。上一版用 `reconcile` 判定的口径是：

> **产品定义 → Reconcile；部署数据 → CreateOnly；历史改名 → MigrateOnce。**

这条口径自洽，但它把"内置对象的展示名与前端 IA"也算成了产品定义，于是代价直接落在控制台的可用性上。

### 逻辑上的硬约束

对**同一个字段**，"控制台可改"与"种子上次启动会收敛它"是互斥的：不可能既每次启动被拉回定义值、又让改动活过重启。因此问题不是"能不能改"，而是：

> **哪些字段值得为「跨版本自愈」付「不可编辑」的代价？**

---

## 合理性分析

### 1. reconcile 只在"种子将来会改这个字段"时才值钱

一个从定义出来就没变过的字段，`reconcile` 与 `create_only` 的运行结果完全一致（新库都写初值、老库都不动），却白白让运维失去编辑能力。对照上一版的矩阵：

| 字段 | 上一版语义 | 实际自愈收益 | 真实代价 |
|------|-----------|--------------|----------|
| 菜单 `name`/`icon`/`sort`/`visibility` | reconcile | ≈0（无任何代码依赖，纯展示） | 菜单页半屏置灰 |
| 菜单 `path`/`component`/`type`/`parent_id` | reconcile | 有限（与前端路由表弱耦合） | 无法调整 IA |
| 应用 `name`/`description`、客户端 `name` | reconcile | ≈0（纯展示） | 内置对象不能改名 |

### 2. 真正需要强制的只有两类

- **定位键**：`pkg/seed` 靠它查行（`tenant`/`application`/`application_client` 的 `code`，`menu` 的 `app_id + code`）。键一旦被改，下次启动**查不到该行** → 按"不存在"再建一行，幂等性直接失效，并丢掉挂在原 `id` 上的关联（`role_menu` 授权、菜单树父子关系）。
- **安全不变式**：`source`（内置标记，平台侧重置内置管理员口令等能力靠它定位内置对象）、`admin_type`（系统管理能力）、平台租户 `status`（被挂起整栈控制台失联）。这些字段**本来就不在前端表单里**，锁住零损失。

### 3. 上一版的划线在同一类需求上是自相矛盾的

平台租户名与根部门名早已因为"运维要改"降级为 `migrate_once` / `create_only`，而同一类诉求（运维按自己的组织与品牌命名）落在应用名和菜单上却被拒绝。这不是有意收紧，是判据用词（"产品身份"）过宽导致的划线漂移。

### 4. 结论

按**「定位键 + 安全不变式」**划线，恰好同时满足三条预期：行的创建仍由种子负责（第 1 条），所有展示与结构字段归还运维（第 2 条），种子的收敛集里不再有"运营想改的字段"（第 3 条）。

---

## 决策：新准入判据

> **reconcile 的准入条件 = 被控制台改写后会导致种子定位失效或鉴权被绕过。**
>
> 不满足该条件的字段一律 `create_only`（归运维）；跨版本改名用 `migrate_once` 值匹配表达，不走 reconcile。

### 目标矩阵（变更行）

| 实体 | 字段 | 改造前 | 改造后 | 理由 |
|------|------|--------|--------|------|
| tenant | `status` | reconcile | reconcile（不变） | 安全不变式：挂起整栈失联 |
| tenant / department | `name` | migrate_once | migrate_once（不变） | 历史种子名一次性改名，运维自定义值不动 |
| application | `source` | reconcile | reconcile（不变） | 安全不变式（且不在 Update DTO 中） |
| application | `name`/`description` | **reconcile** | **create_only** | 纯展示，归运维 |
| application | `code` | migrate_once | migrate_once（不变） | 定位键；不在 Update DTO 中，控制台无入口 |
| application_client | `source`/`app_id` | reconcile | reconcile（不变） | 安全不变式 / 归属关系（无控制台入口） |
| application_client | `name` | **reconcile** | **create_only** | 纯展示，归运维 |
| menu | `app_id`/`code` | 只有 `code` reconcile | **`app_id` + `code` reconcile** | 二者共同构成 `seedMenus` 的查重键 |
| menu | `parent_id`/`name`/`path`/`icon`/`sort`/`component`/`type`/`visibility` | **reconcile** | **create_only** | 展示与 IA，归运维 |
| menu | `status` | create_only | create_only（不变） | 运维可停用菜单 |
| role | `admin_type` | reconcile | reconcile（不变） | 安全不变式 |
| user / person | `source` / 口令 | reconcile / create_only | 不变 | source 安全不变式；口令绝不覆盖 |

### 为什么 `code` 仍然拒写

`code` 不是"身份"，是**查重键**：`seedMenus` 按 `(app_id, code)`、`getOrCreateApplication` 按 `code` 定位内置行。允许改 `code` 的后果不是"字段被收回"，而是**下次启动多出一行**，且挂在原 `id` 上的授权（`role_menu`）与新行失联。所以这条是功能性约束，不是口味问题——且 `code` 在前端本来就不可编辑（应用表单 `disabled={!!editing}`），放开它没有收益。

---

## 详细设计

### 种子侧：只改矩阵，不改执行逻辑

`reconcileFields`（`pkg/seed/seed.go`）本就按矩阵过滤：

```go
if !model.SeedOwnsField(entity, field) { continue }
```

各 upsert 继续把"种子定义的字段"整体放进 desired，**写谁由矩阵决定**。因此本次改动只动 `SeedFieldAuthorities` 的取值，`pkg/seed` 的分支一行未改——矩阵"唯一真相源"的性质因此被正面验证。

唯一的行为性修正：`getOrCreateApplication` 原先在发生收敛时把 `entity.Name/Description` 也改写成种子值，收敛集收窄后这会把内存态与库态写得不一致，已改为只回写 `source`。

### 控制台侧：拒写分支按矩阵自动收缩

| 位置 | 改造前 | 改造后 |
|------|--------|--------|
| `svcapplication.Update` | `applicationSeedFieldsChanged` 拦 name/description | 函数删除：矩阵里 application 的 reconcile 字段（`source`）不在 Update DTO 中，无字段可拦 |
| `svcapplicationclient.Update` | `clientSeedFieldsChanged` 拦 name | 函数删除：同上（`source`/`app_id` 都无入口） |
| `svcpermission`（Update） | 拦 9 个字段 | 只拦定位键 `app_id`/`code`（`menuSeedFieldsChanged` 继续走矩阵判断） |
| `svcpermission`（Create/Delete） | 内置应用整体拒写 | **不变**（理由见下） |

**菜单新增/删除为什么保留限制**：`seedMenus` 只创建不收敛——删掉的行下次启动会被重建（并丢掉 `role_menu` 授权），新增的行则没有对应的前端路由，只会得到死链。需要下线内置菜单请用「停用」（`status` 归运维，种子不回写）。这是唯一保留的"不可编辑"能力，与"修改"无关。

> **后续变更（2026-09-12）**：该限制已解除——[menu-console-crud-20260912.md](menu-console-crud-20260912.md) 把菜单的"行"也交还运维：控制台可新增根菜单/子菜单、可删除任意菜单；删除通过在菜单表留下**软删墓碑**使种子不再复活它（退役菜单则改物理删除、不留墓碑）。错误码 100606 `MenuBuiltInCreateDeleteForbiddenError` 已彻底下线。

**错误码下线**（按 `AGENTS.md` "下线即彻底删代码"）：

- 删除 `ApplicationBuiltInFieldImmutableError`(100749)、`ApplicationClientBuiltInFieldImmutableError`(100821)，编号不复用；
- 保留 `MenuBuiltInFieldImmutableError`(100606)，文案与注释改为"新增/删除，所属应用与编码不可修改"。

### 前端：解除与后端同步的置灰

- 应用管理：内置应用的名称/描述解除置灰（`code` 编辑态仍只读）；
- OAuth 客户端：内置客户端名称解除置灰；
- 菜单管理：编辑弹窗内只保留 `code` 置灰，其余字段可改；「新建根菜单」「新增子级」「删除」对内置应用仍隐藏（与后端一致）；
- 租户管理：平台租户挂起开关保持禁用（纯业务规则：平台租户挂起会导致整栈失联，与种子无关）。

---

## 影响面与兼容

| 维度 | 结论 |
|------|------|
| 存量库 | 无需任何动作：字段语义改变只影响"以后是否回写"，不回填、不删列 |
| 已部署环境中被拒的字段 | 此前根本改不进去（控制台拒写），因此不存在"新的覆盖"；此后改动能稳定保留 |
| 前端契约 | 无接口变更，仅解除置灰；`destroyOnClose` 等无关 |
| 跨版本改名 | 由 `migrate_once` 承担：把 `{entity, field, from, to}` 登记进 `seedMigrations` 即可（当前清单覆盖 tenant/department 的 `name`；application/menu/client 需要时按同一机制扩展 entity 维度） |
| 菜单新增/删除 | 仍归版本：新增菜单由种子创建（创建不冲突），下线由 `retired_menu.go` 的 `retiredMenus` 清理 |

---

## 验收

| 编号 | 内容 | 用例 |
|------|------|------|
| V1 | 矩阵准入判据 | `pkg/model`：`TestSeedOwnsFieldOnlyForReconcile`（展示字段不算种子拥有；定位键与安全不变式算）、`TestSeedReconcileFields`（menu 收敛集只剩 `app_id`/`code`） |
| V2 | 展示名不再回写、历史种子名仍一次性改名 | `pkg/seed`：`TestSeedIamMigratesLegacyNamesOnly` |
| V3 | 内置菜单改名称/路径/组件/图标/排序/父级/可见性/状态后重启不回写，且不触发重建行 | `pkg/seed`：`TestSeedIamKeepsOperatorMenuEdits` |
| V4 | 安全不变式仍收敛（source 被改写回 builtin）、租户 status 仍被纠正 | `pkg/seed`：`TestSeedIamRespectsOperatorOwnedFields` |
| V5 | 控制台放行展示字段 | `svcapplication`：`TestUpdateBuiltInApplicationAllowsDisplayFields`；`svcapplicationclient`：`TestUpdateBuiltInClientAllowsName` |
| V6 | 菜单定位键仍拒写、归属应用改动仍拒写、新增/删除仍拒写、第三方应用不受约束 | `svcpermission`：`TestUpdateBuiltInAppMenuProtectsSeedLookupKey`、`TestCreateAndDeleteBuiltinAppMenuRejected` |
| V7 | 前端不再置灰（仅定位键只读） | `platform-admin-web`：菜单/应用两条回归，55 tests 全绿；`tsc --noEmit` 通过 |

---

## 风险与回滚

| 风险 | 触发 | 应对 |
|------|------|------|
| R1 内置菜单被改坏（`component` 写错致页面白屏） | 运维在菜单页改路径/组件 | 恢复手段是重新编辑（字段已可写）或删库重建；后续可把 `component` 换成前端路由表下拉 + 「恢复默认」动作 |
| R2 存量库既有内置行不再随版本自愈 | 新版本改了既有菜单的路径/组件/名称 | 登记 `seedMigrations` 值匹配条目（运营改过的行不会被动）；新增菜单仍自动播种 |
| R3 想恢复旧行为 | 需要"内置对象只读"的产品形态 | `db.seed=false` 只会停止播种而不恢复拒写；恢复拒写需回退矩阵取值与对应 service 分支（数据无损） |

---

## 落地记录（2026-09-12）

| 层 | 文件 | 内容 |
|----|------|------|
| 真相源 | `pkg/model/seed_authority.go` | 准入判据写入文件头；application `name`/`description`、application_client `name`、menu 9 个字段转 `create_only`；menu 新增 `app_id` reconcile |
| 种子 | `pkg/seed/seed.go` | 注释与包说明同步新语义；`getOrCreateApplication` 收敛时只回写 `source`（不再污染内存态展示名） |
| 控制台 | `svcapplication` / `svcapplicationclient` / `svcpermission` | 删除两处失效的种子字段拦截；菜单拦截收窄到 `app_id`/`code`；新增/删除限制不变 |
| 错误码 | `pkg/code/permission.go` | 删除 100749、100821；100606 文案改为"定位键/新增删除"口径 |
| 前端 | platform-admin-web：application / oauthClient / menu | 展示字段解除置灰；菜单仅 `code` 只读；提示文案同步 |
| 测试 | `pkg/model`、`pkg/seed`、三个 service、platform-admin-web | 6 处用例翻转 + 新增 `TestSeedIamKeepsOperatorMenuEdits` |

### 与上游文档的关系

上游 `seed-initialization-redesign-20260912.md` 的**模型**（单一写者、三分语义、矩阵唯一真相源、控制台拒写、值匹配迁移、并发与报告）全部保留；本文只修订其**矩阵取值与准入判据**——即把该文 Q1（是否允许白标改名）的默认口径从"不允许"改为"允许"：内置控制台的展示与 IA 归运维，跨版本变更走 `migrate_once`。
