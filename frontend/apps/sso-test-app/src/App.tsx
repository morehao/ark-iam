import { useEffect, useRef, useState } from 'react'
import { Routes, Route, Navigate } from 'react-router-dom'
import { Spin } from 'antd'
import MainLayout from './components/MainLayout'
import AuthCallback from './pages/auth/AuthCallback'
import Login from './pages/auth/Login'
import Home from './pages/home'
import { useAuthStore } from './stores/authStore'
import { generatePKCEParams, generateCodeChallenge, buildSilentAuthorizeURL, storePKCEParams } from './utils/oidc'

function FullPageSpinner() {
  return (
    <div style={{ height: '100vh', display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
      <Spin size="large" />
    </div>
  )
}

function App() {
  const { authStage } = useAuthStore()
  const [isChecking, setIsChecking] = useState(false)
  const genRef = useRef(0)
  const searchParams = new URLSearchParams(window.location.search)
  const isCallback = searchParams.has('code') || searchParams.has('error')

  useEffect(() => {
    if (isCallback) return
    if (authStage === 'authenticated') return

    if (sessionStorage.getItem('logged_out') === '1') return

    if (sessionStorage.getItem('oidc_silent_failed') === '1') {
      sessionStorage.removeItem('oidc_silent_failed')
      return
    }

    const gen = ++genRef.current
    setIsChecking(true)

    const run = async () => {
      const params = generatePKCEParams()
      params.codeChallenge = await generateCodeChallenge(params.codeVerifier)
      if (gen !== genRef.current) return
      storePKCEParams(params)
      const url = buildSilentAuthorizeURL(params)
      window.location.replace(url)
    }
    void run()
  }, [])

  if (isCallback) {
    return <AuthCallback />
  }

  if (isChecking) {
    return <FullPageSpinner />
  }

  return (
    <Routes>
      <Route
        path="/login"
        element={authStage === 'authenticated' ? <Navigate to="/" replace /> : <Login />}
      />
      <Route
        path="/"
        element={authStage === 'authenticated' ? <MainLayout /> : <Navigate to="/login" replace />}
      >
        <Route index element={<Home />} />
      </Route>
    </Routes>
  )
}

export default App
