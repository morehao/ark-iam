import { Button, Card } from 'antd'
import { KeyOutlined } from '@ant-design/icons'
import { useAuth } from 'react-oidc-context'

const Login = () => {
  const auth = useAuth()

  return (
    <div style={{ height: '100vh', display: 'flex', alignItems: 'center', justifyContent: 'center', background: '#f0f2f5' }}>
      <Card title="IAM 管理平台" style={{ width: 400 }}>
        <Button
          type="primary"
          icon={<KeyOutlined />}
          onClick={() => void auth.signinRedirect()}
          loading={auth.isLoading}
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
