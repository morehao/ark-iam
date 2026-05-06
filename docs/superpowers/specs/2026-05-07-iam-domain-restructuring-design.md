# IAM 领域重构设计方案

## 目标

将 IAM 应用从现有的多领域结构重组为 5 个核心领域：**tenant、user、permission、auth、audit**

## 一、领域划分

| 领域 | 包含内容 |
|------|---------|
| **tenant** | tenant、department、organization、system |
| **user** | user、user_identity、user_department_relation、user_role、user_login_log |
| **permission** | role、menu、scope、application、role_menu、role_scope、application_role、resource |
| **auth** | auth、connector、sso_connector、refresh_token |
| **audit** | log |

## 二、各层调整方案

### 1. object 层 - 按领域重组（保留 obj 前缀）

```
object/
├── objtenant/      # 原 objtenant + objorganization（组织归入 tenant）+ objsystem 的 system 部分
├── objuser/        # 原 objuser（保持不变）
├── objpermission/  # 原 objpermission + objapplication + objresource
├── objauth/        # 原 objauth（保持不变）
└── objaudit/       # 原 objsystem 的 log 部分
```

**调整说明**：
- `objorganization` 合并到 `objtenant`
- objsystem 拆分为：system 部分归入 objtenant，log 部分归入 objaudit

### 2. service 层 - 按领域重组（保留 svc 前缀）

```
service/
├── svcuser/        # 保持多个文件（user.go、user_role.go 等）
├── svctenant/      # 保持多个文件（tenant.go、department.go 等）
├── svcpermission/  # 保持多个文件（role.go、menu.go 等）
├── svcauth/        # 保持多个文件（auth.go、connector.go 等）
└── svcaudit/       # 保持多个文件（log.go 等）
```

**调整说明**：
- 同领域内的多个文件保持分离（如 svctenant 下 tenant.go、department.go 等保持独立）

### 3. dto 层 - 按领域重组（保留 dto 前缀）

```
dto/
├── dtouser/        # 保持不变
├── dtotenant/      # 保持不变
├── dtopermission/  # 保持不变
├── dtoauth/        # 保持不变
└── dtoaudit/       # 保持不变
```

### 4. controller 层 - 按领域重组（保留 ctr 前缀）

```
controller/
├── ctruser/        # 保持多个文件
├── ctrtenant/      # 保持多个文件
├── ctrpermission/  # 保持多个文件
├── ctrauth/        # 保持多个文件
└── ctraudit/       # 保持多个文件
```

### 5. model/dao 层 - 保持不变

文件名保持不变，按原领域划分。

### 6. router 层 - 按领域拆分

```
router/
├── router.go      # 注册中心
├── user.go        # user 领域所有路由
├── tenant.go      # tenant 领域所有路由
├── permission.go  # permission 领域所有路由
├── auth.go        # auth 领域所有路由
└── audit.go       # audit 领域所有路由
```

**调整说明**：
- 每个领域文件包含该领域下所有的路由注册函数
- 路由注册时按领域分组：`tenantAuthGroup := groups.AuthGroup.Group("/iam/tenant")`

### 7. code 层 - 按领域重组

```
pkg/code/
├── code.go       # 注册中心
├── tenant.go     # 1001XX-1004XX (tenant/department/organization/system)
├── user.go       # 1005XX-1008XX (user/user_identity/user_department_relation/user_role/user_login_log)
├── permission.go # 1006XX-1009XX (menu/role/application/resource)
├── auth.go      # 1010XX-1011XX (connector/sso_connector)
└── audit.go     # 错误码
```

## 三、错误码映射

| 领域 | 错误码段 | 说明 |
|------|---------|------|
| tenant | 1001XX-1004XX | organization(1001), tenant(1002), system(1003), department(1004) |
| user | 1005XX-1008XX | user(1005), user_role(1007), user_login_log(1008), user_department_relation(1041), user_identity(1051) |
| permission | 1006XX-1009XX | menu(1006), role(1007), application/resource(1009) |
| auth | 1010XX-1011XX | connector(1010), sso_connector(1011) |
| audit | - | log |

## 四、实施步骤

1. 创建 object 层新目录结构（objtenant/objaudit）
2. 迁移 object 文件到新目录
3. 创建 router 层新文件（tenant.go/user.go/permission.go/auth.go/audit.go）
4. 迁移 router 内容到新文件
5. 创建 code 层新文件（tenant.go/permission.go/auth.go/audit.go）
6. 迁移 code 内容到新文件
7. 删除旧目录和文件
8. 更新 router.go 注册中心
9. 更新 code.go 注册中心
10. 运行测试验证