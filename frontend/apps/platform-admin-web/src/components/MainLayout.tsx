import { useEffect, useState } from 'react'
import { Layout, Menu, Avatar, Dropdown, Button, message } from 'antd'
import { DashboardOutlined, UserOutlined, TeamOutlined, AppstoreOutlined, MenuFoldOutlined, MenuUnfoldOutlined, LogoutOutlined, BankOutlined, KeyOutlined, SwapOutlined } from '@ant-design/icons'
import { Outlet, useLocation, useNavigate } from 'react-router-dom'
import { useAuth } from 'react-oidc-context'
import { getMyTenants, logoutAllAPI } from '../api/auth'
import { setUserProvider } from '../utils/request'
import { getCurrentTenantId, setCurrentTenantId } from '../tenant'

const { Header, Sider, Content } = Layout

const MainLayout = () => {
  const [collapsed, setCollapsed] = useState(false)
  const [tenants, setTenants] = useState<{ tenantID: number; tenantName: string }[]>([])
  const navigate = useNavigate()
  const location = useLocation()
  const auth = useAuth()

  useEffect(() => {
    setUserProvider(() => auth.user)
    // 从当前 OIDC 用户的 tenant_id claim 同步当前租户，供 oidcConfig extraQueryParams 续期使用
    const claim = (auth.user?.profile as Record<string, unknown> | undefined)?.tenant_id
    if (claim != null && getCurrentTenantId() !== String(claim)) {
      setCurrentTenantId(claim as string | number)
    }
  }, [auth.user])

  const handleSwitchTenant = async (tenantID: number) => {
    if (String(tenantID) === getCurrentTenantId()) return
    setCurrentTenantId(tenantID)
    try {
      await auth.removeUser()
    } catch {
      // 本地清理失败不阻断重授权
    }
    await auth.signinRedirect({ extraQueryParams: { tenant: String(tenantID) }, redirectMethod: 'replace' })
  }

  const loadTenants = async () => {
    try {
      const resp = await getMyTenants()
      setTenants(resp.list || [])
    } catch {
      message.warning('获取可用租户失败')
    }
  }

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
          <Dropdown
            menu={{
              items: [
                {
                  key: 'tenant',
                  icon: <SwapOutlined />,
                  label: '切换租户',
                  children: tenants.map((t) => ({
                    key: `tenant-${t.tenantID}`,
                    label: t.tenantName,
                    disabled: String(t.tenantID) === getCurrentTenantId(),
                    onClick: () => handleSwitchTenant(t.tenantID),
                  })),
                },
                { type: 'divider' },
                { key: 'logout', icon: <LogoutOutlined />, label: '退出登录', onClick: handleLogout },
              ],
            }}
            placement="bottomRight"
            onOpenChange={(open) => {
              if (open) loadTenants()
            }}
          >
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
