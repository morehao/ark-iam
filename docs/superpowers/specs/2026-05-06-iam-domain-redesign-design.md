# IAM 应用功能领域划分设计

## 背景

当前 `backend/apps/iam` 的模块划分过于分散，一个表就创建一个包，导致：
- Controller 层有 22 个模块
- Service 层有 22 个模块
- 大量跨领域逻辑分散在多个包中，难以维护

## 目标

按业务领域划分模块，整合相关实体，减少跨领域耦合。

## 领域划分方案

| 领域 | 说明 | 包含内容 |
|------|------|----------|
| **user** | 用户领域 | UserEntity, UserIdentityEntity, UserDepartmentRelationEntity, UserLoginLogEntity, RefreshTokenEntity, UserRoleEntity |
| **permission** | 权限领域 | RoleEntity, RoleMenuEntity, RoleScopeEntity, ApplicationRoleEntity, MenuEntity |
| **organization** | 组织领域 | OrganizationEntity, OrganizationRoleEntity, OrganizationUserRelationEntity, OrganizationRoleUserRelationEntity |
| **tenant** | 租户领域 | TenantEntity, DepartmentEntity |
| **application** | 应用领域 | ApplicationEntity, ApplicationSecretEntity |
| **resource** | 资源领域 | ResourceEntity, ScopeEntity |
| **auth** | 认证领域 | ConnectorEntity, SsoConnectorEntity |
| **system** | 系统领域 | SystemEntity, LogEntity |

共 **8个核心领域**，从现有的 22 个模块整合。

## 各领域详细说明

### 1. user 用户领域

负责用户全部相关功能：
- 用户基本信息管理
- 用户第三方身份（UserIdentityEntity）
- 用户部门关系（UserDepartmentRelationEntity）
- 用户登录日志（UserLoginLogEntity）
- 刷新令牌管理（RefreshTokenEntity）
- 用户角色分配（UserRoleEntity）

### 2. permission 权限领域

负责权限控制全部相关功能：
- 角色管理（RoleEntity）
- 角色菜单关联（RoleMenuEntity）
- 角色权限范围（RoleScopeEntity）
- 应用角色关联（ApplicationRoleEntity）
- 菜单管理（MenuEntity）

### 3. organization 组织领域

负责组织架构相关功能：
- 组织管理（OrganizationEntity）
- 组织角色（OrganizationRoleEntity）
- 组织用户关系（OrganizationUserRelationEntity）
- 组织角色用户关系（OrganizationRoleUserRelationEntity）

### 4. tenant 租户领域

负责多租户隔离和基础租户配置：
- 租户管理（TenantEntity）
- 部门管理（DepartmentEntity）

### 5. application 应用领域

负责应用管理：
- 应用基本信息（ApplicationEntity）
- 应用密钥（ApplicationSecretEntity）

### 6. resource 资源领域

负责资源管理和权限范围：
- 资源管理（ResourceEntity）
- 权限范围（ScopeEntity）

### 7. auth 认证领域

负责认证连接器：
- 连接器（ConnectorEntity）
- SSO 连接器（SsoConnectorEntity）

### 8. system 系统领域

负责系统配置和日志：
- 系统配置（SystemEntity）
- 日志管理（LogEntity）

## 目录结构

重组后各层目录结构：

```
apps/iam/
├── model/                          # 数据模型层（保持不变）
│   ├── user.go                     # 用户领域所有实体
│   ├── permission.go               # 权限领域所有实体
│   ├── organization.go             # 组织领域所有实体
│   ├── tenant.go                   # 租户领域所有实体（含 DepartmentEntity）
│   ├── application.go              # 应用领域所有实体
│   ├── resource.go                 # 资源领域所有实体
│   ├── auth.go                     # 认证领域所有实体
│   └── system.go                   # 系统领域所有实体
│
├── dao/                            # 数据访问层（按领域划分）
│   ├── user.go                     # 用户领域数据访问
│   ├── permission.go               # 权限领域数据访问
│   ├── organization.go             # 组织领域数据访问
│   ├── tenant.go                   # 租户领域数据访问
│   ├── application.go              # 应用领域数据访问
│   ├── resource.go                 # 资源领域数据访问
│   ├── auth.go                     # 认证领域数据访问
│   └── system.go                   # 系统领域数据访问
│
├── object/                         # 基础对象层（按领域划分）
│   ├── objuser/
│   ├── objpermission/
│   ├── objorganization/
│   ├── objtenant/
│   ├── objapplication/
│   ├── objresource/
│   ├── objauth/
│   └── objsystem/
│
├── internal/
│   ├── dto/                        # DTO 层（按领域划分）
│   │   ├── dtouser/
│   │   ├── dtopermission/
│   │   ├── dtoorganization/
│   │   ├── dtotenant/
│   │   ├── dtoapplication/
│   │   ├── dtoresource/
│   │   ├── dtoauth/
│   │   └── dtosystem/
│   │
│   ├── service/                    # 服务层（按领域划分）
│   │   ├── svcuser/
│   │   ├── svcpermission/
│   │   ├── svcorganization/
│   │   ├── svctenant/
│   │   ├── svcapplication/
│   │   ├── svcresource/
│   │   ├── svcauth/
│   │   └── svcsystem/
│   │
│   ├── controller/                 # 控制器层（按领域划分）
│   │   ├── ctruser/
│   │   ├── ctrpermission/
│   │   ├── ctrorganization/
│   │   ├── ctrtenant/
│   │   ├── ctrapplication/
│   │   ├── ctrresource/
│   │   ├── ctrauth/
│   │   └── ctrsystem/
│   │
│   ├── router/                     # 路由层
│   │   └── router.go
│   │
│   └── constant/                   # 常量层
│
└── docs/                           # 文档
```

## API 路由重组

重组后路由按领域划分：

| 领域 | 路由前缀 |
|------|----------|
| user | /v1/iam/user/* |
| permission | /v1/iam/permission/* |
| organization | /v1/iam/organization/* |
| tenant | /v1/iam/tenant/* |
| application | /v1/iam/application/* |
| resource | /v1/iam/resource/* |
| auth | /v1/iam/auth/* |
| system | /v1/iam/system/* |

## 实施计划

1. 创建新的目录结构
2. 按领域迁移 model 和 dao 层
3. 按领域迁移 object 层
4. 按领域迁移 dto 层
5. 按领域迁移 service 层
6. 按领域迁移 controller 层
7. 更新 router 层路由注册
8. 删除旧的分散目录

## 影响范围

- Model 层：27 个文件 → 8 个文件
- Dao 层：27 个文件 → 8 个文件
- Object 层：15 个目录 → 8 个目录
- DTO 层：13 个目录 → 8 个目录
- Service 层：22 个目录 → 8 个目录
- Controller 层：22 个目录 → 8 个目录
