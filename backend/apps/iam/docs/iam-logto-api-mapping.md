# IAM 与 Logto API 映射对照表

## 1. 用户管理 (User)

### IAM 接口 vs Logto 接口

| IAM 路径 | 方法 | 功能 | Logto 路径 | 方法 | 功能 | 映射状态 |
|---------|------|------|------------|------|------|---------|
| `/v1/user/create` | POST | 创建用户 | `/api/users` | POST | 创建用户 | ✅ 已实现 |
| `/v1/user/delete` | POST | 删除用户 | `/api/users/{userId}` | DELETE | 删除用户 | ✅ 已实现 |
| `/v1/user/update` | POST | 更新用户 | `/api/users/{userId}` | PATCH | 更新用户 | ✅ 已实现 |
| `/v1/user/detail` | GET | 用户详情 | `/api/users/{userId}` | GET | 用户详情 | ✅ 已实现 |
| `/v1/user/pageList` | POST | 用户列表 | `/api/users` | GET | 用户列表(支持搜索) | ⚠️ 部分实现 |
| `/v1/user/updatePassword` | POST | 修改密码 | `/api/users/{userId}/password` | PATCH | 更新密码 | ✅ 已实现 |
| `/v1/user/updateStatus` | POST | 更新状态 | `/api/users/{userId}/is-suspended` | PATCH | 暂停用户 | ✅ 已实现 |
| `/v1/userIdentity/create` | POST | 创建身份 | `/api/users/{userId}/identities` | POST | 链接社交身份 | ✅ 已实现 |
| `/v1/userIdentity/delete` | POST | 删除身份 | `/api/users/{userId}/identities/{target}` | DELETE | 删除社交身份 | ✅ 已实现 |
| `/v1/userIdentity/update` | POST | 更新身份 | `/api/users/{userId}/identities/{target}` | PUT | 更新社交身份 | ✅ 已实现 |
| `/v1/userIdentity/detail` | GET | 身份详情 | `/api/users/{userId}/identities/{target}` | GET | 获取社交身份 | ✅ 已实现 |
| `/v1/userIdentity/pageList` | POST | 身份列表 | - | - | - | ⚠️ 待优化 |
| `/v1/userIdentity/getByUser` | GET | 获取用户身份 | `/api/users/{userId}/all-identities` | GET | 获取所有身份 | ✅ 已实现 |
| - | - | - | `/api/users/{userId}/roles` | GET/POST/PUT | 用户角色管理 | ❌ 未实现 |
| - | - | - | `/api/users/{userId}/mfa-verifications` | GET/POST | MFA 验证 | ❌ 未实现 |
| - | - | - | `/api/users/{userId}/sessions` | GET | 用户会话 | ❌ 未实现 |
| - | - | - | `/api/users/{userId}/custom-data` | GET/PATCH | 自定义数据 | ❌ 未实现 |
| - | - | - | `/api/users/{userId}/password/verify` | POST | 验证密码 | ❌ 未实现 |
| - | - | - | `/api/users/{userId}/has-password` | GET | 检查密码 | ❌ 未实现 |
| - | - | - | `/api/users/{userId}/organizations` | GET | 用户组织 | ❌ 未实现 |
| - | - | - | `/api/users/{userId}/sso-identities` | GET | SSO身份 | ❌ 未实现 |

---

## 2. 角色管理 (Role)

| IAM 路径 | 方法 | 功能 | Logto 路径 | 方法 | 功能 | 映射状态 |
|---------|------|------|------------|------|------|---------|
| `/v1/role/create` | POST | 创建角色 | `/api/roles` | POST | 创建角色 | ✅ 已实现 |
| `/v1/role/delete` | POST | 删除角色 | `/api/roles/{id}` | DELETE | 删除角色 | ✅ 已实现 |
| `/v1/role/update` | POST | 更新角色 | `/api/roles/{id}` | PATCH | 更新角色 | ✅ 已实现 |
| `/v1/role/detail` | GET | 角色详情 | `/api/roles/{id}` | GET | 角色详情 | ✅ 已实现 |
| `/v1/role/pageList` | POST | 角色列表 | `/api/roles` | GET | 角色列表 | ✅ 已实现 |
| - | - | - | `/api/roles/{id}/users` | GET/POST | 角色用户 | ⚠️ 分离到 userRole |
| - | - | - | `/api/roles/{id}/applications` | GET/POST | 角色应用 | ❌ 未实现 |
| - | - | - | `/api/roles/{id}/scopes` | GET/POST | 角色作用域 | ⚠️ 分离到 roleScope |

---

## 3. 应用管理 (Application)

| IAM 路径 | 方法 | 功能 | Logto 路径 | 方法 | 功能 | 映射状态 |
|---------|------|------|------------|------|------|---------|
| `/v1/application/create` | POST | 创建应用 | `/api/applications` | POST | 创建应用 | ✅ 已实现 |
| `/v1/application/delete` | POST | 删除应用 | `/api/applications/{id}` | DELETE | 删除应用 | ✅ 已实现 |
| `/v1/application/update` | POST | 更新应用 | `/api/applications/{id}` | PATCH | 更新应用 | ✅ 已实现 |
| `/v1/application/detail` | GET | 应用详情 | `/api/applications/{id}` | GET | 应用详情 | ✅ 已实现 |
| `/v1/application/pageList` | POST | 应用列表 | `/api/applications` | GET | 应用列表 | ✅ 已实现 |
| - | - | - | `/api/applications/{id}/secrets` | GET/POST | 应用密钥 | ❌ 未实现 |
| - | - | - | `/api/applications/{id}/roles` | GET/POST/PUT | 应用角色 | ❌ 未实现 |
| - | - | - | `/api/applications/{id}/sign-in-experience` | GET/PUT | 登录体验 | ❌ 未实现 |

---

## 4. 资源管理 (Resource)

| IAM 路径 | 方法 | 功能 | Logto 路径 | 方法 | 功能 | 映射状态 |
|---------|------|------|------------|------|------|---------|
| `/v1/resource/create` | POST | 创建资源 | `/api/resources` | POST | 创建资源 | ✅ 已实现 |
| `/v1/resource/delete` | POST | 删除资源 | `/api/resources/{id}` | DELETE | 删除资源 | ✅ 已实现 |
| `/v1/resource/update` | POST | 更新资源 | `/api/resources/{id}` | PATCH | 更新资源 | ✅ 已实现 |
| `/v1/resource/detail` | GET | 资源详情 | `/api/resources/{id}` | GET | 资源详情 | ✅ 已实现 |
| `/v1/resource/pageList` | POST | 资源列表 | `/api/resources` | GET | 资源列表 | ✅ 已实现 |
| - | - | - | `/api/resources/{id}/is-default` | PATCH | 设置默认 | ❌ 未实现 |
| - | - | - | `/api/resources/{resourceId}/scopes` | GET/POST | 资源作用域 | ⚠️ 分离到 scope |

---

## 5. 作用域管理 (Scope)

| IAM 路径 | 方法 | 功能 | Logto 路径 | 方法 | 功能 | 映射状态 |
|---------|------|------|------------|------|------|---------|
| `/v1/scope/create` | POST | 创建作用域 | `/api/resources/{resourceId}/scopes` | POST | 创建资源作用域 | ✅ 已实现 |
| `/v1/scope/delete` | POST | 删除作用域 | `/api/resources/{resourceId}/scopes/{scopeId}` | DELETE | 删除作用域 | ✅ 已实现 |
| `/v1/scope/update` | POST | 更新作用域 | `/api/resources/{resourceId}/scopes/{scopeId}` | PATCH | 更新作用域 | ✅ 已实现 |
| `/v1/scope/detail` | GET | 作用域详情 | `/api/resources/{resourceId}/scopes/{scopeId}` | GET | 获取作用域 | ✅ 已实现 |
| `/v1/scope/pageList` | POST | 作用域列表 | `/api/resources/{resourceId}/scopes` | GET | 作用域列表 | ✅ 已实现 |

---

## 6. 连接器管理 (Connector)

| IAM 路径 | 方法 | 功能 | Logto 路径 | 方法 | 功能 | 映射状态 |
|---------|------|------|------------|------|------|---------|
| `/v1/connector/create` | POST | 创建连接器 | `/api/connectors` | POST | 创建连接器 | ✅ 已实现 |
| `/v1/connector/delete` | POST | 删除连接器 | `/api/connectors/{id}` | DELETE | 删除连接器 | ✅ 已实现 |
| `/v1/connector/update` | POST | 更新连接器 | `/api/connectors/{id}` | PATCH | 更新连接器 | ✅ 已实现 |
| `/v1/connector/detail` | GET | 连接器详情 | `/api/connectors/{id}` | GET | 连接器详情 | ✅ 已实现 |
| `/v1/connector/pageList` | POST | 连接器列表 | `/api/connectors` | GET | 连接器列表 | ✅ 已实现 |
| - | - | - | `/api/connectors/{id}/authorization-uri` | POST | 社交授权URI | ❌ 未实现 |
| - | - | - | `/api/connectors/{connectorId}/config-testing` | POST | 测试配置 | ❌ 未实现 |

---

## 7. SSO 连接器 (SsoConnector)

| IAM 路径 | 方法 | 功能 | Logto 路径 | 方法 | 功能 | 映射状态 |
|---------|------|------|------------|------|------|---------|
| `/v1/ssoConnector/create` | POST | 创建SSO连接器 | `/api/sso-connectors` | POST | 创建SSO连接器 | ✅ 已实现 |
| `/v1/ssoConnector/delete` | POST | 删除SSO连接器 | `/api/sso-connectors/{id}` | DELETE | 删除SSO连接器 | ✅ 已实现 |
| `/v1/ssoConnector/update` | POST | 更新SSO连接器 | `/api/sso-connectors/{id}` | PATCH | 更新SSO连接器 | ✅ 已实现 |
| `/v1/ssoConnector/detail` | GET | SSO连接器详情 | `/api/sso-connectors/{id}` | GET | SSO连接器详情 | ✅ 已实现 |
| `/v1/ssoConnector/pageList` | POST | SSO连接器列表 | `/api/sso-connectors` | GET | SSO连接器列表 | ✅ 已实现 |
| - | - | - | `/api/sso-connector-providers` | GET | SSO提供商 | ❌ 未实现 |

---

## 8. 组织管理 (Organization)

| IAM 路径 | 方法 | 功能 | Logto 路径 | 方法 | 功能 | 映射状态 |
|---------|------|------|------------|------|------|---------|
| `/v1/organization/create` | POST | 创建组织 | `/api/organizations` | POST | 创建组织 | ✅ 已实现 |
| `/v1/organization/delete` | POST | 删除组织 | `/api/organizations/{id}` | DELETE | 删除组织 | ✅ 已实现 |
| `/v1/organization/update` | POST | 更新组织 | `/api/organizations/{id}` | PATCH | 更新组织 | ✅ 已实现 |
| `/v1/organization/detail` | GET | 组织详情 | `/api/organizations/{id}` | GET | 组织详情 | ✅ 已实现 |
| `/v1/organization/pageList` | POST | 组织列表 | `/api/organizations` | GET | 组织列表 | ✅ 已实现 |
| - | - | - | `/api/organizations/{id}/users` | GET/PUT/POST | 组织用户管理 | ⚠️ 分离到 relation |
| - | - | - | `/api/organizations/{id}/applications` | GET | 组织应用 | ❌ 未实现 |
| - | - | - | `/api/organizations/{id}/jit/*` | GET/POST/PUT/DELETE | JIT配置 | ❌ 未实现 |

---

## 9. 组织角色 (OrganizationRole)

| IAM 路径 | 方法 | 功能 | Logto 路径 | 方法 | 功能 | 映射状态 |
|---------|------|------|------------|------|------|---------|
| `/v1/organizationRole/create` | POST | 创建组织角色 | `/api/organization-roles` | POST | 创建组织角色 | ✅ 已实现 |
| `/v1/organizationRole/delete` | POST | 删除组织角色 | `/api/organization-roles/{id}` | DELETE | 删除组织角色 | ✅ 已实现 |
| `/v1/organizationRole/update` | POST | 更新组织角色 | `/api/organization-roles/{id}` | PATCH | 更新组织角色 | ✅ 已实现 |
| `/v1/organizationRole/detail` | GET | 组织角色详情 | `/api/organization-roles/{id}` | GET | 组织角色详情 | ✅ 已实现 |
| `/v1/organizationRole/pageList` | POST | 组织角色列表 | `/api/organization-roles` | GET | 组织角色列表 | ✅ 已实现 |
| - | - | - | `/api/organization-roles/{id}/scopes` | GET/POST/PUT | 角色作用域 | ⚠️ 分离到 relation |
| - | - | - | `/api/organization-roles/{id}/resource-scopes` | GET/POST/PUT | 资源作用域 | ❌ 未实现 |

---

## 10. 部门管理 (Department)

| IAM 路径 | 方法 | 功能 | Logto 路径 | 方法 | 功能 | 映射状态 |
|---------|------|------|------------|------|------|---------|
| `/v1/department/create` | POST | 创建部门 | - | - | - | ⚠️ Logto无直接对应 |
| `/v1/department/delete` | POST | 删除部门 | - | - | - | ⚠️ Logto无直接对应 |
| `/v1/department/update` | POST | 更新部门 | - | - | - | ⚠️ Logto无直接对应 |
| `/v1/department/detail` | GET | 部门详情 | - | - | - | ⚠️ Logto无直接对应 |
| `/v1/department/pageList` | POST | 部门列表 | - | - | - | ⚠️ Logto无直接对应 |
| `/v1/department/tree` | GET | 部门树 | - | - | - | ⚠️ Logto无直接对应 |

---

## 11. 菜单管理 (Menu)

| IAM 路径 | 方法 | 功能 | Logto 路径 | 方法 | 功能 | 映射状态 |
|---------|------|------|------------|------|------|---------|
| `/v1/menu/create` | POST | 创建菜单 | - | - | - | ⚠️ Logto无直接对应 |
| `/v1/menu/delete` | POST | 删除菜单 | - | - | - | ⚠️ Logto无直接对应 |
| `/v1/menu/update` | POST | 更新菜单 | - | - | - | ⚠️ Logto无直接对应 |
| `/v1/menu/detail` | GET | 菜单详情 | - | - | - | ⚠️ Logto无直接对应 |
| `/v1/menu/pageList` | POST | 菜单列表 | - | - | - | ⚠️ Logto无直接对应 |
| `/v1/menu/tree` | GET | 菜单树 | - | - | - | ⚠️ Logto无直接对应 |

---

## 12. 认证管理 (Auth)

| IAM 路径 | 方法 | 功能 | Logto 路径 | 方法 | 功能 | 映射状态 |
|---------|------|------|------------|------|------|---------|
| `/v1/auth/login` | POST | 登录 | `/api/experience` | PUT | 初始化交互 | ✅ 已实现 |
| `/v1/auth/register` | POST | 注册 | `/api/experience` + `/api/experience/submit` | POST | 交互提交流程 | ✅ 已实现 |
| `/v1/auth/refresh-token` | POST | 刷新令牌 | - | - | - | ⚠️ 需确认 |
| `/v1/auth/logout` | POST | 登出 | `/api/my-account/sessions/{sessionId}` | DELETE | 撤销会话 | ✅ 已实现 |
| `/v1/auth/userinfo` | GET | 用户信息 | `/api/my-account` | GET | 账户信息 | ✅ 已实现 |
| `/v1/auth/sso/authorizationUrl` | GET | SSO授权URL | `/api/experience/verification/enterprise-sso` | POST | 企业SSO验证 | ✅ 已实现 |
| `/v1/auth/sso/callback` | GET | SSO回调 | - | - | - | ✅ 已实现 |

---

## 13. 关系类接口映射

### 用户角色关系 (UserRole)

| IAM 路径 | 方法 | 功能 | Logto 路径 | 方法 | 功能 | 映射状态 |
|---------|------|------|------------|------|------|---------|
| `/v1/userRole/create` | POST | 分配角色 | `/api/users/{userId}/roles` | POST | 分配用户角色 | ✅ 已实现 |
| `/v1/userRole/delete` | POST | 移除角色 | `/api/users/{userId}/roles/{roleId}` | DELETE | 移除用户角色 | ✅ 已实现 |
| `/v1/userRole/pageList` | POST | 角色列表 | `/api/users/{userId}/roles` | GET | 获取用户角色 | ✅ 已实现 |

### 角色菜单关系 (RoleMenu)

| IAM 路径 | 方法 | 功能 | Logto 路径 | 方法 | 功能 | 映射状态 |
|---------|------|------|------------|------|------|---------|
| `/v1/roleMenu/create` | POST | 分配菜单 | - | - | - | ⚠️ Logto无直接对应 |
| `/v1/roleMenu/delete` | POST | 移除菜单 | - | - | - | ⚠️ Logto无直接对应 |
| `/v1/roleMenu/pageList` | POST | 菜单列表 | - | - | - | ⚠️ Logto无直接对应 |

### 角色作用域关系 (RoleScope)

| IAM 路径 | 方法 | 功能 | Logto 路径 | 方法 | 功能 | 映射状态 |
|---------|------|------|------------|------|------|---------|
| `/v1/roleScope/create` | POST | 分配作用域 | `/api/roles/{id}/scopes` | POST | 分配角色作用域 | ✅ 已实现 |
| `/v1/roleScope/delete` | POST | 移除作用域 | `/api/roles/{id}/scopes/{scopeId}` | DELETE | 移除角色作用域 | ✅ 已实现 |
| `/v1/roleScope/pageList` | POST | 作用域列表 | `/api/roles/{id}/scopes` | GET | 获取角色作用域 | ✅ 已实现 |

---

## 14. 其他管理模块

### 租户管理 (Tenant)

| IAM 路径 | 方法 | 功能 | Logto 路径 | 方法 | 功能 | 映射状态 |
|---------|------|------|------------|------|------|---------|
| `/v1/tenant/create` | POST | 创建租户 | - | - | - | ⚠️ Logto无直接对应 |
| `/v1/tenant/delete` | POST | 删除租户 | - | - | - | ⚠️ Logto无直接对应 |
| `/v1/tenant/update` | POST | 更新租户 | - | - | - | ⚠️ Logto无直接对应 |
| `/v1/tenant/detail` | GET | 租户详情 | - | - | - | ⚠️ Logto无直接对应 |
| `/v1/tenant/pageList` | POST | 租户列表 | - | - | - | ⚠️ Logto无直接对应 |

### 系统管理 (System)

| IAM 路径 | 方法 | 功能 | Logto 路径 | 方法 | 功能 | 映射状态 |
|---------|------|------|------------|------|------|---------|
| `/v1/system/create` | POST | 创建系统 | - | - | - | ⚠️ Logto无直接对应 |
| `/v1/system/delete` | POST | 删除系统 | - | - | - | ⚠️ Logto无直接对应 |
| `/v1/system/update` | POST | 更新系统 | - | - | - | ⚠️ Logto无直接对应 |
| `/v1/system/detail` | GET | 系统详情 | `/api/systems/application` | GET | 应用常量 | ⚠️ 部分对应 |
| `/v1/system/pageList` | POST | 系统列表 | - | - | - | ⚠️ Logto无直接对应 |

### 日志管理 (Log)

| IAM 路径 | 方法 | 功能 | Logto 路径 | 方法 | 功能 | 映射状态 |
|---------|------|------|------------|------|------|---------|
| `/v1/log/detail` | GET | 日志详情 | `/api/logs/{id}` | GET | 日志详情 | ✅ 已实现 |
| `/v1/log/pageList` | POST | 日志列表 | `/api/logs` | GET | 日志列表 | ✅ 已实现 |

---

## 15. 补充：文档缺失的 Logto API（按模块）

> 说明：以下接口来自 Logto `packages/core/src/routes/**/*.openapi.json`，为本对照文档中原先未覆盖或覆盖不完整的接口。  
> 映射状态说明：`❌ 待映射` 表示 IAM 文档中暂无对应接口；`⚠️ 部分覆盖` 表示已有部分能力但未完整覆盖该组接口。

### 15.1 用户扩展接口（User Extended）

| Logto 路径 | 方法 | 功能 | 映射状态 |
|------------|------|------|---------|
| `/api/users/{userId}/logto-configs` | GET/PATCH | 用户 Logto 配置（MFA、passkey 状态） | ❌ 待映射 |
| `/api/users/{userId}/profile` | PATCH | 更新用户 profile | ❌ 待映射 |
| `/api/users/{userId}/sessions/{sessionId}` | GET/DELETE | 会话详情/会话撤销 | ❌ 待映射 |
| `/api/users/{userId}/mfa-verifications/{verificationId}` | DELETE | 删除指定 MFA 验证 | ❌ 待映射 |

### 15.2 应用扩展接口（Application Extended）

| Logto 路径 | 方法 | 功能 | 映射状态 |
|------------|------|------|---------|
| `/api/applications/{id}/legacy-secret` | DELETE | 删除应用 legacy secret | ❌ 待映射 |
| `/api/applications/{id}/secrets/{name}` | PATCH | 更新指定应用 secret | ❌ 待映射 |
| `/api/applications/{id}/secrets/{name}` | DELETE | 删除指定应用 secret | ❌ 待映射 |
| `/api/applications/{id}/custom-data` | GET/PATCH | 应用自定义数据 | ❌ 待映射 |
| `/api/applications/{id}/organizations` | GET/POST/PUT/DELETE | 应用-组织关系 | ❌ 待映射 |
| `/api/applications/{id}/users/{userId}/consent-organizations` | GET/POST/PUT/DELETE | 用户对应用的组织授权 | ❌ 待映射 |
| `/api/applications/{applicationId}/user-consent-scopes` | GET/POST | 应用用户同意作用域 | ❌ 待映射 |
| `/api/applications/{applicationId}/user-consent-scopes/{scopeType}/{scopeId}` | DELETE | 删除应用用户同意作用域 | ❌ 待映射 |
| `/api/applications/{id}/protected-app-metadata/custom-domains` | GET/POST | 应用自定义域名 | ❌ 待映射 |
| `/api/applications/{id}/protected-app-metadata/custom-domains/{domain}` | DELETE | 删除应用自定义域名 | ❌ 待映射 |

### 15.3 组织扩展接口（Organization Extended）

| Logto 路径 | 方法 | 功能 | 映射状态 |
|------------|------|------|---------|
| `/api/organizations/{id}/applications` | GET/POST/PUT/DELETE | 组织应用管理 | ❌ 待映射 |
| `/api/organizations/{id}/applications/roles` | POST | 批量给组织内应用分配角色 | ❌ 待映射 |
| `/api/organizations/{id}/applications/{applicationId}/roles` | GET/POST/PUT | 组织内应用角色管理 | ❌ 待映射 |
| `/api/organizations/{id}/applications/{applicationId}/roles/{organizationRoleId}` | DELETE | 移除组织内应用角色 | ❌ 待映射 |
| `/api/organizations/{id}/jit/email-domains` | GET/POST/PUT | JIT 邮箱域名配置 | ❌ 待映射 |
| `/api/organizations/{id}/jit/email-domains/{emailDomain}` | DELETE | 删除 JIT 邮箱域名 | ❌ 待映射 |
| `/api/organizations/{id}/jit/sso-connectors` | GET/POST/PUT | JIT SSO 连接器配置 | ❌ 待映射 |
| `/api/organizations/{id}/jit/sso-connectors/{ssoConnectorId}` | DELETE | 删除 JIT SSO 连接器 | ❌ 待映射 |
| `/api/organizations/{id}/users/{userId}/scopes` | GET | 获取组织内用户 scopes | ❌ 待映射 |

### 15.4 组织权限模型补充（Org Scope / Org Invitation）

| Logto 路径 | 方法 | 功能 | 映射状态 |
|------------|------|------|---------|
| `/api/organization-scopes` | GET/POST | 组织级 scope 管理 | ❌ 待映射 |
| `/api/organization-scopes/{id}` | GET/PATCH/DELETE | 组织级 scope 详情/更新/删除 | ❌ 待映射 |
| `/api/organization-invitations` | GET/POST | 组织邀请管理 | ❌ 待映射 |
| `/api/organization-invitations/{id}` | GET/DELETE | 组织邀请详情/删除 | ❌ 待映射 |
| `/api/organization-invitations/{id}/status` | PUT | 更新邀请状态 | ❌ 待映射 |
| `/api/organization-invitations/{id}/message` | POST | 重发邀请消息 | ❌ 待映射 |

### 15.5 登录体验与交互流（Sign-in Experience / Experience）

| Logto 路径 | 方法 | 功能 | 映射状态 |
|------------|------|------|---------|
| `/api/sign-in-exp` | GET/PATCH | 默认登录体验配置 | ❌ 待映射 |
| `/api/sign-in-exp/default/check-password` | POST | 密码策略检查 | ❌ 待映射 |
| `/api/sign-in-exp/default/custom-ui-assets` | POST | 上传自定义登录 UI 资源 | ❌ 待映射 |
| `/api/experience/interaction-event` | PUT | 更新交互事件 | ❌ 待映射 |
| `/api/experience/identification` | POST | 交互流用户识别 | ❌ 待映射 |
| `/api/experience/interaction` | GET | 获取交互公开数据 | ❌ 待映射 |
| `/api/experience/sso-connectors` | GET | 按邮箱域名匹配可用 SSO 连接器 | ❌ 待映射 |

### 15.6 验证与认证增强（Verification / Authn）

| Logto 路径 | 方法 | 功能 | 映射状态 |
|------------|------|------|---------|
| `/api/verifications/password` | POST | 密码验证记录 | ❌ 待映射 |
| `/api/verifications/verification-code` | POST | 发起验证码验证 | ❌ 待映射 |
| `/api/verifications/verification-code/verify` | POST | 校验验证码 | ❌ 待映射 |
| `/api/verifications/social` | POST | 发起社交验证 | ❌ 待映射 |
| `/api/verifications/social/verify` | POST | 社交验证回调校验 | ❌ 待映射 |
| `/api/verifications/web-authn/registration` | POST | 发起 WebAuthn 注册 | ❌ 待映射 |
| `/api/verifications/web-authn/registration/verify` | POST | 校验 WebAuthn 注册 | ❌ 待映射 |
| `/api/authn/hasura` | GET | Hasura Auth Hook | ❌ 待映射 |
| `/api/authn/single-sign-on/saml/{connectorId}` | POST | SAML SSO ACS 端点 | ❌ 待映射 |

### 15.7 密钥与租户配置（Configs / Keys / JWT）

| Logto 路径 | 方法 | 功能 | 映射状态 |
|------------|------|------|---------|
| `/api/configs/admin-console` | GET/PATCH | 控制台全局配置 | ❌ 待映射 |
| `/api/configs/oidc/{keyType}` | GET | OIDC 密钥列表 | ❌ 待映射 |
| `/api/configs/oidc/{keyType}/{keyId}` | DELETE | 删除 OIDC 密钥 | ❌ 待映射 |
| `/api/configs/oidc/{keyType}/rotate` | POST | 轮换 OIDC 密钥 | ❌ 待映射 |
| `/api/configs/oidc/session` | GET/PATCH | OIDC 会话配置 | ❌ 待映射 |
| `/api/configs/jwt-customizer` | GET | JWT 自定义脚本列表 | ❌ 待映射 |
| `/api/configs/jwt-customizer/{tokenTypePath}` | GET/PUT/PATCH/DELETE | JWT 自定义脚本管理 | ❌ 待映射 |
| `/api/configs/jwt-customizer/test` | POST | JWT 自定义脚本测试 | ❌ 待映射 |
| `/api/configs/id-token` | GET/PUT | ID Token 扩展 claims 配置 | ❌ 待映射 |

### 15.8 连接器生态补充（Connector / SSO Connector）

| Logto 路径 | 方法 | 功能 | 映射状态 |
|------------|------|------|---------|
| `/api/connector-factories` | GET | 连接器工厂列表 | ❌ 待映射 |
| `/api/connector-factories/{id}` | GET | 连接器工厂详情 | ❌ 待映射 |
| `/api/captcha-provider` | GET/PUT/DELETE | Captcha 提供商配置 | ❌ 待映射 |
| `/api/sso-connectors/{id}/idp-initiated-auth-config` | GET/PUT/DELETE | IdP 发起登录配置 | ❌ 待映射 |

### 15.9 SAML 应用（SAML Application）

| Logto 路径 | 方法 | 功能 | 映射状态 |
|------------|------|------|---------|
| `/api/saml-applications` | POST | 创建 SAML 应用 | ❌ 待映射 |
| `/api/saml-applications/{id}` | GET/PATCH/DELETE | SAML 应用管理 | ❌ 待映射 |
| `/api/saml-applications/{id}/metadata` | GET | 获取 SAML 元数据 | ❌ 待映射 |
| `/api/saml-applications/{id}/secrets` | GET/POST | SAML 证书管理 | ❌ 待映射 |
| `/api/saml-applications/{id}/secrets/{secretId}` | PATCH/DELETE | 更新/删除 SAML 证书 | ❌ 待映射 |
| `/api/saml-applications/{id}/callback` | GET | SAML 应用回调 | ❌ 待映射 |
| `/api/saml/{id}/authn` | GET/POST | SAML Authn 请求处理 | ❌ 待映射 |

### 15.10 运维与内容管理（Hook / Domain / Template / Phrase / Profile）

| Logto 路径 | 方法 | 功能 | 映射状态 |
|------------|------|------|---------|
| `/api/hooks` | GET/POST | Webhook 管理 | ❌ 待映射 |
| `/api/hooks/{id}` | GET/PATCH/DELETE | Webhook 详情/更新/删除 | ❌ 待映射 |
| `/api/hooks/{id}/recent-logs` | GET | Webhook 最近日志 | ❌ 待映射 |
| `/api/hooks/{id}/test` | POST | Webhook 测试触发 | ❌ 待映射 |
| `/api/hooks/{id}/signing-key` | PATCH | Webhook 签名密钥更新 | ❌ 待映射 |
| `/api/domains` | GET/POST | 自定义域名管理 | ❌ 待映射 |
| `/api/domains/{id}` | GET/DELETE | 域名详情/删除 | ❌ 待映射 |
| `/api/domains/cleanup` | POST | 清理过期域名 | ❌ 待映射 |
| `/api/email-templates` | GET/PUT/DELETE | 邮件模板管理 | ❌ 待映射 |
| `/api/email-templates/{id}` | GET/DELETE | 邮件模板详情/删除 | ❌ 待映射 |
| `/api/email-templates/{id}/details` | PATCH | 邮件模板详情更新 | ❌ 待映射 |
| `/api/custom-phrases` | GET | 全量短语包 | ❌ 待映射 |
| `/api/custom-phrases/{languageTag}` | GET/PUT/DELETE | 多语言短语管理 | ❌ 待映射 |
| `/api/custom-profile-fields` | GET/POST | 自定义资料字段管理 | ❌ 待映射 |
| `/api/custom-profile-fields/batch` | POST | 批量创建自定义字段 | ❌ 待映射 |
| `/api/custom-profile-fields/{name}` | GET/PUT/DELETE | 自定义字段详情/更新/删除 | ❌ 待映射 |
| `/api/custom-profile-fields/properties/sie-order` | POST | 自定义字段显示顺序配置 | ❌ 待映射 |

### 15.11 账户中心与用户自服务（My Account）

| Logto 路径 | 方法 | 功能 | 映射状态 |
|------------|------|------|---------|
| `/api/my-account` | GET/PATCH | 当前用户账户资料 | ⚠️ 部分覆盖 |
| `/api/my-account/identities` | GET/POST | 当前用户身份绑定管理 | ❌ 待映射 |
| `/api/my-account/identities/{target}` | GET/PUT/DELETE | 当前用户单一身份管理 | ❌ 待映射 |
| `/api/my-account/sessions` | GET | 当前用户会话列表 | ❌ 待映射 |
| `/api/my-account/sessions/{sessionId}` | DELETE | 当前用户会话撤销 | ⚠️ 部分覆盖 |
| `/api/my-account/mfa-verifications` | GET/POST | 当前用户 MFA 验证管理 | ❌ 待映射 |

### 15.12 其他平台接口（System / Dashboard / Token / Asset / Meta）

| Logto 路径 | 方法 | 功能 | 映射状态 |
|------------|------|------|---------|
| `/api/dashboard/users/total` | GET | 用户总数统计 | ❌ 待映射 |
| `/api/dashboard/users/new` | GET | 新增用户统计 | ❌ 待映射 |
| `/api/dashboard/users/active` | GET | 活跃用户统计 | ❌ 待映射 |
| `/api/one-time-tokens` | GET/POST | 一次性令牌管理 | ❌ 待映射 |
| `/api/one-time-tokens/{id}` | GET/DELETE | 一次性令牌详情/删除 | ❌ 待映射 |
| `/api/one-time-tokens/verify` | POST | 一次性令牌校验 | ❌ 待映射 |
| `/api/one-time-tokens/{id}/status` | PUT | 一次性令牌状态更新 | ❌ 待映射 |
| `/api/subject-tokens` | POST | Subject token 创建 | ❌ 待映射 |
| `/api/user-assets/service-status` | GET | 用户资产服务状态 | ❌ 待映射 |
| `/api/user-assets` | POST | 用户资产上传 | ❌ 待映射 |
| `/api/secrets/{id}` | DELETE | 通用 secret 删除 | ❌ 待映射 |
| `/api/sentinel-activities/delete` | POST | 安全活动批量删除 | ❌ 待映射 |
| `/api/status` | GET | 服务健康检查 | ❌ 待映射 |
| `/api/swagger.json` | GET | API 文档 JSON | ❌ 待映射 |
| `/api/.well-known/management.openapi.json` | GET | Management API OpenAPI 文档 | ❌ 待映射 |
| `/api/.well-known/experience.openapi.json` | GET | Experience API OpenAPI 文档 | ❌ 待映射 |
| `/api/.well-known/user.openapi.json` | GET | User API OpenAPI 文档 | ❌ 待映射 |

---

## 映射状态汇总

> 说明：以下汇总仅统计第 1-14 章原始映射表；第 15 章为新增补充清单，暂未纳入下表统计。

| 状态 | 数量 | 说明 |
|------|------|------|
| ✅ 已实现 | 58 | 两边都有对应接口且功能一致 |
| ⚠️ 部分实现/需优化 | 15 | 有对应但实现有差异或分离到多个接口 |
| ❌ 未实现 | 20 | Logto有但IAM未实现的功能 |
| ⚠️ Logto无直接对应 | 18 | IAM特有功能，Logto无直接对应 |

**总计 IAM 接口: 109 个**
**Logto 总接口: 约 200+ 个**
