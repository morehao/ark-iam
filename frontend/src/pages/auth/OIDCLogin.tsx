import { useState, useEffect } from 'react'
import { Form, Input, Button, Card, message } from 'antd'
import { UserOutlined, LockOutlined } from '@ant-design/icons'
import { oidcLogin } from '../../api/auth'

const OIDCLogin = () => {
  const [loading, setLoading] = useState(false)
  const [authRequestID, setAuthRequestID] = useState<string | null>(null)

  useEffect(() => {
    const params = new URLSearchParams(window.location.search)
    const id = params.get('authRequestID')
    if (!id) {
      message.error('缺少认证请求 ID')
    }
    setAuthRequestID(id)
  }, [])

  const onFinish = async (values: { identifier: string; password: string }) => {
    if (!authRequestID) {
      message.error('缺少认证请求 ID')
      return
    }

    setLoading(true)
    try {
      const resp = await oidcLogin({
        authRequestID,
        identifier: values.identifier,
        password: values.password,
      })
      window.location.href = resp.continueURL
    } catch (error) {
      console.error('OIDC 登录失败:', error)
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
      <Card title="OIDC 登录" style={{ width: 400 }}>
        <Form
          name="oidcLogin"
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
        </Form>
      </Card>
    </div>
  )
}

export default OIDCLogin
