import { useEffect, useRef, useState } from 'react'
import { Layout, Menu, Avatar, Dropdown, Button } from 'antd'
import { DashboardOutlined, UserOutlined, TeamOutlined, AppstoreOutlined, MenuFoldOutlined, MenuUnfoldOutlined, LogoutOutlined, BankOutlined, KeyOutlined } from '@ant-design/icons'
import { Outlet, useLocation, useNavigate } from 'react-router-dom'
import { getUserinfo, logoutAPI } from '../api/auth'
import { useAuthStore } from '../stores/authStore'
import { buildAuthorizeURL, generateCodeChallenge, generatePKCEParams, getEndSessionURL, storePKCEParams } from '../utils/oidc'

const { Header, Sider, Content } = Layout

const MainLayout = () => {
  const [collapsed, setCollapsed] = useState(false)
  const initializedRef = useRef(false)
  const checkingRef = useRef(false)
  const navigate = useNavigate()
  const location = useLocation()
  const authStage = useAuthStore((state) => state.authStage)
  const clearSession = useAuthStore((state) => state.clearSession)
  const setPersonInfo = useAuthStore((state) => state.setPersonInfo)

  const triggerSilentProbe = async () => {
    if (checkingRef.current) return
    checkingRef.current = true
    try {
      const params = generatePKCEParams()
      params.codeChallenge = await generateCodeChallenge(params.codeVerifier)
      storePKCEParams(params, 'silent')
      window.location.replace(buildAuthorizeURL(params, 'silent'))
    } finally {
      checkingRef.current = false
    }
  }

  const handleLogout = async () => {
    const store = useAuthStore.getState()
    const currentIdToken = store.idToken
    try {
      await logoutAPI(store.refreshToken ?? '')
    } catch {
      // ignore
    }
    clearSession()
    if (currentIdToken) {
      window.location.assign(getEndSessionURL(currentIdToken))
      return
    }
    window.location.assign('/login')
  }

  useEffect(() => {
    if (initializedRef.current || authStage !== 'authenticated') return
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
    const handleVisibilityChange = () => {
      if (document.visibilityState === 'visible') {
        void triggerSilentProbe()
      }
    }
    document.addEventListener('visibilitychange', handleVisibilityChange)
    return () => {
      active = false
      document.removeEventListener('visibilitychange', handleVisibilityChange)
    }
  }, [authStage, setPersonInfo])

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
