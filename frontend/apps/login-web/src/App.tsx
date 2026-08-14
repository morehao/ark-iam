import { useEffect } from 'react'
import { Route, Routes, useLocation, useNavigate } from 'react-router-dom'
import { useAuthGuard, useLogout, FullPageSpinner } from '@ark-iam/auth'
import { Button, Card, Typography } from 'antd'
import { LogoutOutlined, SafetyCertificateOutlined } from '@ant-design/icons'
import LoginPage from './pages/LoginPage'

function PortalHome() {
  const logout = useLogout()
  return (
    <div
      style={{
        minHeight: '100vh',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        background: 'linear-gradient(135deg, #eef2ff 0%, #f5f0ff 100%)',
      }}
    >
      <Card style={{ width: 420, textAlign: 'center', borderRadius: 16, boxShadow: '0 20px 60px rgba(15,23,42,0.12)' }} styles={{ body: { padding: '40px 36px' } }}>
        <div
          style={{
            width: 64,
            height: 64,
            margin: '0 auto 20px',
            borderRadius: 18,
            background: 'linear-gradient(135deg, #4f6ef7 0%, #7a5af8 55%, #a855f7 100%)',
            color: '#fff',
            fontSize: 26,
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
          }}
        >
          <SafetyCertificateOutlined />
        </div>
        <Typography.Title level={4} style={{ marginBottom: 8 }}>
          IAM 登录门户
        </Typography.Title>
        <Typography.Paragraph type="secondary" style={{ marginBottom: 28 }}>
          您已通过统一身份认证登录，可安全退出或返回应用。
        </Typography.Paragraph>
        <Button danger size="large" icon={<LogoutOutlined />} onClick={() => void logout()} block style={{ height: 46, borderRadius: 10, fontWeight: 600 }}>
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
