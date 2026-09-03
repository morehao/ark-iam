import { forwardRef, useMemo, useState } from 'react'
import { Layout, Menu, Avatar, Dropdown, Tooltip, Badge, theme } from 'antd'
import type { MenuProps } from 'antd'
import {
  LogoutOutlined,
  MenuFoldOutlined,
  MenuUnfoldOutlined,
  SafetyCertificateOutlined,
  UserOutlined,
  UserSwitchOutlined,
} from '@ant-design/icons'
import { Outlet, useLocation, useNavigate } from 'react-router-dom'
import { useAuthGuard, useLogout } from '@ark-iam/auth'
import { useAuth } from 'react-oidc-context'
import { TenantSwitcher } from './TenantSwitcher'
import { ProfileCenter } from './ProfileCenter'
import { brand, tokens } from './theme'

const { Header, Sider, Content } = Layout

export interface MainMenuItems {
  key: string
  icon?: React.ReactNode
  label: string
  children?: MainMenuItems[]
}

interface Props {
  title: string
  subtitle?: string
  menuItems: MainMenuItems[]
  hasTenantSwitch?: boolean
}

export function MainLayout({ title, subtitle, menuItems, hasTenantSwitch = true }: Props) {
  const [collapsed, setCollapsed] = useState(false)
  const [profileOpen, setProfileOpen] = useState(false)
  const navigate = useNavigate()
  const location = useLocation()
  const auth = useAuth()
  useAuthGuard()
  const logout = useLogout()
  const { token } = theme.useToken()

  const displayName = useMemo(() => {
    const profile = auth.user?.profile as Record<string, unknown> | undefined
    return (profile?.name as string) || (profile?.preferred_username as string) || 'IAM 用户'
  }, [auth.user])

  const userMenuItems = useMemo(
    () => [
      { key: 'profile', icon: <UserSwitchOutlined />, label: '个人中心', onClick: () => setProfileOpen(true) },
      { type: 'divider' as const },
      { key: 'logout', icon: <LogoutOutlined />, label: '退出登录', onClick: () => void logout() },
    ],
    [logout],
  )

  // 菜单选中态：匹配一级或二级路径
  const selectedKey = useMemo(() => {
    const parts = location.pathname.split('/')
    const level1 = parts[1] ? `/${parts[1]}` : '/'
    const level2 = parts.length > 2 ? `/${parts[1]}/${parts[2]}` : ''
    const flat = (items: MainMenuItems[]): string[] => items.flatMap((i) => [i.key, ...(i.children ? flat(i.children) : [])])
    const all = flat(menuItems)
    if (all.includes(level2)) return level2
    if (all.includes(level1)) return level1
    return '/'
  }, [location.pathname, menuItems])

  return (
    <Layout style={{ minHeight: '100vh' }}>
      <Sider
        trigger={null}
        collapsible
        collapsed={collapsed}
        width={232}
        theme="dark"
        style={{ position: 'sticky', top: 0, height: '100vh' }}
      >
        <div
          style={{
            height: 64,
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            gap: 10,
            background: brand.gradient,
            color: '#fff',
            overflow: 'hidden',
          }}
        >
          <SafetyCertificateOutlined style={{ fontSize: collapsed ? 20 : 24 }} />
          {!collapsed && (
            <div style={{ lineHeight: 1.1 }}>
              <div style={{ fontSize: 16, fontWeight: 700, letterSpacing: 0.5 }}>{title}</div>
              {subtitle && <div style={{ fontSize: 11, opacity: 0.8 }}>{subtitle}</div>}
            </div>
          )}
        </div>
        <Menu
          theme="dark"
          mode="inline"
          selectedKeys={[selectedKey]}
          defaultOpenKeys={[selectedKey]}
          items={menuItems as MenuProps['items']}
          onClick={({ key }) => {
            if (key !== location.pathname) navigate(key)
          }}
          style={{ padding: '8px 10px', borderInlineEnd: 'none' }}
        />
      </Sider>
      <Layout>
        <Header
          style={{
            padding: '0 20px',
            background: '#fff',
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'space-between',
            borderBottom: `1px solid ${tokens.border}`,
            position: 'sticky',
            top: 0,
            zIndex: 10,
            lineHeight: 1,
          }}
        >
          <div style={{ display: 'flex', alignItems: 'center', gap: 12, lineHeight: 'normal' }}>
            <Tooltip title={collapsed ? '展开菜单' : '收起菜单'}>
              <ButtonIcon collapsed={collapsed} onClick={() => setCollapsed(!collapsed)} />
            </Tooltip>
            {hasTenantSwitch && (
              <>
                <div style={{ width: 1, height: 22, background: tokens.borderStrong }} />
                <TenantSwitcher />
              </>
            )}
          </div>
          <div style={{ display: 'flex', alignItems: 'center', gap: 14, lineHeight: 'normal' }}>
            <Dropdown menu={{ items: userMenuItems }} placement="bottomRight" trigger={['click']}>
              <div style={{ display: 'flex', alignItems: 'center', gap: 10, height: 42, padding: '0 8px', lineHeight: 1, borderRadius: 8, cursor: 'pointer' }} className="user-entry">
                <Badge dot color={brand.primary} offset={[-2, 2]}>
                  <Avatar
                    size={34}
                    style={{ background: brand.gradient, fontWeight: 600, fontSize: 14 }}
                    icon={!displayName ? <UserOutlined /> : undefined}
                  >
                    {displayName.charAt(0).toUpperCase()}
                  </Avatar>
                </Badge>
                <span style={{ fontSize: 14, color: token.colorText, fontWeight: 500, lineHeight: 1 }}>{displayName}</span>
              </div>
            </Dropdown>
          </div>
        </Header>
        <Content style={{ margin: 20 }}>
          <Outlet />
        </Content>
      </Layout>
      <ProfileCenter open={profileOpen} onClose={() => setProfileOpen(false)} />
    </Layout>
  )
}

interface ButtonIconProps {
  collapsed: boolean
  onClick: () => void
}

// 必须转发 ref：antd 的 Tooltip 依赖 ref 定位气泡，
// 普通函数组件不支持 ref 会触发 findDOMNode 弃用警告
const ButtonIcon = forwardRef<HTMLButtonElement, ButtonIconProps>(function ButtonIcon({ collapsed, onClick }, ref) {
  return (
    <button
      ref={ref}
      onClick={onClick}
      style={{
        width: 36,
        height: 36,
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        border: 'none',
        borderRadius: 8,
        background: 'transparent',
        cursor: 'pointer',
        fontSize: 16,
        color: brand.textSecondary,
      }}
    >
      {collapsed ? <MenuUnfoldOutlined /> : <MenuFoldOutlined />}
    </button>
  )
})
