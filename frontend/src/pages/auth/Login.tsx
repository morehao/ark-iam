import { useState } from 'react'
import { Form, Input, Button, Card, message } from 'antd'
import { UserOutlined, LockOutlined } from '@ant-design/icons'
import { useNavigate } from 'react-router'
import { login as loginApi, selectTenant } from '../../api/auth'
import { useAuthStore } from '../../stores/authStore'

const Login = () => {
  const [loading, setLoading] = useState(false)
  const navigate = useNavigate()
  const { setPersonSession, setTenantSession, logout } = useAuthStore()

  const onFinish = async (values: { identifier: string; password: string }) => {
    setLoading(true)
    try {
      const resp = await loginApi(values)
      const { personToken, tenants } = resp.data

      setPersonSession({
        personToken: personToken.accessToken,
        refreshToken: personToken.refreshToken,
        tenants,
      })

      if (!tenants || tenants.length === 0) {
        message.error('当前账号未关联租户，请重新登录')
        logout()
        navigate('/login', { replace: true })
        return
      }

      if (tenants.length === 1) {
        const currentTenant = tenants[0]
        const tenantResp = await selectTenant({
          personToken: personToken.accessToken,
          tenantID: currentTenant.tenantID,
        })

        setTenantSession({
          tenantToken: tenantResp.data.tenantToken.accessToken,
          refreshToken: tenantResp.data.tenantToken.refreshToken,
          currentTenant,
        })

        message.success('登录成功')
        navigate('/', { replace: true })
        return
      }

      message.success('登录成功，请选择租户')
      navigate('/select-tenant', { replace: true })
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
