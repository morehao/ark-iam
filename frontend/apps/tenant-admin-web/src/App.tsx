import { useEffect, useMemo, useState } from 'react'
import { Route, Routes, Navigate, useLocation, useNavigate } from 'react-router-dom'
import { useAuthGuard, FullPageSpinner } from '@ark-iam/auth'
import { MainLayout, LoginPage, tokens } from '@ark-iam/ui'
import type { MainMenuItems } from '@ark-iam/ui'
import { ApartmentOutlined, SafetyCertificateOutlined, TeamOutlined, UserOutlined } from '@ant-design/icons'
import type { MenuItem } from '@ark-iam/types'
import { getMyMenuTree } from './api/menu'
import OrganizationList from './pages/organization'
import OrganizationMembersList from './pages/organization-members'
import TenantUserList from './pages/user'
import TenantRoleList from './pages/role'

// 图标映射：后端 menu.icon 存储的字符串 -> antd 图标组件
const ICON_MAP: Record<string, React.ReactNode> = {
  apartment: <ApartmentOutlined />,
  user: <UserOutlined />,
  role: <SafetyCertificateOutlined />,
  team: <TeamOutlined />,
}

// 组件白名单：只有 path 命中才会渲染路由与侧边栏菜单，避免点击进入 404
const COMPONENT_MAP: Record<string, React.ComponentType> = {
  '/organization': OrganizationList,
  '/organization/members': OrganizationMembersList,
  '/user': TenantUserList,
  '/role': TenantRoleList,
}

function iconOf(icon?: string): React.ReactNode {
  return (icon && ICON_MAP[icon]) || <ApartmentOutlined />
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

// 从菜单树收集可渲染路由
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

// 找到第一个可渲染的菜单 path，作为默认首页重定向目标；无可用菜单时返回空串（由调用方渲染空态）。
function firstRoutablePath(list: MenuItem[]): string {
  for (const m of list) {
    if (COMPONENT_MAP[m.path]) return m.path
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
        // 后端不可用：保持菜单为空（不做静态 fallback，管理入口绝不凭空出现）
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
  if (!menuLoaded) return <FullPageSpinner />

  return (
    <Routes>
      <Route path="/login" element={<LoginPage title="租户管理" subtitle="租户自服务控制台" />} />
      <Route path="/auth/callback" element={<FullPageSpinner />} />
      <Route path="/" element={<MainLayout title="Ark IAM" subtitle="租户自服务" menuItems={sidebarMenu} />}>
        {defaultPath ? (
          <Route index element={<Navigate to={defaultPath} replace />} />
        ) : (
          <Route index element={<EmptyAccess />} />
        )}
        {dynamicRoutes}
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
      <p style={{ fontSize: 13 }}>当前账号未被授予任何菜单权限，请联系租户管理员。</p>
    </div>
  )
}

export default App
