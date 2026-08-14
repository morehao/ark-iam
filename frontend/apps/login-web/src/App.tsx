import { useEffect } from 'react'
import { Route, Routes, useLocation, useNavigate } from 'react-router-dom'
import { useAuthGuard, useLogout, FullPageSpinner } from '@ark-iam/auth'
import { Button, Card, Typography } from 'antd'
import { LogoutOutlined } from '@ant-design/icons'
import LoginPage from './pages/LoginPage'

function PortalHome() {
  const logout = useLogout()
  return (
    <div style={{ height: '100vh', display: 'flex', alignItems: 'center', justifyContent: 'center', background: '#f0f2f5' }}>
      <Card style={{ width: 400, textAlign: 'center' }}>
        <Typography.Title level={4}>IAM 登录门户</Typography.Title>
        <Typography.Paragraph type="secondary">您已通过统一身份认证登录。</Typography.Paragraph>
        <Button type="primary" danger icon={<LogoutOutlined />} onClick={() => void logout()} block size="large">
          全局登出
        </Button>
      </Card>
    </div>
  )
}

function App() {
  const auth = useAuthGuard()
  const location = useLocation()
  const navigate = useNavigate()

  // 未认证时跳转 OIDC 授权；/login（凭证表单）与 /auth/callback（回调处理）除外，
  // 避免登录页自身被重定向造成循环。
  useEffect(() => {
    if (
      !auth.isLoading &&
      !auth.activeNavigator &&
      !auth.isAuthenticated &&
      location.pathname !== '/login' &&
      location.pathname !== '/auth/callback'
    ) {
      void auth.signinRedirect()
    }
  }, [auth.isLoading, auth.activeNavigator, auth.isAuthenticated, location.pathname, auth.signinRedirect])

  // 已认证时从回调页回到门户首页
  useEffect(() => {
    if (auth.isAuthenticated && location.pathname === '/auth/callback') {
      navigate('/', { replace: true })
    }
  }, [auth.isAuthenticated, location.pathname, navigate])

  if (auth.isLoading || auth.activeNavigator) return <FullPageSpinner />

  return (
    <Routes>
      <Route path="/login" element={<LoginPage />} />
      <Route path="/auth/callback" element={<FullPageSpinner />} />
      <Route path="/" element={auth.isAuthenticated ? <PortalHome /> : null} />
      <Route path="*" element={null} />
    </Routes>
  )
}

export default App
