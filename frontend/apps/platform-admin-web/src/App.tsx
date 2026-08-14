import { useEffect } from 'react'
import { Route, Routes, useLocation, useNavigate } from 'react-router-dom'
import { useAuthGuard, FullPageSpinner } from '@ark-iam/auth'
import { MainLayout, LoginPage } from '@ark-iam/ui'
import type { MainMenuItems } from '@ark-iam/ui'
import {
  ApartmentOutlined,
  AppstoreOutlined,
  DashboardOutlined,
  FileSearchOutlined,
  GlobalOutlined,
  KeyOutlined,
  MenuOutlined,
  ProfileOutlined,
  SafetyCertificateOutlined,
  SettingOutlined,
  ShoppingOutlined,
  TeamOutlined,
  UserOutlined,
} from '@ant-design/icons'
import Dashboard from './pages/dashboard'
import UserList from './pages/user'
import UserDetail from './pages/user/Detail'
import RoleList from './pages/role'
import DepartmentList from './pages/department'
import ApplicationList from './pages/application'
import TenantList from './pages/tenant'
import TenantApplicationList from './pages/tenantApplication'
import OAuthClientList from './pages/oauthClient'
import OAuthClientDetail from './pages/oauthClient/Detail'
import ApiKeyList from './pages/apiKey'
import MenuList from './pages/menu'
import ScopeList from './pages/scope'
import ResourceList from './pages/resource'
import DomainList from './pages/domain'
import SystemList from './pages/system'
import LogList from './pages/log'

const menuItems: MainMenuItems[] = [
  { key: '/', icon: <DashboardOutlined />, label: '仪表盘' },
  {
    key: 'grp-identity',
    icon: <TeamOutlined />,
    label: '用户与权限',
    children: [
      { key: '/user', icon: <UserOutlined />, label: '用户管理' },
      { key: '/role', icon: <SafetyCertificateOutlined />, label: '角色管理' },
      { key: '/menu', icon: <MenuOutlined />, label: '菜单管理' },
      { key: '/scope', icon: <ProfileOutlined />, label: '权限域' },
      { key: '/resource', icon: <FileSearchOutlined />, label: '资源' },
    ],
  },
  {
    key: 'grp-org',
    icon: <ApartmentOutlined />,
    label: '组织与租户',
    children: [
      { key: '/department', icon: <ApartmentOutlined />, label: '部门管理' },
      { key: '/tenant', icon: <GlobalOutlined />, label: '租户管理' },
      { key: '/tenantApplication', icon: <ShoppingOutlined />, label: '租户应用' },
    ],
  },
  {
    key: 'grp-app',
    icon: <AppstoreOutlined />,
    label: '应用与接入',
    children: [
      { key: '/application', icon: <AppstoreOutlined />, label: '应用管理' },
      { key: '/oauthClient', icon: <KeyOutlined />, label: 'OAuth 客户端' },
      { key: '/domain', icon: <GlobalOutlined />, label: '域名管理' },
    ],
  },
  {
    key: 'grp-ops',
    icon: <SettingOutlined />,
    label: '安全与运维',
    children: [
      { key: '/apiKey', icon: <KeyOutlined />, label: 'API Key' },
      { key: '/system', icon: <SettingOutlined />, label: '系统配置' },
      { key: '/log', icon: <FileSearchOutlined />, label: '审计日志' },
    ],
  },
]

function App() {
  const auth = useAuthGuard()
  const location = useLocation()
  const navigate = useNavigate()

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

  if (auth.isLoading || auth.activeNavigator) return <FullPageSpinner />
  if (!auth.isAuthenticated && location.pathname !== '/auth/callback') return null

  return (
    <Routes>
      <Route path="/login" element={<LoginPage title="IAM 管理平台" subtitle="平台管理控制台" />} />
      <Route path="/auth/callback" element={<FullPageSpinner />} />
      <Route path="/" element={<MainLayout title="Ark IAM" subtitle="平台管理控制台" menuItems={menuItems} />}>
        <Route index element={<Dashboard />} />
        <Route path="user" element={<UserList />} />
        <Route path="user/:id" element={<UserDetail />} />
        <Route path="role" element={<RoleList />} />
        <Route path="department" element={<DepartmentList />} />
        <Route path="application" element={<ApplicationList />} />
        <Route path="tenant" element={<TenantList />} />
        <Route path="tenantApplication" element={<TenantApplicationList />} />
        <Route path="oauthClient" element={<OAuthClientList />} />
        <Route path="oauthClient/:id" element={<OAuthClientDetail />} />
        <Route path="apiKey" element={<ApiKeyList />} />
        <Route path="menu" element={<MenuList />} />
        <Route path="scope" element={<ScopeList />} />
        <Route path="resource" element={<ResourceList />} />
        <Route path="domain" element={<DomainList />} />
        <Route path="system" element={<SystemList />} />
        <Route path="log" element={<LogList />} />
      </Route>
    </Routes>
  )
}

export default App
