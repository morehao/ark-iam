import { useEffect } from 'react'
import { Card, Spin, message } from 'antd'
import { useLocation, useNavigate } from 'react-router'
import { completeConnectorCallback } from '../../api/auth'
import { useAuthStore } from '../../stores/authStore'
import { handleLoginSuccess } from './Login'

const AuthCallback = () => {
  const location = useLocation()
  const navigate = useNavigate()
  const { setPersonSession, setTenantSession, logout } = useAuthStore()

  useEffect(() => {
    const searchParams = new URLSearchParams(location.search)
    const code = searchParams.get('code')
    const state = searchParams.get('state')

    if (!code || !state) {
      message.error('SSO 回调参数缺失，请重新登录')
      navigate('/login', { replace: true })
      return
    }

    let cancelled = false

    const run = async () => {
      try {
        const resp = await completeConnectorCallback({ code, state })
        if (cancelled) {
          return
        }

        await handleLoginSuccess({
          personToken: resp.data.personToken,
          tenants: resp.data.tenants,
          setPersonSession,
          setTenantSession,
          logout,
          navigate,
        })
      } catch (error) {
        if (!cancelled) {
          console.error('SSO 回调处理失败:', error)
          navigate('/login', { replace: true })
        }
      }
    }

    void run()

    return () => {
      cancelled = true
    }
  }, [location.search, logout, navigate, setPersonSession, setTenantSession])

  return (
    <div
      style={{
        minHeight: '100vh',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        background: '#f0f2f5',
        padding: 24,
      }}
    >
      <Card title="正在完成企业登录" style={{ width: 400, maxWidth: '100%' }}>
        <div style={{ display: 'flex', flexDirection: 'column', alignItems: 'center', gap: 12 }}>
          <Spin />
          <span>正在处理 SSO 回调，请稍候...</span>
        </div>
      </Card>
    </div>
  )
}

export default AuthCallback
