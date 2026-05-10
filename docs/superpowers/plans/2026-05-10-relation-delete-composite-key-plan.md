# IAM 关系解绑组合键删除修复 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 修复 5 个 IAM 关系解绑接口把业务 ID 当关系表主键删除的问题，并用回归测试锁定“按租户上下文 + 组合键查关系，再按真实关系主键删除”的行为。

**Architecture:** 保持现有 controller/DTO/DAO 结构不变，只在 service 删除路径增加最小 repo seam，并把删除逻辑改成两步：先用上下文 `tenant_id` 和业务组合键查询关系，再删除查询到的 `entity.ID`。测试沿用仓库现有 service stub 注入模式，不引入数据库级测试基础设施。

**Tech Stack:** Go, Gin, GORM, genericdao, 标准库 testing

---

## File Structure

- Modify: `backend/apps/iam/internal/service/svcpermission/user_role.go`
  责任：为 UserRole 删除引入可替换 repo seam，并改为按 `tenant_id + user_id + role_id` 查询后删除真实关系主键。
- Create: `backend/apps/iam/internal/service/svcpermission/user_role_delete_test.go`
  责任：覆盖 UserRole 删除使用组合键、忽略请求体租户、查不到不删除。
- Modify: `backend/apps/iam/internal/service/svcpermission/role_menu.go`
  责任：为 RoleMenu 删除引入 seam，并改为按 `tenant_id + role_id + menu_id` 查询后删除真实关系主键。
- Create: `backend/apps/iam/internal/service/svcpermission/role_menu_delete_test.go`
  责任：覆盖 RoleMenu 删除的组合键与 tenant 边界行为。
- Modify: `backend/apps/iam/internal/service/svcpermission/role_scope.go`
  责任：为 RoleScope 删除引入 seam，并改为按 `tenant_id + role_id + scope_id` 查询后删除真实关系主键。
- Create: `backend/apps/iam/internal/service/svcpermission/role_scope_delete_test.go`
  责任：覆盖 RoleScope 删除的组合键与 tenant 边界行为。
- Modify: `backend/apps/iam/internal/service/svctenant/organization_user_relation.go`
  责任：为 OrganizationUserRelation 删除引入 seam，并改为按 `tenant_id + organization_id + user_id` 查询后删除真实关系主键。
- Create: `backend/apps/iam/internal/service/svctenant/organization_user_relation_delete_test.go`
  责任：覆盖 OrganizationUserRelation 删除的组合键与 tenant 边界行为。
- Modify: `backend/apps/iam/internal/service/svctenant/organization_role_user_relation.go`
  责任：为 OrganizationRoleUserRelation 删除引入 seam，并改为按 `tenant_id + organization_role_id + user_id` 查询后删除真实关系主键。
- Create: `backend/apps/iam/internal/service/svctenant/organization_role_user_relation_delete_test.go`
  责任：覆盖 OrganizationRoleUserRelation 删除的组合键与 tenant 边界行为。

---

### Task 1: 修复 UserRole 删除

**Files:**
- Modify: `backend/apps/iam/internal/service/svcpermission/user_role.go`
- Test: `backend/apps/iam/internal/service/svcpermission/user_role_delete_test.go`

- [ ] **Step 1: 写失败测试**

在 `backend/apps/iam/internal/service/svcpermission/user_role_delete_test.go` 中新增以下测试和 stub。第一个测试锁定删除应使用上下文租户和组合键查询，并删除真实关系行主键；第二个测试锁定查不到关系时不能调用删除。

```go
package svcpermission

import (
    "context"
    "testing"

    "github.com/gin-gonic/gin"
    "github.com/morehao/ark-iam/iam/dao"
    "github.com/morehao/ark-iam/iam/internal/dto/dtopermission"
    "github.com/morehao/ark-iam/iam/model"
    "github.com/morehao/golib/biz/gcontext"
    "github.com/morehao/golib/biz/genericdao"
    "gorm.io/gorm"
)

func TestDeleteUserRoleUsesTenantScopedCompositeLookup(t *testing.T) {
    ginCtx, _ := gin.CreateTestContext(nil)
    ginCtx.Set(gcontext.KeyTenantID, uint(41))
    ginCtx.Set(gcontext.KeyUserID, uint(9001))

    repo := &stubUserRoleDeleteRepo{
        list: model.UserRoleEntityList{{
            Model:    gorm.Model{ID: 77},
            TenantID: 41,
            UserID:   12,
            RoleID:   34,
        }},
    }
    installUserRoleDeleteRepo(t, repo)

    svc := &userRoleSvc{}
    err := svc.Delete(ginCtx, &dtopermission.UserRoleDeleteReq{TenantID: 999, UserID: 12, RoleID: 34})
    if err != nil {
        t.Fatalf("Delete returned error: %v", err)
    }
    if repo.lastCond == nil {
        t.Fatalf("expected delete lookup condition to be captured")
    }
    if repo.lastCond.TenantID != 41 {
        t.Fatalf("expected tenant lookup 41 from context, got %d", repo.lastCond.TenantID)
    }
    if repo.lastCond.UserID != 12 || repo.lastCond.RoleID != 34 {
        t.Fatalf("unexpected composite lookup: %+v", repo.lastCond)
    }
    if repo.deletedID != 77 {
        t.Fatalf("expected delete by relation id 77, got %d", repo.deletedID)
    }
    if repo.deletedBy != 9001 {
        t.Fatalf("expected deletedBy 9001, got %d", repo.deletedBy)
    }
}

func TestDeleteUserRoleReturnsNotExistWhenCompositeLookupMisses(t *testing.T) {
    ginCtx, _ := gin.CreateTestContext(nil)
    ginCtx.Set(gcontext.KeyTenantID, uint(42))
    ginCtx.Set(gcontext.KeyUserID, uint(9002))

    repo := &stubUserRoleDeleteRepo{}
    installUserRoleDeleteRepo(t, repo)

    svc := &userRoleSvc{}
    err := svc.Delete(ginCtx, &dtopermission.UserRoleDeleteReq{UserID: 12, RoleID: 34})
    if err == nil {
        t.Fatalf("expected not exist error")
    }
    if repo.deletedID != 0 {
        t.Fatalf("expected no delete call, got deletedID=%d", repo.deletedID)
    }
}

type stubUserRoleDeleteRepo struct {
    list     model.UserRoleEntityList
    listErr  error
    deleteErr error
    lastCond *dao.UserRoleCond
    deletedID uint
    deletedBy uint
}

func (r *stubUserRoleDeleteRepo) GetListByCond(ctx context.Context, cond genericdao.Cond) (model.UserRoleEntityList, error) {
    typed, _ := cond.(*dao.UserRoleCond)
    if typed != nil {
        clone := *typed
        if typed.BaseCond != nil {
            base := *typed.BaseCond
            clone.BaseCond = &base
        }
        r.lastCond = &clone
    }
    return r.list, r.listErr
}

func (r *stubUserRoleDeleteRepo) Delete(ctx context.Context, id uint, userID uint) error {
    r.deletedID = id
    r.deletedBy = userID
    return r.deleteErr
}

func installUserRoleDeleteRepo(t *testing.T, repo userRoleDeleteRepository) {
    t.Helper()
    prev := newUserRoleDeleteRepo
    newUserRoleDeleteRepo = func() userRoleDeleteRepository {
        return repo
    }
    t.Cleanup(func() {
        newUserRoleDeleteRepo = prev
    })
}

var _ userRoleDeleteRepository = (*stubUserRoleDeleteRepo)(nil)
```

- [ ] **Step 2: 运行测试确认失败**

Run: `go test ./apps/iam/internal/service/svcpermission -run 'TestDeleteUserRole' -v`

Expected: FAIL，现有实现会把 `req.UserID` 当成关系表主键删除，导致 `deletedID` 断言失败，且 `lastCond` 为空或未按组合键查询。

- [ ] **Step 3: 写最小实现**

在 `backend/apps/iam/internal/service/svcpermission/user_role.go` 中加入 repo seam，并改写 `Delete`：

```go
package svcpermission

import (
    "context"

    "github.com/gin-gonic/gin"
    "github.com/morehao/ark-iam/iam/dao"
    "github.com/morehao/ark-iam/iam/internal/dto/dtopermission"
    "github.com/morehao/ark-iam/iam/model"
    "github.com/morehao/ark-iam/pkg/code"
    "github.com/morehao/golib/biz/gcontext/gincontext"
    "github.com/morehao/golib/biz/genericdao"
    "github.com/morehao/golib/glog"
    "github.com/morehao/golib/gutil"
)

type userRoleDeleteRepository interface {
    GetListByCond(ctx context.Context, cond genericdao.Cond) (model.UserRoleEntityList, error)
    Delete(ctx context.Context, id uint, userID uint) error
}

var newUserRoleDeleteRepo = func() userRoleDeleteRepository {
    return dao.NewUserRoleDao()
}

func (svc *userRoleSvc) Delete(ctx *gin.Context, req *dtopermission.UserRoleDeleteReq) error {
    repo := newUserRoleDeleteRepo()
    list, err := repo.GetListByCond(ctx, &dao.UserRoleCond{
        TenantID: gincontext.GetTenantID(ctx),
        UserID:   req.UserID,
        RoleID:   req.RoleID,
    })
    if err != nil {
        glog.Errorf(ctx, "[svcpermission.DeleteUserRole] dao GetListByCond fail, err:%v, req:%s", err, gutil.ToJsonString(req))
        return code.GetError(code.UserRoleDeleteError)
    }
    if len(list) == 0 || list[0].ID == 0 {
        return code.GetError(code.UserRoleNotExistError)
    }

    userID := gincontext.GetUserID(ctx)
    if err := repo.Delete(ctx, list[0].ID, userID); err != nil {
        glog.Errorf(ctx, "[svcpermission.DeleteUserRole] dao Delete fail, err:%v, req:%s", err, gutil.ToJsonString(req))
        return code.GetError(code.UserRoleDeleteError)
    }
    return nil
}
```

- [ ] **Step 4: 运行测试确认通过**

Run: `go test ./apps/iam/internal/service/svcpermission -run 'TestDeleteUserRole' -v`

Expected: PASS

- [ ] **Step 5: 提交当前最小修复**

```bash
git add apps/iam/internal/service/svcpermission/user_role.go apps/iam/internal/service/svcpermission/user_role_delete_test.go
git commit -m "fix(iam): delete user roles by relation key"
```

---

### Task 2: 修复 RoleMenu 删除

**Files:**
- Modify: `backend/apps/iam/internal/service/svcpermission/role_menu.go`
- Test: `backend/apps/iam/internal/service/svcpermission/role_menu_delete_test.go`

- [ ] **Step 1: 写失败测试**

在 `backend/apps/iam/internal/service/svcpermission/role_menu_delete_test.go` 中新增以下内容：

```go
package svcpermission

import (
    "context"
    "testing"

    "github.com/gin-gonic/gin"
    "github.com/morehao/ark-iam/iam/dao"
    "github.com/morehao/ark-iam/iam/internal/dto/dtopermission"
    "github.com/morehao/ark-iam/iam/model"
    "github.com/morehao/golib/biz/gcontext"
    "github.com/morehao/golib/biz/genericdao"
    "gorm.io/gorm"
)

func TestDeleteRoleMenuUsesTenantScopedCompositeLookup(t *testing.T) {
    ginCtx, _ := gin.CreateTestContext(nil)
    ginCtx.Set(gcontext.KeyTenantID, uint(51))
    ginCtx.Set(gcontext.KeyUserID, uint(9101))

    repo := &stubRoleMenuDeleteRepo{
        list: model.RoleMenuEntityList{{
            Model:    gorm.Model{ID: 88},
            TenantID: 51,
            RoleID:   21,
            MenuID:   43,
        }},
    }
    installRoleMenuDeleteRepo(t, repo)

    svc := &roleMenuSvc{}
    err := svc.Delete(ginCtx, &dtopermission.RoleMenuDeleteReq{TenantID: 999, RoleID: 21, MenuID: 43})
    if err != nil {
        t.Fatalf("Delete returned error: %v", err)
    }
    if repo.lastCond == nil {
        t.Fatalf("expected lookup condition to be captured")
    }
    if repo.lastCond.TenantID != 51 || repo.lastCond.RoleID != 21 || repo.lastCond.MenuID != 43 {
        t.Fatalf("unexpected composite lookup: %+v", repo.lastCond)
    }
    if repo.deletedID != 88 {
        t.Fatalf("expected delete by relation id 88, got %d", repo.deletedID)
    }
}

func TestDeleteRoleMenuReturnsNotExistWhenCompositeLookupMisses(t *testing.T) {
    ginCtx, _ := gin.CreateTestContext(nil)
    ginCtx.Set(gcontext.KeyTenantID, uint(52))
    ginCtx.Set(gcontext.KeyUserID, uint(9102))

    repo := &stubRoleMenuDeleteRepo{}
    installRoleMenuDeleteRepo(t, repo)

    svc := &roleMenuSvc{}
    err := svc.Delete(ginCtx, &dtopermission.RoleMenuDeleteReq{RoleID: 21, MenuID: 43})
    if err == nil {
        t.Fatalf("expected not exist error")
    }
    if repo.deletedID != 0 {
        t.Fatalf("expected no delete call, got deletedID=%d", repo.deletedID)
    }
}

type stubRoleMenuDeleteRepo struct {
    list      model.RoleMenuEntityList
    listErr   error
    deleteErr error
    lastCond  *dao.RoleMenuCond
    deletedID uint
    deletedBy uint
}

func (r *stubRoleMenuDeleteRepo) GetListByCond(ctx context.Context, cond genericdao.Cond) (model.RoleMenuEntityList, error) {
    typed, _ := cond.(*dao.RoleMenuCond)
    if typed != nil {
        clone := *typed
        if typed.BaseCond != nil {
            base := *typed.BaseCond
            clone.BaseCond = &base
        }
        r.lastCond = &clone
    }
    return r.list, r.listErr
}

func (r *stubRoleMenuDeleteRepo) Delete(ctx context.Context, id uint, userID uint) error {
    r.deletedID = id
    r.deletedBy = userID
    return r.deleteErr
}

func installRoleMenuDeleteRepo(t *testing.T, repo roleMenuDeleteRepository) {
    t.Helper()
    prev := newRoleMenuDeleteRepo
    newRoleMenuDeleteRepo = func() roleMenuDeleteRepository {
        return repo
    }
    t.Cleanup(func() {
        newRoleMenuDeleteRepo = prev
    })
}

var _ roleMenuDeleteRepository = (*stubRoleMenuDeleteRepo)(nil)
```

- [ ] **Step 2: 运行测试确认失败**

Run: `go test ./apps/iam/internal/service/svcpermission -run 'TestDeleteRoleMenu' -v`

Expected: FAIL，现有实现会按 `req.RoleID` 调 `GetByID/Delete`，无法通过真实关系主键删除断言。

- [ ] **Step 3: 写最小实现**

在 `backend/apps/iam/internal/service/svcpermission/role_menu.go` 中加入：

```go
type roleMenuDeleteRepository interface {
    GetListByCond(ctx context.Context, cond genericdao.Cond) (model.RoleMenuEntityList, error)
    Delete(ctx context.Context, id uint, userID uint) error
}

var newRoleMenuDeleteRepo = func() roleMenuDeleteRepository {
    return dao.NewRoleMenuDao()
}

func (svc *roleMenuSvc) Delete(ctx *gin.Context, req *dtopermission.RoleMenuDeleteReq) error {
    repo := newRoleMenuDeleteRepo()
    list, err := repo.GetListByCond(ctx, &dao.RoleMenuCond{
        TenantID: gincontext.GetTenantID(ctx),
        RoleID:   req.RoleID,
        MenuID:   req.MenuID,
    })
    if err != nil {
        glog.Errorf(ctx, "[svcpermission.DeleteRoleMenu] dao GetListByCond fail, err:%v, req:%s", err, gutil.ToJsonString(req))
        return code.GetError(code.RoleMenuDeleteError)
    }
    if len(list) == 0 || list[0].ID == 0 {
        return code.GetError(code.RoleMenuNotExistError)
    }
    if err := repo.Delete(ctx, list[0].ID, gincontext.GetUserID(ctx)); err != nil {
        glog.Errorf(ctx, "[svcpermission.DeleteRoleMenu] dao Delete fail, err:%v, req:%s", err, gutil.ToJsonString(req))
        return code.GetError(code.RoleMenuDeleteError)
    }
    return nil
}
```

- [ ] **Step 4: 运行测试确认通过**

Run: `go test ./apps/iam/internal/service/svcpermission -run 'TestDeleteRoleMenu' -v`

Expected: PASS

- [ ] **Step 5: 提交当前最小修复**

```bash
git add apps/iam/internal/service/svcpermission/role_menu.go apps/iam/internal/service/svcpermission/role_menu_delete_test.go
git commit -m "fix(iam): delete role menus by relation key"
```

---

### Task 3: 修复 RoleScope 删除

**Files:**
- Modify: `backend/apps/iam/internal/service/svcpermission/role_scope.go`
- Test: `backend/apps/iam/internal/service/svcpermission/role_scope_delete_test.go`

- [ ] **Step 1: 写失败测试**

在 `backend/apps/iam/internal/service/svcpermission/role_scope_delete_test.go` 中新增：

```go
package svcpermission

import (
    "context"
    "testing"

    "github.com/gin-gonic/gin"
    "github.com/morehao/ark-iam/iam/dao"
    "github.com/morehao/ark-iam/iam/internal/dto/dtopermission"
    "github.com/morehao/ark-iam/iam/model"
    "github.com/morehao/golib/biz/gcontext"
    "github.com/morehao/golib/biz/genericdao"
    "gorm.io/gorm"
)

func TestDeleteRoleScopeUsesTenantScopedCompositeLookup(t *testing.T) {
    ginCtx, _ := gin.CreateTestContext(nil)
    ginCtx.Set(gcontext.KeyTenantID, uint(61))
    ginCtx.Set(gcontext.KeyUserID, uint(9201))

    repo := &stubRoleScopeDeleteRepo{
        list: model.RoleScopeEntityList{{
            Model:    gorm.Model{ID: 99},
            TenantID: 61,
            RoleID:   31,
            ScopeID:  53,
        }},
    }
    installRoleScopeDeleteRepo(t, repo)

    svc := &roleScopeSvc{}
    err := svc.Delete(ginCtx, &dtopermission.RoleScopeDeleteReq{TenantID: 999, RoleID: 31, ScopeID: 53})
    if err != nil {
        t.Fatalf("Delete returned error: %v", err)
    }
    if repo.lastCond == nil {
        t.Fatalf("expected lookup condition to be captured")
    }
    if repo.lastCond.TenantID != 61 || repo.lastCond.RoleID != 31 || repo.lastCond.ScopeID != 53 {
        t.Fatalf("unexpected composite lookup: %+v", repo.lastCond)
    }
    if repo.deletedID != 99 {
        t.Fatalf("expected delete by relation id 99, got %d", repo.deletedID)
    }
}

func TestDeleteRoleScopeReturnsNotExistWhenCompositeLookupMisses(t *testing.T) {
    ginCtx, _ := gin.CreateTestContext(nil)
    ginCtx.Set(gcontext.KeyTenantID, uint(62))
    ginCtx.Set(gcontext.KeyUserID, uint(9202))

    repo := &stubRoleScopeDeleteRepo{}
    installRoleScopeDeleteRepo(t, repo)

    svc := &roleScopeSvc{}
    err := svc.Delete(ginCtx, &dtopermission.RoleScopeDeleteReq{RoleID: 31, ScopeID: 53})
    if err == nil {
        t.Fatalf("expected not exist error")
    }
    if repo.deletedID != 0 {
        t.Fatalf("expected no delete call, got deletedID=%d", repo.deletedID)
    }
}

type stubRoleScopeDeleteRepo struct {
    list      model.RoleScopeEntityList
    listErr   error
    deleteErr error
    lastCond  *dao.RoleScopeCond
    deletedID uint
    deletedBy uint
}

func (r *stubRoleScopeDeleteRepo) GetListByCond(ctx context.Context, cond genericdao.Cond) (model.RoleScopeEntityList, error) {
    typed, _ := cond.(*dao.RoleScopeCond)
    if typed != nil {
        clone := *typed
        if typed.BaseCond != nil {
            base := *typed.BaseCond
            clone.BaseCond = &base
        }
        r.lastCond = &clone
    }
    return r.list, r.listErr
}

func (r *stubRoleScopeDeleteRepo) Delete(ctx context.Context, id uint, userID uint) error {
    r.deletedID = id
    r.deletedBy = userID
    return r.deleteErr
}

func installRoleScopeDeleteRepo(t *testing.T, repo roleScopeDeleteRepository) {
    t.Helper()
    prev := newRoleScopeDeleteRepo
    newRoleScopeDeleteRepo = func() roleScopeDeleteRepository {
        return repo
    }
    t.Cleanup(func() {
        newRoleScopeDeleteRepo = prev
    })
}

var _ roleScopeDeleteRepository = (*stubRoleScopeDeleteRepo)(nil)
```

- [ ] **Step 2: 运行测试确认失败**

Run: `go test ./apps/iam/internal/service/svcpermission -run 'TestDeleteRoleScope' -v`

Expected: FAIL，现有实现会按 `req.RoleID` 删除而不是关系主键。

- [ ] **Step 3: 写最小实现**

在 `backend/apps/iam/internal/service/svcpermission/role_scope.go` 中加入：

```go
type roleScopeDeleteRepository interface {
    GetListByCond(ctx context.Context, cond genericdao.Cond) (model.RoleScopeEntityList, error)
    Delete(ctx context.Context, id uint, userID uint) error
}

var newRoleScopeDeleteRepo = func() roleScopeDeleteRepository {
    return dao.NewRoleScopeDao()
}

func (svc *roleScopeSvc) Delete(ctx *gin.Context, req *dtopermission.RoleScopeDeleteReq) error {
    repo := newRoleScopeDeleteRepo()
    list, err := repo.GetListByCond(ctx, &dao.RoleScopeCond{
        TenantID: gincontext.GetTenantID(ctx),
        RoleID:   req.RoleID,
        ScopeID:  req.ScopeID,
    })
    if err != nil {
        glog.Errorf(ctx, "[svcpermission.DeleteRoleScope] dao GetListByCond fail, err:%v, req:%s", err, gutil.ToJsonString(req))
        return code.GetError(code.RoleScopeDeleteError)
    }
    if len(list) == 0 || list[0].ID == 0 {
        return code.GetError(code.RoleScopeNotExistError)
    }
    if err := repo.Delete(ctx, list[0].ID, gincontext.GetUserID(ctx)); err != nil {
        glog.Errorf(ctx, "[svcpermission.DeleteRoleScope] dao Delete fail, err:%v, req:%s", err, gutil.ToJsonString(req))
        return code.GetError(code.RoleScopeDeleteError)
    }
    return nil
}
```

- [ ] **Step 4: 运行测试确认通过**

Run: `go test ./apps/iam/internal/service/svcpermission -run 'TestDeleteRoleScope' -v`

Expected: PASS

- [ ] **Step 5: 提交当前最小修复**

```bash
git add apps/iam/internal/service/svcpermission/role_scope.go apps/iam/internal/service/svcpermission/role_scope_delete_test.go
git commit -m "fix(iam): delete role scopes by relation key"
```

---

### Task 4: 修复 OrganizationUserRelation 删除

**Files:**
- Modify: `backend/apps/iam/internal/service/svctenant/organization_user_relation.go`
- Test: `backend/apps/iam/internal/service/svctenant/organization_user_relation_delete_test.go`

- [ ] **Step 1: 写失败测试**

在 `backend/apps/iam/internal/service/svctenant/organization_user_relation_delete_test.go` 中新增：

```go
package svctenant

import (
    "context"
    "testing"

    "github.com/gin-gonic/gin"
    "github.com/morehao/ark-iam/iam/dao"
    "github.com/morehao/ark-iam/iam/internal/dto/dtotenant"
    "github.com/morehao/ark-iam/iam/model"
    "github.com/morehao/golib/biz/gcontext"
    "github.com/morehao/golib/biz/genericdao"
    "gorm.io/gorm"
)

func TestDeleteOrganizationUserRelationUsesTenantScopedCompositeLookup(t *testing.T) {
    ginCtx, _ := gin.CreateTestContext(nil)
    ginCtx.Set(gcontext.KeyTenantID, uint(71))
    ginCtx.Set(gcontext.KeyUserID, uint(9301))

    repo := &stubOrganizationUserRelationDeleteRepo{
        list: model.OrganizationUserRelationEntityList{{
            Model:          gorm.Model{ID: 109},
            TenantID:       71,
            OrganizationID: 201,
            UserID:         301,
        }},
    }
    installOrganizationUserRelationDeleteRepo(t, repo)

    svc := &organizationUserRelationSvc{}
    err := svc.Delete(ginCtx, &dtotenant.OrganizationUserRelationDeleteReq{OrganizationID: 201, UserID: 301})
    if err != nil {
        t.Fatalf("Delete returned error: %v", err)
    }
    if repo.lastCond == nil {
        t.Fatalf("expected lookup condition to be captured")
    }
    if repo.lastCond.TenantID != 71 || repo.lastCond.OrganizationID != 201 || repo.lastCond.UserID != 301 {
        t.Fatalf("unexpected composite lookup: %+v", repo.lastCond)
    }
    if repo.deletedID != 109 {
        t.Fatalf("expected delete by relation id 109, got %d", repo.deletedID)
    }
}

func TestDeleteOrganizationUserRelationReturnsNotExistWhenCompositeLookupMisses(t *testing.T) {
    ginCtx, _ := gin.CreateTestContext(nil)
    ginCtx.Set(gcontext.KeyTenantID, uint(72))
    ginCtx.Set(gcontext.KeyUserID, uint(9302))

    repo := &stubOrganizationUserRelationDeleteRepo{}
    installOrganizationUserRelationDeleteRepo(t, repo)

    svc := &organizationUserRelationSvc{}
    err := svc.Delete(ginCtx, &dtotenant.OrganizationUserRelationDeleteReq{OrganizationID: 201, UserID: 301})
    if err == nil {
        t.Fatalf("expected not exist error")
    }
    if repo.deletedID != 0 {
        t.Fatalf("expected no delete call, got deletedID=%d", repo.deletedID)
    }
}

type stubOrganizationUserRelationDeleteRepo struct {
    list      model.OrganizationUserRelationEntityList
    listErr   error
    deleteErr error
    lastCond  *dao.OrganizationUserRelationCond
    deletedID uint
    deletedBy uint
}

func (r *stubOrganizationUserRelationDeleteRepo) GetListByCond(ctx context.Context, cond genericdao.Cond) (model.OrganizationUserRelationEntityList, error) {
    typed, _ := cond.(*dao.OrganizationUserRelationCond)
    if typed != nil {
        clone := *typed
        if typed.BaseCond != nil {
            base := *typed.BaseCond
            clone.BaseCond = &base
        }
        r.lastCond = &clone
    }
    return r.list, r.listErr
}

func (r *stubOrganizationUserRelationDeleteRepo) Delete(ctx context.Context, id uint, userID uint) error {
    r.deletedID = id
    r.deletedBy = userID
    return r.deleteErr
}

func installOrganizationUserRelationDeleteRepo(t *testing.T, repo organizationUserRelationDeleteRepository) {
    t.Helper()
    prev := newOrganizationUserRelationDeleteRepo
    newOrganizationUserRelationDeleteRepo = func() organizationUserRelationDeleteRepository {
        return repo
    }
    t.Cleanup(func() {
        newOrganizationUserRelationDeleteRepo = prev
    })
}

var _ organizationUserRelationDeleteRepository = (*stubOrganizationUserRelationDeleteRepo)(nil)
```

- [ ] **Step 2: 运行测试确认失败**

Run: `go test ./apps/iam/internal/service/svctenant -run 'TestDeleteOrganizationUserRelation' -v`

Expected: FAIL，现有实现按 `req.OrganizationID` 删除而不是关系主键。

- [ ] **Step 3: 写最小实现**

在 `backend/apps/iam/internal/service/svctenant/organization_user_relation.go` 中加入：

```go
type organizationUserRelationDeleteRepository interface {
    GetListByCond(ctx context.Context, cond genericdao.Cond) (model.OrganizationUserRelationEntityList, error)
    Delete(ctx context.Context, id uint, userID uint) error
}

var newOrganizationUserRelationDeleteRepo = func() organizationUserRelationDeleteRepository {
    return dao.NewOrganizationUserRelationDao()
}

func (svc *organizationUserRelationSvc) Delete(ctx *gin.Context, req *dtotenant.OrganizationUserRelationDeleteReq) error {
    repo := newOrganizationUserRelationDeleteRepo()
    list, err := repo.GetListByCond(ctx, &dao.OrganizationUserRelationCond{
        TenantID:       gincontext.GetTenantID(ctx),
        OrganizationID: req.OrganizationID,
        UserID:         req.UserID,
    })
    if err != nil {
        glog.Errorf(ctx, "[svcorganizationuserrelation.Delete] dao GetListByCond fail, err:%v, req:%s", err, gutil.ToJsonString(req))
        return code.GetError(code.OrganizationUserRelationDeleteError)
    }
    if len(list) == 0 || list[0].ID == 0 {
        return code.GetError(code.OrganizationUserRelationNotExistError)
    }
    if err := repo.Delete(ctx, list[0].ID, gincontext.GetUserID(ctx)); err != nil {
        glog.Errorf(ctx, "[svcorganizationuserrelation.Delete] dao Delete fail, err:%v, req:%s", err, gutil.ToJsonString(req))
        return code.GetError(code.OrganizationUserRelationDeleteError)
    }
    return nil
}
```

- [ ] **Step 4: 运行测试确认通过**

Run: `go test ./apps/iam/internal/service/svctenant -run 'TestDeleteOrganizationUserRelation' -v`

Expected: PASS

- [ ] **Step 5: 提交当前最小修复**

```bash
git add apps/iam/internal/service/svctenant/organization_user_relation.go apps/iam/internal/service/svctenant/organization_user_relation_delete_test.go
git commit -m "fix(iam): delete organization users by relation key"
```

---

### Task 5: 修复 OrganizationRoleUserRelation 删除

**Files:**
- Modify: `backend/apps/iam/internal/service/svctenant/organization_role_user_relation.go`
- Test: `backend/apps/iam/internal/service/svctenant/organization_role_user_relation_delete_test.go`

- [ ] **Step 1: 写失败测试**

在 `backend/apps/iam/internal/service/svctenant/organization_role_user_relation_delete_test.go` 中新增：

```go
package svctenant

import (
    "context"
    "testing"

    "github.com/gin-gonic/gin"
    "github.com/morehao/ark-iam/iam/dao"
    "github.com/morehao/ark-iam/iam/internal/dto/dtotenant"
    "github.com/morehao/ark-iam/iam/model"
    "github.com/morehao/golib/biz/gcontext"
    "github.com/morehao/golib/biz/genericdao"
    "gorm.io/gorm"
)

func TestDeleteOrganizationRoleUserRelationUsesTenantScopedCompositeLookup(t *testing.T) {
    ginCtx, _ := gin.CreateTestContext(nil)
    ginCtx.Set(gcontext.KeyTenantID, uint(81))
    ginCtx.Set(gcontext.KeyUserID, uint(9401))

    repo := &stubOrganizationRoleUserRelationDeleteRepo{
        list: model.OrganizationRoleUserRelationEntityList{{
            Model:              gorm.Model{ID: 119},
            TenantID:           81,
            OrganizationID:     401,
            OrganizationRoleID: 501,
            UserID:             601,
        }},
    }
    installOrganizationRoleUserRelationDeleteRepo(t, repo)

    svc := &organizationRoleUserRelationSvc{}
    err := svc.Delete(ginCtx, &dtotenant.OrganizationRoleUserRelationDeleteReq{OrganizationRoleID: 501, UserID: 601})
    if err != nil {
        t.Fatalf("Delete returned error: %v", err)
    }
    if repo.lastCond == nil {
        t.Fatalf("expected lookup condition to be captured")
    }
    if repo.lastCond.TenantID != 81 || repo.lastCond.OrganizationRoleID != 501 || repo.lastCond.UserID != 601 {
        t.Fatalf("unexpected composite lookup: %+v", repo.lastCond)
    }
    if repo.deletedID != 119 {
        t.Fatalf("expected delete by relation id 119, got %d", repo.deletedID)
    }
}

func TestDeleteOrganizationRoleUserRelationReturnsNotExistWhenCompositeLookupMisses(t *testing.T) {
    ginCtx, _ := gin.CreateTestContext(nil)
    ginCtx.Set(gcontext.KeyTenantID, uint(82))
    ginCtx.Set(gcontext.KeyUserID, uint(9402))

    repo := &stubOrganizationRoleUserRelationDeleteRepo{}
    installOrganizationRoleUserRelationDeleteRepo(t, repo)

    svc := &organizationRoleUserRelationSvc{}
    err := svc.Delete(ginCtx, &dtotenant.OrganizationRoleUserRelationDeleteReq{OrganizationRoleID: 501, UserID: 601})
    if err == nil {
        t.Fatalf("expected not exist error")
    }
    if repo.deletedID != 0 {
        t.Fatalf("expected no delete call, got deletedID=%d", repo.deletedID)
    }
}

type stubOrganizationRoleUserRelationDeleteRepo struct {
    list      model.OrganizationRoleUserRelationEntityList
    listErr   error
    deleteErr error
    lastCond  *dao.OrganizationRoleUserRelationCond
    deletedID uint
    deletedBy uint
}

func (r *stubOrganizationRoleUserRelationDeleteRepo) GetListByCond(ctx context.Context, cond genericdao.Cond) (model.OrganizationRoleUserRelationEntityList, error) {
    typed, _ := cond.(*dao.OrganizationRoleUserRelationCond)
    if typed != nil {
        clone := *typed
        if typed.BaseCond != nil {
            base := *typed.BaseCond
            clone.BaseCond = &base
        }
        r.lastCond = &clone
    }
    return r.list, r.listErr
}

func (r *stubOrganizationRoleUserRelationDeleteRepo) Delete(ctx context.Context, id uint, userID uint) error {
    r.deletedID = id
    r.deletedBy = userID
    return r.deleteErr
}

func installOrganizationRoleUserRelationDeleteRepo(t *testing.T, repo organizationRoleUserRelationDeleteRepository) {
    t.Helper()
    prev := newOrganizationRoleUserRelationDeleteRepo
    newOrganizationRoleUserRelationDeleteRepo = func() organizationRoleUserRelationDeleteRepository {
        return repo
    }
    t.Cleanup(func() {
        newOrganizationRoleUserRelationDeleteRepo = prev
    })
}

var _ organizationRoleUserRelationDeleteRepository = (*stubOrganizationRoleUserRelationDeleteRepo)(nil)
```

- [ ] **Step 2: 运行测试确认失败**

Run: `go test ./apps/iam/internal/service/svctenant -run 'TestDeleteOrganizationRoleUserRelation' -v`

Expected: FAIL，现有实现按 `req.OrganizationRoleID` 删除而不是关系主键。

- [ ] **Step 3: 写最小实现**

在 `backend/apps/iam/internal/service/svctenant/organization_role_user_relation.go` 中加入：

```go
type organizationRoleUserRelationDeleteRepository interface {
    GetListByCond(ctx context.Context, cond genericdao.Cond) (model.OrganizationRoleUserRelationEntityList, error)
    Delete(ctx context.Context, id uint, userID uint) error
}

var newOrganizationRoleUserRelationDeleteRepo = func() organizationRoleUserRelationDeleteRepository {
    return dao.NewOrganizationRoleUserRelationDao()
}

func (svc *organizationRoleUserRelationSvc) Delete(ctx *gin.Context, req *dtotenant.OrganizationRoleUserRelationDeleteReq) error {
    repo := newOrganizationRoleUserRelationDeleteRepo()
    list, err := repo.GetListByCond(ctx, &dao.OrganizationRoleUserRelationCond{
        TenantID:           gincontext.GetTenantID(ctx),
        OrganizationRoleID: req.OrganizationRoleID,
        UserID:             req.UserID,
    })
    if err != nil {
        glog.Errorf(ctx, "[svcorganizationroleuserrelation.Delete] dao GetListByCond fail, err:%v, req:%s", err, gutil.ToJsonString(req))
        return code.GetError(code.OrganizationRoleUserRelationDeleteError)
    }
    if len(list) == 0 || list[0].ID == 0 {
        return code.GetError(code.OrganizationRoleUserRelationNotExistError)
    }
    if err := repo.Delete(ctx, list[0].ID, gincontext.GetUserID(ctx)); err != nil {
        glog.Errorf(ctx, "[svcorganizationroleuserrelation.Delete] dao Delete fail, err:%v, req:%s", err, gutil.ToJsonString(req))
        return code.GetError(code.OrganizationRoleUserRelationDeleteError)
    }
    return nil
}
```

- [ ] **Step 4: 运行测试确认通过**

Run: `go test ./apps/iam/internal/service/svctenant -run 'TestDeleteOrganizationRoleUserRelation' -v`

Expected: PASS

- [ ] **Step 5: 提交当前最小修复**

```bash
git add apps/iam/internal/service/svctenant/organization_role_user_relation.go apps/iam/internal/service/svctenant/organization_role_user_relation_delete_test.go
git commit -m "fix(iam): delete organization role users by relation key"
```

---

### Task 6: 全量回归与审计收口

**Files:**
- Modify: `docs/superpowers/plans/2026-05-10-relation-delete-composite-key-plan.md`

- [ ] **Step 1: 运行 permission 定向测试**

Run: `go test ./apps/iam/internal/service/svcpermission -run 'TestDelete(UserRole|RoleMenu|RoleScope)' -v`

Expected: PASS

- [ ] **Step 2: 运行 tenant 定向测试**

Run: `go test ./apps/iam/internal/service/svctenant -run 'TestDeleteOrganization(UserRelation|RoleUserRelation)' -v`

Expected: PASS

- [ ] **Step 3: 运行 IAM service 全量测试**

Run: `go test ./apps/iam/internal/service/...`

Expected: PASS

- [ ] **Step 4: 运行 IAM 应用全量测试**

Run: `go test ./apps/iam/...`

Expected: PASS

- [ ] **Step 5: 更新计划勾选状态并准备合并说明**

确认每个任务都已勾选完成，并记录以下结果：

```text
- 5 个关系解绑点全部改为按上下文 tenant + 组合键查询
- 删除时统一使用关系表真实主键 entity.ID
- 请求体中的 TenantID 不再影响删除边界
- 定向 service 测试与 IAM 全量测试均通过
```

---

## Self-Review

- Spec coverage：5 个删除点都各有独立任务，且每个任务都覆盖组合键查询、tenant 边界、查不到不删除。
- Placeholder scan：计划中没有 `TODO/TBD/后续再补` 一类占位词。
- Type consistency：所有测试都围绕现有 DTO 字段名与 DAO 条件结构，删除 seam 统一使用 `GetListByCond + Delete`，与当前仓库接口一致。
