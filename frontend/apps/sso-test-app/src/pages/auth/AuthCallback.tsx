import { useEffect, useRef } from 'react'
import { Card, Spin, message } from 'antd'
import { useLocation, useNavigate } from 'react-router-dom'
import {
  loadPKCEParams,
  clearPKCEParams,
  exchangeCodeForTokens,
} from '../../utils/oidc'
import { useAuthStore } from '../../stores/authStore'

const AuthCallback = () => {
  const location = useLocation()
  const navigate = useNavigate()
  const { setSession } = useAuthStore()
  const processedRef = useRef(false)

  useEffect(() => {
    if (processedRef.current) return
    processedRef.current = true

    const searchParams = new URLSearchParams(location.search)
    const code = searchParams.get('code')
    const state = searchParams.get('state')
    const error = searchParams.get('error')

    if (error) {
      if (error === 'login_required') {
        sessionStorage.setItem('oidc_silent_failed', '1')
      } else {
        message.error(`登录失败: ${searchParams.get('error_description') || error}`)
      }
      clearPKCEParams()
      navigate('/login', { replace: true })
      return
    }

    if (!code || !state) {
      clearPKCEParams()
      navigate('/login', { replace: true })
      return
    }

    const pkceParams = loadPKCEParams()
    if (!pkceParams || pkceParams.state !== state) {
      clearPKCEParams()
      navigate('/login', { replace: true })
      return
    }

    const run = async () => {
      try {
        const resp = await exchangeCodeForTokens(code, pkceParams.codeVerifier)
        clearPKCEParams()
        sessionStorage.removeItem('oidc_silent_failed')
        setSession({
          accessToken: resp.access_token,
          idToken: resp.id_token,
          refreshToken: resp.refresh_token,
          expiresIn: resp.expires_in,
        })
        message.success('登录成功')
        navigate('/', { replace: true })
      } catch {
        clearPKCEParams()
        message.error('登录失败，请重试')
        navigate('/login', { replace: true })
      }
    }

    void run()
  }, [location.search, navigate, setSession])

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
      <Card title="正在完成登录" style={{ width: 400, maxWidth: '100%' }}>
        <div style={{ display: 'flex', flexDirection: 'column', alignItems: 'center', gap: 12 }}>
          <Spin />
          <span>正在处理登录回调，请稍候...</span>
        </div>
      </Card>
    </div>
  )
}

export default AuthCallback
