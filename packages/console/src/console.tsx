import { createContext, useContext, useEffect, useMemo, useState } from 'react'
import { BrowserRouter, Route, Routes, Navigate, useLocation, useNavigate } from 'react-router-dom'
import { useAuthGuard, FullPageSpinner, createAuthProvider, fetchRuntimeConfig } from '@ark-iam/auth'
import { MainLayout, SsoLoginPage, tokens, menuIcon, AppShell } from '@ark-iam/ui'
import type { MainMenuItems } from '@ark-iam/ui'
import type { MenuItem } from '@ark-iam/types'
import { getMyMenuTree, getTenantMenuTree } from '@ark-iam/api'
import { OAUTH_CLIENT_DETAIL_ROUTE } from './routes'

import Dashboard from './pages/platform/dashboard'
import ApplicationList from './pages/platform/application'
import TenantList from './pages/platform/tenant'
import TenantApplicationList from './pages/platform/tenantApplication'
import OAuthClientList from './pages/platform/oauthClient'
import OAuthClientDetail from './pages/platform/oauthClient/Detail'
import MenuList from './pages/platform/menu'
import DomainList from './pages/platform/domain'

import DepartmentList from './pages/tenant/department'
import TenantUserList from './pages/tenant/user'
import TenantRoleList from './pages/tenant/role'
import TenantApiKeyList from './pages/tenant/apiKey'

/**
 * 页面级门面包对外配置（宿主一字接入的最小画像）。
 *
 * 优先级与 `createAuthProvider` 一致：props > /config.json > VITE_* > 相对默认值。
 */
export interface IamConsoleConfig {
  /** OIDC client_id（默认从 /config.json.clientId 取） */
  clientId?: string
  issuer?: string
  /** API 基址（默认 /config.json.apiBase → '/v1'） */
  apiBase?: string
  /** 已下发的域白名单（G4）；缺省视为全部域 */
  enabledDomains?: string[]
  /** 挂载基路径，默认 '/'（子路径部署传 '/iam'） */
  basePath?: string
  /** 控制台形态，默认 'platform'；'tenant' 时侧栏启用租户切换、登录页副标题不同 */
  variant?: 'platform' | 'tenant'
}

/** 单条可渲染路由定义（供宿主可选地覆盖 / 收窄默认页面集）。 */
export interface ConsoleRouteDef {
  /** 与后端菜单 path 一致（绝对路径 kebab-case） */
  path: string
  Component: React.ComponentType
  /** 该页归属的域（供 enabledDomains 裁剪） */
  domain?: string
}

// 平台端组件白名单：path 命中才渲染路由与侧边栏菜单，避免点击进入 404。
// path 必须与后端动态菜单的 path 一致（绝对路径）。用户与角色按租户归属，平台端不提供入口。
const PLATFORM_COMPONENT_MAP: Record<string, { Component: React.ComponentType; domain?: string }> = {
  '/dashboard': { Component: Dashboard },
  '/application': { Component: ApplicationList, domain: 'application' },
  '/tenant': { Component: TenantList, domain: 'tenant' },
  '/tenant-application': { Component: TenantApplicationList, domain: 'tenant-app' },
  '/oauth-client': { Component: OAuthClientList, domain: 'oauth-client' },
  '/menu': { Component: MenuList, domain: 'menu' },
  '/domain': { Component: DomainList, domain: 'domain' },
  // 详情页（不进侧边栏菜单，由静态路由单独注册；path 取自 routes.ts 单一真相源）
  [OAUTH_CLIENT_DETAIL_ROUTE]: { Component: OAuthClientDetail, domain: 'oauth-client' },
}

// 租户端组件白名单
const TENANT_COMPONENT_MAP: Record<string, { Component: React.ComponentType; domain?: string }> = {
  '/department': { Component: DepartmentList, domain: 'department' },
  '/user': { Component: TenantUserList, domain: 'user' },
  '/role': { Component: TenantRoleList, domain: 'role' },
  '/api-key': { Component: TenantApiKeyList, domain: 'api-key' },
}

/** 平台端 / 租户端的形态级差异（标题、登录页副标题、菜单拉取、组件白名单）。 */
interface VariantMeta {
  title: string
  subtitle: string
  loginTitle: string
  loginSubtitle: string
  fetchMenuTree: () => Promise<{ list: MenuItem[] } | null | undefined>
  componentMap: Record<string, { Component: React.ComponentType; domain?: string }>
}

const VARIANTS: Record<'platform' | 'tenant', VariantMeta> = {
  platform: {
    title: 'Ark IAM',
    subtitle: '平台管理后台',
    loginTitle: 'IAM 管理平台',
    loginSubtitle: '平台管理后台',
    fetchMenuTree: getMyMenuTree,
    componentMap: PLATFORM_COMPONENT_MAP,
  },
  tenant: {
    title: 'Ark IAM',
    subtitle: '租户管理后台',
    loginTitle: '租户管理',
    loginSubtitle: '租户管理后台',
    fetchMenuTree: getTenantMenuTree,
    componentMap: TENANT_COMPONENT_MAP,
  },
}

// 依据 enabledDomains 白名单过滤组件白名单：未下发（undefined/空）视为全部域可用；
// 声明了 domain 的页面在 domain 不在白名单时被裁掉（既不出现在侧栏也不注册路由）。
function filterByDomains(
  map: Record<string, { Component: React.ComponentType; domain?: string }>,
  enabledDomains?: string[],
): Record<string, { Component: React.ComponentType; domain?: string }> {
  if (!enabledDomains || enabledDomains.length === 0) return map
  const allow = new Set(enabledDomains)
  const out: Record<string, { Component: React.ComponentType; domain?: string }> = {}
  for (const [path, def] of Object.entries(map)) {
    if (def.domain && !allow.has(def.domain)) continue
    out[path] = def
  }
  return out
}

// 将后端菜单树转换为 MainLayout 侧边栏菜单；叶子菜单需命中组件白名单，目录仅保留有可渲染子项的
function buildMenuItems(list: MenuItem[], map: Record<string, { Component: React.ComponentType }>): MainMenuItems[] {
  const result: MainMenuItems[] = []
  list
    .filter((m) => m.type !== 'button')
    .forEach((m) => {
      const children = m.children?.length ? buildMenuItems(m.children, map) : undefined
      if (children && children.length > 0) {
        result.push({ key: m.path || `/${m.code}`, label: m.name, icon: menuIcon(m.icon), children })
      } else if (map[m.path]) {
        result.push({ key: m.path || `/${m.code}`, label: m.name, icon: menuIcon(m.icon) })
      }
    })
  return result
}

// 从后端菜单树收集可渲染路由
function collectRoutes(list: MenuItem[], map: Record<string, { Component: React.ComponentType }>): React.ReactNode[] {
  const result: React.ReactNode[] = []
  const walk = (items: MenuItem[]) => {
    items.forEach((m) => {
      const def = map[m.path]
      if (def) {
        const Comp = def.Component
        result.push(<Route key={m.path} path={m.path} element={<Comp />} />)
      }
      if (m.children?.length) walk(m.children)
    })
  }
  walk(list)
  return result
}

// 找到第一个可渲染的菜单 path，作为默认首页重定向目标；无可用菜单时返回空串（渲染空态）。
function firstRoutablePath(list: MenuItem[], map: Record<string, { Component: React.ComponentType }>): string {
  for (const m of list) {
    if (map[m.path] && !m.path.includes(':')) return m.path
    if (m.children?.length) {
      const sub = firstRoutablePath(m.children, map)
      if (sub) return sub
    }
  }
  return ''
}

/**
 * ConsoleRuntimeContext：IamProvider 在拿到 `/config.json` 后把运行期下发的域白名单注入，
 * 供 IamRoutes 在 props 未显式指定 enabledDomains 时作为兜底来源（props 覆盖 runtime）。
 */
const ConsoleRuntimeContext = createContext<{ enabledDomains?: string[] }>({})
const useConsoleRuntime = () => useContext(ConsoleRuntimeContext)

// resolveConfig 把 props config 与默认值收敛为最终生效配置。
function resolveConfig(config?: IamConsoleConfig): Required<Pick<IamConsoleConfig, 'variant' | 'basePath'>> & IamConsoleConfig {
  return {
    variant: config?.variant ?? 'platform',
    basePath: config?.basePath ?? '/',
    ...config,
  }
}

/**
 * IamProvider：认证 + 主题外壳（createAuthProvider + AppShell）。
 *
 * ★不内置额外 Router：`createAuthProvider` 自带的历史 BrowserRouter 是既有形态，这里不再叠加，
 * 宿主须在外层自备 Router（浏览器整站部署改用 `IamConsoleApp` 自托管入口）。
 */
export function IamProvider({ config, children }: { config?: IamConsoleConfig; children: React.ReactNode }) {
  const resolved = resolveConfig(config)
  const AuthProvider = useMemo(
    () =>
      // 延迟到运行时构造，避免在模块顶层读取 import.meta（SSR / 非 Vite 场景安全）。
      createAuthProviderFor(resolved),
    [resolved.clientId, resolved.issuer, resolved.apiBase],
  )

  // 运行期域白名单：优先 props（config.enabledDomains），否则从 /config.json 读。
  // props 显式给（含空数组）即视为宿主已拍板，不再回退 runtime；未给才去拉 config.json。
  const [runtimeDomains, setRuntimeDomains] = useState<string[] | undefined>(undefined)
  useEffect(() => {
    if (resolved.enabledDomains !== undefined) return
    let cancelled = false
    void fetchRuntimeConfig().then((cfg) => {
      if (!cancelled && cfg?.enabledDomains) setRuntimeDomains(cfg.enabledDomains)
    })
    return () => {
      cancelled = true
    }
  }, [resolved.enabledDomains])

  const enabledDomains = resolved.enabledDomains !== undefined ? resolved.enabledDomains : runtimeDomains

  const runtimeValue = useMemo(() => ({ enabledDomains }), [enabledDomains])

  return (
    <AppShell>
      <ConsoleRuntimeContext.Provider value={runtimeValue}>
        <AuthProvider>{children}</AuthProvider>
      </ConsoleRuntimeContext.Provider>
    </AppShell>
  )
}

// createAuthProviderFor 是 createAuthProvider 的薄包装，把 IamConsoleConfig 中的认证相关字段透传。
function createAuthProviderFor(config: IamConsoleConfig) {
  return createAuthProvider({
    clientID: config.clientId,
    issuer: config.issuer,
    useRuntimeConfig: true,
  })
}

/**
 * IamRoutes：与现 App.tsx 行为等价的路由 + 侧栏渲染。
 *
 * - 内部 useAuthGuard 拉取 /me/menu-tree → 用 variant 决定的 COMPONENT_MAP 渲染路由与侧栏；
 * - 支持 `config.enabledDomains` 白名单裁剪（G4）；
 * - 宿主可传 `routes` 覆盖 / 收窄默认页面集（ConsoleRouteDef[]）。
 * ★要求外层已有 Router（不内置）。
 */
export function IamRoutes({ config, routes }: { config?: IamConsoleConfig; routes?: ConsoleRouteDef[] }) {
  const resolved = resolveConfig(config)
  const variantMeta = VARIANTS[resolved.variant]
  // props 未指定 enabledDomains 时，回退到 IamProvider 从 /config.json 注入的 runtime 白名单。
  const runtime = useConsoleRuntime()
  const enabledDomains = resolved.enabledDomains !== undefined ? resolved.enabledDomains : runtime.enabledDomains

  const auth = useAuthGuard()
  const location = useLocation()
  const navigate = useNavigate()

  const [menuTree, setMenuTree] = useState<MenuItem[] | null>(null)

  // 组件白名单：variant 决定；宿主传 routes 时以 routes 为准（可按需收窄/覆盖）；再按 enabledDomains 裁剪。
  const componentMap = useMemo(() => {
    if (routes) {
      const m: Record<string, { Component: React.ComponentType; domain?: string }> = {}
      for (const r of routes) m[r.path] = { Component: r.Component, domain: r.domain }
      return filterByDomains(m, enabledDomains)
    }
    return filterByDomains(variantMeta.componentMap, enabledDomains)
  }, [routes, variantMeta, enabledDomains])

  // 仅在认证完成后加载动态菜单；未认证时发起请求会 401，
  // 进而触发全局 session-expired 处理（removeUser + 跳转 /），造成刷新死循环。
  useEffect(() => {
    if (!auth.isAuthenticated || auth.isLoading || auth.activeNavigator) return
    let mounted = true
    variantMeta
      .fetchMenuTree()
      .then((resp) => {
        if (mounted) setMenuTree(resp?.list || [])
      })
      .catch(() => {
        // 后端不可用：保持菜单为空（不提供静态兜底，管理入口绝不凭空出现）
        if (mounted) setMenuTree([])
      })
    return () => {
      mounted = false
    }
  }, [auth.isAuthenticated, auth.isLoading, auth.activeNavigator, variantMeta])

  useEffect(() => {
    if (!auth.isLoading && !auth.activeNavigator && !auth.isAuthenticated && location.pathname !== '/auth/callback') {
      void auth.signinRedirect()
    }
  }, [auth.isLoading, auth.activeNavigator, auth.isAuthenticated, location.pathname, auth.signinRedirect])

  useEffect(() => {
    if (auth.isAuthenticated && location.pathname === '/auth/callback') {
      navigate('/', { replace: true })
    }
  }, [auth.isAuthenticated, location.pathname, navigate])

  // 菜单严格依赖后端按角色下发：后端未返回则不渲染侧边栏菜单，不提供任何静态兜底。
  const menuLoaded = menuTree !== null
  const effectiveTree = useMemo(() => menuTree ?? [], [menuTree])

  const sidebarMenu = useMemo(() => buildMenuItems(effectiveTree, componentMap), [effectiveTree, componentMap])
  const dynamicRoutes = useMemo(() => collectRoutes(effectiveTree, componentMap), [effectiveTree, componentMap])
  const defaultPath = useMemo(() => firstRoutablePath(effectiveTree, componentMap), [effectiveTree, componentMap])

  if (auth.isLoading || auth.activeNavigator) return <FullPageSpinner />
  // 回调/登录失败：必须给出可见的错误与恢复入口，绝不无限转圈。
  if (auth.error) {
    return <AuthErrorView error={auth.error} onRetry={() => void retryLogin(auth)} />
  }
  if (!auth.isAuthenticated && location.pathname !== '/auth/callback') return null
  // 登录页不依赖动态菜单，无需等待菜单加载。
  if (location.pathname !== '/login' && !menuLoaded) return <FullPageSpinner />

  // platform 形态额外静态注册 OAuth 客户端详情路由（不进侧边栏）；tenant 形态无详情路由。
  const detailRoute = resolved.variant === 'platform' ? OAUTH_CLIENT_DETAIL_ROUTE : null

  return (
    <Routes>
      <Route
        path="/login"
        element={<SsoLoginPage title={variantMeta.loginTitle} subtitle={variantMeta.loginSubtitle} />}
      />
      <Route path="/auth/callback" element={<FullPageSpinner />} />
      <Route
        path="/"
        element={<MainLayout title={variantMeta.title} subtitle={variantMeta.subtitle} menuItems={sidebarMenu} />}
      >
        {defaultPath ? (
          <Route index element={<Navigate to={defaultPath} replace />} />
        ) : (
          <Route index element={<EmptyAccess />} />
        )}
        {dynamicRoutes}
        {detailRoute ? <Route path={detailRoute} element={<OAuthClientDetail />} /> : null}
      </Route>
    </Routes>
  )
}

/**
 * IamConsoleApp：自托管入口（无路由栈的宿主 / 仓内两个 app 直接用）。
 *
 * 内部自带 `BrowserRouter` + `IamProvider` + `IamRoutes` 一把装好。
 * ★宿主若已有 Router（推荐形态），请改用 `IamProvider` + `IamRoutes`，否则会「Router 套 Router」报错。
 */
export function IamConsoleApp({ config }: { config?: IamConsoleConfig }) {
  return (
    <BrowserRouter>
      <IamProvider config={config}>
        <IamRoutes config={config} />
      </IamProvider>
    </BrowserRouter>
  )
}

// clearOidcStorage 彻底清掉 oidc-client-ts 在本地留下的全部痕迹（重试前的一次干净重登）。
function clearOidcStorage() {
  for (const store of [window.sessionStorage, window.localStorage]) {
    const keys: string[] = []
    for (let i = 0; i < store.length; i++) {
      const k = store.key(i)
      if (k && k.startsWith('oidc.')) keys.push(k)
    }
    keys.forEach((k) => store.removeItem(k))
  }
}

// retryLogin 重新发起一次登录：先彻底清掉本地残留（失效的 user/state/code），再整页跳回根路径。
async function retryLogin(auth: ReturnType<typeof useAuthGuard>) {
  try {
    await auth.removeUser()
  } catch {
    // 本地清理失败不阻断重试
  }
  clearOidcStorage()
  window.location.assign('/')
}

// AuthErrorView 是认证失败（OIDC 回调 / 静默续期出错）时的兜底页，绝不停在无限 spinner。
function AuthErrorView({ error, onRetry }: { error: { message?: string; name?: string }; onRetry: () => void }) {
  return (
    <div
      style={{
        minHeight: '100vh',
        display: 'flex',
        flexDirection: 'column',
        alignItems: 'center',
        justifyContent: 'center',
        gap: 12,
        padding: 24,
        textAlign: 'center',
        color: tokens.textPlaceholder,
      }}
    >
      <div style={{ fontSize: 40 }}>⚠️</div>
      <div style={{ fontSize: 16, color: tokens.text }}>登录失败</div>
      <div style={{ fontSize: 13, maxWidth: 520 }}>
        认证过程未能完成，可能是登录链接已过期或本次登录信息已失效，请重新登录。
      </div>
      {error?.message ? (
        <div style={{ fontSize: 12, color: tokens.textPlaceholder, maxWidth: 520, wordBreak: 'break-all' }}>
          {error.message}
        </div>
      ) : null}
      <button
        type="button"
        onClick={onRetry}
        style={{
          marginTop: 8,
          padding: '6px 20px',
          fontSize: 14,
          color: '#fff',
          background: tokens.primary,
          border: 'none',
          borderRadius: 6,
          cursor: 'pointer',
        }}
      >
        重新登录
      </button>
    </div>
  )
}

// EmptyAccess 菜单为空（后端未下发任何权限菜单）时的空态占位，避免重定向到 404 或白屏。
function EmptyAccess() {
  return (
    <div style={{ padding: 80, textAlign: 'center', color: tokens.textPlaceholder }}>
      <div style={{ fontSize: 40 }}>🔒</div>
      <p>你尚未被授予本应用的任何功能权限</p>
      <p style={{ fontSize: 13 }}>当前账号没有可见菜单：可能是未获得本应用的准入，或已准入但未被授予任何菜单权限。</p>
      <p style={{ fontSize: 13 }}>请联系管理员分配应用准入与角色。</p>
    </div>
  )
}

// 纯函数导出：供单测与宿主复用（行为与内部渲染一致）。
export { buildMenuItems, collectRoutes, firstRoutablePath, filterByDomains }
