import { useEffect, useRef } from 'react'
import { Card, Spin, message } from 'antd'
import { useLocation, useNavigate } from 'react-router-dom'
import { clearPKCEParams, exchangeCodeForTokens, getOIDCFlowMode, loadPKCEParams } from '../../utils/oidc'
import { useAuthStore } from '../../stores/authStore'

const AuthCallback = () => {
  const location = useLocation()
  const navigate = useNavigate()
  const setAuthenticatedSession = useAuthStore((state) => state.setAuthenticatedSession)
  const markAnonymous = useAuthStore((state) => state.markAnonymous)
  const processedRef = useRef(false)

  useEffect(() => {
    if (processedRef.current) return
    processedRef.current = true
    const searchParams = new URLSearchParams(location.search)
    const code = searchParams.get('code')
    const state = searchParams.get('state')
    const error = searchParams.get('error')
    const flowMode = getOIDCFlowMode()

    if (error) {
      clearPKCEParams()
      if (flowMode === 'silent' && error === 'login_required') {
        markAnonymous()
        navigate('/login', { replace: true })
        return
      }
      message.error(`登录失败: ${searchParams.get('error_description') || error}`)
      markAnonymous()
      navigate('/login', { replace: true })
      return
    }
    if (!code || !state) {
      clearPKCEParams()
      markAnonymous()
      navigate('/login', { replace: true })
      return
    }
    const pkceParams = loadPKCEParams()
    if (!pkceParams || pkceParams.state !== state) {
      clearPKCEParams()
      markAnonymous()
      navigate('/login', { replace: true })
      return
    }
    const run = async () => {
      try {
        const resp = await exchangeCodeForTokens(code, pkceParams.codeVerifier)
        clearPKCEParams()
        setAuthenticatedSession({ accessToken: resp.access_token, idToken: resp.id_token, refreshToken: resp.refresh_token, expiresIn: resp.expires_in })
        if (flowMode !== 'silent') message.success('登录成功')
        navigate('/', { replace: true })
      } catch {
        clearPKCEParams()
        markAnonymous()
        if (flowMode !== 'silent') message.error('登录失败，请重试')
        navigate('/login', { replace: true })
      }
    }
    void run()
  }, [location.search, navigate, setAuthenticatedSession, markAnonymous])

  return (
    <div style={{ minHeight: '100vh', display: 'flex', alignItems: 'center', justifyContent: 'center', background: '#f0f2f5', padding: 24 }}>
      <Card title="正在处理登录回调" style={{ width: 400, maxWidth: '100%' }}>
        <div style={{ display: 'flex', flexDirection: 'column', alignItems: 'center', gap: 12 }}>
          <Spin />
          <span>正在处理登录回调，请稍候...</span>
        </div>
      </Card>
    </div>
  )
}

export default AuthCallback
