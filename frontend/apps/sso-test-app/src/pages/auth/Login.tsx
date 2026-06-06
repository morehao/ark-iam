import { useEffect, useRef, useState } from 'react'
import { Button, Card } from 'antd'
import { KeyOutlined } from '@ant-design/icons'
import {
  generatePKCEParams,
  generateCodeChallenge,
  buildAuthorizeURL,
  storePKCEParams,
} from '../../utils/oidc'

const Login = () => {
  const [loading, setLoading] = useState(false)
  const genRef = useRef(0)

  useEffect(() => {
    sessionStorage.removeItem('logged_out')
  }, [])

  const handleOIDCLogin = async () => {
    setLoading(true)
    const gen = ++genRef.current
    try {
      const params = generatePKCEParams()
      params.codeChallenge = await generateCodeChallenge(params.codeVerifier)
      if (gen !== genRef.current) return
      storePKCEParams(params)
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
      <Card title="SSO 测试应用" style={{ width: 400 }}>
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
      </Card>
    </div>
  )
}

export default Login
