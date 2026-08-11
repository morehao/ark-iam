import { useEffect } from 'react'
import { Route, Routes, Navigate, useLocation, useNavigate } from 'react-router-dom'
import { useAuthGuard, FullPageSpinner } from '@ark-iam/auth'
import { MainLayout, LoginPage } from '@ark-iam/ui'
import OrganizationList from './pages/organization'
import OrganizationRoleList from './pages/organizationRole'
import OrganizationUserList from './pages/organizationUser'
import OrganizationRoleUserList from './pages/organizationRoleUser'

const menuItems = [
  { key: '/organization', label: '组织管理' },
  { key: '/organizationRole', label: '组织角色' },
  { key: '/organizationUser', label: '组织用户' },
  { key: '/organizationRoleUser', label: '组织角色用户' },
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
      <Route path="/login" element={<LoginPage title="租户管理" />} />
      <Route path="/auth/callback" element={<FullPageSpinner />} />
      <Route path="/" element={<MainLayout title="租户管理" menuItems={menuItems} />}>
        <Route index element={<Navigate to="/organization" replace />} />
        <Route path="organization" element={<OrganizationList />} />
        <Route path="organizationRole" element={<OrganizationRoleList />} />
        <Route path="organizationUser" element={<OrganizationUserList />} />
        <Route path="organizationRoleUser" element={<OrganizationRoleUserList />} />
      </Route>
    </Routes>
  )
}

export default App
