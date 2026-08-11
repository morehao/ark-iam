import { useEffect } from 'react'
import { Route, Routes, useLocation, useNavigate } from 'react-router-dom'
import { useAuthGuard, FullPageSpinner } from '@ark-iam/auth'
import { MainLayout, LoginPage } from '@ark-iam/ui'
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

const menuItems = [
  { key: '/', label: '仪表盘' },
  { key: '/user', label: '用户管理' },
  { key: '/role', label: '角色管理' },
  { key: '/department', label: '部门管理' },
  { key: '/application', label: '应用管理' },
  { key: '/tenant', label: '租户管理' },
  { key: '/tenantApplication', label: '租户应用' },
  { key: '/oauthClient', label: 'OAuth 客户端' },
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
      <Route path="/login" element={<LoginPage title="IAM 管理平台" />} />
      <Route path="/auth/callback" element={<FullPageSpinner />} />
      <Route path="/" element={<MainLayout title="IAM 管理平台" menuItems={menuItems} />}>
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
      </Route>
    </Routes>
  )
}

export default App
