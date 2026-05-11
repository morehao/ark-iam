# IAM 前端项目实施计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 搭建完整的 React 前端项目，实现 IAM 管理界面

**Architecture:** 基于 React 18 + Vite，使用 Ant Design 5.x 作为 UI 组件库，Zustand 管理状态，React Router 6 管理路由，Axios 处理 HTTP 请求。

**Tech Stack:** React 18, Vite, Ant Design 5.x, Zustand, React Router 6, Axios, TypeScript

---

## 文件结构

```
frontend/
├── public/
├── src/
│   ├── api/
│   │   ├── auth.ts
│   │   ├── user.ts
│   │   ├── role.ts
│   │   ├── department.ts
│   │   └── application.ts
│   ├── assets/
│   ├── components/
│   │   └── MainLayout.tsx
│   ├── pages/
│   │   ├── auth/
│   │   │   ├── Login.tsx
│   │   │   └── Register.tsx
│   │   ├── dashboard/
│   │   │   └── index.tsx
│   │   ├── user/
│   │   │   ├── index.tsx
│   │   │   └── Detail.tsx
│   │   ├── role/
│   │   │   └── index.tsx
│   │   ├── department/
│   │   │   └── index.tsx
│   │   └── application/
│   │       └── index.tsx
│   ├── stores/
│   │   ├── authStore.ts
│   │   └── appStore.ts
│   ├── router/
│   │   └── index.tsx
│   ├── utils/
│   │   └── request.ts
│   ├── App.tsx
│   └── main.tsx
├── index.html
├── vite.config.ts
├── tsconfig.json
├── tsconfig.node.json
├── package.json
└── README.md
```

---

## Task 1: 项目脚手架和配置文件

**Files:**
- Create: `frontend/package.json`
- Create: `frontend/vite.config.ts`
- Create: `frontend/tsconfig.json`
- Create: `frontend/tsconfig.node.json`
- Create: `frontend/index.html`

- [ ] **Step 1: 创建 package.json**

```json
{
  "name": "ark-iam-frontend",
  "private": true,
  "version": "1.0.0",
  "type": "module",
  "scripts": {
    "dev": "vite",
    "build": "tsc -b && vite build",
    "preview": "vite preview"
  },
  "dependencies": {
    "react": "^18.3.1",
    "react-dom": "^18.3.1",
    "react-router": "^6.28.0",
    "antd": "^5.22.0",
    "axios": "^1.7.7",
    "zustand": "^5.0.1",
    "@ant-design/icons": "^5.4.0"
  },
  "devDependencies": {
    "@types/react": "^18.3.12",
    "@types/react-dom": "^18.3.1",
    "@vitejs/plugin-react": "^4.3.3",
    "typescript": "^5.6.3",
    "vite": "^5.4.10"
  }
}
```

- [ ] **Step 2: 创建 vite.config.ts**

```typescript
import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

export default defineConfig({
  plugins: [react()],
  server: {
    port: 3000,
    proxy: {
      '/v1': {
        target: 'http://localhost:8080',
        changeOrigin: true,
      },
    },
  },
})
```

- [ ] **Step 3: 创建 tsconfig.json**

```json
{
  "compilerOptions": {
    "target": "ES2020",
    "useDefineForClassFields": true,
    "lib": ["ES2020", "DOM", "DOM.Iterable"],
    "module": "ESNext",
    "skipLibCheck": true,
    "moduleResolution": "bundler",
    "allowImportingTsExtensions": true,
    "isolatedModules": true,
    "moduleDetection": "force",
    "noEmit": true,
    "jsx": "react-jsy",
    "strict": true,
    "noUnusedLocals": true,
    "noUnusedParameters": true,
    "noFallthroughCasesInSwitch": true,
    "noUncheckedSideEffectImports": true
  },
  "include": ["src"]
}
```

- [ ] **Step 4: 创建 tsconfig.node.json**

```json
{
  "compilerOptions": {
    "target": "ES2022",
    "lib": ["ES2023"],
    "module": "ESNext",
    "skipLibCheck": true,
    "moduleResolution": "bundler",
    "allowImportingTsExtensions": true,
    "isolatedModules": true,
    "moduleDetection": "force",
    "noEmit": true,
    "strict": true,
    "noUnusedLocals": true,
    "noUnusedParameters": true,
    "noFallthroughCasesInSwitch": true,
    "noUncheckedSideEffectImports": true
  },
  "include": ["vite.config.ts"]
}
```

- [ ] **Step 5: 创建 index.html**

```html
<!doctype html>
<html lang="zh-CN">
  <head>
    <meta charset="UTF-8" />
    <link rel="icon" type="image/svg+xml" href="/vite.svg" />
    <meta name="viewport" content="width=device-width, initial-scale=1.0" />
    <title>IAM 管理平台</title>
  </head>
  <body>
    <div id="root"></div>
    <script type="module" src="/src/main.tsx"></script>
  </body>
</html>
```

---

## Task 2: 搭建项目基础文件

**Files:**
- Create: `frontend/src/main.tsx`
- Create: `frontend/src/App.tsx`
- Create: `frontend/src/vite-env.d.ts`

- [ ] **Step 1: 创建 src/main.tsx**

```typescript
import React from 'react'
import ReactDOM from 'react-dom/client'
import { BrowserRouter } from 'react-router'
import App from './App'

ReactDOM.createRoot(document.getElementById('root')!).render(
  <React.StrictMode>
    <BrowserRouter>
      <App />
    </BrowserRouter>
  </React.StrictMode>,
)
```

- [ ] **Step 2: 创建 src/App.tsx**

```typescript
import { Routes, Route, Navigate } from 'react-router'
import MainLayout from './components/MainLayout'
import Login from './pages/auth/Login'
import Register from './pages/auth/Register'
import Dashboard from './pages/dashboard'
import UserList from './pages/user'
import RoleList from './pages/role'
import DepartmentList from './pages/department'
import ApplicationList from './pages/application'
import { useAuthStore } from './stores/authStore'

function App() {
  const { accessToken } = useAuthStore()

  return (
    <Routes>
      <Route path="/login" element={<Login />} />
      <Route path="/register" element={<Register />} />
      <Route
        path="/"
        element={
          accessToken ? <MainLayout /> : <Navigate to="/login" replace />
        }
      >
        <Route index element={<Dashboard />} />
        <Route path="user" element={<UserList />} />
        <Route path="role" element={<RoleList />} />
        <Route path="department" element={<DepartmentList />} />
        <Route path="application" element={<ApplicationList />} />
      </Route>
    </Routes>
  )
}

export default App
```

- [ ] **Step 3: 创建 src/vite-env.d.ts**

```typescript
/// <reference types="vite/client" />
```

---

## Task 3: API 请求模块

**Files:**
- Create: `frontend/src/utils/request.ts`
- Create: `frontend/src/api/auth.ts`

- [ ] **Step 1: 创建 utils/request.ts**

```typescript
import axios, { AxiosError, InternalAxiosRequestConfig } from 'axios'
import { message } from 'antd'

const request = axios.create({
  baseURL: '/v1/iam',
  timeout: 30000,
})

request.interceptors.request.use(
  (config: InternalAxiosRequestConfig) => {
    const token = localStorage.getItem('accessToken')
    if (token && config.headers) {
      config.headers.Authorization = `Bearer ${token}`
    }
    return config
  },
  (error: AxiosError) => {
    return Promise.reject(error)
  }
)

request.interceptors.response.use(
  (response) => response.data,
  async (error: AxiosError) => {
    const status = error.response?.status
    const data = error.response?.data as any

    if (status === 401) {
      localStorage.removeItem('accessToken')
      localStorage.removeItem('refreshToken')
      window.location.href = '/login'
      return Promise.reject(error)
    }

    if (data?.msg) {
      message.error(data.msg)
    } else {
      message.error('请求失败')
    }

    return Promise.reject(error)
  }
)

export default request
```

- [ ] **Step 2: 创建 api/auth.ts**

```typescript
import request from '../utils/request'

export interface LoginReq {
  identifier: string
  password: string
}

export interface LoginResp {
  personToken: {
    accessToken: string
    refreshToken: string
    expiresIn: number
    tokenType: string
  }
  tenants: Array<{
    tenantID: number
    name: string
    tag: string
    userID: number
    isOwner: number
  }>
}

export interface RegisterReq {
  username: string
  password: string
  name: string
  primaryEmail?: string
  primaryPhone?: string
  tenantID: number
}

export interface RegisterResp {
  userID: number
}

export interface UserinfoResp {
  personInfo: {
    personID: number
    name: string
    avatar: string
  }
  userInfo: {
    userID: number
    name: string
    tenantID: number
    isOwner: number
  }
}

export const login = (data: LoginReq) => {
  return request.post<any, any>('/auth/login', data)
}

export const register = (data: RegisterReq) => {
  return request.post<any, any>('/auth/register', data)
}

export const logout = (refreshToken: string) => {
  return request.post<any, any>('/auth/logout', { refreshToken })
}

export const refreshToken = (refreshToken: string) => {
  return request.post<any, any>('/auth/refreshToken', { refreshToken })
}

export const getUserinfo = () => {
  return request.get<any, any>('/auth/userinfo')
}
```

---

## Task 4: Zustand Store

**Files:**
- Create: `frontend/src/stores/authStore.ts`
- Create: `frontend/src/stores/appStore.ts`

- [ ] **Step 1: 创建 stores/authStore.ts**

```typescript
import { create } from 'zustand'
import { persist } from 'zustand/middleware'

interface AuthState {
  accessToken: string | null
  refreshToken: string | null
  userInfo: {
    userID: number
    name: string
    tenantID: number
    isOwner: number
  } | null
  personInfo: {
    personID: number
    name: string
    avatar: string
  } | null
  currentTenant: {
    tenantID: number
    name: string
    tag: string
    userID: number
    isOwner: number
  } | null
  tenants: Array<{
    tenantID: number
    name: string
    tag: string
    userID: number
    isOwner: number
  }>

  setTokens: (accessToken: string, refreshToken: string) => void
  setUserinfo: (userInfo: AuthState['userInfo'], personInfo: AuthState['personInfo']) => void
  setTenants: (tenants: AuthState['tenants']) => void
  setCurrentTenant: (tenant: AuthState['currentTenant']) => void
  logout: () => void
}

export const useAuthStore = create<AuthState>()(
  persist(
    (set) => ({
      accessToken: null,
      refreshToken: null,
      userInfo: null,
      personInfo: null,
      currentTenant: null,
      tenants: [],

      setTokens: (accessToken, refreshToken) => {
        localStorage.setItem('accessToken', accessToken)
        localStorage.setItem('refreshToken', refreshToken)
        set({ accessToken, refreshToken })
      },

      setUserinfo: (userInfo, personInfo) => set({ userInfo, personInfo }),

      setTenants: (tenants) => set({ tenants }),

      setCurrentTenant: (tenant) => set({ currentTenant: tenant }),

      logout: () => {
        localStorage.removeItem('accessToken')
        localStorage.removeItem('refreshToken')
        set({
          accessToken: null,
          refreshToken: null,
          userInfo: null,
          personInfo: null,
          currentTenant: null,
          tenants: [],
        })
      },
    }),
    {
      name: 'auth-storage',
      partialize: (state) => ({
        accessToken: state.accessToken,
        refreshToken: state.refreshToken,
        userInfo: state.userInfo,
        personInfo: state.personInfo,
        currentTenant: state.currentTenant,
        tenants: state.tenants,
      }),
    }
  )
)
```

- [ ] **Step 2: 创建 stores/appStore.ts**

```typescript
import { create } from 'zustand'

interface AppState {
  collapsed: boolean
  toggleCollapsed: () => void
}

export const useAppStore = create<AppState>((set) => ({
  collapsed: false,
  toggleCollapsed: () => set((state) => ({ collapsed: !state.collapsed })),
}))
```

---

## Task 5: MainLayout 布局组件

**Files:**
- Create: `frontend/src/components/MainLayout.tsx`

- [ ] **Step 1: 创建 components/MainLayout.tsx**

```typescript
import { useState } from 'react'
import { Layout, Menu, Avatar, Dropdown, Button } from 'antd'
import {
  DashboardOutlined,
  UserOutlined,
  TeamOutlined,
  AppstoreOutlined,
  MenuFoldOutlined,
  MenuUnfoldOutlined,
  LogoutOutlined,
} from '@ant-design/icons'
import { Outlet, useNavigate, useLocation } from 'react-router'
import { useAuthStore } from '../stores/authStore'

const { Header, Sider, Content } = Layout

const MainLayout = () => {
  const [collapsed, setCollapsed] = useState(false)
  const navigate = useNavigate()
  const location = useLocation()
  const { userInfo, personInfo, logout } = useAuthStore()

  const menuItems = [
    { key: '/', icon: <DashboardOutlined />, label: '仪表盘' },
    { key: '/user', icon: <UserOutlined />, label: '用户管理' },
    { key: '/role', icon: <TeamOutlined />, label: '角色管理' },
    { key: '/department', icon: <TeamOutlined />, label: '部门管理' },
    { key: '/application', icon: <AppstoreOutlined />, label: '应用管理' },
  ]

  const handleLogout = () => {
    logout()
    navigate('/login')
  }

  const userMenuItems = [
    { key: 'logout', icon: <LogoutOutlined />, label: '退出登录', onClick: handleLogout },
  ]

  return (
    <Layout style={{ minHeight: '100vh' }}>
      <Sider trigger={null} collapsible collapsed={collapsed}>
        <div style={{
          height: 64,
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'center',
          color: '#fff',
          fontSize: collapsed ? 14 : 18,
          fontWeight: 'bold',
        }}>
          {collapsed ? 'IAM' : 'IAM 管理平台'}
        </div>
        <Menu
          theme="dark"
          mode="inline"
          selectedKeys={[location.pathname]}
          items={menuItems}
          onClick={({ key }) => navigate(key)}
        />
      </Sider>
      <Layout>
        <Header style={{ padding: '0 16px', background: '#fff', display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
          <Button
            type="text"
            icon={collapsed ? <MenuUnfoldOutlined /> : <MenuFoldOutlined />}
            onClick={() => setCollapsed(!collapsed)}
          />
          <Dropdown menu={{ items: userMenuItems }} placement="bottomRight">
            <Avatar style={{ cursor: 'pointer' }} icon={<UserOutlined />} />
          </Dropdown>
        </Header>
        <Content style={{ margin: 24, padding: 24, background: '#fff' }}>
          <Outlet />
        </Content>
      </Layout>
    </Layout>
  )
}

export default MainLayout
```

---

## Task 6: 登录页面

**Files:**
- Create: `frontend/src/pages/auth/Login.tsx`

- [ ] **Step 1: 创建 pages/auth/Login.tsx**

```typescript
import { useState } from 'react'
import { Form, Input, Button, Card, message } from 'antd'
import { UserOutlined, LockOutlined } from '@ant-design/icons'
import { useNavigate } from 'react-router'
import { login as loginApi } from '../../api/auth'
import { useAuthStore } from '../../stores/authStore'

const Login = () => {
  const [loading, setLoading] = useState(false)
  const navigate = useNavigate()
  const { setTokens, setTenants, setCurrentTenant } = useAuthStore()

  const onFinish = async (values: { identifier: string; password: string }) => {
    setLoading(true)
    try {
      const resp = await loginApi(values)
      const { personToken, tenants } = resp.data

      setTokens(personToken.accessToken, personToken.refreshToken)
      setTenants(tenants)

      if (tenants && tenants.length > 0) {
        setCurrentTenant(tenants[0])
      }

      message.success('登录成功')
      navigate('/')
    } catch (error) {
      console.error('登录失败:', error)
    } finally {
      setLoading(false)
    }
  }

  return (
    <div style={{
      height: '100vh',
      display: 'flex',
      alignItems: 'center',
      justifyContent: 'center',
      background: '#f0f2f5',
    }}>
      <Card title="IAM 管理平台" style={{ width: 400 }}>
        <Form
          name="login"
          onFinish={onFinish}
          autoComplete="off"
        >
          <Form.Item
            name="identifier"
            rules={[{ required: true, message: '请输入用户名/邮箱/手机号' }]}
          >
            <Input prefix={<UserOutlined />} placeholder="用户名/邮箱/手机号" />
          </Form.Item>

          <Form.Item
            name="password"
            rules={[{ required: true, message: '请输入密码' }]}
          >
            <Input.Password prefix={<LockOutlined />} placeholder="密码" />
          </Form.Item>

          <Form.Item>
            <Button type="primary" htmlType="submit" loading={loading} block>
              登录
            </Button>
          </Form.Item>

          <div style={{ textAlign: 'center' }}>
            <a onClick={() => navigate('/register')}>还没有账号？立即注册</a>
          </div>
        </Form>
      </Card>
    </div>
  )
}

export default Login
```

---

## Task 7: 注册页面

**Files:**
- Create: `frontend/src/pages/auth/Register.tsx`

- [ ] **Step 1: 创建 pages/auth/Register.tsx**

```typescript
import { useState } from 'react'
import { Form, Input, Button, Card, message } from 'antd'
import { UserOutlined, LockOutlined, MailOutlined, PhoneOutlined } from '@ant-design/icons'
import { useNavigate } from 'react-router'
import { register as registerApi } from '../../api/auth'

const Register = () => {
  const [loading, setLoading] = useState(false)
  const navigate = useNavigate()

  const onFinish = async (values: {
    username: string
    password: string
    name: string
    primaryEmail?: string
    primaryPhone?: string
    tenantID: number
  }) => {
    setLoading(true)
    try {
      await registerApi(values)
      message.success('注册成功，请登录')
      navigate('/login')
    } catch (error) {
      console.error('注册失败:', error)
    } finally {
      setLoading(false)
    }
  }

  return (
    <div style={{
      height: '100vh',
      display: 'flex',
      alignItems: 'center',
      justifyContent: 'center',
      background: '#f0f2f5',
    }}>
      <Card title="用户注册" style={{ width: 400 }}>
        <Form
          name="register"
          onFinish={onFinish}
          autoComplete="off"
        >
          <Form.Item
            name="username"
            rules={[{ required: true, message: '请输入用户名' }]}
          >
            <Input prefix={<UserOutlined />} placeholder="用户名" />
          </Form.Item>

          <Form.Item
            name="password"
            rules={[
              { required: true, message: '请输入密码' },
              { min: 6, message: '密码至少6位' },
            ]}
          >
            <Input.Password prefix={<LockOutlined />} placeholder="密码" />
          </Form.Item>

          <Form.Item
            name="name"
            rules={[{ required: true, message: '请输入姓名' }]}
          >
            <Input placeholder="姓名" />
          </Form.Item>

          <Form.Item
            name="primaryEmail"
          >
            <Input prefix={<MailOutlined />} placeholder="邮箱（可选）" />
          </Form.Item>

          <Form.Item
            name="primaryPhone"
          >
            <Input prefix={<PhoneOutlined />} placeholder="手机号（可选）" />
          </Form.Item>

          <Form.Item
            name="tenantID"
            rules={[{ required: true, message: '请输入租户ID' }]}
          >
            <Input type="number" placeholder="租户ID" />
          </Form.Item>

          <Form.Item>
            <Button type="primary" htmlType="submit" loading={loading} block>
              注册
            </Button>
          </Form.Item>

          <div style={{ textAlign: 'center' }}>
            <a onClick={() => navigate('/login')}>已有账号？立即登录</a>
          </div>
        </Form>
      </Card>
    </div>
  )
}

export default Register
```

---

## Task 8: 仪表盘页面

**Files:**
- Create: `frontend/src/pages/dashboard/index.tsx`

- [ ] **Step 1: 创建 pages/dashboard/index.tsx**

```typescript
import { Card, Row, Col } from 'antd'

const Dashboard = () => {
  return (
    <div>
      <h1>仪表盘</h1>
      <Row gutter={16}>
        <Col span={6}>
          <Card title="用户总数">
            <p style={{ fontSize: 24, textAlign: 'center' }}>-</p>
          </Card>
        </Col>
        <Col span={6}>
          <Card title="角色总数">
            <p style={{ fontSize: 24, textAlign: 'center' }}>-</p>
          </Card>
        </Col>
        <Col span={6}>
          <Card title="部门总数">
            <p style={{ fontSize: 24, textAlign: 'center' }}>-</p>
          </Card>
        </Col>
        <Col span={6}>
          <Card title="应用总数">
            <p style={{ fontSize: 24, textAlign: 'center' }}>-</p>
          </Card>
        </Col>
      </Row>
    </div>
  )
}

export default Dashboard
```

---

## Task 9: 用户管理页面

**Files:**
- Create: `frontend/src/pages/user/index.tsx`
- Create: `frontend/src/pages/user/Detail.tsx`
- Create: `frontend/src/api/user.ts`

- [ ] **Step 1: 创建 api/user.ts**

```typescript
import request from '../utils/request'

export interface User {
  id: number
  username: string
  name: string
  email: string
  phone: string
  status: string
  createdAt: string
}

export interface UserPageListReq {
  page: number
  pageSize: number
  keyword?: string
}

export const getUserPageList = (data: UserPageListReq) => {
  return request.post<any, any>('/user/pageList', data)
}

export const getUserDetail = (id: number) => {
  return request.get<any, any>('/user/detail', { params: { userID: id } })
}
```

- [ ] **Step 2: 创建 pages/user/index.tsx**

```typescript
import { useEffect, useState } from 'react'
import { Table, Button, Space, Input, Modal, message } from 'antd'
import { useNavigate } from 'react-router'
import type { ColumnsType } from 'antd/es/table'
import { getUserPageList, User } from '../../api/user'

interface UserTableItem extends User {}

const UserList = () => {
  const navigate = useNavigate()
  const [data, setData] = useState<UserTableItem[]>([])
  const [loading, setLoading] = useState(false)
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(10)
  const [total, setTotal] = useState(0)
  const [keyword, setKeyword] = useState('')

  const fetchData = async () => {
    setLoading(true)
    try {
      const resp = await getUserPageList({ page, pageSize, keyword })
      setData(resp.data?.list || [])
      setTotal(resp.data?.total || 0)
    } catch (error) {
      console.error('获取用户列表失败:', error)
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    fetchData()
  }, [page, pageSize, keyword])

  const columns: ColumnsType<UserTableItem> = [
    { title: 'ID', dataIndex: 'id', key: 'id' },
    { title: '用户名', dataIndex: 'username', key: 'username' },
    { title: '姓名', dataIndex: 'name', key: 'name' },
    { title: '邮箱', dataIndex: 'email', key: 'email' },
    { title: '手机号', dataIndex: 'phone', key: 'phone' },
    { title: '状态', dataIndex: 'status', key: 'status' },
    {
      title: '操作',
      key: 'action',
      render: (_, record) => (
        <Space>
          <Button type="link" onClick={() => navigate(`/user/${record.id}`)}>
            详情
          </Button>
        </Space>
      ),
    },
  ]

  return (
    <div>
      <h1>用户管理</h1>
      <div style={{ marginBottom: 16 }}>
        <Input.Search
          placeholder="搜索用户"
          style={{ width: 200 }}
          onSearch={(value) => setKeyword(value)}
        />
      </div>
      <Table
        columns={columns}
        dataSource={data}
        rowKey="id"
        loading={loading}
        pagination={{
          current: page,
          pageSize,
          total,
          onChange: (p, ps) => {
            setPage(p)
            setPageSize(ps)
          },
        }}
      />
    </div>
  )
}

export default UserList
```

- [ ] **Step 3: 创建 pages/user/Detail.tsx**

```typescript
import { useEffect, useState } from 'react'
import { useParams, useNavigate } from 'react-router'
import { Card, Descriptions, Button, Spin } from 'antd'
import { getUserDetail } from '../../api/user'

const UserDetail = () => {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const [loading, setLoading] = useState(false)
  const [user, setUser] = useState<any>(null)

  useEffect(() => {
    const fetchData = async () => {
      if (!id) return
      setLoading(true)
      try {
        const resp = await getUserDetail(Number(id))
        setUser(resp.data)
      } catch (error) {
        console.error('获取用户详情失败:', error)
      } finally {
        setLoading(false)
      }
    }
    fetchData()
  }, [id])

  if (loading) return <Spin />

  if (!user) return null

  return (
    <div>
      <Button onClick={() => navigate('/user')}>返回</Button>
      <Card title="用户详情" style={{ marginTop: 16 }}>
        <Descriptions column={2} bordered>
          <Descriptions.Item label="ID">{user.id}</Descriptions.Item>
          <Descriptions.Item label="用户名">{user.username}</Descriptions.Item>
          <Descriptions.Item label="姓名">{user.name}</Descriptions.Item>
          <Descriptions.Item label="邮箱">{user.email}</Descriptions.Item>
          <Descriptions.Item label="手机号">{user.phone}</Descriptions.Item>
          <Descriptions.Item label="状态">{user.status}</Descriptions.Item>
        </Descriptions>
      </Card>
    </div>
  )
}

export default UserDetail
```

---

## Task 10: 角色管理页面

**Files:**
- Create: `frontend/src/pages/role/index.tsx`
- Create: `frontend/src/api/role.ts`

- [ ] **Step 1: 创建 api/role.ts**

```typescript
import request from '../utils/request'

export interface Role {
  id: number
  name: string
  code: string
  description: string
  status: string
  createdAt: string
}

export interface RolePageListReq {
  page: number
  pageSize: number
  keyword?: string
}

export const getRolePageList = (data: RolePageListReq) => {
  return request.post<any, any>('/role/pageList', data)
}

export const getRoleDetail = (id: number) => {
  return request.get<any, any>('/role/detail', { params: { roleID: id } })
}
```

- [ ] **Step 2: 创建 pages/role/index.tsx**

```typescript
import { useEffect, useState } from 'react'
import { Table, Button, Space, Input } from 'antd'
import type { ColumnsType } from 'antd/es/table'
import { getRolePageList, Role } from '../../api/role'

interface RoleTableItem extends Role {}

const RoleList = () => {
  const [data, setData] = useState<RoleTableItem[]>([])
  const [loading, setLoading] = useState(false)
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(10)
  const [total, setTotal] = useState(0)
  const [keyword, setKeyword] = useState('')

  const fetchData = async () => {
    setLoading(true)
    try {
      const resp = await getRolePageList({ page, pageSize, keyword })
      setData(resp.data?.list || [])
      setTotal(resp.data?.total || 0)
    } catch (error) {
      console.error('获取角色列表失败:', error)
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    fetchData()
  }, [page, pageSize, keyword])

  const columns: ColumnsType<RoleTableItem> = [
    { title: 'ID', dataIndex: 'id', key: 'id' },
    { title: '角色名称', dataIndex: 'name', key: 'name' },
    { title: '角色编码', dataIndex: 'code', key: 'code' },
    { title: '描述', dataIndex: 'description', key: 'description' },
    { title: '状态', dataIndex: 'status', key: 'status' },
  ]

  return (
    <div>
      <h1>角色管理</h1>
      <div style={{ marginBottom: 16 }}>
        <Input.Search
          placeholder="搜索角色"
          style={{ width: 200 }}
          onSearch={(value) => setKeyword(value)}
        />
      </div>
      <Table
        columns={columns}
        dataSource={data}
        rowKey="id"
        loading={loading}
        pagination={{
          current: page,
          pageSize,
          total,
          onChange: (p, ps) => {
            setPage(p)
            setPageSize(ps)
          },
        }}
      />
    </div>
  )
}

export default RoleList
```

---

## Task 11: 部门管理页面

**Files:**
- Create: `frontend/src/pages/department/index.tsx`
- Create: `frontend/src/api/department.ts`

- [ ] **Step 1: 创建 api/department.ts**

```typescript
import request from '../utils/request'

export interface Department {
  id: number
  name: string
  parentID: number
  orderNum: number
  status: string
  createdAt: string
}

export const getDepartmentList = () => {
  return request.get<any, any>('/department/list')
}

export const createDepartment = (data: any) => {
  return request.post<any, any>('/department/create', data)
}

export const updateDepartment = (data: any) => {
  return request.post<any, any>('/department/update', data)
}

export const deleteDepartment = (id: number) => {
  return request.post<any, any>('/department/delete', { id })
}
```

- [ ] **Step 2: 创建 pages/department/index.tsx**

```typescript
import { useEffect, useState } from 'react'
import { Tree, Button, Modal, Form, Input, message } from 'antd'
import type { DataNode } from 'antd/es/tree'
import { getDepartmentList, createDepartment, deleteDepartment, Department } from '../../api/department'

const DepartmentList = () => {
  const [data, setData] = useState<Department[]>([])
  const [loading, setLoading] = useState(false)
  const [modalVisible, setModalVisible] = useState(false)
  const [form] = Form.useForm()

  const fetchData = async () => {
    setLoading(true)
    try {
      const resp = await getDepartmentList()
      setData(resp.data || [])
    } catch (error) {
      console.error('获取部门列表失败:', error)
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    fetchData()
  }, [])

  const buildTreeData = (list: Department[]): DataNode[] => {
    return list.map((item) => ({
      title: item.name,
      key: item.id,
      children: [],
    }))
  }

  const handleAdd = () => {
    form.resetFields()
    setModalVisible(true)
  }

  const handleSubmit = async () => {
    try {
      const values = await form.validateFields()
      await createDepartment(values)
      message.success('创建成功')
      setModalVisible(false)
      fetchData()
    } catch (error) {
      console.error('创建部门失败:', error)
    }
  }

  return (
    <div>
      <h1>部门管理</h1>
      <Button type="primary" onClick={handleAdd} style={{ marginBottom: 16 }}>
        新建部门
      </Button>
      <Tree treeData={buildTreeData(data)} />

      <Modal
        title="新建部门"
        open={modalVisible}
        onOk={handleSubmit}
        onCancel={() => setModalVisible(false)}
      >
        <Form form={form} layout="vertical">
          <Form.Item name="name" label="部门名称" rules={[{ required: true }]}>
            <Input />
          </Form.Item>
          <Form.Item name="parentID" label="上级部门ID">
            <Input type="number" />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  )
}

export default DepartmentList
```

---

## Task 12: 应用管理页面

**Files:**
- Create: `frontend/src/pages/application/index.tsx`
- Create: `frontend/src/api/application.ts`

- [ ] **Step 1: 创建 api/application.ts**

```typescript
import request from '../utils/request'

export interface Application {
  id: number
  name: string
  code: string
  description: string
  status: string
  createdAt: string
}

export interface ApplicationPageListReq {
  page: number
  pageSize: number
  keyword?: string
}

export const getApplicationPageList = (data: ApplicationPageListReq) => {
  return request.post<any, any>('/application/pageList', data)
}

export const getApplicationDetail = (id: number) => {
  return request.get<any, any>('/application/detail', { params: { applicationID: id } })
}

export const createApplication = (data: any) => {
  return request.post<any, any>('/application/create', data)
}

export const updateApplication = (data: any) => {
  return request.post<any, any>('/application/update', data)
}

export const deleteApplication = (id: number) => {
  return request.post<any, any>('/application/delete', { id })
}

export const getApplicationRoles = (applicationId: number) => {
  return request.get<any, any>('/application/roles', { params: { applicationId } })
}
```

- [ ] **Step 2: 创建 pages/application/index.tsx**

```typescript
import { useEffect, useState } from 'react'
import { Table, Button, Space, Input } from 'antd'
import type { ColumnsType } from 'antd/es/table'
import { getApplicationPageList, Application } from '../../api/application'

interface ApplicationTableItem extends Application {}

const ApplicationList = () => {
  const [data, setData] = useState<ApplicationTableItem[]>([])
  const [loading, setLoading] = useState(false)
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(10)
  const [total, setTotal] = useState(0)
  const [keyword, setKeyword] = useState('')

  const fetchData = async () => {
    setLoading(true)
    try {
      const resp = await getApplicationPageList({ page, pageSize, keyword })
      setData(resp.data?.list || [])
      setTotal(resp.data?.total || 0)
    } catch (error) {
      console.error('获取应用列表失败:', error)
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    fetchData()
  }, [page, pageSize, keyword])

  const columns: ColumnsType<ApplicationTableItem> = [
    { title: 'ID', dataIndex: 'id', key: 'id' },
    { title: '应用名称', dataIndex: 'name', key: 'name' },
    { title: '应用编码', dataIndex: 'code', key: 'code' },
    { title: '描述', dataIndex: 'description', key: 'description' },
    { title: '状态', dataIndex: 'status', key: 'status' },
  ]

  return (
    <div>
      <h1>应用管理</h1>
      <div style={{ marginBottom: 16 }}>
        <Input.Search
          placeholder="搜索应用"
          style={{ width: 200 }}
          onSearch={(value) => setKeyword(value)}
        />
      </div>
      <Table
        columns={columns}
        dataSource={data}
        rowKey="id"
        loading={loading}
        pagination={{
          current: page,
          pageSize,
          total,
          onChange: (p, ps) => {
            setPage(p)
            setPageSize(ps)
          },
        }}
      />
    </div>
  )
}

export default ApplicationList
```

---

## Task 13: 更新 README.md

**Files:**
- Create: `frontend/README.md`

- [ ] **Step 1: 创建 frontend/README.md**

```markdown
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

1. 登录：`POST /v1/iam/auth/login`
2. 获取用户信息：`GET /v1/iam/auth/userinfo`
3. 刷新 Token：`POST /v1/iam/auth/refreshToken`
4. 退出登录：`POST /v1/iam/auth/logout`
```

---

## 自检清单

- [ ] 所有文件路径已确认
- [ ] 所有 API 接口与后端文档匹配
- [ ] 所有 TypeScript 类型定义完整
- [ ] 页面组件结构清晰
- [ ] README 文档完整

**Plan complete.**