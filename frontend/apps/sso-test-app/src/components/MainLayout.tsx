import { useEffect } from 'react'
import { Layout, Avatar, Dropdown } from 'antd'
import { UserOutlined, LogoutOutlined } from '@ant-design/icons'
import { Outlet } from 'react-router-dom'
import { useAuth } from 'react-oidc-context'
import { logoutAllAPI } from '../api/auth'
import { setUserProvider } from '../utils/request'

const { Header, Content } = Layout

const MainLayout = () => {
  const auth = useAuth()

  useEffect(() => {
    setUserProvider(() => auth.user)
  }, [auth.user])

  const handleLogout = async () => {
    try {
      await logoutAllAPI(auth.user?.refresh_token ?? '')
    } catch {
      // ignore：撤销自有 refresh token 为尽力而为，失败不阻断登出
    }
    // 触发 OIDC 全局登出：先清除本地 user，再用 signoutRedirect 携带
    // id_token_hint + post_logout_redirect_uri 跳转 IdP end_session 端点，
    // 后端据此定位 personID 并撤销 Redis SSO session，实现"一处登出、处处登出"
    await auth.removeUser()
    await auth.signoutRedirect()
  }

  const userMenuItems = [
    { key: 'logout', icon: <LogoutOutlined />, label: '退出登录', onClick: handleLogout },
  ]

  return (
    <Layout style={{ minHeight: '100vh' }}>
      <Header style={{ padding: '0 24px', background: '#fff', display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
        <span style={{ fontSize: 18, fontWeight: 'bold' }}>SSO 测试应用</span>
        <Dropdown menu={{ items: userMenuItems }} placement="bottomRight">
          <Avatar style={{ cursor: 'pointer' }} icon={<UserOutlined />} />
        </Dropdown>
      </Header>
      <Content style={{ margin: 24, padding: 24, background: '#fff', borderRadius: 8 }}>
        <Outlet />
      </Content>
    </Layout>
  )
}

export default MainLayout
