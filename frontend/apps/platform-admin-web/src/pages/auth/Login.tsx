import { useRef, useState } from 'react'
import { Button, Card } from 'antd'
import { KeyOutlined } from '@ant-design/icons'
import { buildAuthorizeURL, generateCodeChallenge, generatePKCEParams, storePKCEParams } from '../../utils/oidc'

const Login = () => {
  const [loading, setLoading] = useState(false)
  const genRef = useRef(0)

  const handleOIDCLogin = async () => {
    setLoading(true)
    const gen = ++genRef.current
    try {
      const params = generatePKCEParams()
      params.codeChallenge = await generateCodeChallenge(params.codeVerifier)
      if (gen !== genRef.current) return
      storePKCEParams(params, 'interactive')
      window.location.assign(buildAuthorizeURL(params, 'interactive'))
    } catch {
      setLoading(false)
    }
  }

  return (
    <div style={{ height: '100vh', display: 'flex', alignItems: 'center', justifyContent: 'center', background: '#f0f2f5' }}>
      <Card title="IAM 管理平台" style={{ width: 400 }}>
        <Button type="primary" icon={<KeyOutlined />} onClick={handleOIDCLogin} loading={loading} block size="large">
          IAM 账号登录
        </Button>
      </Card>
    </div>
  )
}

export default Login
