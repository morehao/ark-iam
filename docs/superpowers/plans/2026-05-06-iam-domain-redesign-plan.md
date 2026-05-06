# IAM 应用功能领域划分实施计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 将 IAM 应用的 model、dao、object、dto、service、controller 层从分散的按表划分重组为按业务领域划分

**Architecture:** 按 8 个业务领域（user、permission、organization、tenant、application、resource、auth、system）重组各层代码，每个领域将相关实体、数据访问、DTO、服务、控制器合并到统一目录

**Tech Stack:** Go, Gin, GORM

---

## 文件结构映射

### Model 层重组

**现有文件 → 新文件：**

| 新文件 | 合并的实体 |
|--------|-----------|
| model/user.go | user.go, user_identity.go, user_department_relation.go, user_login_log.go, refresh_token.go, user_role.go |
| model/permission.go | role.go, role_menu.go, role_scope.go, application_role.go, menu.go |
| model/organization.go | organization.go, organization_role.go, organization_user_relation.go, organization_role_user_relation.go |
| model/tenant.go | tenant.go, department.go |
| model/application.go | application.go, application_secret.go |
| model/resource.go | resource.go, scope.go |
| model/auth.go | connector.go, sso_connector.go |
| model/system.go | system.go, log.go |

### Dao 层重组

**现有文件 → 新文件：** 与 model 层一一对应，8 个文件。

### Object 层重组

**现有目录 → 新目录：**

| 新目录 | 合并的目录 |
|--------|-----------|
| object/objuser/ | objuser/ |
| object/objpermission/ | objrole/, objmenu/ |
| object/objorganization/ | objorganization/ |
| object/objtenant/ | objtenant/, objdepartment/ |
| object/objapplication/ | objapplication/ |
| object/objresource/ | objresource/, objscope/ |
| object/objauth/ | objauth/, objsso/, objconnector/ |
| object/objsystem/ | objsystem/, objlog/ |

### DTO 层重组

**现有目录 → 新目录：**

| 新目录 | 合并的目录 |
|--------|-----------|
| internal/dto/dtouser/ | dtouser/ |
| internal/dto/dtopermission/ | dtorole/, dtomenu/ |
| internal/dto/dtoorganization/ | dtoorganization/ |
| internal/dto/dtotenant/ | dtotenant/, dtodepartment/ |
| internal/dto/dtoapplication/ | dtoapplication/ |
| internal/dto/dtoresource/ | dtoresource/ |
| internal/dto/dtoauth/ | dtoauth/, dtosso/ |
| internal/dto/dtosystem/ | dtosystem/, dtolog/ |

### Service 层重组

**现有目录 → 新目录：**

| 新目录 | 合并的目录 |
|--------|-----------|
| internal/service/svcuser/ | svcuser/, svcuserrole/, svcuseridentity/, svcuserdepartmentrelation/, svcuserloginlog/, svcrefreshtoken/ |
| internal/service/svcpermission/ | svcrole/, svcrolemenu/, svcrolescope/, svcapplicationrole/, svcmenu/ |
| internal/service/svcorganization/ | svcorganization/, svcorganizationrole/, svcorganizationuserrelation/, svcorganizationroleuserrelation/ |
| internal/service/svctenant/ | svctenant/, svcdepartment/ |
| internal/service/svcapplication/ | svcapplication/, svcapplicationsecret/ |
| internal/service/svcresource/ | svcresource/, svcscope/ |
| internal/service/svcauth/ | svcauth/, svcsso/, svcconnector/ |
| internal/service/svcsystem/ | svcsystem/, svclog/ |

### Controller 层重组

**现有目录 → 新目录：** 与 Service 层一一对应，8 个目录。

### Router 层重组

现有 23 个路由文件合并为 8 个领域路由文件，通过 router.go 统一注册。

---

## 任务列表

### 阶段一：Model + Dao 层重组

- [ ] **Task 1: 创建 user 领域 model 和 dao 文件**

Files:
- Create: `backend/apps/iam/model/user.go`
- Create: `backend/apps/iam/dao/user.go`
- Delete: `backend/apps/iam/model/user.go` (原 single entity file)
- Delete: `backend/apps/iam/model/user_identity.go`
- Delete: `backend/apps/iam/model/user_department_relation.go`
- Delete: `backend/apps/iam/model/user_login_log.go`
- Delete: `backend/apps/iam/model/refresh_token.go`
- Delete: `backend/apps/iam/model/user_role.go`
- Delete: `backend/apps/iam/dao/user.go` (原 single entity file)
- Delete: `backend/apps/iam/dao/user_identity.go`
- Delete: `backend/apps/iam/dao/user_department_relation.go`
- Delete: `backend/apps/iam/dao/user_login_log.go`
- Delete: `backend/apps/iam/dao/refresh_token.go`
- Delete: `backend/apps/iam/dao/user_role.go`

- [ ] **Task 2: 创建 permission 领域 model 和 dao 文件**

Files:
- Create: `backend/apps/iam/model/permission.go`
- Create: `backend/apps/iam/dao/permission.go`
- Delete: `backend/apps/iam/model/role.go`, `role_menu.go`, `role_scope.go`, `application_role.go`, `menu.go`
- Delete: `backend/apps/iam/dao/role.go`, `role_menu.go`, `role_scope.go`, `application_role.go`, `menu.go`

- [ ] **Task 3: 创建 organization 领域 model 和 dao 文件**

Files:
- Create: `backend/apps/iam/model/organization.go`
- Create: `backend/apps/iam/dao/organization.go`
- Delete: `backend/apps/iam/model/organization.go`, `organization_role.go`, `organization_user_relation.go`, `organization_role_user_relation.go`
- Delete: `backend/apps/iam/dao/organization.go`, `organization_role.go`, `organization_user_relation.go`, `organization_role_user_relation.go`

- [ ] **Task 4: 创建 tenant 领域 model 和 dao 文件**

Files:
- Create: `backend/apps/iam/model/tenant.go`
- Create: `backend/apps/iam/dao/tenant.go`
- Delete: `backend/apps/iam/model/tenant.go`, `department.go`
- Delete: `backend/apps/iam/dao/tenant.go`, `department.go`

- [ ] **Task 5: 创建 application 领域 model 和 dao 文件**

Files:
- Create: `backend/apps/iam/model/application.go`
- Create: `backend/apps/iam/dao/application.go`
- Delete: `backend/apps/iam/model/application.go`, `application_secret.go`, `application_role.go`
- Delete: `backend/apps/iam/dao/application.go`, `application_secret.go`, `application_role.go`

- [ ] **Task 6: 创建 resource 领域 model 和 dao 文件**

Files:
- Create: `backend/apps/iam/model/resource.go`
- Create: `backend/apps/iam/dao/resource.go`
- Delete: `backend/apps/iam/model/resource.go`, `scope.go`
- Delete: `backend/apps/iam/dao/resource.go`, `scope.go`

- [ ] **Task 7: 创建 auth 领域 model 和 dao 文件**

Files:
- Create: `backend/apps/iam/model/auth.go`
- Create: `backend/apps/iam/dao/auth.go`
- Delete: `backend/apps/iam/model/connector.go`, `sso_connector.go`
- Delete: `backend/apps/iam/dao/connector.go`, `sso_connector.go`

- [ ] **Task 8: 创建 system 领域 model 和 dao 文件**

Files:
- Create: `backend/apps/iam/model/system.go`
- Create: `backend/apps/iam/dao/system.go`
- Delete: `backend/apps/iam/model/system.go`, `log.go`
- Delete: `backend/apps/iam/dao/system.go`, `log.go`

### 阶段二：Object 层重组

- [ ] **Task 9: 创建 user 领域 object 目录**

Files:
- Create: `backend/apps/iam/object/objuser/` (保留原 objuser 内容)
- Modify: `backend/apps/iam/object/` 目录结构
- Delete: `backend/apps/iam/object/objuser/` 外层目录重组

- [ ] **Task 10: 创建 permission 领域 object 目录**

Files:
- Create: `backend/apps/iam/object/objpermission/`
- Merge: `backend/apps/iam/object/objrole/`, `backend/apps/iam/object/objmenu/`
- Delete: `backend/apps/iam/object/objrole/`, `backend/apps/iam/object/objmenu/`

- [ ] **Task 11: 创建 organization 领域 object 目录**

Files:
- Create: `backend/apps/iam/object/objorganization/`
- Delete: `backend/apps/iam/object/objorganization/` (已存在，内容合并)

- [ ] **Task 12: 创建 tenant 领域 object 目录**

Files:
- Create: `backend/apps/iam/object/objtenant/`
- Merge: `backend/apps/iam/object/objtenant/`, `backend/apps/iam/object/objdepartment/`
- Delete: `backend/apps/iam/object/objdepartment/`

- [ ] **Task 13: 创建 application 领域 object 目录**

Files:
- Create: `backend/apps/iam/object/objapplication/`
- Delete: `backend/apps/iam/object/objapplication/` (已存在)

- [ ] **Task 14: 创建 resource 领域 object 目录**

Files:
- Create: `backend/apps/iam/object/objresource/`
- Merge: `backend/apps/iam/object/objresource/`, `backend/apps/iam/object/objscope/`
- Delete: `backend/apps/iam/object/objscope/`

- [ ] **Task 15: 创建 auth 领域 object 目录**

Files:
- Create: `backend/apps/iam/object/objauth/`
- Merge: `backend/apps/iam/object/objauth/`, `backend/apps/iam/object/objconnector/`, `backend/apps/iam/object/objsso/`
- Delete: `backend/apps/iam/object/objconnector/`, `backend/apps/iam/object/objsso/`

- [ ] **Task 16: 创建 system 领域 object 目录**

Files:
- Create: `backend/apps/iam/object/objsystem/`
- Merge: `backend/apps/iam/object/objsystem/`, `backend/apps/iam/object/objlog/`
- Delete: `backend/apps/iam/object/objlog/`

### 阶段三：DTO 层重组

- [ ] **Task 17: 创建 user 领域 dto 目录**
- [ ] **Task 18: 创建 permission 领域 dto 目录**
- [ ] **Task 19: 创建 organization 领域 dto 目录**
- [ ] **Task 20: 创建 tenant 领域 dto 目录**
- [ ] **Task 21: 创建 application 领域 dto 目录**
- [ ] **Task 22: 创建 resource 领域 dto 目录**
- [ ] **Task 23: 创建 auth 领域 dto 目录**
- [ ] **Task 24: 创建 system 领域 dto 目录**

### 阶段四：Service 层重组

- [ ] **Task 25: 创建 user 领域 service 目录**
- [ ] **Task 26: 创建 permission 领域 service 目录**
- [ ] **Task 27: 创建 organization 领域 service 目录**
- [ ] **Task 28: 创建 tenant 领域 service 目录**
- [ ] **Task 29: 创建 application 领域 service 目录**
- [ ] **Task 30: 创建 resource 领域 service 目录**
- [ ] **Task 31: 创建 auth 领域 service 目录**
- [ ] **Task 32: 创建 system 领域 service 目录**

### 阶段五：Controller 层重组

- [ ] **Task 33: 创建 user 领域 controller 目录**
- [ ] **Task 34: 创建 permission 领域 controller 目录**
- [ ] **Task 35: 创建 organization 领域 controller 目录**
- [ ] **Task 36: 创建 tenant 领域 controller 目录**
- [ ] **Task 37: 创建 application 领域 controller 目录**
- [ ] **Task 38: 创建 resource 领域 controller 目录**
- [ ] **Task 39: 创建 auth 领域 controller 目录**
- [ ] **Task 40: 创建 system 领域 controller 目录**

### 阶段六：Router 层重组

- [ ] **Task 41: 重组 router 层，按领域注册路由**

### 阶段七：清理和验证

- [ ] **Task 42: 删除旧的空目录**
- [ ] **Task 43: 运行 make lint 和 make test 验证**

---

## 实施顺序建议

1. 先完成 Model + Dao 层重组（Task 1-8），因为其他层依赖 model 定义
2. 再完成 Object 层重组（Task 9-16）
3. DTO、Service、Controller 层可以并行重组（Task 17-40）
4. 最后处理 Router 层和清理（Task 41-43）

**注意：** 每个 Task 完成后建议运行 `go build ./...` 验证编译通过。
