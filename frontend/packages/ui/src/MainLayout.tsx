import { useMemo, useState } from 'react'
import { Layout, Menu, Avatar, Dropdown, Button } from 'antd'
import type { MenuProps } from 'antd'
import {
  LogoutOutlined,
  MenuFoldOutlined,
  MenuUnfoldOutlined,
  UserOutlined,
  UserSwitchOutlined,
} from '@ant-design/icons'
import { Outlet, useLocation, useNavigate } from 'react-router-dom'
import { useAuthGuard, useLogout } from '@ark-iam/auth'
import { TenantSwitcher } from './TenantSwitcher'
import { ProfileCenter } from './ProfileCenter'

const { Header, Sider, Content } = Layout

export interface MainMenuItems {
  key: string
  icon?: React.ReactNode
  label: string
}

interface Props {
  title: string
  menuItems: MainMenuItems[]
  hasTenantSwitch?: boolean
}

export function MainLayout({ title, menuItems, hasTenantSwitch = true }: Props) {
  const [collapsed, setCollapsed] = useState(false)
  const [profileOpen, setProfileOpen] = useState(false)
  const navigate = useNavigate()
  const location = useLocation()
  useAuthGuard()
  const logout = useLogout()

  const userMenuItems = useMemo(
    () => [
      { key: 'profile', icon: <UserSwitchOutlined />, label: '个人中心', onClick: () => setProfileOpen(true) },
      { type: 'divider' as const },
      { key: 'logout', icon: <LogoutOutlined />, label: '退出登录', onClick: () => void logout() },
    ],
    [logout],
  )

  return (
    <Layout style={{ minHeight: '100vh' }}>
      <Sider trigger={null} collapsible collapsed={collapsed}>
        <div style={{ height: 64, display: 'flex', alignItems: 'center', justifyContent: 'center', color: '#fff', fontSize: collapsed ? 14 : 18, fontWeight: 'bold' }}>
          {collapsed ? 'IAM' : title}
        </div>
        <Menu theme="dark" mode="inline" selectedKeys={[location.pathname]} items={menuItems as MenuProps['items']} onClick={({ key }) => navigate(key)} />
      </Sider>
      <Layout>
        <Header style={{ padding: '0 16px', background: '#fff', display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
          <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
            <Button type="text" icon={collapsed ? <MenuUnfoldOutlined /> : <MenuFoldOutlined />} onClick={() => setCollapsed(!collapsed)} />
            {hasTenantSwitch && <TenantSwitcher />}
          </div>
          <Dropdown menu={{ items: userMenuItems }} placement="bottomRight">
            <Avatar style={{ cursor: 'pointer' }} icon={<UserOutlined />} />
          </Dropdown>
        </Header>
        <Content style={{ margin: 24, padding: 24, background: '#fff' }}>
          <Outlet />
        </Content>
      </Layout>
      <ProfileCenter open={profileOpen} onClose={() => setProfileOpen(false)} />
    </Layout>
  )
}
