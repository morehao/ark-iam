import { useEffect } from 'react'
import { Route, Routes, useLocation } from 'react-router-dom'
import { Spin } from 'antd'
import { useAuth } from 'react-oidc-context'
import MainLayout from './components/MainLayout'
import Login from './pages/auth/Login'
import Home from './pages/home'

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

  useEffect(() => {
    if (!auth.isLoading && !auth.activeNavigator && !auth.isAuthenticated && location.pathname !== '/auth/callback') {
      void auth.signinRedirect()
    }
  }, [auth.isLoading, auth.activeNavigator, auth.isAuthenticated, location.pathname, auth.signinRedirect])

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
