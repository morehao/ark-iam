import { useEffect, useRef, useState } from 'react'
import { Layout, Menu, Avatar, Dropdown, Button } from 'antd'
import {
  DashboardOutlined,
  UserOutlined,
  TeamOutlined,
  AppstoreOutlined,
  MenuFoldOutlined,
  MenuUnfoldOutlined,
  LogoutOutlined,
  BankOutlined,
  KeyOutlined,
} from '@ant-design/icons'
import { Outlet, useNavigate, useLocation } from 'react-router-dom'
import { getUserinfo } from '../api/auth'
import { useAuthStore } from '../stores/authStore'
import { getEndSessionURL } from '../utils/oidc'

const { Header, Sider, Content } = Layout

const MainLayout = () => {
  const [collapsed, setCollapsed] = useState(false)
  const initializedRef = useRef(false)
  const navigate = useNavigate()
  const location = useLocation()
  const authStage = useAuthStore((state) => state.authStage)
  const idToken = useAuthStore((state) => state.idToken)
  const logout = useAuthStore((state) => state.logout)
  const setPersonInfo = useAuthStore((state) => state.setPersonInfo)

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

  const handleLogout = () => {
    logout()
    if (idToken) {
      window.location.assign(getEndSessionURL(idToken))
    } else {
      navigate('/login')
    }
  }

  useEffect(() => {
    if (initializedRef.current || authStage !== 'authenticated') return
    initializedRef.current = true

    let active = true
    const loadUserContext = async () => {
      try {
        const userinfoResp = await getUserinfo()
        if (!active) return
        const personInfo = userinfoResp?.personInfo ?? null
        setPersonInfo(personInfo)
      } catch {
        return
      }
    }
    void loadUserContext()
    return () => { active = false }
  }, [authStage, setPersonInfo])

  const userMenuItems = [
    { key: 'logout', icon: <LogoutOutlined />, label: '退出登录', onClick: handleLogout },
  ]

  return (
    <Layout style={{ minHeight: '100vh' }}>
      <Sider trigger={null} collapsible collapsed={collapsed}>
        <div style={{
          height: 64,
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'center',
          color: '#fff',
          fontSize: collapsed ? 14 : 18,
          fontWeight: 'bold',
        }}>
          {collapsed ? 'IAM' : 'IAM 管理平台'}
        </div>
        <Menu
          theme="dark"
          mode="inline"
          selectedKeys={[location.pathname]}
          items={menuItems}
          onClick={({ key }) => navigate(key)}
        />
      </Sider>
      <Layout>
        <Header style={{ padding: '0 16px', background: '#fff', display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
          <Button
            type="text"
            icon={collapsed ? <MenuUnfoldOutlined /> : <MenuFoldOutlined />}
            onClick={() => setCollapsed(!collapsed)}
          />
          <div style={{ display: 'flex', alignItems: 'center', gap: 12 }}>
            <Dropdown menu={{ items: userMenuItems }} placement="bottomRight">
              <Avatar style={{ cursor: 'pointer' }} icon={<UserOutlined />} />
            </Dropdown>
          </div>
        </Header>
        <Content style={{ margin: 24, padding: 24, background: '#fff' }}>
          <Outlet />
        </Content>
      </Layout>
    </Layout>
  )
}

export default MainLayout
