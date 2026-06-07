import { useEffect, useRef } from 'react'
import { Navigate, Route, Routes, useLocation } from 'react-router-dom'
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
  const markAnonymous = useAuthStore((state) => state.markAnonymous)
  const location = useLocation()
  const silentInitiatedRef = useRef(false)

  useEffect(() => {
    if (silentInitiatedRef.current) return
    if (authStage === 'authenticated' || authStage === 'checking') return
    if (location.pathname === '/auth/callback' || location.pathname === '/login') return

    silentInitiatedRef.current = true
    beginChecking()

    void (async () => {
      try {
        const params = generatePKCEParams()
        params.codeChallenge = await generateCodeChallenge(params.codeVerifier)
        storePKCEParams(params, 'silent')
        window.location.replace(buildAuthorizeURL(params, 'silent'))
      } catch {
        silentInitiatedRef.current = false
        markAnonymous()
      }
    })()
  }, [authStage, beginChecking, markAnonymous, location])

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
