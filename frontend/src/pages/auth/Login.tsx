import { useState } from 'react'
import { Form, Input, Button, Card, message } from 'antd'
import { UserOutlined, LockOutlined } from '@ant-design/icons'
import type { NavigateFunction } from 'react-router'
import { useNavigate } from 'react-router'
import { getConnectorAuthorizationUrl, login as loginApi, selectTenant } from '../../api/auth'
import { useAuthStore } from '../../stores/authStore'
import type { TenantMembership } from '../../types/auth'

const parseSsoConnectorID = (value?: string) => {
  if (!value || !/^\d+$/.test(value)) {
    return null
  }

  const connectorID = Number(value)
  if (!Number.isSafeInteger(connectorID) || connectorID <= 0) {
    return null
  }

  return connectorID
}

interface LoginFlowParams {
  personToken: {
    accessToken: string
    refreshToken: string
  }
  tenants: TenantMembership[]
  setPersonSession: (payload: {
    personToken: string
    refreshToken: string
    tenants: TenantMembership[]
  }) => void
  setTenantSession: (payload: {
    tenantToken: string
    refreshToken: string
    currentTenant: TenantMembership
  }) => void
  logout: () => void
  navigate: NavigateFunction
}

export const handleLoginSuccess = async ({
  personToken,
  tenants,
  setPersonSession,
  setTenantSession,
  logout,
  navigate,
}: LoginFlowParams) => {
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
}

const Login = () => {
  const [loading, setLoading] = useState(false)
  const [ssoLoading, setSsoLoading] = useState(false)
  const navigate = useNavigate()
  const { setPersonSession, setTenantSession, logout } = useAuthStore()
  const ssoConnectorID = parseSsoConnectorID(import.meta.env.VITE_SSO_CONNECTOR_ID)
  const hasSsoConnector = ssoConnectorID !== null

  const onFinish = async (values: { identifier: string; password: string }) => {
    setLoading(true)
    try {
      const resp = await loginApi(values)
      await handleLoginSuccess({
        personToken: resp.data.personToken,
        tenants: resp.data.tenants,
        setPersonSession,
        setTenantSession,
        logout,
        navigate,
      })
    } catch (error) {
      console.error('登录失败:', error)
    } finally {
      setLoading(false)
    }
  }

  const handleSsoLogin = async () => {
    if (!hasSsoConnector) {
      return
    }

    setSsoLoading(true)
    try {
      const redirectUri = new URL('/auth/callback', window.location.origin).toString()
      const resp = await getConnectorAuthorizationUrl({
        connectorId: ssoConnectorID,
        redirectUri,
      })

      window.location.assign(resp.data.authorizationUrl)
    } catch (error) {
      console.error('获取 SSO 授权地址失败:', error)
    } finally {
      setSsoLoading(false)
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

          {hasSsoConnector ? (
            <Form.Item>
              <Button onClick={handleSsoLogin} loading={ssoLoading} block>
                企业 SSO 登录
              </Button>
            </Form.Item>
          ) : null}

          <div style={{ textAlign: 'center' }}>
            <a onClick={() => navigate('/register')}>还没有账号？立即注册</a>
          </div>
        </Form>
      </Card>
    </div>
  )
}

export default Login
