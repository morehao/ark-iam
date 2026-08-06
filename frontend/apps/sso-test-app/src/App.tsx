import { useEffect } from 'react'
import { Route, Routes, useLocation, useNavigate } from 'react-router-dom'
import { Spin } from 'antd'
import { useAuth } from 'react-oidc-context'
import MainLayout from './components/MainLayout'
import Login from './pages/auth/Login'
import Home from './pages/home'
import { useSSOSessionProbe } from './utils/ssoSessionProbe'

function FullPageSpinner() {
  return (
    <div style={{ height: '100vh', display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
      <Spin size="large" />
    </div>
  )
}

function App() {
  const auth = useAuth()
  const location = useLocation()
  const navigate = useNavigate()

  // 页面加载时校验 SSO 会话是否仍有效，实现"一处登出、处处登出"
  useSSOSessionProbe(auth)

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

  if (auth.isLoading || auth.activeNavigator) {
    return <FullPageSpinner />
  }

  if (!auth.isAuthenticated && location.pathname !== '/auth/callback') {
    return null
  }

  return (
    <Routes>
      <Route path="/login" element={<Login />} />
      <Route path="/auth/callback" element={<FullPageSpinner />} />
      <Route path="/" element={<MainLayout />}>
        <Route index element={<Home />} />
      </Route>
    </Routes>
  )
}

export default App
