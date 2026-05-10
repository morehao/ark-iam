# IAM 关系解绑组合键删除修复设计

## 文档信息

- **创建时间**：2026-05-10
- **版本**：v1.0
- **状态**：待审阅

---

## 一、问题背景

当前 `backend/apps/iam` 中有一批关系解绑接口，把业务字段 `userID`、`roleID`、`organizationID`、`organizationRoleID` 当成关系表主键使用，直接调用通用 `GetByID/Delete`。

这会导致两个严重问题：

1. 删除目标与请求语义不一致，可能误删主键刚好碰撞的无关关系记录。
2. 删除条件丢失 `tenant_id` 与其他业务键，无法保证解绑的是当前租户下的唯一关系。

这类问题属于高风险的数据破坏问题，且接口语义已经对外暴露，修复应优先保持请求结构不变，只修正服务层定位关系记录的方式。

---

## 二、目标

本次修复目标仅限于以下 5 个关系解绑点：

1. `svcpermission/user_role.go`
2. `svcpermission/role_menu.go`
3. `svcpermission/role_scope.go`
4. `svctenant/organization_user_relation.go`
5. `svctenant/organization_role_user_relation.go`

修复后统一满足：

1. 从上下文读取 `tenant_id`，不信任前端删除请求传入的租户边界。
2. 先按组合键查唯一关系记录。
3. 再按真实关系表主键 `entity.ID` 调用现有删除逻辑。
4. 查不到时保持现有“不存在”错误语义。

本次不包含：

- 大范围 tenant-scoped `GetByIDAndTenant` 重构
- 新增 DAO 基础设施 `DeleteByCond`
- 变更现有 DTO/API 形态
- 修复其他对象级 IDOR 问题

---

## 三、设计方案

### 方案 A：服务层两步删除（推荐）

做法：

1. 在 service 中构造对应 `Cond`。
2. 用 `GetListByCond` 或 `GetByCond` 查出关系记录。
3. 取真实 `entity.ID` 调用已有 `Delete(ctx, id, operatorID)`。

优点：

- 改动最小
- 不改 DAO 通用接口
- 易于逐点补测试

缺点：

- 每个 service 要重复一小段“按组合键查唯一关系”的逻辑

### 方案 B：DAO 新增 `DeleteByCond`

做法：

1. 扩展各关系 DAO，新增按条件删除能力。
2. service 直接调用 `DeleteByCond`。

优点：

- 删除语义更直接

缺点：

- 牵涉 DAO 扩展，超出本轮最小修复范围

### 方案 C：接口改为传关系表主键

做法：

1. 修改 DTO，让删除接口显式接收 relation row id。

优点：

- 实现最简单

缺点：

- 改变对外 API 语义
- 不兼容当前调用方

### 结论

采用方案 A。

---

## 四、修复映射

### 4.1 UserRole 删除

当前错误语义：

- 直接用 `req.UserID` 调 `GetByID/Delete`

目标语义：

- 按 `tenant_id + user_id + role_id` 查唯一关系
- 删除查到的 `entity.ID`

### 4.2 RoleMenu 删除

目标语义：

- 按 `tenant_id + role_id + menu_id` 查唯一关系
- 删除查到的 `entity.ID`

### 4.3 RoleScope 删除

目标语义：

- 按 `tenant_id + role_id + scope_id` 查唯一关系
- 删除查到的 `entity.ID`

### 4.4 OrganizationUserRelation 删除

目标语义：

- 按 `tenant_id + organization_id + user_id` 查唯一关系
- 删除查到的 `entity.ID`

### 4.5 OrganizationRoleUserRelation 删除

目标语义：

- 按 `tenant_id + organization_role_id + user_id` 查唯一关系
- 删除查到的 `entity.ID`

---

## 五、测试策略

采用 TDD：先补失败测试，再修 service。

### 5.1 测试目标

每个删除点至少覆盖：

1. 当前代码会把业务键当主键删错记录的场景
2. 修复后能按组合键删对记录的场景
3. tenant 不匹配时不会删到别的租户记录

### 5.2 推荐测试方式

优先使用 sqlite 或 stub repo：

1. 插入两条容易产生主键/业务键碰撞的关系记录
2. 调用删除 service
3. 断言只有组合键匹配的那条被删

### 5.3 验证标准

修复后应满足：

1. 关系解绑不再依赖业务字段碰巧等于表主键。
2. 删除条件具备 tenant 边界。
3. 相关 service 测试通过。
4. IAM 全量测试不回归。

---

## 六、风险与约束

### 风险

1. 若关系表历史上存在重复脏数据，`GetListByCond` 可能返回多条，需要明确取第一条还是报错。
2. 某些删除 DTO 自带 `TenantID`，但当前设计统一以上下文租户为准，可能暴露出旧测试对请求体租户的隐式依赖。

### 约束

1. 不顺手修复其他 detail/update/delete 的 tenant 校验问题。
2. 不改变外部接口字段。
3. 不新增新的 DAO 抽象，除非测试替身需要极小注入缝。

---

## 七、预期结果

完成后，这 5 个关系解绑接口将按组合键和租户边界精确定位关系记录，再按真实主键删除，消除“把业务 ID 当关系表主键删除”的高风险错误。 
