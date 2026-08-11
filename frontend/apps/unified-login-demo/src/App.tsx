import { useEffect } from 'react'
import { Route, Routes, useLocation, useNavigate } from 'react-router-dom'
import { useAuthGuard, FullPageSpinner } from '@ark-iam/auth'
import { MainLayout, LoginPage } from '@ark-iam/ui'
import Home from './pages/home'

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
      <Route path="/login" element={<LoginPage title="统一登录演示应用" />} />
      <Route path="/auth/callback" element={<FullPageSpinner />} />
      <Route path="/" element={<MainLayout title="统一登录演示应用" menuItems={[]} hasTenantSwitch={false} />}>
        <Route index element={<Home />} />
      </Route>
    </Routes>
  )
}

export default App
