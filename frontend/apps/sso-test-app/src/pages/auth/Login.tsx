import { useEffect, useRef, useState } from 'react'
import { Button, Card, Spin } from 'antd'
import { KeyOutlined } from '@ant-design/icons'
import {
  generatePKCEParams,
  generateCodeChallenge,
  buildAuthorizeURL,
  storePKCEParams,
} from '../../utils/oidc'
import { useAuthStore } from '../../stores/authStore'

const SILENT_TIMEOUT = 5000

const Login = () => {
  const [loading, setLoading] = useState(false)
  const [silentChecking, setSilentChecking] = useState(true)
  const genRef = useRef(0)
  const iframeRef = useRef<HTMLIFrameElement>(null)

  useEffect(() => {
    let active = true
    const timeoutId = setTimeout(() => {
      if (active) setSilentChecking(false)
    }, SILENT_TIMEOUT)

    const handleMessage = (event: MessageEvent) => {
      if (event.origin !== window.location.origin) return
      if (event.data?.type !== 'oidc-silent') return
      if (event.data?.status === 'success' && event.data?.tokens) {
        clearTimeout(timeoutId)
        useAuthStore.getState().setAuthenticatedSession(event.data.tokens)
        window.location.href = '/'
        return
      }
      clearTimeout(timeoutId)
      if (active) setSilentChecking(false)
    }
    window.addEventListener('message', handleMessage)

    const run = async () => {
      const gen = ++genRef.current
      try {
        const params = generatePKCEParams()
        params.codeChallenge = await generateCodeChallenge(params.codeVerifier)
        if (gen !== genRef.current || !active) return
        sessionStorage.setItem('oidc_silent', '1')
        storePKCEParams(params, 'silent')
        const url = buildAuthorizeURL(params, 'silent')
        if (iframeRef.current) {
          iframeRef.current.src = url
        }
      } catch {
        if (active) setSilentChecking(false)
      }
    }
    void run().catch(() => {
      if (active) setSilentChecking(false)
    })

    return () => {
      active = false
      clearTimeout(timeoutId)
      window.removeEventListener('message', handleMessage)
    }
  }, [])

  const handleOIDCLogin = async () => {
    setLoading(true)
    const gen = ++genRef.current
    try {
      const params = generatePKCEParams()
      params.codeChallenge = await generateCodeChallenge(params.codeVerifier)
      if (gen !== genRef.current) return
      storePKCEParams(params, 'interactive')
      const url = buildAuthorizeURL(params) + '&prompt=login'
      window.location.assign(url)
    } catch {
      setLoading(false)
    }
  }

  return (
    <div
      style={{
        height: '100vh',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        background: '#f0f2f5',
      }}
    >
      <iframe
        ref={iframeRef}
        style={{ display: 'none' }}
        title="oidc-silent-check"
        sandbox="allow-scripts allow-same-origin allow-forms"
      />
      <Card title="SSO 测试应用" style={{ width: 400 }}>
        {silentChecking ? (
          <div style={{ display: 'flex', flexDirection: 'column', alignItems: 'center', gap: 12 }}>
            <Spin />
            <span style={{ color: '#888' }}>检测登录状态...</span>
          </div>
        ) : (
          <Button
            type="primary"
            icon={<KeyOutlined />}
            onClick={handleOIDCLogin}
            loading={loading}
            block
            size="large"
          >
            IAM 账号登录
          </Button>
        )}
      </Card>
    </div>
  )
}

export default Login
