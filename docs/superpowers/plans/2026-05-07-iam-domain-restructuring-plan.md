# IAM 领域重构实施计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 将 IAM 应用从现有的多领域结构重组为 5 个核心领域：tenant、user、permission、auth、audit

**Architecture:** 按领域合并 object/router/code 层，model/dao/service/controller/dto 层保持现有文件结构不变

**Tech Stack:** Go, GORM

---

## 文件变更概览

### 需要创建的目录/文件

| 操作 | 路径 |
|------|------|
| 创建 | `backend/apps/iam/object/objaudit/` |
| 创建 | `backend/apps/iam/internal/router/tenant.go` |
| 创建 | `backend/apps/iam/internal/router/user.go` |
| 创建 | `backend/apps/iam/internal/router/permission.go` |
| 创建 | `backend/apps/iam/internal/router/auth.go` |
| 创建 | `backend/apps/iam/internal/router/audit.go` |
| 创建 | `backend/pkg/code/tenant.go` |
| 创建 | `backend/pkg/code/user.go` |
| 创建 | `backend/pkg/code/permission.go` |
| 创建 | `backend/pkg/code/auth.go` |
| 创建 | `backend/pkg/code/audit.go` |

### 需要删除的目录/文件

| 操作 | 路径 |
|------|------|
| 删除 | `backend/apps/iam/object/objorganization/` |
| 删除 | `backend/apps/iam/internal/router/department.go` |
| 删除 | `backend/apps/iam/internal/router/system.go` |
| 删除 | `backend/apps/iam/internal/router/log.go` |
| 删除 | `backend/apps/iam/internal/router/connector.go` |
| 删除 | `backend/apps/iam/internal/router/sso_connector.go` |
| 删除 | `backend/apps/iam/internal/router/user_role.go` |
| 删除 | `backend/apps/iam/internal/router/role.go` |
| 删除 | `backend/apps/iam/internal/router/role_menu.go` |
| 删除 | `backend/apps/iam/internal/router/role_scope.go` |
| 删除 | `backend/apps/iam/internal/router/menu.go` |
| 删除 | `backend/apps/iam/internal/router/scope.go` |
| 删除 | `backend/apps/iam/internal/router/resource.go` |
| 删除 | `backend/apps/iam/internal/router/application.go` |
| 删除 | `backend/apps/iam/internal/router/organization.go` |
| 删除 | `backend/apps/iam/internal/router/organization_role.go` |
| 删除 | `backend/apps/iam/internal/router/organization_user_relation.go` |
| 删除 | `backend/apps/iam/internal/router/organization_role_user_relation.go` |
| 删除 | `backend/pkg/code/department.go` |
| 删除 | `backend/pkg/code/system.go` |
| 删除 | `backend/pkg/code/log.go` |
| 删除 | `backend/pkg/code/connector.go` |
| 删除 | `backend/pkg/code/sso_connector.go` |
| 删除 | `backend/pkg/code/menu.go` |
| 删除 | `backend/pkg/code/role.go` |
| 删除 | `backend/pkg/code/auth.go` |
| 删除 | `backend/pkg/code/application.go` |
| 删除 | `backend/pkg/code/resource.go` |
| 删除 | `backend/pkg/code/scope.go` |
| 删除 | `backend/pkg/code/organization.go` |
| 删除 | `backend/pkg/code/organization_role.go` |
| 删除 | `backend/pkg/code/organization_user_relation.go` |
| 删除 | `backend/pkg/code/organization_role_user_relation.go` |

### 需要修改的文件

| 文件 | 变更 |
|------|------|
| `backend/apps/iam/internal/router/router.go` | 重写为按领域注册 |
| `backend/pkg/code/code.go` | 重写为按领域注册错误码 |

---

## Task 1: Object 层重组

### 1.1 创建 objaudit 目录并迁移 log.go

- [ ] **Step 1: 创建 objaudit 目录**

Run: `mkdir -p backend/apps/iam/object/objaudit`

- [ ] **Step 2: 迁移 log.go 到 objaudit**

从 `backend/apps/iam/object/objsystem/log.go` 迁移到 `backend/apps/iam/object/objaudit/log.go`

- [ ] **Step 3: 提交**

```bash
git add backend/apps/iam/object/objaudit/
git commit -m "feat(iam): create objaudit directory and migrate log.go"
```

### 1.2 创建 objtenant/system.go（从 objsystem 迁移）

- [ ] **Step 1: 创建 objtenant/system.go**

从 `backend/apps/iam/object/objsystem/system.go` 迁移到 `backend/apps/iam/object/objtenant/system.go`

- [ ] **Step 2: 提交**

```bash
git add backend/apps/iam/object/objtenant/system.go
git commit -m "feat(iam): migrate system.go to objtenant"
```

### 1.3 将 objorganization 合并到 objtenant

- [ ] **Step 1: 迁移 organization.go 到 objtenant**

从 `backend/apps/iam/object/objorganization/organization.go` 迁移到 `backend/apps/iam/object/objtenant/organization.go`

- [ ] **Step 2: 删除 objorganization 目录**

Run: `rm -rf backend/apps/iam/object/objorganization`

- [ ] **Step 3: 提交**

```bash
git add backend/apps/iam/object/objtenant/organization.go
git rm -r backend/apps/iam/object/objorganization
git commit -m "feat(iam): merge objorganization into objtenant"
```

### 1.4 删除 objsystem 目录

- [ ] **Step 1: 删除 objsystem 目录**

Run: `rm -rf backend/apps/iam/object/objsystem`

- [ ] **Step 2: 提交**

```bash
git rm -r backend/apps/iam/object/objsystem
git commit -m "feat(iam): remove objsystem directory"
```

---

## Task 2: Router 层重组

### 2.1 创建 router/tenant.go

- [ ] **Step 1: 创建 tenant.go**

合并以下路由到一个文件：
- 原 `router/tenant.go` (tenantRouter)
- 原 `router/department.go` (departmentRouter)
- 原 `router/organization.go` (organizationRouter)
- 原 `router/system.go` (systemRouter)

路由注册时使用：`tenantGroup := groups.AuthGroup.Group("/iam/tenant")`

- [ ] **Step 2: 提交**

```bash
git add backend/apps/iam/internal/router/tenant.go
git commit -m "feat(iam): create router/tenant.go with tenant/department/organization/system"
```

### 2.2 创建 router/user.go

- [ ] **Step 1: 创建 user.go**

合并以下路由到一个文件：
- 原 `router/user.go` (userRouter)
- 原 `router/user_role.go` (userRoleRouter)

路由注册时使用：`userGroup := groups.AuthGroup.Group("/iam/user")`

- [ ] **Step 2: 提交**

```bash
git add backend/apps/iam/internal/router/user.go
git commit -m "feat(iam): create router/user.go with user/userRole"
```

### 2.3 创建 router/permission.go

- [ ] **Step 1: 创建 permission.go**

合并以下路由到一个文件：
- 原 `router/role.go` (roleRouter)
- 原 `router/menu.go` (menuRouter)
- 原 `router/scope.go` (scopeRouter)
- 原 `router/application.go` (applicationRouter)
- 原 `router/resource.go` (resourceRouter)
- 原 `router/role_menu.go` (roleMenuRouter)
- 原 `router/role_scope.go` (roleScopeRouter)

路由注册时使用：`permissionGroup := groups.AuthGroup.Group("/iam/permission")`

- [ ] **Step 2: 提交**

```bash
git add backend/apps/iam/internal/router/permission.go
git commit -m "feat(iam): create router/permission.go with role/menu/scope/application/resource"
```

### 2.4 创建 router/auth.go

- [ ] **Step 1: 创建 auth.go**

合并以下路由到一个文件：
- 原 `router/auth.go` (authRouter)
- 原 `router/connector.go` (connectorRouter)
- 原 `router/sso_connector.go` (ssoConnectorRouter)

路由注册时使用：`authGroup := groups.AuthGroup.Group("/iam/auth")`

- [ ] **Step 2: 提交**

```bash
git add backend/apps/iam/internal/router/auth.go
git commit -m "feat(iam): create router/auth.go with auth/connector/ssoConnector"
```

### 2.5 创建 router/audit.go

- [ ] **Step 1: 创建 audit.go**

迁移 `router/log.go` (logRouter) 到 `router/audit.go`

路由注册时使用：`auditGroup := groups.AuthGroup.Group("/iam/audit")`

- [ ] **Step 2: 提交**

```bash
git add backend/apps/iam/internal/router/audit.go
git commit -m "feat(iam): create router/audit.go with log"
```

### 2.6 更新 router/router.go

- [ ] **Step 1: 重写 router.go**

修改为按领域调用：

```go
package router

import "github.com/morehao/golib/biz/gserver/ginserver"

func RegisterRouter(groups *ginserver.RouterGroups, appName string) {
	tenantRouter(groups)
	userRouter(groups)
	permissionRouter(groups)
	authRouter(groups)
	auditRouter(groups)
}
```

- [ ] **Step 2: 提交**

```bash
git add backend/apps/iam/internal/router/router.go
git commit -m "feat(iam): update router.go to register by domain"
```

### 2.7 删除旧的 router 文件

- [ ] **Step 1: 删除已合并的 router 文件**

Run:
```bash
rm backend/apps/iam/internal/router/department.go
rm backend/apps/iam/internal/router/system.go
rm backend/apps/iam/internal/router/log.go
rm backend/apps/iam/internal/router/connector.go
rm backend/apps/iam/internal/router/sso_connector.go
rm backend/apps/iam/internal/router/user_role.go
rm backend/apps/iam/internal/router/role.go
rm backend/apps/iam/internal/router/role_menu.go
rm backend/apps/iam/internal/router/role_scope.go
rm backend/apps/iam/internal/router/menu.go
rm backend/apps/iam/internal/router/scope.go
rm backend/apps/iam/internal/router/resource.go
rm backend/apps/iam/internal/router/application.go
rm backend/apps/iam/internal/router/organization.go
rm backend/apps/iam/internal/router/organization_role.go
rm backend/apps/iam/internal/router/organization_user_relation.go
rm backend/apps/iam/internal/router/organization_role_user_relation.go
```

- [ ] **Step 2: 提交**

```bash
git add -A
git commit -m "feat(iam): remove old router files after domain merge"
```

---

## Task 3: Code 层重组

### 3.1 创建 pkg/code/tenant.go

- [ ] **Step 1: 创建 tenant.go**

合并以下错误码到一个文件：
- 原 `code/tenant.go` (tenantErrorMsgMap)
- 原 `code/department.go` (departmentErrorMsgMap)
- 原 `code/organization.go` (organizationErrorMsgMap)
- 原 `code/system.go` (systemErrorMsgMap)

- [ ] **Step 2: 提交**

```bash
git add backend/pkg/code/tenant.go
git commit -m "feat(iam): create pkg/code/tenant.go with tenant/department/organization/system errors"
```

### 3.2 更新 pkg/code/user.go

- [ ] **Step 1: 更新 user.go**

合并以下错误码到一个文件：
- 原 `code/user.go` (userErrorMsgMap)
- 原 `code/user.go` 中需添加 userIdentityErrorMsgMap、userLoginLogErrorMsgMap、userDepartmentRelationErrorMsgMap、userRoleErrorMsgMap

从以下文件迁移：
- `userIdentityErrorMsgMap` from `code/user_identity.go`
- `userLoginLogErrorMsgMap` from `code/user_login_log.go`
- `userDepartmentRelationErrorMsgMap` from `code/user_department_relation.go`
- `userRoleErrorMsgMap` from `code/user_role.go`

- [ ] **Step 2: 提交**

```bash
git add backend/pkg/code/user.go
git commit -m "feat(iam): update pkg/code/user.go with all user-related errors"
```

### 3.3 创建 pkg/code/permission.go

- [ ] **Step 1: 创建 permission.go**

合并以下错误码到一个文件：
- 原 `code/role.go` (roleErrorMsgMap)
- 原 `code/menu.go` (menuErrorMsgMap)
- 原 `code/scope.go` (scopeErrorMsgMap)
- 原 `code/application.go` (applicationErrorMsgMap)
- 原 `code/resource.go` (resourceErrorMsgMap)
- 原 `code/role_menu.go` (roleMenuErrorMsgMap)
- 原 `code/role_scope.go` (roleScopeErrorMsgMap)

- [ ] **Step 2: 提交**

```bash
git add backend/pkg/code/permission.go
git commit -m "feat(iam): create pkg/code/permission.go with role/menu/scope/application/resource errors"
```

### 3.4 创建 pkg/code/auth.go

- [ ] **Step 1: 创建 auth.go**

合并以下错误码到一个文件：
- 原 `code/connector.go` (connectorErrorMsgMap)
- 原 `code/sso_connector.go` (ssoConnectorErrorMsgMap)
- 原 `code/auth.go` (authErrorMsgMap)

- [ ] **Step 2: 提交**

```bash
git add backend/pkg/code/auth.go
git commit -m "feat(iam): create pkg/code/auth.go with connector/ssoConnector/auth errors"
```

### 3.5 创建 pkg/code/audit.go

- [ ] **Step 1: 创建 audit.go**

迁移 `code/log.go` (logErrorMsgMap) 到 `code/audit.go`

- [ ] **Step 2: 提交**

```bash
git add backend/pkg/code/audit.go
git commit -m "feat(iam): create pkg/code/audit.go with log errors"
```

### 3.6 更新 pkg/code/code.go

- [ ] **Step 1: 重写 code.go**

修改为按领域注册错误码：

```go
package code

import (
	"fmt"

	"github.com/morehao/golib/biz/gconstant"
	"github.com/morehao/golib/biz/genericdao"
	"github.com/morehao/golib/gerror"
)

var errorMap = gerror.ErrorMap{}

func registerError(codeMsgMap gerror.CodeMsgMap) {
	for code, msg := range codeMsgMap {
		if _, ok := errorMap[code]; ok {
			panic(fmt.Sprintf("error code %d already exists", code))
		}
		errorMap[code] = gerror.Error{
			Code:	code,
			Msg:	msg,
		}
	}
}

func GetError(code int) *gerror.Error {
	err := errorMap[code]
	return &err
}

func init() {
	// 业务错误码规范: 从 1001XX 开始
	// 领域划分: tenant(1001XX-1004XX) user(1005XX-1008XX) permission(1006XX-1009XX) auth(1010XX-1011XX) audit
	registerError(genericdao.DBErrorMsgMap)
	registerError(gconstant.SystemErrorMsgMap)
	registerError(gconstant.AuthErrorMsgMap)
	registerError(tenantErrorMsgMap)
	registerError(userErrorMsgMap)
	registerError(permissionErrorMsgMap)
	registerError(authErrorMsgMap)
	registerError(auditErrorMsgMap)
}
```

- [ ] **Step 2: 提交**

```bash
git add backend/pkg/code/code.go
git commit -m "feat(iam): update pkg/code/code.go to register by domain"
```

### 3.7 删除旧的 code 文件

- [ ] **Step 1: 删除已合并的 code 文件**

Run:
```bash
rm backend/pkg/code/department.go
rm backend/pkg/code/system.go
rm backend/pkg/code/log.go
rm backend/pkg/code/connector.go
rm backend/pkg/code/sso_connector.go
rm backend/pkg/code/menu.go
rm backend/pkg/code/role.go
rm backend/pkg/code/auth.go
rm backend/pkg/code/application.go
rm backend/pkg/code/resource.go
rm backend/pkg/code/scope.go
rm backend/pkg/code/organization.go
rm backend/pkg/code/organization_role.go
rm backend/pkg/code/organization_user_relation.go
rm backend/pkg/code/organization_role_user_relation.go
rm backend/pkg/code/user_identity.go
rm backend/pkg/code/user_login_log.go
rm backend/pkg/code/user_department_relation.go
rm backend/pkg/code/user_role.go
rm backend/pkg/code/role_menu.go
rm backend/pkg/code/role_scope.go
```

- [ ] **Step 2: 提交**

```bash
git add -A
git commit -m "feat(iam): remove old code files after domain merge"
```

---

## Task 4: 验证

- [ ] **Step 1: 运行 linter**

Run: `make lint`

- [ ] **Step 2: 运行测试**

Run: `make test APP=iam`

- [ ] **Step 3: 提交所有变更**

```bash
git add -A
git commit -m "feat(iam): complete domain restructuring - tenant/user/permission/auth/audit"
```