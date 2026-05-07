# IAM & Logto API 接口文档

## 文档说明

本文档将 Ark-IAM 与 Logto 的 API 接口进行合并，建立两个服务之间的接口映射关系，目的是：
1. 分析 IAM 还缺什么功能
2. 验证 IAM 的逻辑是否完整
3. 借助 Logto 完善 IAM

---

## 一、模块对应关系总览

| IAM 模块 | IAM 接口数 | Logto 模块 | Logto 接口数 | 对应关系 | 完整性 |
|----------|-----------|------------|--------------|----------|--------|
| 应用管理 | 5 | Applications | 20+ | 部分对应 | **缺失较多** |
| 认证 | 7 | Experience + Authn | 30+ | 部分对应 | **缺失较多** |
| 连接器 | 5 | Connector | 9 | 部分对应 | **缺失较多** |
| SSO连接器 | 5 | SSO Connector | 9 | 部分对应 | **缺失较多** |
| 部门 | 6 | - | - | 无对应 | IAM特有 |
| 日志 | 2 | Log | 2 | 对应 | 基本完整 |
| 菜单管理 | 6 | - | - | 无对应 | IAM特有 |
| 组织 | 5 | Organization | 40+ | 对应 | **缺失较多** |
| 组织角色 | 5 | Organization Role | 10+ | 对应 | **缺失较多** |
| 组织角色用户关联 | 3 | Organization | 40+ | 部分对应 | **缺失较多** |
| 组织用户关联 | 3 | Organization | 40+ | 部分对应 | **缺失较多** |
| 权限范围 | 5 | Resource + Organization Scope | 14 | 对应 | 基本完整 |
| 系统配置 | 5 | Logto Config | 14+ | 对应 | **缺失较多** |
| 租户管理 | 5 | - | - | 无对应 | IAM特有 |
| 用户管理 | 19 | Admin User | 50+ | 对应 | **缺失较多** |
| 用户角色 | 3 | Role | 13 | 对应 | 基本完整 |
| 资源管理 | 5 | Resource | 9 | 对应 | 基本完整 |
| 角色管理 | 5 | Role | 13 | 对应 | 基本完整 |
| 角色菜单 | 3 | - | - | 无对应 | IAM特有 |
| 角色权限范围 | 3 | Role + Resource Scope | 13 | 对应 | 基本完整 |

---

## 二、接口映射详情

### 2.1 应用管理 (Application)

| 功能 | IAM 接口 | Logto 接口 | 映射状态 | 缺失说明 |
|------|----------|------------|----------|----------|
| 创建应用 | POST /v1/iam/application/create | POST /api/applications | ✅ 对应 | Logto支持更多应用类型 |
| 删除应用 | POST /v1/iam/application/delete | DELETE /api/applications/{id} | ✅ 对应 | - |
| 应用详情 | GET /v1/iam/application/detail | GET /api/applications/{id} | ✅ 对应 | - |
| 应用列表 | POST /v1/iam/application/pageList | GET /api/applications | ✅ 对应 | - |
| 修改应用 | POST /v1/iam/application/update | PATCH /api/applications/{id} | ✅ 对应 | - |
| 应用角色 | - | GET /api/applications/{id}/roles | ❌ 缺失 | 需要实现 |
| 角色分配 | - | POST /api/applications/{id}/roles | ❌ 缺失 | 需要实现 |
| 应用组织 | - | GET /api/applications/{id}/organizations | ❌ 缺失 | 需要实现 |
| 应用密钥 | - | GET/POST/DELETE /api/applications/{id}/secrets | ❌ 缺失 | 需要实现 |
| 自定义域名 | - | GET/POST/DELETE /api/applications/{id}/protected-app-metadata/custom-domains | ❌ 缺失 | 需要实现 |
| 用户同意范围 | - | GET/POST/DELETE /api/applications/{id}/user-consent-scopes | ❌ 缺失 | 需要实现 |

**IAM 缺失功能汇总：**
- 应用角色管理（分配/移除角色给应用）
- 应用关联组织
- 应用密钥管理（创建、删除、轮换）
- 自定义域名管理
- 用户同意范围管理

---

### 2.2 认证 (Authentication)

| 功能 | IAM 接口 | Logto 接口 | 映射状态 | 缺失说明 |
|------|----------|------------|----------|----------|
| 用户登录 | POST /v1/iam/login | POST /api/experience (submit) | ✅ 对应 | Logto流程更复杂 |
| 用户注册 | POST /v1/iam/register | POST /api/experience (submit) | ✅ 对应 | Logto流程更复杂 |
| 登出 | POST /v1/iam/logout | DELETE /api/my-account/sessions/{id} | ✅ 对应 | - |
| 刷新令牌 | POST /v1/iam/refreshToken | - | ✅ 特有 | - |
| 获取用户信息 | GET /v1/iam/userinfo | GET /api/my-account | ✅ 对应 | - |
| SSO授权URL | GET /v1/iam/authorizationUrl | - | ✅ 特有 | - |
| SSO回调 | GET /v1/iam/callback | - | ✅ 特有 | - |
| 密码验证 | - | POST /api/verifications/password | ❌ 缺失 | 需要实现 |
| 验证码 | - | POST /api/verifications/verification-code | ❌ 缺失 | 需要实现 |
| MFA设置 | - | GET/PATCH /api/my-account/mfa-settings | ❌ 缺失 | 需要实现 |
| TOTP | - | POST /api/my-account/mfa-verifications/totp | ❌ 缺失 | 需要实现 |
| WebAuthn | - | POST /api/experience/verification/web-authn/* | ❌ 缺失 | 需要实现 |
| 社交登录 | - | POST /api/experience/verification/social/* | ❌ 缺失 | 需要实现 |
| 登录体验 | - | GET/PATCH /api/sign-in-exp | ❌ 缺失 | 需要实现 |

**IAM 缺失功能汇总：**
- 密码强度验证
- 验证码发送与验证
- MFA 多因素认证（TOTP、备用码）
- WebAuthn/Passkey 认证
- 社交登录验证
- 登录体验配置（品牌色、语言、sign-in/sign-up表单等）

---

### 2.3 用户管理 (User)

| 功能 | IAM 接口 | Logto 接口 | 映射状态 | 缺失说明 |
|------|----------|------------|----------|----------|
| 创建用户 | POST /v1/iam/user/create | POST /api/users | ✅ 对应 | - |
| 删除用户 | POST /v1/iam/user/delete | DELETE /api/users/{id} | ✅ 对应 | - |
| 用户详情 | GET /v1/iam/user/detail | GET /api/users/{id} | ✅ 对应 | Logto支持更多查询参数 |
| 用户列表 | POST /v1/iam/user/pageList | GET /api/users | ✅ 对应 | - |
| 修改用户 | POST /v1/iam/user/update | PATCH /api/users/{id} | ✅ 对应 | - |
| 修改密码 | POST /v1/iam/user/updatePassword | PATCH /api/users/{id}/password | ✅ 对应 | - |
| 修改状态 | POST /v1/iam/user/updateStatus | PATCH /api/users/{id}/is-suspended | ✅ 对应 | - |
| 分配部门 | POST /v1/iam/user/assignDepartments | - | ✅ 特有 | - |
| 创建身份 | POST /v1/iam/user/createUserIdentity | POST /api/users/{id}/identities | ✅ 对应 | - |
| 删除身份 | POST /v1/iam/user/deleteUserIdentity | DELETE /api/users/{id}/identities/{target} | ✅ 对应 | - |
| 身份详情 | GET /v1/iam/user/detailUserIdentity | GET /api/users/{id}/identities/{target} | ✅ 对应 | - |
| 身份列表 | GET /v1/iam/user/getUserIdentityByUser | GET /api/users/{id}/all-identities | ✅ 对应 | - |
| 部门关联详情 | GET /v1/iam/user/detailUserDepartmentRelation | - | ✅ 特有 | - |
| 部门关联列表 | GET /v1/iam/user/getUserDepartmentRelationByUser | - | ✅ 特有 | - |
| 部门关联创建 | POST /v1/iam/user/createUserDepartmentRelation | - | ✅ 特有 | - |
| 部门关联删除 | POST /v1/iam/user/deleteUserDepartmentRelation | - | ✅ 特有 | - |
| 部门关联修改 | POST /v1/iam/user/updateUserDepartmentRelation | - | ✅ 特有 | - |
| 部门关联分页 | POST /v1/iam/user/pageListUserDepartmentRelation | - | ✅ 特有 | - |
| 登录日志 | GET /v1/iam/user/getUserLoginLogByUser | - | ✅ 特有 | - |
| 登录日志详情 | GET /v1/iam/user/detailUserLoginLog | - | ✅ 特有 | - |
| 登录日志分页 | POST /v1/iam/user/pageListUserLoginLog | - | ✅ 特有 | - |
| 自定义数据 | - | GET/PATCH /api/users/{id}/custom-data | ❌ 缺失 | 需要实现 |
| 用户MFA | - | GET/POST/DELETE /api/users/{id}/mfa-verifications | ❌ 缺失 | 需要实现 |
| 用户会话 | - | GET/DELETE /api/users/{id}/sessions | ❌ 缺失 | 需要实现 |
| 用户角色 | POST /v1/iam/userRole/* | POST/DELETE /api/users/{id}/roles | ✅ 对应 | - |
| 用户SSO身份 | - | GET /api/users/{id}/sso-identities/{id} | ❌ 缺失 | 需要实现 |
| 个人访问令牌 | - | GET/POST/DELETE /api/users/{id}/personal-access-tokens | ❌ 缺失 | 需要实现 |
| 用户授权 | - | GET/DELETE /api/users/{id}/grants | ❌ 缺失 | 需要实现 |

**IAM 缺失功能汇总：**
- 用户自定义数据管理
- 用户MFA验证管理
- 用户会话管理（查看、撤销）
- 用户SSO身份管理
- 个人访问令牌（PAT）
- 用户授权管理

---

### 2.4 角色管理 (Role)

| 功能 | IAM 接口 | Logto 接口 | 映射状态 | 缺失说明 |
|------|----------|------------|----------|----------|
| 创建角色 | POST /v1/iam/role/create | POST /api/roles | ✅ 对应 | - |
| 删除角色 | POST /v1/iam/role/delete | DELETE /api/roles/{id} | ✅ 对应 | - |
| 角色详情 | GET /v1/iam/role/detail | GET /api/roles/{id} | ✅ 对应 | - |
| 角色列表 | POST /v1/iam/role/pageList | GET /api/roles | ✅ 对应 | - |
| 修改角色 | POST /v1/iam/role/update | PATCH /api/roles/{id} | ✅ 对应 | - |
| 角色用户 | - | GET /api/roles/{id}/users | ❌ 缺失 | 需要实现 |
| 用户分配 | - | POST /api/roles/{id}/users | ❌ 缺失 | 需要实现 |
| 移除用户 | - | DELETE /api/roles/{id}/users/{userId} | ❌ 缺失 | 需要实现 |
| 角色应用 | - | GET /api/roles/{id}/applications | ❌ 缺失 | 需要实现 |
| 应用分配 | - | POST /api/roles/{id}/applications | ❌ 缺失 | 需要实现 |
| 角色范围 | POST /v1/iam/roleScope/* | GET/POST/DELETE /api/roles/{id}/scopes | ✅ 对应 | - |

**IAM 缺失功能汇总：**
- 角色关联用户查询
- 角色分配给用户
- 角色关联应用查询
- 角色分配给应用

---

### 2.5 组织管理 (Organization)

| 功能 | IAM 接口 | Logto 接口 | 映射状态 | 缺失说明 |
|------|----------|------------|----------|----------|
| 创建组织 | POST /v1/iam/organization/create | POST /api/organizations | ✅ 对应 | - |
| 删除组织 | POST /v1/iam/organization/delete | DELETE /api/organizations/{id} | ✅ 对应 | - |
| 组织详情 | GET /v1/iam/organization/detail | GET /api/organizations/{id} | ✅ 对应 | - |
| 组织列表 | POST /v1/iam/organization/pageList | GET /api/organizations | ✅ 对应 | - |
| 修改组织 | POST /v1/iam/organization/update | PATCH /api/organizations/{id} | ✅ 对应 | - |
| 组织用户 | POST /v1/iam/organizationUser/* | GET/POST/PUT/DELETE /api/organizations/{id}/users | ✅ 对应 | - |
| 组织角色 | POST /v1/iam/organizationRole/* | GET/POST/PATCH/DELETE /api/organization-roles | ✅ 对应 | - |
| 角色用户 | POST /v1/iam/organizationRoleUser/* | POST /api/organizations/{id}/users/roles | ✅ 对应 | - |
| JIT配置 | - | GET/POST/PUT/DELETE /api/organizations/{id}/jit/* | ❌ 缺失 | 需要实现 |
| 组织邀请 | - | GET/POST /api/organization-invitations | ❌ 缺失 | 需要实现 |
| 组织应用 | - | GET/POST /api/organizations/{id}/applications | ❌ 缺失 | 需要实现 |
| 应用角色 | - | POST /api/organizations/{id}/applications/roles | ❌ 缺失 | 需要实现 |

**IAM 缺失功能汇总：**
- 组织JIT（即时加入）配置（默认角色、SSO连接器、邮件域）
- 组织邀请管理
- 组织关联应用
- 组织应用角色分配

---

### 2.6 资源管理 (Resource)

| 功能 | IAM 接口 | Logto 接口 | 映射状态 | 缺失说明 |
|------|----------|------------|----------|----------|
| 创建资源 | POST /v1/iam/resource/create | POST /api/resources | ✅ 对应 | - |
| 删除资源 | POST /v1/iam/resource/delete | DELETE /api/resources/{id} | ✅ 对应 | - |
| 资源详情 | GET /v1/iam/resource/detail | GET /api/resources/{id} | ✅ 对应 | - |
| 资源列表 | POST /v1/iam/resource/pageList | GET /api/resources | ✅ 对应 | - |
| 修改资源 | POST /v1/iam/resource/update | PATCH /api/resources/{id} | ✅ 对应 | - |
| 设置默认 | - | PATCH /api/resources/{id}/is-default | ❌ 缺失 | 需要实现 |
| 资源范围 | POST /v1/iam/scope/* | GET/POST /api/resources/{id}/scopes | ✅ 对应 | - |

---

### 2.7 连接器 (Connector)

| 功能 | IAM 接口 | Logto 接口 | 映射状态 | 缺失说明 |
|------|----------|------------|----------|----------|
| 创建连接器 | POST /v1/iam/connector/create | POST /api/connectors | ✅ 对应 | - |
| 删除连接器 | POST /v1/iam/connector/delete | DELETE /api/connectors/{id} | ✅ 对应 | - |
| 连接器详情 | GET /v1/iam/connector/detail | GET /api/connectors/{id} | ✅ 对应 | - |
| 连接器列表 | POST /v1/iam/connector/pageList | GET /api/connectors | ✅ 对应 | - |
| 修改连接器 | POST /v1/iam/connector/update | PATCH /api/connectors/{id} | ✅ 对应 | - |
| 连接器工厂 | - | GET /api/connector-factories | ❌ 缺失 | 需要实现 |
| 测试连接器 | - | POST /api/connectors/{id}/test | ❌ 缺失 | 需要实现 |
| 授权URI | - | POST /api/connectors/{id}/authorization-uri | ❌ 缺失 | 需要实现 |

---

### 2.8 SSO连接器 (SSO Connector)

| 功能 | IAM 接口 | Logto 接口 | 映射状态 | 缺失说明 |
|------|----------|------------|----------|----------|
| 创建SSO连接器 | POST /v1/iam/ssoConnector/create | POST /api/sso-connectors | ✅ 对应 | - |
| 删除SSO连接器 | POST /v1/iam/ssoConnector/delete | DELETE /api/sso-connectors/{id} | ✅ 对应 | - |
| SSO连接器详情 | GET /v1/iam/ssoConnector/detail | GET /api/sso-connectors/{id} | ✅ 对应 | - |
| SSO连接器列表 | POST /v1/iam/ssoConnector/pageList | GET /api/sso-connectors | ✅ 对应 | - |
| 修改SSO连接器 | POST /v1/iam/ssoConnector/update | PATCH /api/sso-connectors/{id} | ✅ 对应 | - |
| SSO提供商 | - | GET /api/sso-connector-providers | ❌ 缺失 | 需要实现 |
| IdP发起的认证配置 | - | GET/PUT/DELETE /api/sso-connectors/{id}/idp-initiated-auth-config | ❌ 缺失 | 需要实现 |

---

### 2.9 日志 (Log)

| 功能 | IAM 接口 | Logto 接口 | 映射状态 | 缺失说明 |
|------|----------|------------|----------|----------|
| 日志详情 | GET /v1/iam/log/detail | GET /api/logs/{id} | ✅ 对应 | - |
| 日志列表 | POST /v1/iam/log/pageList | GET /api/logs | ✅ 对应 | - |

---

### 2.10 系统配置 (System Config)

| 功能 | IAM 接口 | Logto 接口 | 映射状态 | 缺失说明 |
|------|----------|------------|----------|----------|
| 创建配置 | POST /v1/iam/system/create | - | ✅ 特有 | - |
| 删除配置 | POST /v1/iam/system/delete | - | ✅ 特有 | - |
| 配置详情 | GET /v1/iam/system/detail | - | ✅ 特有 | - |
| 配置列表 | POST /v1/iam/system/pageList | - | ✅ 特有 | - |
| 修改配置 | POST /v1/iam/system/update | - | ✅ 特有 | - |
| 控制台配置 | - | GET/PATCH /api/configs/admin-console | ❌ 缺失 | 需要实现 |
| OIDC密钥 | - | GET/DELETE/POST /api/configs/oidc/* | ❌ 缺失 | 需要实现 |
| JWT自定义 | - | GET/PUT/DELETE /api/configs/jwt-customizer | ❌ 缺失 | 需要实现 |
| ID令牌配置 | - | GET/PUT /api/configs/id-token | ❌ 缺失 | 需要实现 |

---

## 三、功能缺失汇总

### 3.1 高优先级缺失（核心功能）

| 缺失功能 | 说明 | 对应Logto模块 |
|----------|------|---------------|
| MFA多因素认证 | TOTP、备用码、WebAuthn | Account, Experience |
| 用户自定义数据 | customData管理 | Admin User |
| 登录体验配置 | 品牌、语言、sign-in/up表单 | Sign-in Experience |
| 用户会话管理 | 查看和撤销用户会话 | Admin User |
| 应用角色管理 | 给应用分配角色 | Applications |
| 社交登录验证 | 验证码流程 | Experience |
| 密码强度验证 | 密码策略检查 | Sign-in Experience |

### 3.2 中优先级缺失（重要功能）

| 缺失功能 | 说明 | 对应Logto模块 |
|----------|------|---------------|
| 用户SSO身份 | SSO身份管理 | Admin User |
| 个人访问令牌 | PAT管理 | Admin User |
| 组织JIT配置 | 即时加入配置 | Organization |
| 组织邀请管理 | 邀请码/链接 | Organization Invitation |
| 角色-用户关系 | 查看角色下用户 | Role |
| 角色-应用关系 | 查看角色下应用 | Role |
| 连接器工厂 | 查看可用连接器类型 | Connector |
| SSO提供商列表 | 支持的SSO提供商 | SSO Connector |
| IdP认证配置 | SSO高级配置 | SSO Connector |
| 用户授权(Grant) | OAuth授权管理 | Admin User |

### 3.3 低优先级缺失（增强功能）

| 缺失功能 | 说明 | 对应Logto模块 |
|----------|------|---------------|
| Webhook钩子 | 事件通知 | Hooks |
| 自定义短语 | 国际化 | Custom Phrases |
| 自定义字段 | 用户自定义字段 | Custom Profile Fields |
| 邮件模板 | 邮件内容定制 | Email Template |
| 验证码提供商 | 第三方验证码 | Captcha Provider |
| 健康检查 | /status端点 | Status |
| 域名管理 | 自定义域名 | Domain |
| 统计数据 | Dashboard | Dashboard |

---

## 四、IAM特有功能

以下功能是IAM有而Logto没有的，是IAM的差异化优势：

| 功能 | 模块 | 说明 |
|------|------|------|
| 部门管理 | Department | 树形部门结构 |
| 菜单管理 | Menu | 前端菜单权限 |
| 角色菜单 | RoleMenu | 角色菜单关联 |
| 租户管理 | Tenant | 多租户隔离 |
| 用户部门关联 | UserDepartment | 用户多部门归属 |
| 用户登录日志 | UserLoginLog | 详细的登录记录 |

---

## 五、接口数量对比

| 模块类别 | IAM接口数 | Logto接口数 | 差距 |
|----------|-----------|-------------|------|
| 应用管理 | 5 | 20+ | -15 |
| 认证 | 7 | 30+ | -23 |
| 用户管理 | 19 | 50+ | -31 |
| 角色管理 | 5 | 13 | -8 |
| 组织管理 | 5 | 40+ | -35 |
| 资源管理 | 5 | 9 | -4 |
| 连接器 | 5 | 9 | -4 |
| SSO连接器 | 5 | 9 | -4 |
| 系统配置 | 5 | 14+ | -9 |
| **总计** | **~65** | **~200+** | **~135** |

---

## 六、完善建议

### 第一阶段：核心功能补全（建议优先）

1. **MFA多因素认证**
   - 实现TOTP绑定与验证
   - 实现备用码生成与验证
   - 实现WebAuthn/Passkey支持

2. **用户自定义数据**
   - 实现customData的CRUD

3. **登录体验配置**
   - 实现sign-in-exp的读写
   - 实现品牌色、语言配置
   - 实现sign-in/sign-up表单配置

4. **应用角色管理**
   - 实现应用角色分配API

### 第二阶段：重要功能补全

5. **会话与授权管理**
   - 实现用户会话查看与撤销
   - 实现用户授权(Grant)管理

6. **SSO功能增强**
   - 实现SSO提供商列表
   - 实现IdP发起认证配置

7. **组织功能增强**
   - 实现JIT配置
   - 实现组织邀请

### 第三阶段：增强功能

8. Webhook钩子
9. 自定义短语
10. 邮件模板定制
11. 统计数据

---

*文档生成时间：2026-05-07*