import { useEffect, useMemo, useState } from 'react'
import { Route, Routes, Navigate, useLocation, useNavigate } from 'react-router-dom'
import { useAuthGuard, FullPageSpinner } from '@morehao/ark-iam-react/auth'
import { MainLayout, SsoLoginPage, tokens, menuIcon } from '@morehao/ark-iam-react/ui'
import type { MainMenuItems } from '@morehao/ark-iam-react/ui'
import type { MenuItem } from '@morehao/ark-iam-react/types'
import { getMyMenuTree } from '@morehao/ark-iam-react/api'
import Dashboard from './pages/dashboard'
import ApplicationList from './pages/application'
import TenantList from './pages/tenant'
import TenantApplicationList from './pages/tenantApplication'
import OAuthClientList from './pages/oauthClient'
import OAuthClientDetail from './pages/oauthClient/Detail'
import MenuList from './pages/menu'
import DomainList from './pages/domain'
import { OAUTH_CLIENT_DETAIL_ROUTE } from './routes'

// 组件白名单：只有 path 命中才会渲染路由与侧边栏菜单，避免点击进入 404。
// 关键：path 必须与后端动态菜单的 path 一致（均为绝对路径）。
// 用户与角色不在其中：两者按租户归属，平台端不提供入口。
const COMPONENT_MAP: Record<string, React.ComponentType> = {
  '/dashboard': Dashboard,
  '/application': ApplicationList,
  '/tenant': TenantList,
  '/tenant-application': TenantApplicationList,
  '/oauth-client': OAuthClientList,
  '/menu': MenuList,
  '/domain': DomainList,
  // 详情页（不进侧边栏菜单，由静态路由单独注册；path 取自 routes.ts 单一真相源）
  [OAUTH_CLIENT_DETAIL_ROUTE]: OAuthClientDetail,
}

// 将后端菜单树转换为 MainLayout 侧边栏菜单；叶子菜单需命中组件白名单，目录仅保留有可渲染子项的
function buildMenuItems(list: MenuItem[]): MainMenuItems[] {
  const result: MainMenuItems[] = []
  list
    .filter((m) => m.type !== 'button')
    .forEach((m) => {
      const children = m.children?.length ? buildMenuItems(m.children) : undefined
      if (children && children.length > 0) {
        result.push({ key: m.path || `/${m.code}`, label: m.name, icon: menuIcon(m.icon), children })
      } else if (COMPONENT_MAP[m.path]) {
        result.push({ key: m.path || `/${m.code}`, label: m.name, icon: menuIcon(m.icon) })
      }
    })
  return result
}

// 从后端菜单树收集可渲染路由
function collectRoutes(list: MenuItem[]): React.ReactNode[] {
  const result: React.ReactNode[] = []
  const walk = (items: MenuItem[]) => {
    items.forEach((m) => {
      const Comp = COMPONENT_MAP[m.path]
      if (Comp) result.push(<Route key={m.path} path={m.path} element={<Comp />} />)
      if (m.children?.length) walk(m.children)
    })
  }
  walk(list)
  return result
}

// 找到第一个可渲染的菜单 path，作为默认首页重定向目标；无可用菜单时返回空串（渲染空态）。
function firstRoutablePath(list: MenuItem[]): string {
  for (const m of list) {
    if (COMPONENT_MAP[m.path] && !m.path.includes(':')) return m.path
    if (m.children?.length) {
      const sub = firstRoutablePath(m.children)
      if (sub) return sub
    }
  }
  return ''
}

function App() {
  const auth = useAuthGuard()
  const location = useLocation()
  const navigate = useNavigate()

  const [menuTree, setMenuTree] = useState<MenuItem[] | null>(null)

  // 仅在认证完成后加载动态菜单；未认证时发起请求会 401，
  // 进而触发全局 session-expired 处理（removeUser + 跳转 /），造成刷新死循环。
  useEffect(() => {
    if (!auth.isAuthenticated || auth.isLoading || auth.activeNavigator) return
    let mounted = true
    getMyMenuTree()
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
  }, [auth.isAuthenticated, auth.isLoading, auth.activeNavigator])

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

  const sidebarMenu = useMemo(() => buildMenuItems(effectiveTree), [effectiveTree])
  const dynamicRoutes = useMemo(() => collectRoutes(effectiveTree), [effectiveTree])
  const defaultPath = useMemo(() => firstRoutablePath(effectiveTree), [effectiveTree])

  if (auth.isLoading || auth.activeNavigator) return <FullPageSpinner />
  // 回调/登录失败：必须给出可见的错误与恢复入口，绝不无限转圈。
  // react-oidc-context 在 signinCallback / silent renew 失败时会置 auth.error；
  // 此时 isAuthenticated 为 false，若没有这一分支就会永远停在 /auth/callback 的 spinner 上。
  if (auth.error) {
    return <AuthErrorView error={auth.error} onRetry={() => void retryLogin(auth)} />
  }
  if (!auth.isAuthenticated && location.pathname !== '/auth/callback') return null
  // 登录页不依赖动态菜单，无需等待菜单加载。
  if (location.pathname !== '/login' && !menuLoaded) return <FullPageSpinner />

  return (
    <Routes>
      <Route path="/login" element={<SsoLoginPage title="IAM 管理平台" subtitle="平台管理后台" />} />
      <Route path="/auth/callback" element={<FullPageSpinner />} />
      <Route path="/" element={<MainLayout title="Ark IAM" subtitle="平台管理后台" menuItems={sidebarMenu} />}>
        {defaultPath ? (
          <Route index element={<Navigate to={defaultPath} replace />} />
        ) : (
          <Route index element={<EmptyAccess />} />
        )}
        {dynamicRoutes}
        {/* 详情路由（静态注册，不进侧边栏菜单） */}
        <Route path={OAUTH_CLIENT_DETAIL_ROUTE} element={<OAuthClientDetail />} />
      </Route>
    </Routes>
  )
}

// clearOidcStorage 彻底清掉 oidc-client-ts 在本地留下的全部痕迹。
//
// 为什么不能只 removeUser()：oidc-client-ts 除了 `oidc.user:*`，还会把授权流程的
// state/nonce/回调参数存在带 `oidc.` 前缀的键里（本次实现落在 localStorage 的
// `oidc.<hash>`，用户令牌在 sessionStorage 的 `oidc.user:*`）。若只删 user 不清这些
// 残留 state，则跳回 `/` 后 oidc-client-ts 会读到残留 state 再次判定「这是回调」，
// 带上已失效的旧 code 触发 signinCallback → user not found → 又回到错误页，形成死循环。
// 因此重试时把两个 storage 里所有 `oidc.` 前缀的键清干净，确保是一次干净的重登。
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

// retryLogin 重新发起一次登录：先彻底清掉本地残留（失效的 user/state/code），
// 再整页跳回根路径由 signinRedirect 重新走 authorize。
// 用整页跳转而非 router navigate，确保 URL 上残留的 ?code=&state= 被丢弃。
async function retryLogin(auth: ReturnType<typeof useAuthGuard>) {
  try {
    await auth.removeUser()
  } catch {
    // 本地清理失败不阻断重试
  }
  clearOidcStorage()
  window.location.assign('/')
}

// AuthErrorView 是认证失败（OIDC 回调 / 静默续期出错）时的兜底页。
// 核心目的：绝不让失败表现为「无限加载的 spinner」——必须给出可见原因与「重新登录」入口。
function AuthErrorView({
  error,
  onRetry,
}: {
  error: { message?: string; name?: string } 
  onRetry: () => void
}) {
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
// 菜单为空有两类成因，提示必须同时覆盖：
//   1) 应用准入缺失（准入是进入应用的前提，无准入者连登录都会被拒）；
//   2) 有准入但该应用下未授予任何功能菜单（无角色 / 角色无菜单）。
function EmptyAccess() {
  return (
    <div style={{ padding: 80, textAlign: 'center', color: tokens.textPlaceholder }}>
      <div style={{ fontSize: 40 }}>🔒</div>
      <p>你尚未被授予本应用的任何功能权限</p>
      <p style={{ fontSize: 13 }}>当前账号没有可见菜单：可能是未获得本应用的准入，或已准入但未被授予任何菜单权限。</p>
      <p style={{ fontSize: 13 }}>请联系平台管理员分配应用准入与角色。</p>
    </div>
  )
}

export default App
