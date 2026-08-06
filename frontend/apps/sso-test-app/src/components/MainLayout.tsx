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
    // 触发 OIDC 全局登出：先 signoutRedirect 再清除本地 user。
    // signoutRedirect 需要从当前 user 读取 id_token_hint 拼入 end_session 请求，
    // 后端据此定位 personID 并撤销 Redis SSO session，实现"一处登出、处处登出"。
    // 若先 removeUser，id_token 被清空，end_session 将丢失 id_token_hint，
    // 导致后端无法定位 SSO session 而跳过撤销，兄弟应用刷新后仍保持登录。
    try {
      await auth.signoutRedirect()
    } finally {
      await auth.removeUser()
    }
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
