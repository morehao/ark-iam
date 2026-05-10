# IAM Tenant-Scoped 对象访问修复设计

## 文档信息

- **创建时间**：2026-05-10
- **版本**：v1.0
- **状态**：已按用户指令直接执行

---

## 一、问题背景

`backend/apps/iam/internal/service` 中存在一类模式性高风险问题：多个 detail/update/delete/create 入口直接使用 `dao.GetByID` 按主键读取对象，但没有把当前请求上下文中的 `tenant_id` 纳入对象存在性校验。

这会导致两类风险：

1. 当前租户可以通过已知主键读取或修改别的租户对象，形成对象级越权。
2. 某些关联创建场景只校验对象主键存在，不校验对象是否属于当前租户，导致跨租户建立关联。

前一批已经修复了“关系解绑误按主键删除”的数据破坏问题；本批继续收敛“对象可见性和可操作边界”问题。

---

## 二、目标

本次修复目标：

1. 对所有应受租户边界约束的对象访问，统一以 `gincontext.GetTenantID(ctx)` 作为真实租户边界。
2. detail/update/delete 在执行前，必须确认目标对象属于当前租户。
3. create 中引用外部对象时，若该对象应租户隔离，也必须确认该对象属于当前租户。
4. 保持现有 DTO、controller 路由和对外 API 语义不变。
5. 每个模块都补最小回归测试，锁定“不允许跨租户访问”。

本次按用户要求依次修复 4 个子批次：

1. `svctenant`
2. `svcpermission`
3. `svcapplication` + `svcauth`
4. `svcuser`

---

## 三、非目标

本次不包含：

- 重构 DAO 通用基建为 `GetByIDAndTenant`
- 调整任何 DTO 字段、路由、controller 绑定逻辑
- 处理与租户无关的系统级对象访问策略
- 顺手做大范围重构或风格清理

---

## 四、根因分析

当前多个 service 使用如下模式：

1. `entity, err := dao.NewXxxDao().GetByID(ctx, req.ID)`
2. `if entity == nil { return NotExist }`
3. 继续返回详情、更新、删除，或把该对象作为关联创建前置条件

问题在于：

1. `GetByID` 只按表主键查询，不带 `tenant_id`。
2. service 没有在读到对象后校验 `entity.TenantID == gincontext.GetTenantID(ctx)`。
3. 部分页查列表依赖请求体里的 `TenantID`，而不是上下文租户。

因此，只要调用方拿到其他租户对象主键，就可能越权访问或引用该对象。

---

## 五、设计方案

### 方案 A：service 层显式 tenant 校验（推荐）

做法：

1. 保留现有 DAO `GetByID/GetListByCond`。
2. 在 service 层增加极小的 tenant-scoped 校验辅助函数或 repo seam。
3. 对于 detail/update/delete：
   - 先按 ID 读取对象，或按 `GetByCond/GetListByCond` 查对象
   - 再校验对象 `TenantID` 与上下文租户一致
4. 对于 pageList/tree：优先改为使用上下文租户，而不是信任请求参数中的 `TenantID`。

优点：

- 改动最小
- 不扩 DAO 公共接口
- 便于按模块逐批 TDD 落地

缺点：

- 会在多个 service 中重复一小段 tenant 校验逻辑

### 方案 B：DAO 增加统一 `GetByIDAndTenant`

优点：

- 语义更集中

缺点：

- 需要扩展大量 DAO，超出当前最小修复范围

### 方案 C：依赖 middleware 或 controller 做统一前置拦截

优点：

- service 代码变动少

缺点：

- 当前 controller/DTO 无法表达所有对象语义
- 很难覆盖 create 中的关联对象引用校验

### 结论

采用方案 A：在 service 层做最小、显式、可测试的 tenant-scoped 校验。

---

## 六、统一修复规则

### 6.1 对 detail/update/delete

统一规则：

1. 从 `gincontext.GetTenantID(ctx)` 取得当前租户。
2. 目标对象读取后，若 `entity == nil`、`entity.ID == 0`、或 `entity.TenantID != currentTenantID`，统一按现有“不存在”错误返回。
3. 只有通过校验后，才允许返回详情、更新、删除。

### 6.2 对 create 中的关联对象引用

统一规则：

1. 若 create 会引用已有对象，如组织、角色、菜单、资源、用户、连接器、应用等：
2. 对应对象若属于租户隔离资源，则必须校验其 `TenantID == currentTenantID`。
3. 校验失败按现有“对象不存在”错误返回。

### 6.3 对 pageList/tree

统一规则：

1. 租户隔离资源的列表查询优先使用上下文租户。
2. 不再信任请求参数中的 `TenantID` 作为实际权限边界。
3. 请求参数中的 `TenantID` 如保留，仅作为兼容字段，不作为越权边界依据。

---

## 七、分批范围

### 7.1 第一批：`svctenant`

优先修复：

- `tenant.go`
- `department.go`
- `organization.go`
- `organization_role.go`
- `system.go`
- `log.go`

重点：detail/update/delete 和 tree/pageList 的 tenant 边界。

### 7.2 第二批：`svcpermission`

优先修复：

- `menu.go`
- `role.go`
- `resource.go`
- `scope.go`
- 以及本轮已修过删除语义的 `user_role.go` / `role_menu.go` / `role_scope.go` 的 create 引用校验

重点：角色/菜单/资源/权限对象 detail/update/delete 与跨租户引用。

### 7.3 第三批：`svcapplication` + `svcauth`

优先修复：

- `application.go`
- `connector.go`

重点：应用、密钥、连接器等对象的 detail/update/delete 与引用边界。

### 7.4 第四批：`svcuser`

优先修复：

- `user.go`
- `user_identity.go`

重点：用户、身份、日志等对象的 detail/update/delete 与列表租户边界。

---

## 八、测试策略

采用 TDD，统一使用 service 层 stub/repo seam 测试模式，避免引入数据库级新基建。

每个模块至少覆盖：

1. 当前租户访问本租户对象成功。
2. 当前租户访问其他租户对象时返回既有 `NotExist` 错误。
3. create 引用其他租户对象时返回既有 `NotExist` 错误。
4. pageList/tree 忽略请求里的其他租户参数，实际按上下文租户查询。

---

## 九、风险与约束

### 风险

1. 部分 service 当前把请求体中的 `TenantID` 同时当作数据字段和权限边界，修复后可能暴露旧测试或旧调用方的隐式依赖。
2. `tenant.go` 本身是租户对象服务，是否完全受租户上下文限制需要以现有业务模型为准，避免把平台级管理员场景误伤。
3. `log.go`、`system.go` 等列表与详情可能已有系统级访问预期，需要只对明显 tenant-scoped 对象收紧，不扩大到全局对象。

### 约束

1. 保持对外 API 不变。
2. 默认最小改动，优先在 service 层加校验与测试。
3. 每个子批次修完后都要跑对应定向测试和 IAM 全量测试。

---

## 十、预期结果

完成后，IAM 中 tenant-scoped 资源的 detail/update/delete/create/list 将统一受当前上下文租户约束，调用方即使知道其他租户对象主键，也无法跨租户读取、修改、删除或建立关联。 
