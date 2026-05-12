# 认证、多租户与 SSO 统一设计

## 1. 背景

当前项目已经具备 IAM 前后端基础能力，后端已提供账号密码登录、租户选择、租户切换、我的租户、连接器授权与回调等接口，前端已有基础登录页和简单认证状态存储。

现阶段需要将以下能力收敛成一套完整且一致的登录体验：

- 账号密码登录
- 登录时多租户选择
- 登录后租户切换
- SSO 登录

本设计的目标不是重做整套认证体系，而是在尽量复用现有接口与代码结构的前提下，补齐前端状态模型、页面流转和 SSO 收口方式，使账号密码与 SSO 共享同一条多租户登录链路。

## 2. 目标与范围

### 2.1 目标

- 统一账号密码登录与 SSO 登录后的租户进入流程
- 当人员关联多个租户时，显式要求用户先选择租户，再进入系统
- 支持登录后从系统右上角切换租户
- 让前端认证状态明确区分“人已认证”和“已进入租户”两个阶段
- 最大化复用现有后端接口，避免大规模推翻重做

### 2.2 不在本次范围

- MFA、多因素认证
- 验证码登录或找回密码
- 社交登录泛化能力
- 基于邮箱域名自动匹配 SSO 连接器
- 多 SSO 提供商选择器
- 登录体验装修、品牌配置、多语言

## 3. 现状总结

### 3.1 后端现状

后端已存在以下接口与能力：

- `POST /v1/iam/auth/login`：账号密码登录，返回 `personToken + tenants`
- `GET /v1/iam/auth/myTenants`：按人员查询可访问租户
- `POST /v1/iam/auth/selectTenant`：基于 `personToken` 选择租户并签发租户令牌
- `POST /v1/iam/auth/switchTenant`：已登录租户态下切换租户
- `POST /v1/iam/auth/refreshToken`：刷新租户令牌
- `POST /v1/iam/auth/logout`：退出当前会话
- `GET /v1/iam/auth/userinfo`：获取当前租户用户信息
- `POST /v1/iam/connector/:connectorId/authorize`：发起连接器授权
- `GET /v1/iam/connector/callback`：处理连接器回调，返回 `personToken + tenants`

这说明后端核心模型已经是“人”和“租户用户”分离，只是前端尚未把这套模型完整消费起来。

### 3.2 前端现状

前端当前问题主要有：

- 登录成功后直接把返回结果写成单一 `accessToken` 语义
- 多租户场景默认选第一个租户，没有显式选择页
- 路由守卫只判断是否存在一个 token，无法区分登录阶段
- 页面头部没有租户切换入口
- 还没有 SSO 登录入口与回调承接页
- 请求拦截器只处理单一 token，无法区分 `personToken` 与 `tenantToken`

## 4. 核心设计决策

### 4.1 采用双阶段认证模型

本次采用“双阶段认证”作为统一模型：

1. 人员认证阶段：账号密码或 SSO 成功后，获取 `personToken`
2. 租户进入阶段：用户确认进入某个租户后，获取 `tenantToken`

该模型适用于账号密码与 SSO 两条链路，避免为 SSO 单独设计第二套流程。

### 4.2 多租户进入规则

- 仅有 1 个租户：前端自动选择该租户并进入系统
- 多于 1 个租户：前端跳转租户选择页，由用户显式选择
- 没有租户：提示“当前账号未加入任何租户”，不进入系统

### 4.3 登录后租户切换规则

- 在系统右上角展示当前租户
- 点击下拉可查看并切换到其他租户
- 切换后更新当前租户令牌、当前租户信息和页面数据
- 若切换失败，保留原租户上下文，不破坏当前会话

### 4.4 SSO 规则

- 本期只支持后台已配置好的企业 SSO 连接器
- 前端只负责提供入口与承接回调
- SSO 成功后与账号密码登录完全复用同一套租户选择流程

## 5. 认证架构

### 5.1 令牌职责

#### `personToken`

表示“这个人已经通过身份认证，但尚未进入具体租户”。只用于以下场景：

- `GET /auth/myTenants`
- `POST /auth/selectTenant`
- SSO 回调后的短流程承接

`personToken` 不用于普通业务接口访问。

#### `tenantToken`

表示“该人员已以某个租户用户身份进入系统”。用于：

- `GET /auth/userinfo`
- `POST /auth/switchTenant`
- 所有业务接口，如用户、角色、部门、应用等模块接口

#### `refreshToken`

与当前 `tenantToken` 配套，专门用于刷新租户态会话。它不承担人员认证阶段的职责。

### 5.2 前端认证状态

前端将认证状态拆为三段：

- `anonymous`：未登录
- `authenticated_person`：已拿到 `personToken`，但尚未选租户
- `authenticated_tenant`：已拿到 `tenantToken`，已进入系统

这样可避免当前仅凭一个 token 判断全部登录状态的粗放做法。

## 6. 页面与路由设计

### 6.1 页面清单

- `/login`：账号密码登录页，同时提供企业 SSO 登录入口
- `/register`：注册页，维持现有能力
- `/select-tenant`：租户选择页，仅服务“已认证但未入租户”状态
- `/auth/callback`：前端 SSO 回调承接页
- `/` 及其子路由：业务系统页，需要有效 `tenantToken`

### 6.2 页面流转

#### 账号密码登录

1. 用户在 `/login` 输入账号和密码
2. 前端调用 `POST /auth/login`
3. 后端返回 `personToken + tenants`
4. 前端根据租户数量分流：
   - 1 个租户：自动调用 `POST /auth/selectTenant`
   - 多个租户：跳转 `/select-tenant`
   - 0 个租户：提示不可进入系统
5. 选定租户后获取 `tenantToken`，进入首页

#### SSO 登录

1. 用户在 `/login` 点击企业 SSO 登录
2. 前端调用连接器授权接口，跳转第三方身份提供商
3. 第三方认证成功后回到后端回调
4. 后端完成身份解析，生成 `personToken + tenants`
5. 前端回调页再按与账号密码登录相同的规则分流

#### 登录后切租户

1. 用户在右上角点击当前租户下拉
2. 前端调用 `GET /auth/myTenants` 或使用已缓存租户列表展示可选租户
3. 用户选定目标租户
4. 前端调用 `POST /auth/switchTenant`
5. 成功后更新 `tenantToken`、`currentTenant`、`userInfo`
6. 刷新当前页数据；若新租户下当前页无权限，则回首页或展示无权限提示

## 7. 前端状态管理设计

### 7.1 `authStore` 调整

建议将现有 `authStore` 调整为以下核心字段：

- `personToken: string | null`
- `tenantToken: string | null`
- `refreshToken: string | null`
- `authStage: 'anonymous' | 'authenticated_person' | 'authenticated_tenant'`
- `tenants: TenantOption[]`
- `currentTenant: TenantOption | null`
- `personInfo: PersonInfo | null`
- `userInfo: TenantUserInfo | null`

其中：

- `personToken` 用于租户选择阶段
- `tenantToken` 用于主系统业务访问
- `tenants` 和 `currentTenant` 共同支持首次选租户和后续切换租户

### 7.2 持久化策略

建议持久化以下数据：

- `personToken`
- `tenantToken`
- `refreshToken`
- `authStage`
- `tenants`
- `currentTenant`

这样用户刷新页面后仍能恢复到正确阶段：

- 如果处于选租户阶段，可回到 `/select-tenant`
- 如果处于业务阶段，可恢复主系统

### 7.3 请求头策略

- 业务接口默认带 `tenantToken`
- 只有租户选择链路才显式使用 `personToken`
- 不允许在业务接口里混用 `personToken`

请求封装应显式区分调用上下文，而不是继续默认读取单一 `accessToken`。

## 8. 接口契约与职责边界

### 8.1 复用现有接口

本次优先复用现有后端接口，不重做认证域：

- `POST /v1/iam/auth/login`
- `GET /v1/iam/auth/myTenants`
- `POST /v1/iam/auth/selectTenant`
- `POST /v1/iam/auth/switchTenant`
- `POST /v1/iam/auth/refreshToken`
- `POST /v1/iam/auth/logout`
- `GET /v1/iam/auth/userinfo`
- `POST /v1/iam/connector/:connectorId/authorize`
- `GET /v1/iam/connector/callback`

### 8.2 关键请求语义

#### 登录

`POST /v1/iam/auth/login`

请求：

```json
{
  "identifier": "string",
  "password": "string"
}
```

响应：

```json
{
  "personToken": {
    "accessToken": "string",
    "refreshToken": "string",
    "expiresIn": 86400,
    "tokenType": "Bearer"
  },
  "tenants": [
    {
      "tenantID": 1,
      "name": "租户A",
      "tag": "tenant-a",
      "userID": 10,
      "isOwner": 1
    }
  ]
}
```

#### 选择租户

`POST /v1/iam/auth/selectTenant`

请求：

```json
{
  "personToken": "string",
  "tenantID": 1
}
```

响应：

```json
{
  "tokenInfo": {
    "accessToken": "string",
    "refreshToken": "string",
    "expiresIn": 604800,
    "tokenType": "Bearer"
  }
}
```

这里的 `tokenInfo` 在前端语义上就是 `tenantToken` 对应的会话结果。

#### 切换租户

`POST /v1/iam/auth/switchTenant`

请求：

```json
{
  "tenantID": 2
}
```

请求头使用当前 `tenantToken`，响应继续返回新的租户态 `tokenInfo`。

### 8.3 前后端职责边界

前端负责：

- 维护认证状态机
- 处理租户分流
- 在正确阶段调用正确接口
- 提供租户切换交互
- 承接 SSO 回流页面

后端负责：

- 账号密码校验
- SSO 第三方交互和身份映射
- 生成 `personToken` 与 `tenantToken`
- 约束不同 token 的可访问边界
- 根据租户生成用户上下文

## 9. SSO 收口设计

### 9.1 当前问题

后端当前已有 `GET /v1/iam/connector/callback`，但浏览器型登录流程要顺畅落到前端页面，仍需要一个清晰、固定、可消费的回流方式。

### 9.2 推荐方案

推荐采用“后端回调成功后重定向到前端回调页，前端再完成结果交换”的模式。

推荐流程：

1. 前端调用授权接口并跳转第三方
2. 第三方回到后端 `connector/callback`
3. 后端完成身份确认后，不直接把 token 暴露在 URL 上
4. 后端重定向到前端 `/auth/callback`，并附带一次性短期 code 或状态标识
5. 前端在 `/auth/callback` 通过专用交换动作拿到 `personToken + tenants`
6. 前端复用账号密码登录后的分流逻辑

### 9.3 边界要求

- 不在 URL query 中直接暴露 token
- 一次性 code 必须短时有效且仅可消费一次
- 失败时应能明确回到 `/login` 并展示错误提示

### 9.4 最小后端补强

如果当前 `connector/callback` 只能直接返回 JSON，则建议补一个前端友好的收口能力。可接受的形式只有一种，避免混用：

- 为当前回调流程增加重定向和一次性 code 交换能力

本设计不要求重写整个连接器体系，只要求补齐浏览器登录闭环。

## 10. 路由守卫与权限边界

### 10.1 路由守卫规则

- `/login`、`/register`、`/auth/callback`：公开访问
- `/select-tenant`：要求存在有效 `personToken`
- 所有业务页：要求存在有效 `tenantToken`

### 10.2 Token 边界

- `personToken` 访问业务接口时必须被拒绝
- `tenantToken` 才是系统内业务权限校验的基础
- 切租户后，必须以新的 `tenantToken` 重新请求用户信息和业务数据

## 11. 异常处理

### 11.1 登录链路

- 密码错误：停留登录页，按后端错误码提示
- 用户被停用：停留登录页并提示
- 无租户：提示账号未加入任何租户
- 自动选租户失败：不进入首页，回退到登录链路或租户选择页

### 11.2 租户链路

- 选择租户失败：保留 `personToken`，允许用户重新选择
- 切换租户失败：保留原 `tenantToken` 和原页面状态
- 新租户无当前页权限：回到首页或展示无权限态

### 11.3 SSO 链路

- 第三方认证失败：回到 `/login`
- 回调状态失效：回到 `/login` 并提示登录超时
- 身份映射失败：提示当前 SSO 账号未绑定系统身份或不可自动建号

### 11.4 Token 过期

- `tenantToken` 过期时优先尝试刷新
- 刷新失败时清理租户态并跳回登录页
- 不自动用 `personToken` 恢复业务态，避免状态错乱

## 12. 测试设计

### 12.1 后端测试重点

- 账号密码登录返回多租户列表
- 单租户 `selectTenant` 成功签发租户 token
- 多租户 `selectTenant` 正确校验人员归属
- `switchTenant` 返回新租户 token
- `personToken` 误访问业务接口被拒绝
- SSO callback 返回 `personToken + tenants`
- SSO 一次性 code 或回调状态只能消费一次

### 12.2 前端测试重点

- 登录页单租户自动进入系统
- 登录页多租户跳转选择页
- 选择租户成功后进入首页
- 切换租户成功后更新头部状态与页面数据
- 切换租户失败时保留当前上下文
- SSO 回调成功后正确分流
- SSO 回调失败后回到登录页
- 页面刷新后按 `authStage` 正确恢复路由

### 12.3 集成验收用例

- 账号密码 + 单租户
- 账号密码 + 多租户
- SSO + 单租户
- SSO + 多租户
- 已登录系统内租户切换

## 13. 实施建议

建议实现顺序如下：

1. 重构前端 `authStore`，明确双阶段 token 语义
2. 增加 `/select-tenant` 页面和相应路由守卫
3. 调整登录页逻辑，接入自动选租户与显式选租户分流
4. 在主布局头部增加当前租户展示与切换入口
5. 增加前端 SSO 入口与 `/auth/callback` 页
6. 如有必要，补后端 SSO 回调收口能力
7. 补前后端测试

## 14. 方案取舍说明

本设计明确不采用以下方案：

- 登录后默认直接进入第一个租户：会掩盖多租户用户的明确选择，且与你确认的业务规则不一致
- 只保留一个 token 混用所有阶段：会让前端状态和接口语义继续混乱
- SSO 单独走另一套租户逻辑：会造成维护成本上升，后续扩展也更难统一

因此，双阶段认证流是本次最小且正确的方案。
