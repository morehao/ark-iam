import { useEffect, useRef, useState } from 'react'
import { Layout, Menu, Avatar, Dropdown, Button } from 'antd'
import { DashboardOutlined, UserOutlined, TeamOutlined, AppstoreOutlined, MenuFoldOutlined, MenuUnfoldOutlined, LogoutOutlined, BankOutlined, KeyOutlined } from '@ant-design/icons'
import { Outlet, useLocation, useNavigate } from 'react-router-dom'
import { useAuth } from 'react-oidc-context'
import type { PersonInfo } from '@ark-iam/shared'
import { getUserinfo, logoutAllAPI } from '../api/auth'
import { setUserProvider } from '../utils/request'

const { Header, Sider, Content } = Layout

const MainLayout = () => {
  const [collapsed, setCollapsed] = useState(false)
  const [personInfo, setPersonInfo] = useState<PersonInfo | null>(null)
  const initializedRef = useRef(false)
  const navigate = useNavigate()
  const location = useLocation()
  const auth = useAuth()

  useEffect(() => {
    setUserProvider(() => auth.user)
  }, [auth.user])

  useEffect(() => {
    if (initializedRef.current || !auth.isAuthenticated) return
    initializedRef.current = true
    let active = true
    const loadUserContext = async () => {
      try {
        const userinfoResp = await getUserinfo()
        if (!active) return
        setPersonInfo(userinfoResp?.personInfo ?? null)
      } catch {
        return
      }
    }
    void loadUserContext()
    return () => {
      active = false
    }
  }, [auth.isAuthenticated])

  const handleLogout = async () => {
    try {
      await logoutAllAPI(auth.user?.refresh_token ?? '')
    } catch {
      // ignore
    }
    // 触发 OIDC 全局登出：清除本地 user 并跳转 IdP end_session 端点，
    // 后端清除 SSO cookie 并撤销 Redis SSO session，实现"一处登出、处处登出"
    await auth.removeUser()
    window.location.href = '/v1/iam/oidc/end_session'
  }

  const menuItems = [
    { key: '/', icon: <DashboardOutlined />, label: '仪表盘' },
    { key: '/user', icon: <UserOutlined />, label: '用户管理' },
    { key: '/role', icon: <TeamOutlined />, label: '角色管理' },
    { key: '/department', icon: <TeamOutlined />, label: '部门管理' },
    { key: '/application', icon: <AppstoreOutlined />, label: '应用管理' },
    { key: '/tenant', icon: <BankOutlined />, label: '租户管理' },
    { key: '/tenantApplication', icon: <AppstoreOutlined />, label: '租户应用' },
    { key: '/oauthClient', icon: <KeyOutlined />, label: 'OAuth 客户端' },
  ]

  return (
    <Layout style={{ minHeight: '100vh' }}>
      <Sider trigger={null} collapsible collapsed={collapsed}>
        <div style={{ height: 64, display: 'flex', alignItems: 'center', justifyContent: 'center', color: '#fff', fontSize: collapsed ? 14 : 18, fontWeight: 'bold' }}>
          {collapsed ? 'IAM' : 'IAM 管理平台'}
        </div>
        <Menu theme="dark" mode="inline" selectedKeys={[location.pathname]} items={menuItems} onClick={({ key }) => navigate(key)} />
      </Sider>
      <Layout>
        <Header style={{ padding: '0 16px', background: '#fff', display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
          <Button type="text" icon={collapsed ? <MenuUnfoldOutlined /> : <MenuFoldOutlined />} onClick={() => setCollapsed(!collapsed)} />
          <Dropdown menu={{ items: [{ key: 'logout', icon: <LogoutOutlined />, label: '退出登录', onClick: handleLogout }] }} placement="bottomRight">
            <Avatar style={{ cursor: 'pointer' }} icon={<UserOutlined />} />
          </Dropdown>
        </Header>
        <Content style={{ margin: 24, padding: 24, background: '#fff' }}>
          <Outlet />
        </Content>
      </Layout>
    </Layout>
  )
}

export default MainLayout
