# SSO Test App 精简重构实施计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 将 sso-test-app 从功能复杂的 SSO 测试应用精简为仅含登录、主页、登出的轻量应用

**Architecture:** 参照 platform-admin-web 的代码架构，引入 axios + 拦截器，对齐 authStore/oidc 模式，页面用 Header+Content 布局（无 Sider），首页仅展示用户信息

**Tech Stack:** React 18, TypeScript, Vite, Ant Design 5, react-router-dom v7, Zustand 5, axios

---

## 说明

OIDC 回调：sso-test-app 的后端 `test-rp-client` 注册的 redirect_uri 为 `http://localhost:3001/`（根路径），因此回调在根路径处理，不属于路由表。AuthCallback 由 App.tsx 在 Routes 之前条件渲染。

---

### Task 1: 添加依赖

**Files:**
- Modify: `frontend/apps/sso-test-app/package.json`

- [ ] **Step 1: 添加 axios 和 @ark-iam/shared 依赖**

```bash
cd frontend/apps/sso-test-app && pnpm add axios && pnpm add -D @ark-iam/shared@workspace:*
```

- [ ] **Step 2: 提交**

```bash
git add frontend/apps/sso-test-app/package.json frontend/apps/sso-test-app/pnpm-lock.yaml
git commit -m "feat(sso-test-app): add axios and shared dependencies"
```

---

### Task 2: 创建 src/utils/response.ts

**Files:**
- Create: `frontend/apps/sso-test-app/src/utils/response.ts`

- [ ] **Step 1: 创建 response.ts**

```ts
export interface ApiResponse<T> {
  code: number
  msg: string
  data: T
}

export enum BizCode {
  Success          = 0,
  Unauthorized     = 110000,
  Forbidden        = 110001,
  TokenInvalid     = 110002,
  TokenExpired     = 110003,
  PermissionDenied = 110004,
  ParamInvalid     = 100104,
}
```

- [ ] **Step 2: 提交**

```bash
git add frontend/apps/sso-test-app/src/utils/response.ts
git commit -m "feat(sso-test-app): add ApiResponse and BizCode"
```

---

### Task 3: 改写 src/utils/oidc.ts

**Files:**
- Modify: `frontend/apps/sso-test-app/src/utils/oidc.ts`

- [ ] **Step 1: 改写 oidc.ts，对齐 platform-admin-web 结构，保留 sso-test-app 的 OIDC 配置**

```ts
const OIDC_ISSUER = import.meta.env.VITE_OIDC_ISSUER ?? '/v1/iam/oidc'
const OIDC_CLIENT_ID = import.meta.env.VITE_OIDC_CLIENT_ID ?? 'test-rp-client'
const OIDC_CLIENT_SECRET = 'my-test-client-secret'
const OIDC_REDIRECT_URI = `${window.location.origin}/`
const OIDC_SCOPE = 'openid profile email'

function base64URLEncode(buffer: ArrayBuffer): string {
  return btoa(String.fromCharCode(...new Uint8Array(buffer)))
    .replace(/\+/g, '-')
    .replace(/\//g, '_')
    .replace(/=+$/, '')
}

async function sha256(plain: string): Promise<ArrayBuffer> {
  const encoder = new TextEncoder()
  return crypto.subtle.digest('SHA-256', encoder.encode(plain))
}

export interface PKCEParams {
  codeVerifier: string
  codeChallenge: string
  state: string
}

export function generatePKCEParams(): PKCEParams {
  const array = new Uint8Array(32)
  crypto.getRandomValues(array)
  const codeVerifier = base64URLEncode(array.buffer)
  const state = crypto.randomUUID()

  return { codeVerifier, codeChallenge: '', state }
}

export async function generateCodeChallenge(verifier: string): Promise<string> {
  return base64URLEncode(await sha256(verifier))
}

export function buildAuthorizeURL(params: PKCEParams): string {
  const url = new URL(`${OIDC_ISSUER}/authorize`, window.location.origin)
  url.searchParams.set('client_id', OIDC_CLIENT_ID)
  url.searchParams.set('redirect_uri', OIDC_REDIRECT_URI)
  url.searchParams.set('response_type', 'code')
  url.searchParams.set('scope', OIDC_SCOPE)
  url.searchParams.set('state', params.state)
  url.searchParams.set('code_challenge', params.codeChallenge)
  url.searchParams.set('code_challenge_method', 'S256')
  return url.toString()
}

export function storePKCEParams(params: PKCEParams): void {
  sessionStorage.setItem('oidc_pkce', JSON.stringify(params))
}

export function loadPKCEParams(): PKCEParams | null {
  const raw = sessionStorage.getItem('oidc_pkce')
  if (!raw) return null
  try {
    return JSON.parse(raw) as PKCEParams
  } catch {
    return null
  }
}

export function clearPKCEParams(): void {
  sessionStorage.removeItem('oidc_pkce')
}

export interface TokenResponse {
  access_token: string
  id_token: string
  refresh_token: string
  expires_in: number
  token_type: string
}

export async function exchangeCodeForTokens(code: string, codeVerifier: string): Promise<TokenResponse> {
  const form = new URLSearchParams()
  form.append('grant_type', 'authorization_code')
  form.append('code', code)
  form.append('redirect_uri', OIDC_REDIRECT_URI)
  form.append('client_id', OIDC_CLIENT_ID)
  form.append('code_verifier', codeVerifier)

  const basicAuth = btoa(`${OIDC_CLIENT_ID}:${OIDC_CLIENT_SECRET}`)
  const resp = await fetch(`${OIDC_ISSUER}/oauth/token`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/x-www-form-urlencoded',
      'Authorization': `Basic ${basicAuth}`,
    },
    body: form.toString(),
  })

  if (!resp.ok) {
    throw new Error(`Token exchange failed: ${resp.status}`)
  }
  return resp.json()
}

export async function refreshTokens(refreshToken: string): Promise<TokenResponse> {
  const form = new URLSearchParams()
  form.append('grant_type', 'refresh_token')
  form.append('refresh_token', refreshToken)
  form.append('client_id', OIDC_CLIENT_ID)

  const basicAuth = btoa(`${OIDC_CLIENT_ID}:${OIDC_CLIENT_SECRET}`)
  const resp = await fetch(`${OIDC_ISSUER}/oauth/token`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/x-www-form-urlencoded',
      'Authorization': `Basic ${basicAuth}`,
    },
    body: form.toString(),
  })

  if (!resp.ok) {
    throw new Error(`Token refresh failed: ${resp.status}`)
  }
  return resp.json()
}

export function parseJWT(token: string): Record<string, unknown> | null {
  try {
    const parts = token.split('.')
    if (parts.length !== 3) return null
    const payload = parts[1]
    return JSON.parse(atob(payload.replace(/-/g, '+').replace(/_/g, '/')))
  } catch {
    return null
  }
}

export function buildSilentAuthorizeURL(params: PKCEParams): string {
  const url = buildAuthorizeURL(params)
  return url + '&prompt=none'
}

export function getEndSessionURL(idToken: string): string {
  const url = new URL(`${OIDC_ISSUER}/end_session`, window.location.origin)
  url.searchParams.set('id_token_hint', idToken)
  url.searchParams.set('post_logout_redirect_uri', OIDC_REDIRECT_URI)
  return url.toString()
}
```

- [ ] **Step 2: 提交**

```bash
git add frontend/apps/sso-test-app/src/utils/oidc.ts
git commit -m "refactor(sso-test-app): align oidc utils with platform-admin-web pattern"
```

---

### Task 4: 改写 src/stores/authStore.ts

**Files:**
- Modify: `frontend/apps/sso-test-app/src/stores/authStore.ts`

- [ ] **Step 1: 改写 authStore，添加 personInfo/tenantInfo/setPersonInfo/setTenantInfo**

```ts
import { create } from 'zustand'
import { persist } from 'zustand/middleware'
import type { AuthStage, PersonInfo, TenantInfo } from '@ark-iam/shared'
import { parseJWT } from '../utils/oidc'

interface AuthState {
  authStage: AuthStage
  accessToken: string | null
  idToken: string | null
  refreshToken: string | null
  expiresAt: number | null
  personInfo: PersonInfo | null
  tenantInfo: TenantInfo | null

  setSession: (tokens: {
    accessToken: string
    idToken: string
    refreshToken: string
    expiresIn: number
  }) => void
  updateTokens: (tokens: {
    accessToken: string
    idToken?: string
    refreshToken: string
    expiresIn: number
  }) => void
  setPersonInfo: (info: PersonInfo | null) => void
  setTenantInfo: (info: TenantInfo | null) => void
  logout: () => void
}

function extractTenantFromToken(accessToken: string): TenantInfo | null {
  const claims = parseJWT(accessToken)
  if (!claims) return null
  const tenantID = claims['tenant_id'] as number
  if (!tenantID) return null
  return { tenantID, tenantName: '' }
}

export const useAuthStore = create<AuthState>()(
  persist(
    (set) => ({
      authStage: 'anonymous',
      accessToken: null,
      idToken: null,
      refreshToken: null,
      expiresAt: null,
      personInfo: null,
      tenantInfo: null,

      setSession: ({ accessToken, idToken, refreshToken, expiresIn }) => {
        const expiresAt = Date.now() + expiresIn * 1000
        const tenantInfo = extractTenantFromToken(accessToken)
        set({
          authStage: 'authenticated',
          accessToken,
          idToken,
          refreshToken,
          expiresAt,
          tenantInfo,
          personInfo: null,
        })
      },

      updateTokens: ({ accessToken, idToken, refreshToken, expiresIn }) => {
        const expiresAt = Date.now() + expiresIn * 1000
        const tenantInfo = extractTenantFromToken(accessToken)
        set((state) => ({
          accessToken,
          idToken: idToken ?? state.idToken,
          refreshToken,
          expiresAt,
          tenantInfo: tenantInfo ?? state.tenantInfo,
        }))
      },

      setPersonInfo: (personInfo) => set({ personInfo }),
      setTenantInfo: (tenantInfo) => set({ tenantInfo }),

      logout: () => {
        set({
          authStage: 'anonymous',
          accessToken: null,
          idToken: null,
          refreshToken: null,
          expiresAt: null,
          personInfo: null,
          tenantInfo: null,
        })
      },
    }),
    {
      name: 'auth-storage',
      partialize: (state) => ({
        authStage: state.authStage,
        accessToken: state.accessToken,
        idToken: state.idToken,
        refreshToken: state.refreshToken,
        expiresAt: state.expiresAt,
        personInfo: state.personInfo,
        tenantInfo: state.tenantInfo,
      }),
    }
  )
)
```

- [ ] **Step 2: 提交**

```bash
git add frontend/apps/sso-test-app/src/stores/authStore.ts
git commit -m "refactor(sso-test-app): add personInfo/tenantInfo to authStore"
```

---

### Task 5: 创建 src/utils/request.ts

**Files:**
- Create: `frontend/apps/sso-test-app/src/utils/request.ts`

- [ ] **Step 1: 创建 axios 实例 + 请求/响应拦截器**

```ts
import axios, { AxiosError, InternalAxiosRequestConfig } from 'axios'
import { message } from 'antd'
import { useAuthStore } from '../stores/authStore'
import { BizCode } from './response'
import { refreshTokens } from './oidc'

const request = axios.create({
  baseURL: '/v1/iam',
  timeout: 30000,
})

let isRefreshing = false
let pendingRequests: Array<(token: string) => void> = []

async function handleTokenExpired(originalConfig: InternalAxiosRequestConfig): Promise<any> {
  const store = useAuthStore.getState()

  if (!store.refreshToken) {
    store.logout()
    window.location.href = '/login'
    return Promise.reject(new Error('no refresh token'))
  }

  if (!isRefreshing) {
    isRefreshing = true
    try {
      const resp = await refreshTokens(store.refreshToken)
      store.updateTokens({
        accessToken: resp.access_token,
        idToken: resp.id_token,
        refreshToken: resp.refresh_token,
        expiresIn: resp.expires_in,
      })

      pendingRequests.forEach((cb) => cb(resp.access_token))
      pendingRequests = []

      originalConfig.headers!.Authorization = `Bearer ${resp.access_token}`
      return request(originalConfig)
    } catch {
      store.logout()
      pendingRequests = []
      window.location.href = '/login'
      return Promise.reject(new Error('token refresh failed'))
    } finally {
      isRefreshing = false
    }
  } else {
    return new Promise((resolve) => {
      pendingRequests.push((newToken: string) => {
        originalConfig.headers!.Authorization = `Bearer ${newToken}`
        resolve(request(originalConfig))
      })
    })
  }
}

request.interceptors.request.use(
  (config: InternalAxiosRequestConfig) => {
    const token = useAuthStore.getState().accessToken
    if (token && config.headers) {
      config.headers.Authorization = `Bearer ${token}`
    }
    return config
  },
  (error: AxiosError) => Promise.reject(error)
)

request.interceptors.response.use(
  (response) => {
    const { code, msg } = response.data as { code: number; msg: string }
    if (code === BizCode.Success) return response.data.data

    if (code === BizCode.TokenExpired) return handleTokenExpired(response.config)

    if (code === BizCode.TokenInvalid || code === BizCode.Unauthorized) {
      useAuthStore.getState().logout()
      window.location.href = '/login'
      return Promise.reject(new Error(msg || '未认证'))
    }

    if (code === BizCode.Forbidden || code === BizCode.PermissionDenied) {
      message.warning('暂无权限访问')
      return Promise.reject(new Error(msg || '暂无权限'))
    }

    message.error(msg || '请求失败')
    return Promise.reject(new Error(msg || '请求失败'))
  },
  async (error: AxiosError) => {
    const status = error.response?.status
    const data = error.response?.data as any
    if (status === 401) {
      useAuthStore.getState().logout()
      window.location.href = '/login'
      return Promise.reject(error)
    }
    message.error(data?.msg || '请求失败')
    return Promise.reject(error)
  }
)

export default request
```

- [ ] **Step 2: 提交**

```bash
git add frontend/apps/sso-test-app/src/utils/request.ts
git commit -m "feat(sso-test-app): add axios request with interceptors"
```

---

### Task 6: 创建 src/api/auth.ts

**Files:**
- Create: `frontend/apps/sso-test-app/src/api/auth.ts`

- [ ] **Step 1: 创建认证 API 模块**

```ts
import request from '../utils/request'
import type { PersonInfo } from '@ark-iam/shared'

export interface UserinfoResp {
  personInfo: PersonInfo
  userInfo: { userID: number; tenantID: number; name: string; isOwner: number }
}

export const getUserinfo = () => {
  return request.get<any, UserinfoResp>('/auth/userinfo')
}

export const logoutAPI = (refreshToken: string) => {
  return request.post<any, void>('/auth/logout', { refreshToken })
}
```

- [ ] **Step 2: 提交**

```bash
git add frontend/apps/sso-test-app/src/api/auth.ts
git commit -m "feat(sso-test-app): add auth API module"
```

---

### Task 7: 创建 src/components/MainLayout.tsx

**Files:**
- Create: `frontend/apps/sso-test-app/src/components/MainLayout.tsx`

- [ ] **Step 1: 创建 MainLayout（Header + Content，无 Sider）**

```tsx
import { useEffect, useRef } from 'react'
import { Layout, Avatar, Dropdown } from 'antd'
import { UserOutlined, LogoutOutlined } from '@ant-design/icons'
import { Outlet } from 'react-router-dom'
import { getUserinfo, logoutAPI } from '../api/auth'
import { useAuthStore } from '../stores/authStore'
import { getEndSessionURL } from '../utils/oidc'

const { Header, Content } = Layout

const MainLayout = () => {
  const initializedRef = useRef(false)
  const authStage = useAuthStore((state) => state.authStage)
  const logout = useAuthStore((state) => state.logout)
  const setPersonInfo = useAuthStore((state) => state.setPersonInfo)

  const handleLogout = async () => {
    const store = useAuthStore.getState()
    const currentIdToken = store.idToken

    try {
      await logoutAPI(store.refreshToken ?? '')
    } catch {
      // 即使接口调用失败也继续退出流程
    }

    sessionStorage.setItem('logged_out', '1')
    logout()

    if (currentIdToken) {
      const el = document.createElement('script')
      el.src = getEndSessionURL(currentIdToken)
      el.onload = () => el.remove()
      el.onerror = () => el.remove()
      document.head.appendChild(el)
    }
  }

  useEffect(() => {
    if (initializedRef.current || authStage !== 'authenticated') return
    initializedRef.current = true

    let active = true
    const loadUserContext = async () => {
      try {
        const userinfoResp = await getUserinfo()
        if (!active) return
        const personInfo = userinfoResp?.personInfo ?? null
        setPersonInfo(personInfo)
      } catch {
        return
      }
    }
    void loadUserContext()
    return () => { active = false }
  }, [authStage, setPersonInfo])

  const userMenuItems = [
    { key: 'logout', icon: <LogoutOutlined />, label: '退出登录', onClick: handleLogout },
  ]

  return (
    <Layout style={{ minHeight: '100vh' }}>
      <Header style={{ padding: '0 24px', background: '#fff', display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
        <span style={{ fontSize: 18, fontWeight: 'bold' }}>SSO 测试应用</span>
        <Dropdown menu={{ items: userMenuItems }} placement="bottomRight">
          <Avatar style={{ cursor: 'pointer' }} icon={<UserOutlined />} />
        </Dropdown>
      </Header>
      <Content style={{ margin: 24, padding: 24, background: '#fff', borderRadius: 8 }}>
        <Outlet />
      </Content>
    </Layout>
  )
}

export default MainLayout
```

- [ ] **Step 2: 提交**

```bash
git add frontend/apps/sso-test-app/src/components/MainLayout.tsx
git commit -m "feat(sso-test-app): add MainLayout with Header+Content"
```

---

### Task 8: 改写 Login 页面

**Files:**
- Create: `frontend/apps/sso-test-app/src/pages/auth/Login.tsx`

- [ ] **Step 1: 创建简化 Login 页面，对齐 platform-admin-web 风格**

```tsx
import { useEffect, useRef, useState } from 'react'
import { Button, Card } from 'antd'
import { KeyOutlined } from '@ant-design/icons'
import {
  generatePKCEParams,
  generateCodeChallenge,
  buildAuthorizeURL,
  storePKCEParams,
} from '../../utils/oidc'

const Login = () => {
  const [loading, setLoading] = useState(false)
  const genRef = useRef(0)

  useEffect(() => {
    sessionStorage.removeItem('logged_out')
  }, [])

  const handleOIDCLogin = async () => {
    setLoading(true)
    const gen = ++genRef.current
    try {
      const params = generatePKCEParams()
      params.codeChallenge = await generateCodeChallenge(params.codeVerifier)
      if (gen !== genRef.current) return
      storePKCEParams(params)
      const url = buildAuthorizeURL(params) + '&prompt=login'
      window.location.assign(url)
    } catch {
      setLoading(false)
    }
  }

  return (
    <div
      style={{
        height: '100vh',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        background: '#f0f2f5',
      }}
    >
      <Card title="SSO 测试应用" style={{ width: 400 }}>
        <Button
          type="primary"
          icon={<KeyOutlined />}
          onClick={handleOIDCLogin}
          loading={loading}
          block
          size="large"
        >
          IAM 账号登录
        </Button>
      </Card>
    </div>
  )
}

export default Login
```

- [ ] **Step 2: 提交**

```bash
git add frontend/apps/sso-test-app/src/pages/auth/Login.tsx
git commit -m "refactor(sso-test-app): simplify Login page"
```

---

### Task 9: 改写 AuthCallback 页面

**Files:**
- Create: `frontend/apps/sso-test-app/src/pages/auth/AuthCallback.tsx`

- [ ] **Step 1: 创建 AuthCallback，对齐 platform-admin-web 的模式**

```tsx
import { useEffect, useRef } from 'react'
import { Card, Spin, message } from 'antd'
import { useLocation, useNavigate } from 'react-router-dom'
import {
  loadPKCEParams,
  clearPKCEParams,
  exchangeCodeForTokens,
} from '../../utils/oidc'
import { useAuthStore } from '../../stores/authStore'

const AuthCallback = () => {
  const location = useLocation()
  const navigate = useNavigate()
  const { setSession } = useAuthStore()
  const processedRef = useRef(false)

  useEffect(() => {
    if (processedRef.current) return
    processedRef.current = true

    const searchParams = new URLSearchParams(location.search)
    const code = searchParams.get('code')
    const state = searchParams.get('state')
    const error = searchParams.get('error')

    if (error) {
      if (error === 'login_required') {
        sessionStorage.setItem('oidc_silent_failed', '1')
      } else {
        message.error(`登录失败: ${searchParams.get('error_description') || error}`)
      }
      clearPKCEParams()
      navigate('/login', { replace: true })
      return
    }

    if (!code || !state) {
      clearPKCEParams()
      navigate('/login', { replace: true })
      return
    }

    const pkceParams = loadPKCEParams()
    if (!pkceParams || pkceParams.state !== state) {
      clearPKCEParams()
      navigate('/login', { replace: true })
      return
    }

    const run = async () => {
      try {
        const resp = await exchangeCodeForTokens(code, pkceParams.codeVerifier)
        clearPKCEParams()
        sessionStorage.removeItem('oidc_silent_failed')
        setSession({
          accessToken: resp.access_token,
          idToken: resp.id_token,
          refreshToken: resp.refresh_token,
          expiresIn: resp.expires_in,
        })
        message.success('登录成功')
        navigate('/', { replace: true })
      } catch {
        clearPKCEParams()
        message.error('登录失败，请重试')
        navigate('/login', { replace: true })
      }
    }

    void run()
  }, [location.search, navigate, setSession])

  return (
    <div
      style={{
        minHeight: '100vh',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        background: '#f0f2f5',
        padding: 24,
      }}
    >
      <Card title="正在完成登录" style={{ width: 400, maxWidth: '100%' }}>
        <div style={{ display: 'flex', flexDirection: 'column', alignItems: 'center', gap: 12 }}>
          <Spin />
          <span>正在处理登录回调，请稍候...</span>
        </div>
      </Card>
    </div>
  )
}

export default AuthCallback
```

- [ ] **Step 2: 提交**

```bash
git add frontend/apps/sso-test-app/src/pages/auth/AuthCallback.tsx
git commit -m "refactor(sso-test-app): update AuthCallback to match platform-admin-web"
```

---

### Task 10: 改写 Home 页面

**Files:**
- Create: `frontend/apps/sso-test-app/src/pages/home/index.tsx`

- [ ] **Step 1: 创建极简 Home 页面，仅展示用户信息**

```tsx
import { Card, Avatar } from 'antd'
import { UserOutlined } from '@ant-design/icons'
import { useAuthStore } from '../../stores/authStore'

const Home = () => {
  const personInfo = useAuthStore((state) => state.personInfo)

  return (
    <Card title="用户信息">
      <div style={{ display: 'flex', alignItems: 'center', gap: 16 }}>
        <Avatar
          size={64}
          src={personInfo?.avatar}
          icon={!personInfo?.avatar ? <UserOutlined /> : undefined}
        />
        <div>
          <div style={{ fontSize: 18, fontWeight: 500 }}>{personInfo?.name ?? '-'}</div>
          <div style={{ color: '#888', marginTop: 4 }}>
            用户ID: {personInfo?.personID ?? '-'}
          </div>
        </div>
      </div>
    </Card>
  )
}

export default Home
```

- [ ] **Step 2: 提交**

```bash
git add frontend/apps/sso-test-app/src/pages/home/index.tsx
git commit -m "refactor(sso-test-app): simplify Home page to user info only"
```

---

### Task 11: 改写 App.tsx

**Files:**
- Modify: `frontend/apps/sso-test-app/src/App.tsx`

- [ ] **Step 1: 改写 App.tsx，路由守卫 + 静默登录**

```tsx
import { useEffect, useRef, useState } from 'react'
import { Routes, Route, Navigate } from 'react-router-dom'
import { Spin } from 'antd'
import MainLayout from './components/MainLayout'
import AuthCallback from './pages/auth/AuthCallback'
import Login from './pages/auth/Login'
import Home from './pages/home'
import { useAuthStore } from './stores/authStore'
import { generatePKCEParams, generateCodeChallenge, buildSilentAuthorizeURL, storePKCEParams } from './utils/oidc'

function FullPageSpinner() {
  return (
    <div style={{ height: '100vh', display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
      <Spin size="large" />
    </div>
  )
}

function App() {
  const { authStage } = useAuthStore()
  const [isChecking, setIsChecking] = useState(false)
  const genRef = useRef(0)
  const searchParams = new URLSearchParams(window.location.search)
  const isCallback = searchParams.has('code') || searchParams.has('error')

  useEffect(() => {
    if (isCallback) return
    if (authStage === 'authenticated') return

    if (sessionStorage.getItem('logged_out') === '1') return

    if (sessionStorage.getItem('oidc_silent_failed') === '1') {
      sessionStorage.removeItem('oidc_silent_failed')
      return
    }

    const gen = ++genRef.current
    setIsChecking(true)

    const run = async () => {
      const params = generatePKCEParams()
      params.codeChallenge = await generateCodeChallenge(params.codeVerifier)
      if (gen !== genRef.current) return
      storePKCEParams(params)
      const url = buildSilentAuthorizeURL(params)
      window.location.replace(url)
    }
    void run()
  }, [])

  if (isCallback) {
    return <AuthCallback />
  }

  if (isChecking) {
    return <FullPageSpinner />
  }

  return (
    <Routes>
      <Route
        path="/login"
        element={authStage === 'authenticated' ? <Navigate to="/" replace /> : <Login />}
      />
      <Route
        path="/"
        element={authStage === 'authenticated' ? <MainLayout /> : <Navigate to="/login" replace />}
      >
        <Route index element={<Home />} />
      </Route>
    </Routes>
  )
}

export default App
```

- [ ] **Step 2: 提交**

```bash
git add frontend/apps/sso-test-app/src/App.tsx
git commit -m "refactor(sso-test-app): update App with layout route and guards"
```

---

### Task 12: 删除旧页面文件

**Files:**
- Delete: `frontend/apps/sso-test-app/src/pages/Login.tsx`
- Delete: `frontend/apps/sso-test-app/src/pages/AuthCallback.tsx`
- Delete: `frontend/apps/sso-test-app/src/pages/Home.tsx`

- [ ] **Step 1: 删除旧文件**

```bash
rm frontend/apps/sso-test-app/src/pages/Login.tsx
rm frontend/apps/sso-test-app/src/pages/AuthCallback.tsx
rm frontend/apps/sso-test-app/src/pages/Home.tsx
```

- [ ] **Step 2: 提交**

```bash
git add frontend/apps/sso-test-app/src/pages/Login.tsx frontend/apps/sso-test-app/src/pages/AuthCallback.tsx frontend/apps/sso-test-app/src/pages/Home.tsx
git commit -m "refactor(sso-test-app): remove old page files"
```

---

### Task 13: 验证编译

- [ ] **Step 1: 运行 TypeScript 类型检查**

```bash
cd frontend/apps/sso-test-app && pnpm exec tsc --noEmit
```
Expected: 无类型错误

- [ ] **Step 2: 运行 Vite 构建**

```bash
cd frontend/apps/sso-test-app && pnpm run build
```
Expected: 构建成功，无错误
