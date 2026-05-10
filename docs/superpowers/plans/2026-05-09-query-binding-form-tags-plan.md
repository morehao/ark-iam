# IAM Query 绑定 Form 标签修复 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 修复 `ShouldBindQuery` 场景下 DTO 缺少 `form` 标签导致的 query 参数绑定不稳定问题，并用回归测试锁定行为。

**Architecture:** 保持现有 controller/service/dao 结构不变，只在实际被 `ShouldBindQuery` 使用的 DTO 上补齐 `form` 标签。先通过控制器级绑定测试证明当前 query 参数无法稳定绑定，再做最小标签修改，最后运行相关模块测试验证未引入回归。

**Tech Stack:** Go, Gin, GORM, Testify/标准库 testing, swag 静态文档

---

## File Structure

- Modify: `backend/apps/iam/internal/dto/dtouser/request.go`
  责任：为 user 模块 query DTO 补 `form` 标签。
- Modify: `backend/apps/iam/internal/dto/dtotenant/request.go`
  责任：为 tenant/department 模块 query DTO 补 `form` 标签。
- Modify: `backend/apps/iam/internal/dto/dtotenant/system_request.go`
  责任：为 system/log 模块 query DTO 补 `form` 标签。
- Modify: `backend/apps/iam/internal/dto/dtoauth/request.go`
  责任：为 connector 模块 query DTO 补 `form` 标签。
- Modify: `backend/apps/iam/internal/dto/dtopermission/request.go`
  责任：为 menu 等 permission 模块 query DTO 补 `form` 标签。
- Create: `backend/apps/iam/internal/controller/query_binding_test.go`
  责任：新增控制器级 query 绑定回归测试，覆盖 user/tenant/connector 等典型路径。

---

### Task 1: 写 Query 绑定失败测试

**Files:**
- Create: `backend/apps/iam/internal/controller/query_binding_test.go`

- [ ] **Step 1: 写失败测试**

在 `backend/apps/iam/internal/controller/query_binding_test.go` 中新增绑定测试，只验证 `ShouldBindQuery` 是否能把 query 参数绑定到 DTO，不依赖真实 service。

```go
package controller_test

import (
    "net/http"
    "net/http/httptest"
    "testing"

    "github.com/gin-gonic/gin"
    "github.com/morehao/ark-iam/iam/internal/dto/dtoauth"
    "github.com/morehao/ark-iam/iam/internal/dto/dtotenant"
    "github.com/morehao/ark-iam/iam/internal/dto/dtouser"
)

func TestQueryBindingUserDetailReqUsesFormTags(t *testing.T) {
    gin.SetMode(gin.TestMode)
    engine := gin.New()
    var got dtouser.UserDetailReq
    engine.GET("/user/detail", func(ctx *gin.Context) {
        if err := ctx.ShouldBindQuery(&got); err != nil {
            ctx.String(http.StatusBadRequest, err.Error())
            return
        }
        ctx.Status(http.StatusNoContent)
    })

    req := httptest.NewRequest(http.MethodGet, "/user/detail?userID=12", nil)
    resp := httptest.NewRecorder()
    engine.ServeHTTP(resp, req)

    if resp.Code != http.StatusNoContent {
        t.Fatalf("expected status %d, got %d body=%s", http.StatusNoContent, resp.Code, resp.Body.String())
    }
    if got.UserID != 12 {
        t.Fatalf("expected userID 12, got %d", got.UserID)
    }
}

func TestQueryBindingTenantDetailReqUsesFormTags(t *testing.T) {
    gin.SetMode(gin.TestMode)
    engine := gin.New()
    var got dtotenant.TenantDetailReq
    engine.GET("/tenant/detail", func(ctx *gin.Context) {
        if err := ctx.ShouldBindQuery(&got); err != nil {
            ctx.String(http.StatusBadRequest, err.Error())
            return
        }
        ctx.Status(http.StatusNoContent)
    })

    req := httptest.NewRequest(http.MethodGet, "/tenant/detail?tenantID=34", nil)
    resp := httptest.NewRecorder()
    engine.ServeHTTP(resp, req)

    if resp.Code != http.StatusNoContent {
        t.Fatalf("expected status %d, got %d body=%s", http.StatusNoContent, resp.Code, resp.Body.String())
    }
    if got.TenantID != 34 {
        t.Fatalf("expected tenantID 34, got %d", got.TenantID)
    }
}

func TestQueryBindingConnectorDetailReqUsesFormTags(t *testing.T) {
    gin.SetMode(gin.TestMode)
    engine := gin.New()
    var got dtoauth.ConnectorDetailReq
    engine.GET("/connector/detail", func(ctx *gin.Context) {
        if err := ctx.ShouldBindQuery(&got); err != nil {
            ctx.String(http.StatusBadRequest, err.Error())
            return
        }
        ctx.Status(http.StatusNoContent)
    })

    req := httptest.NewRequest(http.MethodGet, "/connector/detail?connectorId=56", nil)
    resp := httptest.NewRecorder()
    engine.ServeHTTP(resp, req)

    if resp.Code != http.StatusNoContent {
        t.Fatalf("expected status %d, got %d body=%s", http.StatusNoContent, resp.Code, resp.Body.String())
    }
    if got.ConnectorID != 56 {
        t.Fatalf("expected connectorId 56, got %d", got.ConnectorID)
    }
}
```

- [ ] **Step 2: 运行测试确认失败**

Run: `go test ./apps/iam/internal/controller -run 'TestQueryBinding(UserDetailReqUsesFormTags|TenantDetailReqUsesFormTags|ConnectorDetailReqUsesFormTags)' -v`

Expected: 至少一个测试因 query 参数未绑定到 DTO 而失败，常见表现为 `400` 或字段仍为 `0`。

- [ ] **Step 3: 不改生产代码，先确认失败原因正确**

确认失败是由于缺少 `form` 标签，而不是测试路径、Gin 模式、或断言写错。

- [ ] **Step 4: 记录失败点后进入最小实现**

继续 Task 2。

---

### Task 2: 为实际 Query DTO 补齐 `form` 标签

**Files:**
- Modify: `backend/apps/iam/internal/dto/dtouser/request.go`
- Modify: `backend/apps/iam/internal/dto/dtotenant/request.go`
- Modify: `backend/apps/iam/internal/dto/dtotenant/system_request.go`
- Modify: `backend/apps/iam/internal/dto/dtoauth/request.go`
- Modify: `backend/apps/iam/internal/dto/dtopermission/request.go`

- [ ] **Step 1: 修改 `dtouser/request.go` 中 query DTO 标签**

补齐以下结构的 `form` 标签，名称与 `json` 一致：

```go
type UserDetailReq struct {
    UserID uint `json:"userID" form:"userID" binding:"required"`
}

type UserIdentityDetailReq struct {
    UserIdentityID uint `json:"userIdentityID" form:"userIdentityID" binding:"required"`
}

type UserIdentityByUserReq struct {
    UserID uint `json:"userID" form:"userID" binding:"required"`
}

type UserLoginLogDetailReq struct {
    UserLoginLogID uint `json:"userLoginLogID" form:"userLoginLogID" binding:"required"`
}

type UserLoginLogByUserReq struct {
    UserID uint `json:"userID" form:"userID" binding:"required"`
}

type UserDepartmentRelationByUserReq struct {
    UserID uint `json:"userID" form:"userID" binding:"required"`
}
```

- [ ] **Step 2: 修改 `dtotenant/request.go` 与 `system_request.go`**

为实际 query 使用的结构补齐 `form`：

```go
type TenantDetailReq struct {
    TenantID uint `json:"tenantID" form:"tenantID" binding:"required"`
}

type DepartmentDetailReq struct {
    DepartmentID uint `json:"departmentID" form:"departmentID" binding:"required"`
}

type DepartmentPageListReq struct {
    gobject.PageQuery
    TenantID uint   `json:"tenantID" form:"tenantID"`
    ParentID uint   `json:"parentID" form:"parentID"`
    Name     string `json:"name" form:"name"`
    Code     string `json:"code" form:"code"`
}

type DepartmentTreeReq struct {
    TenantID uint `json:"tenantID" form:"tenantID"`
}

type SystemDetailReq struct {
    SystemID uint `json:"systemID" form:"systemID" binding:"required"`
}

type SystemPageListReq struct {
    gobject.PageQuery
    TenantID uint   `json:"tenantID" form:"tenantID"`
    Key      string `json:"key" form:"key"`
}

type LogDetailReq struct {
    LogID uint `json:"logID" form:"logID" binding:"required"`
}

type LogPageListReq struct {
    gobject.PageQuery
    TenantID uint   `json:"tenantID" form:"tenantID"`
    Key      string `json:"key" form:"key"`
}
```

- [ ] **Step 3: 修改 `dtoauth/request.go` 与 `dtopermission/request.go`**

补齐 connector/menu 等 query DTO 的 `form` 标签：

```go
type ConnectorDetailReq struct {
    ConnectorID uint `json:"connectorId" form:"connectorId" binding:"required"`
}

type ConnectorPageListReq struct {
    gobject.PageQuery
    TenantID    uint   `json:"tenantId" form:"tenantId"`
    Protocol    string `json:"protocol" form:"protocol"`
    Provider    string `json:"provider" form:"provider"`
    Status      string `json:"status" form:"status"`
    Name        string `json:"name" form:"name"`
    DisplayName string `json:"displayName" form:"displayName"`
}

type MenuDetailReq struct {
    MenuID uint `json:"menuID" form:"menuID" binding:"required"`
}

type MenuPageListReq struct {
    gobject.PageQuery
    TenantID uint   `json:"tenantID" form:"tenantID"`
    ParentID uint   `json:"parentID" form:"parentID"`
    Name     string `json:"name" form:"name"`
    Code     string `json:"code" form:"code"`
    Type     string `json:"type" form:"type"`
    Status   string `json:"status" form:"status"`
}

type MenuTreeReq struct {
    TenantID uint `json:"tenantID" form:"tenantID"`
}
```

- [ ] **Step 4: 运行失败测试确认转绿**

Run: `go test ./apps/iam/internal/controller -run 'TestQueryBinding(UserDetailReqUsesFormTags|TenantDetailReqUsesFormTags|ConnectorDetailReqUsesFormTags)' -v`

Expected: 全部 PASS。

---

### Task 3: 扩展验证并做相关回归

**Files:**
- Modify: `backend/apps/iam/internal/controller/query_binding_test.go`

- [ ] **Step 1: 增加一个列表型 query 绑定测试**

在同一测试文件中新增一个列表型场景，覆盖多个字段：

```go
func TestQueryBindingMenuPageListReqUsesFormTags(t *testing.T) {
    gin.SetMode(gin.TestMode)
    engine := gin.New()
    var got dtopermission.MenuPageListReq
    engine.GET("/menu/pageList", func(ctx *gin.Context) {
        if err := ctx.ShouldBindQuery(&got); err != nil {
            ctx.String(http.StatusBadRequest, err.Error())
            return
        }
        ctx.Status(http.StatusNoContent)
    })

    req := httptest.NewRequest(http.MethodGet, "/menu/pageList?tenantID=9&parentID=2&name=ops&code=OPS&type=menu&status=enable&page=3&pageSize=20", nil)
    resp := httptest.NewRecorder()
    engine.ServeHTTP(resp, req)

    if resp.Code != http.StatusNoContent {
        t.Fatalf("expected status %d, got %d body=%s", http.StatusNoContent, resp.Code, resp.Body.String())
    }
    if got.TenantID != 9 || got.ParentID != 2 || got.Name != "ops" || got.Code != "OPS" || got.Type != "menu" || got.Status != "enable" || got.Page != 3 || got.PageSize != 20 {
        t.Fatalf("unexpected bound req: %#v", got)
    }
}
```

- [ ] **Step 2: 跑 controller 绑定测试集合**

Run: `go test ./apps/iam/internal/controller -run 'TestQueryBinding' -v`

Expected: 全部 PASS。

- [ ] **Step 3: 跑相关模块测试**

Run: `go test ./apps/iam/internal/router -v && go test ./apps/iam/internal/controller/... && go test ./apps/iam/internal/service/...`

Expected: 相关模块通过；若某些已有测试存在无关日志，不影响退出码即可，但不得有失败。

- [ ] **Step 4: 跑 IAM 全量测试**

Run: `go test ./apps/iam/...`

Expected: PASS。

- [ ] **Step 5: Commit**

```bash
git add docs/superpowers/specs/2026-05-09-query-binding-form-tags-design.md docs/superpowers/plans/2026-05-09-query-binding-form-tags-plan.md backend/apps/iam/internal/dto/dtouser/request.go backend/apps/iam/internal/dto/dtotenant/request.go backend/apps/iam/internal/dto/dtotenant/system_request.go backend/apps/iam/internal/dto/dtoauth/request.go backend/apps/iam/internal/dto/dtopermission/request.go backend/apps/iam/internal/controller/query_binding_test.go
git commit -m "fix: stabilize query binding for IAM DTOs"
```

只有在用户明确要求提交时才执行本步。
