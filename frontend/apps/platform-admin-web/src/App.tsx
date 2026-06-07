import { useEffect } from 'react'
import { Navigate, Route, Routes } from 'react-router-dom'
import { Spin } from 'antd'
import MainLayout from './components/MainLayout'
import AuthCallback from './pages/auth/AuthCallback'
import Login from './pages/auth/Login'
import Dashboard from './pages/dashboard'
import UserList from './pages/user'
import RoleList from './pages/role'
import DepartmentList from './pages/department'
import ApplicationList from './pages/application'
import TenantList from './pages/tenant'
import TenantApplicationList from './pages/tenantApplication'
import OAuthClientList from './pages/oauthClient'
import OAuthClientDetail from './pages/oauthClient/Detail'
import { useAuthStore } from './stores/authStore'
import { buildAuthorizeURL, generateCodeChallenge, generatePKCEParams, storePKCEParams } from './utils/oidc'

function FullPageSpinner() {
  return (
    <div style={{ height: '100vh', display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
      <Spin size="large" />
    </div>
  )
}

function App() {
  const authStage = useAuthStore((state) => state.authStage)
  const beginChecking = useAuthStore((state) => state.beginChecking)

  useEffect(() => {
    if (window.location.pathname === '/auth/callback') return
    if (authStage === 'authenticated' || authStage === 'checking') return

    let active = true
    beginChecking()

    const run = async () => {
      const params = generatePKCEParams()
      params.codeChallenge = await generateCodeChallenge(params.codeVerifier)
      if (!active) return
      storePKCEParams(params, 'silent')
      window.location.replace(buildAuthorizeURL(params, 'silent'))
    }

    void run()
    return () => {
      active = false
    }
  }, [authStage, beginChecking])

  if (authStage === 'checking') {
    return <FullPageSpinner />
  }

  return (
    <Routes>
      <Route path="/login" element={authStage === 'authenticated' ? <Navigate to="/" replace /> : <Login />} />
      <Route path="/auth/callback" element={<AuthCallback />} />
      <Route path="/" element={authStage === 'authenticated' ? <MainLayout /> : <Navigate to="/login" replace />}>
        <Route index element={<Dashboard />} />
        <Route path="user" element={<UserList />} />
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
