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
  const { setAuthenticatedSession } = useAuthStore()
  const processedRef = useRef(false)
  const isInIframe = window.self !== window.top

  useEffect(() => {
    if (processedRef.current) return
    processedRef.current = true

    const searchParams = new URLSearchParams(location.search)
    const code = searchParams.get('code')
    const state = searchParams.get('state')
    const error = searchParams.get('error')

    if (error) {
      if (error === 'login_required') {
        if (isInIframe) {
          sessionStorage.removeItem('oidc_silent')
          clearPKCEParams()
          window.parent.postMessage(
            { type: 'oidc-silent', status: 'expired' },
            window.location.origin
          )
          return
        }
      } else {
        message.error(`登录失败: ${searchParams.get('error_description') || error}`)
      }
      clearPKCEParams()
      if (!isInIframe) navigate('/login', { replace: true })
      return
    }

    if (!code || !state) {
      clearPKCEParams()
      if (isInIframe) {
        sessionStorage.removeItem('oidc_silent')
        window.parent.postMessage(
          { type: 'oidc-silent', status: 'expired' },
          window.location.origin
        )
      } else {
        navigate('/login', { replace: true })
      }
      return
    }

    const pkceParams = loadPKCEParams()
    if (!pkceParams || pkceParams.state !== state) {
      clearPKCEParams()
      if (isInIframe) {
        sessionStorage.removeItem('oidc_silent')
        window.parent.postMessage(
          { type: 'oidc-silent', status: 'expired' },
          window.location.origin
        )
      } else {
        navigate('/login', { replace: true })
      }
      return
    }

    const isSilent = sessionStorage.getItem('oidc_silent') === '1'

    const run = async () => {
      try {
        const resp = await exchangeCodeForTokens(code, pkceParams.codeVerifier)
        clearPKCEParams()

        if (isInIframe && isSilent) {
          sessionStorage.removeItem('oidc_silent')
          window.parent.postMessage(
            {
              type: 'oidc-silent',
              status: 'success',
              tokens: {
                accessToken: resp.access_token,
                idToken: resp.id_token,
                refreshToken: resp.refresh_token,
                expiresIn: resp.expires_in,
              },
            },
            window.location.origin
          )
          return
        }

        setAuthenticatedSession({
          accessToken: resp.access_token,
          idToken: resp.id_token,
          refreshToken: resp.refresh_token,
          expiresIn: resp.expires_in,
        })
        message.success('登录成功')
        navigate('/', { replace: true })
      } catch {
        clearPKCEParams()
        if (isInIframe) {
          window.parent.postMessage(
            { type: 'oidc-silent', status: 'expired' },
            window.location.origin
          )
        } else {
          message.error('登录失败，请重试')
          navigate('/login', { replace: true })
        }
      }
    }

    void run()
  }, [location.search, navigate, setAuthenticatedSession, isInIframe])

  if (isInIframe) return null

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
