# IAM 轻量级优化设计方案

## 文档信息

- **创建时间**：2026-05-07
- **版本**：v1.0
- **状态**：已批准

---

## 一、项目背景

### 1.1 目标

基于 Logto 实现思路，对 ark-iam 进行轻量级优化和功能补全，打造一个简洁实用的 IAM 系统。

### 1.2 核心功能需求

1. **多租户** - 共享数据库，tenant_id 隔离
2. **SSO 登录** - OIDC 协议
3. **RBAC** - 角色权限管理
4. **会话管理** - 用户会话查看与撤销
5. **OAuth2/OIDC 基础** - 授权码流程、令牌刷新

### 1.3 非优先级功能（延后）

- 社交登录（微信）- 优先级低
- MFA 多因素认证 - 暂不实现

---

## 二、接口处理策略

### 2.1 保留的接口（无需修改）

| 模块 | 接口数 | 说明 |
|------|--------|------|
| 认证 | 5 | 登录、注册、刷新令牌、登出、用户信息 |
| 用户管理 | 22 | 用户 CRUD、身份管理、部门关联、登录日志 |
| 权限管理 | 20+ | 角色、菜单、资源、范围及关联管理 |
| 租户/组织/部门/系统/日志 | ~50 | 全部保留现有实现 |

### 2.2 需增强的接口

| 接口 | 当前状态 | 需要完善 |
|------|----------|----------|
| `GET /v1/iam/authorizationUrl` | stub实现 | 实现真正的 OIDC 授权URL构建 |
| `GET /v1/iam/callback` | stub实现 | 实现真正的 code 换取 userInfo |

### 2.3 需新增的接口

| 模块 | 接口数 | 功能 |
|------|--------|------|
| 会话管理 | 4 | 用户会话查看、撤销 |
| 应用角色管理 | 3 | 应用角色分配/移除/列表 |
| 角色用户管理 | 3 | 角色用户分配/移除/列表 |
| 角色应用管理 | 2 | 角色应用分配/列表 |
| SSO连接器增强 | 2 | 提供商列表、IdP配置 |
| 连接器增强 | 3 | 工厂列表、测试、授权URI |
| 应用密钥管理 | 3 | 密钥的创建/删除/列表 |

---

## 三、功能详细设计

### 3.1 会话管理 (Session)

#### 3.1.1 数据模型

使用现有的 `refresh_token` 表存储会话信息：

```sql
CREATE TABLE `refresh_token`
(
    `id`              BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '自增ID',
    `tenant_id`      BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '租户id',
    `user_id`        BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '用户ID',
    `application_id` BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '应用ID',
    `token`          VARCHAR(256) NOT NULL DEFAULT '' COMMENT 'token哈希',
    `expires_at`     DATETIME DEFAULT NULL COMMENT '过期时间',
    `revoked_at`     DATETIME DEFAULT NULL COMMENT '撤销时间',
    `created_at`     DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `updated_at`     DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    `deleted_at`     DATETIME DEFAULT NULL COMMENT '删除时间',
    `created_by`     BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '创建人ID',
    `updated_by`     BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '更新人ID',
    `deleted_by`     BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '删除人ID',
    PRIMARY KEY (`id`),
    KEY              `idx_tenant_id` (`tenant_id`),
    KEY              `idx_user_id` (`user_id`),
    KEY              `idx_token` (`token`),
    KEY              `idx_tenant_user` (`tenant_id`, `user_id`),
    KEY              `idx_expires_at` (`expires_at`),
    KEY              `idx_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='刷新令牌表';
```

#### 3.1.2 新增接口

| 接口 | 方法 | 路径 | 说明 |
|------|------|------|------|
| 会话列表 | GET | `/v1/iam/user/sessions` | 获取当前用户的会话列表 |
| 会话撤销 | DELETE | `/v1/iam/user/sessions/{sessionId}` | 撤销指定会话 |
| 全部会话撤销 | DELETE | `/v1/iam/user/sessions` | 撤销用户所有会话 |
| RefreshToken撤销 | DELETE | `/v1/iam/refreshToken/{id}` | 撤销刷新令牌 |

#### 3.1.3 DTO 设计

**请求**：
```go
// SessionListReq - 会话列表请求
type SessionListReq struct {
    gobject.PageQuery
}

// SessionRevokeReq - 会话撤销请求
type SessionRevokeReq struct {
    SessionID uint64 `json:"sessionId" path:"sessionId"`
}
```

**响应**：
```go
// SessionResp - 会话响应
type SessionResp struct {
    ID           uint64     `json:"id"`
    ApplicationID uint64    `json:"applicationId"`
    TenantID     uint64     `json:"tenantId"`
    ExpiresAt    *time.Time `json:"expiresAt"`
    CreatedAt    time.Time  `json:"createdAt"`
    IsActive     bool       `json:"isActive"`
}

// SessionListResp - 会话列表响应
type SessionListResp struct {
    gobject.PageResp
    Sessions []SessionResp `json:"sessions"`
}
```

---

### 3.2 应用角色管理 (Application Role)

#### 3.2.1 数据模型

使用现有的 `application_role` 表：

```sql
CREATE TABLE `application_role`
(
    `id`             BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '自增ID',
    `tenant_id`      BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '租户id',
    `application_id` BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '应用ID',
    `role_id`        BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '角色ID',
    `created_at`     DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `updated_at`     DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    `deleted_at`     DATETIME DEFAULT NULL COMMENT '删除时间',
    `created_by`     BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '创建人ID',
    `updated_by`     BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '更新人ID',
    `deleted_by`     BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '删除人ID',
    PRIMARY KEY (`id`),
    KEY              `idx_tenant_id` (`tenant_id`),
    KEY              `idx_application_id` (`application_id`),
    KEY              `idx_role_id` (`role_id`),
    KEY              `idx_tenant_app_role` (`tenant_id`, `application_id`, `role_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='应用角色关联表';
```

#### 3.2.2 新增接口

| 接口 | 方法 | 路径 | 说明 |
|------|------|------|------|
| 应用角色列表 | GET | `/v1/iam/application/roles` | 获取应用的角色列表 |
| 分配角色 | POST | `/v1/iam/application/assignRoles` | 分配角色给应用 |
| 移除角色 | DELETE | `/v1/iam/application/roles/{roleId}` | 从应用移除角色 |

#### 3.2.3 DTO 设计

**请求**：
```go
// ApplicationRoleListReq - 应用角色列表请求
type ApplicationRoleListReq struct {
    ApplicationID uint64 `json:"applicationId" path:"applicationId" binding:"required"`
}

// AssignApplicationRolesReq - 分配角色请求
type AssignApplicationRolesReq struct {
    ApplicationID uint64   `json:"applicationId" binding:"required"`
    RoleIDs        []uint64 `json:"roleIds" binding:"required,min=1"`
}
```

**响应**：
```go
// ApplicationRoleResp - 应用角色响应
type ApplicationRoleResp struct {
    RoleID        uint64    `json:"roleId"`
    RoleName      string    `json:"roleName"`
    RoleCode      string    `json:"roleCode"`
    ApplicationID uint64    `json:"applicationId"`
    CreatedAt     time.Time `json:"createdAt"`
}
```

---

### 3.3 角色用户管理 (Role User)

#### 3.3.1 新增接口

| 接口 | 方法 | 路径 | 说明 |
|------|------|------|------|
| 角色用户列表 | GET | `/v1/iam/role/users` | 获取角色的用户列表 |
| 分配用户 | POST | `/v1/iam/role/assignUsers` | 分配用户给角色 |
| 移除用户 | DELETE | `/v1/iam/role/users/{userId}` | 从角色移除用户 |

#### 3.3.2 DTO 设计

**请求**：
```go
// RoleUserListReq - 角色用户列表请求
type RoleUserListReq struct {
    RoleID uint64 `json:"roleId" form:"roleId" binding:"required"`
}

// AssignRoleUsersReq - 分配用户请求
type AssignRoleUsersReq struct {
    RoleID  uint64   `json:"roleId" binding:"required"`
    UserIDs []uint64 `json:"userIds" binding:"required,min=1"`
}
```

**响应**：
```go
// RoleUserResp - 角色用户响应
type RoleUserResp struct {
    UserID    uint64    `json:"userId"`
    Username  string    `json:"username"`
    Name      string    `json:"name"`
    Email     string    `json:"email"`
    RoleID    uint64    `json:"roleId"`
    CreatedAt time.Time `json:"createdAt"`
}
```

---

### 3.4 角色应用管理 (Role Application)

#### 3.4.1 新增接口

| 接口 | 方法 | 路径 | 说明 |
|------|------|------|------|
| 角色应用列表 | GET | `/v1/iam/role/applications` | 获取角色关联的应用列表 |
| 分配应用 | POST | `/v1/iam/role/assignApplications` | 分配应用给角色 |

#### 3.4.2 DTO 设计

**请求**：
```go
// RoleApplicationListReq - 角色应用列表请求
type RoleApplicationListReq struct {
    RoleID uint64 `json:"roleId" form:"roleId" binding:"required"`
}

// AssignRoleApplicationsReq - 分配应用请求
type AssignRoleApplicationsReq struct {
    RoleID         uint64   `json:"roleId" binding:"required"`
    ApplicationIDs []uint64 `json:"applicationIds" binding:"required,min=1"`
}
```

---

### 3.5 SSO 连接器增强

#### 3.5.1 新增接口

| 接口 | 方法 | 路径 | 说明 |
|------|------|------|------|
| 提供商列表 | GET | `/v1/iam/ssoConnector/providers` | 获取SSO提供商列表 |
| IdP配置 | GET/PUT | `/v1/iam/ssoConnector/{id}/idp-config` | 获取/设置IdP配置 |

#### 3.5.2 DTO 设计

**响应**：
```go
// SsoProviderResp - SSO提供商响应
type SsoProviderResp struct {
    ProviderName string `json:"providerName"`
    DisplayName  string `json:"displayName"`
    Logo         string `json:"logo"`
}

// SsoIdpConfigResp - IdP配置响应
type SsoIdpConfigResp struct {
    ClientID     string `json:"clientId"`
    ClientSecret string `json:"clientSecret"`
    Issuer       string `json:"issuer"`
    AuthURL      string `json:"authUrl"`
    TokenURL     string `json:"tokenUrl"`
    UserInfoURL  string `json:"userInfoUrl"`
    Scopes       []string `json:"scopes"`
}
```

---

### 3.6 连接器增强

#### 3.6.1 新增接口

| 接口 | 方法 | 路径 | 说明 |
|------|------|------|------|
| 工厂列表 | GET | `/v1/iam/connector/factories` | 获取可用连接器类型 |
| 测试连接 | POST | `/v1/iam/connector/{id}/test` | 测试连接器配置 |
| 授权URI | POST | `/v1/iam/connector/{id}/authorization-uri` | 获取社交登录授权URI |

---

### 3.7 应用密钥管理 (Application Secret)

#### 3.7.1 数据模型

使用现有的 `application_secret` 表：

```sql
CREATE TABLE `application_secret`
(
    `id`              BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '自增ID',
    `tenant_id`       BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '租户id',
    `application_id`  BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '应用ID',
    `name`            VARCHAR(256) NOT NULL DEFAULT '' COMMENT '密钥名称',
    `value`           VARCHAR(64) NOT NULL DEFAULT '' COMMENT '密钥值',
    `created_at`      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `updated_at`      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    `expires_at`      DATETIME DEFAULT NULL COMMENT '过期时间',
    `deleted_at`      DATETIME DEFAULT NULL COMMENT '删除时间',
    `created_by`      BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '创建人ID',
    `updated_by`      BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '更新人ID',
    `deleted_by`      BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '删除人ID',
    PRIMARY KEY (`id`),
    KEY               `idx_tenant_id` (`tenant_id`),
    KEY               `idx_application_id` (`application_id`),
    KEY               `idx_tenant_app_name` (`tenant_id`, `application_id`, `name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='应用密钥表';
```

#### 3.7.2 新增接口

| 接口 | 方法 | 路径 | 说明 |
|------|------|------|------|
| 密钥列表 | GET | `/v1/iam/application/secrets` | 获取应用密钥列表 |
| 创建密钥 | POST | `/v1/iam/application/secrets` | 创建新密钥 |
| 删除密钥 | DELETE | `/v1/iam/application/secrets/{secretId}` | 删除密钥 |

---

### 3.8 SSO 逻辑完善

#### 3.8.1 当前问题

`svcauth/auth.go` 中的 `buildAuthorizationUrl` 和 `exchangeCodeForUserInfo` 是 stub 实现，返回假数据。

#### 3.8.2 需要完善的功能

1. **buildAuthorizationUrl** - 构建真正的 OIDC 授权URL
   - 从 sso_connector 表获取 IdP 配置
   - 生成 state 参数
   - 构建符合 OIDC 规范的授权请求 URL

2. **exchangeCodeForUserInfo** - 交换 code 获取用户信息
   - 使用 code 向 IdP 的 token endpoint 请求 access_token
   - 使用 access_token 获取用户信息
   - 创建或更新用户身份

---

## 四、文件变更清单

### 4.1 新增文件

| 文件路径 | 说明 |
|----------|------|
| `internal/controller/ctrsession/` | 会话管理控制器 |
| `internal/service/svcsession/` | 会话管理服务 |
| `internal/dto/dtouser/session.go` | 会话相关 DTO |
| `dao/session.go` | 会话 DAO |

### 4.2 修改文件

| 文件路径 | 修改内容 |
|----------|----------|
| `internal/controller/ctrpermission/application.go` | 添加应用角色管理接口 |
| `internal/controller/ctrpermission/role.go` | 添加角色用户/应用管理接口 |
| `internal/controller/ctrauth/sso_connector.go` | 添加提供商列表、IdP配置接口 |
| `internal/controller/ctrauth/connector.go` | 添加工厂列表、测试、授权URI接口 |
| `internal/service/svcauth/auth.go` | 完善 SSO 逻辑 |
| `internal/router/permission.go` | 注册新路由 |
| `internal/router/auth.go` | 注册新路由 |

---

## 五、实施顺序

### Phase 1 - 核心功能 (P0)

1. **会话管理** - 最高优先级
   - DAO 层：session.go
   - Service 层：session.go
   - Controller 层：session.go
   - 路由注册

2. **SSO 逻辑完善**
   - 完善 buildAuthorizationUrl
   - 完善 exchangeCodeForUserInfo

### Phase 2 - 重要功能 (P1)

3. **应用角色管理**
   - 新增 assignRoles、removeRole、listRoles 接口

4. **角色用户管理**
   - 新增 assignUsers、removeUser、listUsers 接口

5. **角色应用管理**
   - 新增 assignApplications、listApplications 接口

6. **SSO连接器增强**
   - 新增 providers、idp-config 接口

7. **连接器增强**
   - 新增 factories、test、authorization-uri 接口

8. **应用密钥管理**
   - 新增 secrets CRUD 接口

---

## 六、数据库变更

本次优化**无需数据库变更**，所有功能基于现有表结构实现：

- `refresh_token` - 会话管理
- `application_role` - 应用角色关联
- `user_role` - 角色用户关联
- `role` - 角色信息
- `application` - 应用信息
- `sso_connector` - SSO连接器
- `connector` - 连接器
- `application_secret` - 应用密钥

---

## 七、测试策略

1. **单元测试**
   - Service 层每个方法独立测试
   - DAO 层使用 mock 测试

2. **集成测试**
   - API 端到端测试
   - SSO 流程测试（需模拟 IdP）

3. **测试覆盖**
   - 核心业务逻辑覆盖率 > 80%

---

*文档结束*