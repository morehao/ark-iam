import { useEffect } from 'react'
import { Navigate, Route, Routes } from 'react-router-dom'
import { Spin } from 'antd'
import MainLayout from './components/MainLayout'
import AuthCallback from './pages/auth/AuthCallback'
import Login from './pages/auth/Login'
import Home from './pages/home'
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
    return () => { active = false }
  }, [authStage, beginChecking])

  if (authStage === 'checking') return <FullPageSpinner />

  return (
    <Routes>
      <Route path="/login" element={authStage === 'authenticated' ? <Navigate to="/" replace /> : <Login />} />
      <Route path="/auth/callback" element={<AuthCallback />} />
      <Route path="/" element={authStage === 'authenticated' ? <MainLayout /> : <Navigate to="/login" replace />}>
        <Route index element={<Home />} />
      </Route>
    </Routes>
  )
}

export default App
