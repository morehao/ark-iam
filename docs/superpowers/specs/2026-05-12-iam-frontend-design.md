# IAM 前端项目设计文档

## 1. 项目概述

基于 React 18 + Vite 构建的 IAM（身份认证和管理）前端管理平台，提供完整的用户、角色、部门、应用管理界面。

## 2. 技术选型

| 类别 | 选择 | 说明 |
|------|------|------|
| 框架 | React 18 + Vite | 快速开发体验 |
| UI 组件库 | Ant Design 5.x | 企业级组件 |
| 状态管理 | Zustand | 轻量级 |
| 路由 | React Router 6 | React 官方推荐 |
| HTTP 客户端 | Axios | 成熟稳定 |
| CSS 方案 | Ant Design 默认主题 + CSS 变量 | 快速定制 |

## 3. 项目结构

```
frontend/
├── public/                  # 静态资源
├── src/
│   ├── api/                 # API 请求模块（按模块划分）
│   │   ├── auth.ts          # 认证相关 API
│   │   ├── user.ts          # 用户相关 API
│   │   ├── role.ts          # 角色相关 API
│   │   ├── department.ts    # 部门相关 API
│   │   └── application.ts   # 应用相关 API
│   ├── assets/              # 资源文件
│   ├── components/          # 公共组件
│   ├── layouts/             # 布局组件
│   │   └── MainLayout.tsx   # 主布局（侧边栏 + 头部）
│   ├── pages/               # 页面（按业务模块划分）
│   │   ├── auth/            # 认证页面
│   │   │   ├── Login.tsx
│   │   │   ├── Register.tsx
│   │   │   └── Logout.tsx
│   │   ├── user/             # 用户管理页面
│   │   ├── role/             # 角色管理页面
│   │   ├── department/        # 部门管理页面
│   │   ├── application/      # 应用管理页面
│   │   └── dashboard/        # 仪表盘
│   ├── stores/              # Zustand stores
│   │   ├── authStore.ts     # 认证状态（token、用户信息）
│   │   └── appStore.ts      # 应用全局状态
│   ├── router/              # 路由配置
│   │   └── index.tsx
│   ├── utils/               # 工具函数
│   │   └── request.ts       # Axios 实例封装
│   ├── App.tsx
│   └── main.tsx
├── index.html
├── vite.config.ts
├── package.json
└── README.md
```

## 4. 核心模块设计

### 4.1 认证模块 (auth)

**登录接口**: `POST /v1/iam/auth/login`
- 请求: `{ identifier: string, password: string }`
- 响应: `{ personToken: TokenInfo, tenants: TenantOption[] }`

**注册接口**: `POST /v1/iam/auth/register`
- 请求: `{ username, password, name, primaryEmail?, primaryPhone?, tenantID }`
- 响应: `{ userID }`

**退出登录**: `POST /v1/iam/auth/logout`
- 请求: `{ refreshToken: string }`
- 响应: `string`

**刷新令牌**: `POST /v1/iam/auth/refreshToken`
- 请求: `{ refreshToken: string }`
- 响应: `{ accessToken, refreshToken, expiresIn, tokenType }`

**获取用户信息**: `GET /v1/iam/auth/userinfo`
- 响应: `{ personInfo: PersonInfo, userInfo: TenantUserInfo }`

### 4.2 用户管理模块 (user)

- 用户列表（分页）
- 用户详情
- 用户登录日志

### 4.3 角色管理模块 (role)

- 角色列表
- 角色详情/编辑

### 4.4 部门管理模块 (department)

- 部门树形列表
- 部门增删改

### 4.5 应用管理模块 (application)

- 应用列表
- 应用详情
- 角色分配

## 5. 路由设计

| 路径 | 页面 | 权限 |
|------|------|------|
| /login | 登录页 | 公开 |
| /register | 注册页 | 公开 |
| / | 首页/仪表盘 | 需登录 |
| /user | 用户列表 | 需登录 |
| /role | 角色列表 | 需登录 |
| /department | 部门管理 | 需登录 |
| /application | 应用列表 | 需登录 |

## 6. 状态管理

**authStore**: 存储认证相关信息
- accessToken / refreshToken
- 用户信息
- 租户信息

**appStore**: 存储应用全局状态
- 侧边栏折叠状态
- 主题设置等

## 7. API 请求封装

使用 Axios 封装请求实例，统一处理：
- 请求拦截器：添加 Authorization header
- 响应拦截器：统一处理错误、刷新 Token
- 错误处理：根据后端错误码返回统一错误信息