# IAM Tenant-Scoped 对象访问修复 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 按 `svctenant -> svcpermission -> svcapplication/svcauth -> svcuser` 顺序修复 tenant-scoped 对象的 detail/update/delete/create/list 越权问题，并用最小 service 层测试锁定行为。

**Architecture:** 保持现有 DTO、controller、DAO 对外接口不变，只在 service 层补充基于 `gincontext.GetTenantID(ctx)` 的对象归属校验，以及必要的 query repo seam。detail/update/delete 统一执行“读取对象后校验 tenant”；pageList/tree 统一优先采用上下文租户；create 中若引用租户资源，则校验被引用对象租户归属。

**Tech Stack:** Go, Gin, genericdao, GORM, 标准库 testing

---

## File Structure

- Modify: `backend/apps/iam/internal/service/svctenant/tenant.go`
  责任：为租户模块补充最小 tenant-scoped 行为判断；若租户对象确认为平台级资源，则保留现状并在测试中锁定。
- Modify: `backend/apps/iam/internal/service/svctenant/department.go`
  责任：为部门 detail/update/delete/tree/pageList 强制使用上下文租户边界。
- Modify: `backend/apps/iam/internal/service/svctenant/organization.go`
  责任：为组织 detail/update/delete/pageList 强制使用上下文租户边界。
- Modify: `backend/apps/iam/internal/service/svctenant/organization_role.go`
  责任：为组织角色 create/detail/update/delete/pageList 强制使用上下文租户边界。
- Modify: `backend/apps/iam/internal/service/svctenant/system.go`
  责任：为系统配置 create/detail/update/delete/pageList 强制使用上下文租户边界。
- Modify: `backend/apps/iam/internal/service/svctenant/log.go`
  责任：为审计日志 detail/pageList 强制使用上下文租户边界。
- Create: `backend/apps/iam/internal/service/svctenant/tenant_scope_test.go`
  责任：覆盖 `svctenant` 中 detail/update/delete/create/list 的越权阻断和上下文租户透传。

- Modify: `backend/apps/iam/internal/service/svcpermission/menu.go`
- Modify: `backend/apps/iam/internal/service/svcpermission/role.go`
- Modify: `backend/apps/iam/internal/service/svcpermission/resource.go`
- Modify: `backend/apps/iam/internal/service/svcpermission/scope.go`
- Modify: `backend/apps/iam/internal/service/svcpermission/user_role.go`
- Modify: `backend/apps/iam/internal/service/svcpermission/role_menu.go`
- Modify: `backend/apps/iam/internal/service/svcpermission/role_scope.go`
- Create: `backend/apps/iam/internal/service/svcpermission/tenant_scope_test.go`
  责任：覆盖权限模块对象校验、引用校验、列表租户透传。

- Modify: `backend/apps/iam/internal/service/svcapplication/application.go`
- Modify: `backend/apps/iam/internal/service/svcauth/connector.go`
- Create: `backend/apps/iam/internal/service/svcapplication/application_tenant_scope_test.go`
- Create: `backend/apps/iam/internal/service/svcauth/connector_tenant_scope_test.go`
  责任：覆盖应用、密钥、连接器的 tenant-scoped 访问。

- Modify: `backend/apps/iam/internal/service/svcuser/user.go`
- Modify: `backend/apps/iam/internal/service/svcuser/user_identity.go`
- Create: `backend/apps/iam/internal/service/svcuser/user_object_scope_test.go`
  责任：覆盖用户、身份、日志等对象的 tenant-scoped 访问。

---

### Task 1: 修复 `svctenant` 的 tenant-scoped 对象访问

**Files:**
- Modify: `backend/apps/iam/internal/service/svctenant/department.go`
- Modify: `backend/apps/iam/internal/service/svctenant/organization.go`
- Modify: `backend/apps/iam/internal/service/svctenant/organization_role.go`
- Modify: `backend/apps/iam/internal/service/svctenant/system.go`
- Modify: `backend/apps/iam/internal/service/svctenant/log.go`
- Test: `backend/apps/iam/internal/service/svctenant/tenant_scope_test.go`

- [ ] **Step 1: 写失败测试**

在 `backend/apps/iam/internal/service/svctenant/tenant_scope_test.go` 中新增一组最小 service 单测，覆盖：

1. `Department.Detail` 读取到其他租户部门时返回 `DepartmentNotExistError`
2. `Department.PageList` 和 `Department.Tree` 查询条件使用 `gincontext.GetTenantID(ctx)`，忽略请求中的其他租户值
3. `Organization.Detail` / `OrganizationRole.Detail` / `System.Detail` / `Log.Detail` 在对象租户不匹配时返回各自 `NotExistError`
4. `OrganizationRole.Create` 引用不属于当前租户的组织时返回 `OrganizationNotExistError`

测试风格沿用现有 repo seam + stub，示例片段：

```go
func TestDepartmentDetailRejectsCrossTenantEntity(t *testing.T) {
    ginCtx, _ := gin.CreateTestContext(nil)
    ginCtx.Set(gcontext.KeyTenantID, uint(11))

    repo := &stubDepartmentScopedRepo{
        detail: &model.DepartmentEntity{Model: gorm.Model{ID: 7}, TenantID: 22},
    }
    installDepartmentScopedRepo(t, repo)

    svc := &departmentSvc{}
    resp, err := svc.Detail(ginCtx, &dtotenant.DepartmentDetailReq{DepartmentID: 7})
    if err == nil {
        t.Fatalf("expected cross-tenant detail to fail, resp=%+v", resp)
    }
}

func TestDepartmentPageListUsesContextTenant(t *testing.T) {
    ginCtx, _ := gin.CreateTestContext(nil)
    ginCtx.Set(gcontext.KeyTenantID, uint(33))

    repo := &stubDepartmentScopedRepo{}
    installDepartmentScopedRepo(t, repo)

    svc := &departmentSvc{}
    _, _ = svc.PageList(ginCtx, &dtotenant.DepartmentPageListReq{TenantID: 99})
    if repo.lastCond == nil || repo.lastCond.TenantID != 33 {
        t.Fatalf("expected tenant 33 from context, got %+v", repo.lastCond)
    }
}
```

- [ ] **Step 2: 运行测试确认失败**

Run: `go test ./apps/iam/internal/service/svctenant -run 'Test(Department|Organization|OrganizationRole|System|Log)' -v`

Expected: FAIL，因为当前实现直接 `GetByID` 或透传请求 `TenantID`，不会拒绝跨租户对象，也不会统一使用上下文租户。

- [ ] **Step 3: 写最小实现**

在 `department.go`、`organization.go`、`organization_role.go`、`system.go`、`log.go` 中：

1. 为需要观测查询条件的路径增加最小 repo seam
2. 增加 tenant 校验辅助逻辑，形式保持简单：

```go
func departmentVisibleToTenant(entity *model.DepartmentEntity, tenantID uint) bool {
    return entity != nil && entity.ID != 0 && entity.TenantID == tenantID
}
```

3. `Detail/Update/Delete` 在 `GetByID` 后统一检查：

```go
tenantID := gincontext.GetTenantID(ctx)
if !departmentVisibleToTenant(departmentEntity, tenantID) {
    return nil, code.GetError(code.DepartmentNotExistError)
}
```

4. `PageList/Tree` 统一把条件改为：

```go
cond := &dao.DepartmentCond{
    BaseCond: &genericdao.BaseCond{Page: req.Page, PageSize: req.PageSize},
    TenantID: gincontext.GetTenantID(ctx),
    ParentID: req.ParentID,
    Name: req.Name,
    Code: req.Code,
}
```

5. `OrganizationRole.Create` 在引用组织前校验该组织 `TenantID` 与上下文一致。

- [ ] **Step 4: 运行测试确认通过**

Run: `go test ./apps/iam/internal/service/svctenant -run 'Test(Department|Organization|OrganizationRole|System|Log)' -v`

Expected: PASS

- [ ] **Step 5: 运行 `svctenant` 全量测试**

Run: `go test ./apps/iam/internal/service/svctenant -v`

Expected: PASS

---

### Task 2: 修复 `svcpermission` 的 tenant-scoped 对象访问

**Files:**
- Modify: `backend/apps/iam/internal/service/svcpermission/menu.go`
- Modify: `backend/apps/iam/internal/service/svcpermission/role.go`
- Modify: `backend/apps/iam/internal/service/svcpermission/resource.go`
- Modify: `backend/apps/iam/internal/service/svcpermission/scope.go`
- Modify: `backend/apps/iam/internal/service/svcpermission/user_role.go`
- Modify: `backend/apps/iam/internal/service/svcpermission/role_menu.go`
- Modify: `backend/apps/iam/internal/service/svcpermission/role_scope.go`
- Test: `backend/apps/iam/internal/service/svcpermission/tenant_scope_test.go`

- [ ] **Step 1: 写失败测试**

在 `backend/apps/iam/internal/service/svcpermission/tenant_scope_test.go` 中新增测试，覆盖：

1. `Menu/Role/Resource/Scope` 的 `Detail/Update/Delete` 读取到其他租户对象时返回对应 `NotExistError`
2. `Menu/Role/Resource/Scope` 的 `PageList` 使用上下文租户，而不是请求里的 `TenantID`
3. `UserRole/Create`、`RoleMenu/Create`、`RoleScope/Create` 在引用其他租户 `Role/Menu/Scope/Resource` 时返回对应 `NotExistError`

示例片段：

```go
func TestRoleDetailRejectsCrossTenantEntity(t *testing.T) {
    ginCtx, _ := gin.CreateTestContext(nil)
    ginCtx.Set(gcontext.KeyTenantID, uint(21))

    repo := &stubRoleScopedRepo{detail: &model.RoleEntity{Model: gorm.Model{ID: 5}, TenantID: 88}}
    installRoleScopedRepo(t, repo)

    svc := &roleSvc{}
    _, err := svc.Detail(ginCtx, &dtopermission.RoleDetailReq{RoleID: 5})
    if err == nil {
        t.Fatalf("expected cross-tenant role detail to fail")
    }
}
```

- [ ] **Step 2: 运行测试确认失败**

Run: `go test ./apps/iam/internal/service/svcpermission -run 'Test(Menu|Role|Resource|Scope|UserRole|RoleMenu|RoleScope)' -v`

Expected: FAIL，当前实现仍有直接主键读取与请求租户透传。

- [ ] **Step 3: 写最小实现**

1. 在 `menu.go`、`role.go`、`resource.go`、`scope.go` 中加入 tenant 可见性判断
2. 在各自 `PageList` 中强制使用 `gincontext.GetTenantID(ctx)`
3. 在 `user_role.go`、`role_menu.go`、`role_scope.go` 的 `Create` 中校验被引用的 role/menu/scope/resource 属于当前租户

实现模式统一为：

```go
tenantID := gincontext.GetTenantID(ctx)
if entity == nil || entity.ID == 0 || entity.TenantID != tenantID {
    return nil, code.GetError(code.RoleNotExistError)
}
```

- [ ] **Step 4: 运行测试确认通过**

Run: `go test ./apps/iam/internal/service/svcpermission -run 'Test(Menu|Role|Resource|Scope|UserRole|RoleMenu|RoleScope)' -v`

Expected: PASS

- [ ] **Step 5: 运行 `svcpermission` 全量测试**

Run: `go test ./apps/iam/internal/service/svcpermission -v`

Expected: PASS

---

### Task 3: 修复 `svcapplication` 与 `svcauth` 的 tenant-scoped 对象访问

**Files:**
- Modify: `backend/apps/iam/internal/service/svcapplication/application.go`
- Modify: `backend/apps/iam/internal/service/svcauth/connector.go`
- Test: `backend/apps/iam/internal/service/svcapplication/application_tenant_scope_test.go`
- Test: `backend/apps/iam/internal/service/svcauth/connector_tenant_scope_test.go`

- [ ] **Step 1: 写失败测试**

新增测试覆盖：

1. `Application.Detail/Update/Delete` 跨租户对象拒绝访问
2. `Application` 相关 secret 读取/删除拒绝跨租户 secret
3. `Connector.Detail/Update/Delete/Test/AuthURI` 拒绝跨租户 connector
4. `PageList` 使用上下文租户

- [ ] **Step 2: 运行测试确认失败**

Run: `go test ./apps/iam/internal/service/svcapplication ./apps/iam/internal/service/svcauth -run 'Test(Application|Connector)' -v`

Expected: FAIL

- [ ] **Step 3: 写最小实现**

1. 在 `application.go` 和 `connector.go` 增加最小 repo seam
2. 统一 detail/update/delete/test/auth URI 的对象租户校验
3. 统一列表查询使用上下文租户

- [ ] **Step 4: 运行测试确认通过**

Run: `go test ./apps/iam/internal/service/svcapplication ./apps/iam/internal/service/svcauth -run 'Test(Application|Connector)' -v`

Expected: PASS

- [ ] **Step 5: 运行模块全量测试**

Run: `go test ./apps/iam/internal/service/svcapplication ./apps/iam/internal/service/svcauth -v`

Expected: PASS

---

### Task 4: 修复 `svcuser` 的 tenant-scoped 对象访问

**Files:**
- Modify: `backend/apps/iam/internal/service/svcuser/user.go`
- Modify: `backend/apps/iam/internal/service/svcuser/user_identity.go`
- Test: `backend/apps/iam/internal/service/svcuser/user_object_scope_test.go`

- [ ] **Step 1: 写失败测试**

新增测试覆盖：

1. `User.Detail/Update/Delete` 拒绝跨租户用户
2. `UserLoginLog.Detail` 拒绝跨租户日志
3. `UserIdentity.Detail/Update/Delete` 拒绝跨租户身份
4. `PageList/GetByUser` 等列表查询统一使用上下文租户

- [ ] **Step 2: 运行测试确认失败**

Run: `go test ./apps/iam/internal/service/svcuser -run 'Test(User|UserIdentity|UserLoginLog)' -v`

Expected: FAIL

- [ ] **Step 3: 写最小实现**

1. 为 `user.go`、`user_identity.go` 增加必要 repo seam
2. detail/update/delete 统一做 `TenantID == gincontext.GetTenantID(ctx)` 判断
3. 列表和按用户查询统一使用上下文租户

- [ ] **Step 4: 运行测试确认通过**

Run: `go test ./apps/iam/internal/service/svcuser -run 'Test(User|UserIdentity|UserLoginLog)' -v`

Expected: PASS

- [ ] **Step 5: 运行模块全量测试**

Run: `go test ./apps/iam/internal/service/svcuser -v`

Expected: PASS

---

### Task 5: 全量回归

**Files:**
- Modify: `docs/superpowers/plans/2026-05-10-tenant-scoped-object-access-plan.md`

- [ ] **Step 1: 运行 service 全量测试**

Run: `go test ./apps/iam/internal/service/...`

Expected: PASS

- [ ] **Step 2: 运行 IAM 全量测试**

Run: `go test ./apps/iam/...`

Expected: PASS

- [ ] **Step 3: 更新计划勾选状态**

完成后确认记录以下结果：

```text
- svctenant 对象访问改为上下文租户边界
- svcpermission 对象访问改为上下文租户边界
- svcapplication 与 svcauth 对象访问改为上下文租户边界
- svcuser 对象访问改为上下文租户边界
- 所有定向与 IAM 全量测试通过
```

---

## Self-Review

- Spec coverage：4 个子批次都已落成独立任务，并覆盖 detail/update/delete/create/list 的租户边界。
- Placeholder scan：无 TBD/TODO/“类似上一任务”占位项。
- Type consistency：计划统一采用 `gincontext.GetTenantID(ctx)`、service repo seam、`NotExistError` 语义，与现有仓库模式一致。
