# Ark IAM Frontend

IAM 管理平台前端，基于 React 18 + Vite + Ant Design 5.x 的 pnpm monorepo。

## 技术栈

- 框架：React 18 + TypeScript（strict）
- 构建工具：Vite 5
- 路由：React Router 7
- 认证：react-oidc-context / oidc-client-ts（OIDC Authorization Code + PKCE）
- UI 组件库：Ant Design 5.x（自定义主题 + 中文语言包）
- HTTP 客户端：Axios（统一 `{code, msg, data}` 信封处理 + 401 兜底登出）

## 项目结构

```
frontend/
├── packages/
│   ├── tsconfig/          # 共享 TS 配置
│   ├── types/             # 领域类型（auth / organization / platform）
│   ├── api/               # Axios 实例 + 全量领域 API 资源
│   ├── auth/              # OIDC Provider、租户切换、登出、鉴权守卫
│   └── ui/                # 设计系统：主题、MainLayout、LoginPage、PageContainer 等
└── apps/
    ├── login-web/         # 登录页（:3000）— OP 登录 UI：凭证登录 / 多租户选择（非 OIDC Client）
    ├── platform-admin-web/# 平台管理控制台（:3001）— 用户/角色/菜单/部门/应用/OAuth客户端/租户/API Key/域名/系统配置/审计日志
    └── tenant-admin-web/  # 租户自服务控制台（:3003）— 组织/组织角色/组织用户/组织角色用户
```

依赖方向：`ui → auth → api → types`，业务 app 消费共享包。

## 构建与运行

```bash
# 安装依赖
pnpm install

# 开发模式（单应用）
pnpm dev              # 平台管理（:3001）
pnpm dev:login        # 登录门户（:3000）
pnpm dev:tenant       # 租户自服务（:3003）

# 同时启动三个应用
pnpm dev:all

# 构建全部 / 单应用
pnpm build
pnpm build:web

# 类型检查
pnpm typecheck

# 测试
pnpm test:all
```

Vite dev server 将 `/v1` 与 `/oidc` 代理到后端网关 `http://localhost:8100`。

## 主要功能

### 平台管理控制台（platform-admin-web）
- 用户管理：用户 CRUD、状态启停、重置密码、第三方身份绑定、登录日志、部门分配
- 角色管理：角色 CRUD、成员分配/移除
- 菜单管理：菜单树 + CRUD（目录/菜单/按钮）
- 权限域 / 资源：细粒度权限配置
- 部门管理：组织架构树 + CRUD
- 应用管理：应用 CRUD + 详情
- OAuth 客户端：客户端 CRUD、密钥轮换（创建即显示明文）
- 租户管理：租户 CRUD（类型/挂起）
- 租户应用：租户订阅关系管理
- API Key：创建（展示明文一次）、吊销、删除
- 域名管理：域名 CRUD + 验证状态
- 系统配置：键值配置管理
- 审计日志：只读查询 + 详情
- 个人中心：个人信息、修改密码、会话管理（撤销单条/全部）
- 租户切换：多租户上下文切换

### 租户自服务控制台（tenant-admin-web）
- 组织管理、组织角色、组织用户、组织角色用户

### 登录页（login-web）
- OIDC 凭证登录、多租户选择
- 定位：IAM OP 的登录 UI（非 OIDC Client/RP），凭证直接提交到 OP 内部端点 `/oidc/login`，不参与授权码流程；业务应用（platform-admin-web / tenant-admin-web）才是 OIDC Client

## 认证流程

1. 未登录访问任意页面 → `signinRedirect()` 跳转 `/oidc/authorize`
2. OP 校验 SSO 会话 → 凭证登录（login-web `/oidc/login`）或自动放行（`/oidc/sso-login`）
3. 多租户时选择租户（`/oidc/login/selectTenant`）→ 回跳 `/auth/callback` 拿 token
4. API 请求携带 `Bearer access_token`；401 / token 失效 → 清除本地 user 并回登录页
5. 全局登出：撤销 refresh token + SSO 会话 + back-channel 通知各已登录应用
