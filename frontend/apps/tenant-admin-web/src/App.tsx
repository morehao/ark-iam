import { useEffect, useMemo, useState } from 'react'
import { Route, Routes, Navigate, useLocation, useNavigate } from 'react-router-dom'
import { useAuthGuard, FullPageSpinner } from '@ark-iam/auth'
import { MainLayout, LoginPage } from '@ark-iam/ui'
import type { MainMenuItems } from '@ark-iam/ui'
import {
  ApartmentOutlined,
  SafetyCertificateOutlined,
  TeamOutlined,
  UserSwitchOutlined,
} from '@ant-design/icons'
import type { MenuItem } from '@ark-iam/types'
import { getMyMenuTree } from './api/menu'
import OrganizationList from './pages/organization'
import OrganizationRoleList from './pages/organizationRole'
import OrganizationUserList from './pages/organizationUser'
import OrganizationRoleUserList from './pages/organizationRoleUser'

// 图标映射：后端 menu.icon 存储的字符串 -> antd 图标组件
const ICON_MAP: Record<string, React.ReactNode> = {
  apartment: <ApartmentOutlined />,
  safety: <SafetyCertificateOutlined />,
  team: <TeamOutlined />,
  'user-switch': <UserSwitchOutlined />,
}

// 组件白名单：只有 path 命中才会渲染路由与侧边栏菜单，避免点击进入 404
const COMPONENT_MAP: Record<string, React.ComponentType> = {
  '/organization': OrganizationList,
  '/organizationRole': OrganizationRoleList,
  '/organizationUser': OrganizationUserList,
  '/organizationRoleUser': OrganizationRoleUserList,
}

function iconOf(icon?: string): React.ReactNode {
  return (icon && ICON_MAP[icon]) || <ApartmentOutlined />
}

// 静态 fallback 菜单：后端不可用或未配置时保持页面可用
function makeStaticMenu(id: string, name: string, code: string, path: string, icon: string): MenuItem {
  return {
    menuID: id,
    appID: "2",
    parentID: "",
    name,
    code,
    path,
    icon,
    sort: Number(id),
    type: 'menu',
    component: '',
    redirect: '',
    hidden: 0,
    externalLink: 0,
    keepAlive: 0,
    permission: '',
    status: 'enable',
  }
}

const STATIC_MENU_TREE: MenuItem[] = [
  makeStaticMenu("1", '组织管理', 'organization', '/organization', 'apartment'),
  makeStaticMenu("2", '组织角色', 'organizationRole', '/organizationRole', 'safety'),
  makeStaticMenu("3", '组织用户', 'organizationUser', '/organizationUser', 'team'),
  makeStaticMenu("4", '组织角色用户', 'organizationRoleUser', '/organizationRoleUser', 'user-switch'),
]

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

// 找到第一个可渲染的菜单 path，作为默认首页重定向目标
function firstRoutablePath(list: MenuItem[]): string {
  for (const m of list) {
    if (COMPONENT_MAP[m.path]) return m.path
    if (m.children?.length) {
      const sub = firstRoutablePath(m.children)
      if (sub) return sub
    }
  }
  return '/organization'
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
        /* 后端不可用时保持静态 fallback */
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

  const effectiveTree = useMemo(() => {
    if (menuTree !== null && menuTree.length > 0) return menuTree
    return STATIC_MENU_TREE
  }, [menuTree])

  const sidebarMenu = useMemo(() => buildMenuItems(effectiveTree), [effectiveTree])
  const dynamicRoutes = useMemo(() => collectRoutes(effectiveTree), [effectiveTree])
  const defaultPath = useMemo(() => firstRoutablePath(effectiveTree), [effectiveTree])

  if (auth.isLoading || auth.activeNavigator) return <FullPageSpinner />
  if (!auth.isAuthenticated && location.pathname !== '/auth/callback') return null

  return (
    <Routes>
      <Route path="/login" element={<LoginPage title="租户管理" subtitle="租户自服务控制台" />} />
      <Route path="/auth/callback" element={<FullPageSpinner />} />
      <Route path="/" element={<MainLayout title="Ark IAM" subtitle="租户自服务" menuItems={sidebarMenu} />}>
        <Route index element={<Navigate to={defaultPath} replace />} />
        {dynamicRoutes}
      </Route>
    </Routes>
  )
}

export default App
