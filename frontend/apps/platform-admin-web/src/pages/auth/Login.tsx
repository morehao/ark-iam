import { useEffect, useState } from 'react'
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
  const isLoggedOut = sessionStorage.getItem('logged_out') === '1'

  useEffect(() => {
    sessionStorage.removeItem('logged_out')
    document.cookie = 'iam_sso_session=; expires=Thu, 01 Jan 1970 00:00:00 UTC; path=/;'
  }, [])

  const handleOIDCLogin = async () => {
    setLoading(true)
    try {
      const params = generatePKCEParams()
      params.codeChallenge = await generateCodeChallenge(params.codeVerifier)
      storePKCEParams(params)
      let authorizeURL = buildAuthorizeURL(params)
      if (isLoggedOut) {
        authorizeURL += '&prompt=login'
      }
      window.location.assign(authorizeURL)
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
      <Card title="IAM 管理平台" style={{ width: 400 }}>
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
