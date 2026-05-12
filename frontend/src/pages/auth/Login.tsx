import { useState } from 'react'
import { Form, Input, Button, Card, message } from 'antd'
import { UserOutlined, LockOutlined } from '@ant-design/icons'
import { useNavigate } from 'react-router'
import { login as loginApi } from '../../api/auth'
import { useAuthStore } from '../../stores/authStore'

const Login = () => {
  const [loading, setLoading] = useState(false)
  const navigate = useNavigate()
  const { setTokens, setTenants, setCurrentTenant } = useAuthStore()

  const onFinish = async (values: { identifier: string; password: string }) => {
    setLoading(true)
    try {
      const resp = await loginApi(values)
      const { personToken, tenants } = resp.data

      setTokens(personToken.accessToken, personToken.refreshToken)
      setTenants(tenants)

      if (tenants && tenants.length > 0) {
        setCurrentTenant(tenants[0])
      }

      message.success('登录成功')
      navigate('/')
    } catch (error) {
      console.error('登录失败:', error)
    } finally {
      setLoading(false)
    }
  }

  return (
    <div style={{
      height: '100vh',
      display: 'flex',
      alignItems: 'center',
      justifyContent: 'center',
      background: '#f0f2f5',
    }}>
      <Card title="IAM 管理平台" style={{ width: 400 }}>
        <Form
          name="login"
          onFinish={onFinish}
          autoComplete="off"
        >
          <Form.Item
            name="identifier"
            rules={[{ required: true, message: '请输入用户名/邮箱/手机号' }]}
          >
            <Input prefix={<UserOutlined />} placeholder="用户名/邮箱/手机号" />
          </Form.Item>

          <Form.Item
            name="password"
            rules={[{ required: true, message: '请输入密码' }]}
          >
            <Input.Password prefix={<LockOutlined />} placeholder="密码" />
          </Form.Item>

          <Form.Item>
            <Button type="primary" htmlType="submit" loading={loading} block>
              登录
            </Button>
          </Form.Item>

          <div style={{ textAlign: 'center' }}>
            <a onClick={() => navigate('/register')}>还没有账号？立即注册</a>
          </div>
        </Form>
      </Card>
    </div>
  )
}

export default Login