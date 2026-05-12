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
} from '@ant-design/icons'
import { Outlet, useNavigate, useLocation } from 'react-router'
import { getMyTenants, getUserinfo, switchTenant } from '../api/auth'
import { useAuthStore } from '../stores/authStore'

const { Header, Sider, Content } = Layout

const MainLayout = () => {
  const [collapsed, setCollapsed] = useState(false)
  const [switchingTenantID, setSwitchingTenantID] = useState<number | null>(null)
  const initializedRef = useRef(false)
  const navigate = useNavigate()
  const location = useLocation()
  const personToken = useAuthStore((state) => state.personToken)
  const refreshToken = useAuthStore((state) => state.refreshToken)
  const tenants = useAuthStore((state) => state.tenants)
  const currentTenant = useAuthStore((state) => state.currentTenant)
  const logout = useAuthStore((state) => state.logout)
  const setPersonInfo = useAuthStore((state) => state.setPersonInfo)
  const setUserInfo = useAuthStore((state) => state.setUserInfo)
  const setTenants = useAuthStore((state) => state.setTenants)
  const setCurrentTenant = useAuthStore((state) => state.setCurrentTenant)
  const setTenantSession = useAuthStore((state) => state.setTenantSession)

  const menuItems = [
    { key: '/', icon: <DashboardOutlined />, label: '仪表盘' },
    { key: '/user', icon: <UserOutlined />, label: '用户管理' },
    { key: '/role', icon: <TeamOutlined />, label: '角色管理' },
    { key: '/department', icon: <TeamOutlined />, label: '部门管理' },
    { key: '/application', icon: <AppstoreOutlined />, label: '应用管理' },
  ]

  const handleLogout = () => {
    logout()
    navigate('/login')
  }

  useEffect(() => {
    if (initializedRef.current) {
      return
    }
    initializedRef.current = true

    let active = true

    const loadUserContext = async () => {
      let nextTenants = tenants

      try {
        const userinfoResp = await getUserinfo()
        if (!active) {
          return
        }

        const personInfo = userinfoResp.data?.personInfo ?? null
        const userInfo = userinfoResp.data?.userInfo ?? null
        setPersonInfo(personInfo)
        setUserInfo(userInfo)

        if (personToken && nextTenants.length === 0) {
          const tenantsResp = await getMyTenants({ personToken })
          if (!active) {
            return
          }

          nextTenants = tenantsResp.data?.list ?? []
          setTenants(nextTenants)
        }

        if (userInfo?.tenantID) {
          const matchedTenant = nextTenants.find((tenant) => tenant.tenantID === userInfo.tenantID)
          if (matchedTenant) {
            setCurrentTenant(matchedTenant)
          }
        }
      } catch {
        return
      }
    }

    void loadUserContext()

    return () => {
      active = false
    }
  }, [personToken, setCurrentTenant, setPersonInfo, setTenants, setUserInfo, tenants])

  const handleSwitchTenant = async (tenantID: number) => {
    const nextTenant = tenants.find((tenant) => tenant.tenantID === tenantID)
    if (!nextTenant || currentTenant?.tenantID === tenantID) {
      return
    }

    setSwitchingTenantID(tenantID)

    try {
      const switchResp = await switchTenant({ tenantID })
      setTenantSession({
        tenantToken: switchResp.data.tenantToken.accessToken,
        refreshToken: switchResp.data.tenantToken.refreshToken || refreshToken || '',
        currentTenant: nextTenant,
        userInfo: useAuthStore.getState().userInfo,
      })

      try {
        const userinfoResp = await getUserinfo()
        const personInfo = userinfoResp.data?.personInfo ?? null
        const userInfo = userinfoResp.data?.userInfo ?? null

        setPersonInfo(personInfo)
        setUserInfo(userInfo)
      } catch {
        return
      }
    } finally {
      setSwitchingTenantID(null)
    }
  }

  const userMenuItems = [
    { key: 'logout', icon: <LogoutOutlined />, label: '退出登录', onClick: handleLogout },
  ]

  const tenantMenuItems = tenants.map((tenant) => ({
    key: String(tenant.tenantID),
    label: tenant.name,
    disabled: tenant.tenantID === currentTenant?.tenantID,
  }))

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
            <Dropdown
              trigger={['click']}
              menu={{
                items: tenantMenuItems,
                onClick: ({ key }) => void handleSwitchTenant(Number(key)),
              }}
              placement="bottomRight"
            >
              <Button loading={switchingTenantID !== null}>
                {currentTenant?.name || '选择租户'}
              </Button>
            </Dropdown>
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
