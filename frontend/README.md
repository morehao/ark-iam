# Ark IAM Frontend

IAM 管理平台前端项目，基于 React 18 + Vite + Ant Design 5.x。

## 技术栈

- 框架：React 18
- 构建工具：Vite
- 路由：React Router 6
- 状态管理：Zustand
- UI 组件库：Ant Design 5.x
- HTTP 客户端：Axios

## 项目结构

```
src/
├── api/              # API 请求模块
├── components/       # 公共组件
├── pages/            # 页面组件
│   ├── auth/         # 认证页面
│   ├── dashboard/    # 仪表盘
│   ├── user/         # 用户管理
│   ├── role/         # 角色管理
│   ├── department/   # 部门管理
│   └── application/  # 应用管理
├── stores/           # Zustand 状态管理
├── router/           # 路由配置
├── utils/            # 工具函数
├── App.tsx
└── main.tsx
```

## 构建与运行

```bash
# 安装依赖
npm install

# 开发模式运行
npm run dev

# 构建生产版本
npm run build

# 预览生产版本
npm run preview
```

## 访问地址

- 前端：http://localhost:3000
- 后端 API：http://localhost:8080

## 主要功能

- 用户管理：用户列表、用户详情、用户登录日志
- 角色管理：角色列表、角色详情
- 部门管理：部门树形列表、部门增删改
- 应用管理：应用列表、应用详情、角色分配

## 认证流程

1. 登录：`POST /v1/auth/login`
2. 获取用户信息：`GET /v1/auth/userinfo`
3. 刷新 Token：`POST /v1/auth/refreshToken`
4. 退出登录：`POST /v1/auth/logout`