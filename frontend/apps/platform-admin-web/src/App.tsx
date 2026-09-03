import { useEffect, useMemo, useState } from 'react'
import { Route, Routes, Navigate, useLocation, useNavigate } from 'react-router-dom'
import { useAuthGuard, FullPageSpinner } from '@ark-iam/auth'
import { MainLayout, LoginPage, tokens } from '@ark-iam/ui'
import type { MainMenuItems } from '@ark-iam/ui'
import {
  ApartmentOutlined,
  AppstoreOutlined,
  DashboardOutlined,
  FileSearchOutlined,
  GlobalOutlined,
  KeyOutlined,
  MenuOutlined,
  SafetyCertificateOutlined,
  SettingOutlined,
  ShoppingOutlined,
  TeamOutlined,
  UserOutlined,
} from '@ant-design/icons'
import type { MenuItem } from '@ark-iam/types'
import { getMyMenuTree } from '@ark-iam/api'
import Dashboard from './pages/dashboard'
import UserList from './pages/user'
import UserDetail from './pages/user/Detail'
import RoleList from './pages/role'
import ApplicationList from './pages/application'
import TenantList from './pages/tenant'
import TenantApplicationList from './pages/tenantApplication'
import OAuthClientList from './pages/oauthClient'
import OAuthClientDetail from './pages/oauthClient/Detail'
import ApiKeyList from './pages/apiKey'
import MenuList from './pages/menu'
import DomainList from './pages/domain'
import LogList from './pages/log'

// 图标映射：后端 menu.icon 存储的字符串 -> antd 图标组件
const ICON_MAP: Record<string, React.ReactNode> = {
  dashboard: <DashboardOutlined />,
  user: <UserOutlined />,
  team: <TeamOutlined />,
  apartment: <ApartmentOutlined />,
  role: <SafetyCertificateOutlined />,
  app: <AppstoreOutlined />,
  menu: <MenuOutlined />,
  global: <GlobalOutlined />,
  shopping: <ShoppingOutlined />,
  key: <KeyOutlined />,
  setting: <SettingOutlined />,
  file: <FileSearchOutlined />,
}

// 组件白名单：只有 path 命中才会渲染路由与侧边栏菜单，避免点击进入 404。
// 关键：path 必须与后端动态菜单的 path 一致（均为绝对路径）。
const COMPONENT_MAP: Record<string, React.ComponentType> = {
  '/dashboard': Dashboard,
  '/user': UserList,
  '/role': RoleList,
  '/application': ApplicationList,
  '/tenant': TenantList,
  '/tenant-application': TenantApplicationList,
  '/oauth-client': OAuthClientList,
  '/api-key': ApiKeyList,
  '/menu': MenuList,
  '/domain': DomainList,
  '/log': LogList,
  // 详情页（不进侧边栏菜单，由静态路由单独注册）
  '/user/:id': UserDetail,
  '/oauth-client/:id': OAuthClientDetail,
}

function iconOf(icon?: string): React.ReactNode {
  return (icon && ICON_MAP[icon]) || <AppstoreOutlined />
}

// 将后端菜单树转换为 MainLayout 侧边栏菜单；叶子菜单需命中组件白名单，目录仅保留有可渲染子项的
function buildMenuItems(list: MenuItem[]): MainMenuItems[] {
  const result: MainMenuItems[] = []
  list
    .filter((m) => m.type !== 'button')
    .forEach((m) => {
      const children = m.children?.length ? buildMenuItems(m.children) : undefined
      if (children && children.length > 0) {
        result.push({ key: m.path || `/${m.code}`, label: m.name, icon: iconOf(m.icon), children })
      } else if (COMPONENT_MAP[m.path]) {
        result.push({ key: m.path || `/${m.code}`, label: m.name, icon: iconOf(m.icon) })
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
  if (!auth.isAuthenticated && location.pathname !== '/auth/callback') return null
  // 登录页不依赖动态菜单，无需等待菜单加载。
  if (location.pathname !== '/login' && !menuLoaded) return <FullPageSpinner />

  return (
    <Routes>
      <Route path="/login" element={<LoginPage title="IAM 管理平台" subtitle="平台管理控制台" />} />
      <Route path="/auth/callback" element={<FullPageSpinner />} />
      <Route path="/" element={<MainLayout title="Ark IAM" subtitle="平台管理控制台" menuItems={sidebarMenu} />}>
        {defaultPath ? (
          <Route index element={<Navigate to={defaultPath} replace />} />
        ) : (
          <Route index element={<EmptyAccess />} />
        )}
        {dynamicRoutes}
        {/* 详情路由（静态注册，不进侧边栏菜单） */}
        <Route path="/user/:id" element={<UserDetail />} />
        <Route path="/oauth-client/:id" element={<OAuthClientDetail />} />
      </Route>
    </Routes>
  )
}

// EmptyAccess 菜单为空（后端未下发任何权限菜单）时的空态占位，避免重定向到 404 或白屏。
function EmptyAccess() {
  return (
    <div style={{ padding: 80, textAlign: 'center', color: tokens.textPlaceholder }}>
      <div style={{ fontSize: 40 }}>🔒</div>
      <p>暂无可用菜单</p>
      <p style={{ fontSize: 13 }}>当前账号未被授予任何菜单权限，请联系平台管理员。</p>
    </div>
  )
}

export default App
