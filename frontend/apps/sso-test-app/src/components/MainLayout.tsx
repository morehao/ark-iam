import { useEffect, useRef, useState } from 'react'
import { Layout, Avatar, Dropdown } from 'antd'
import { UserOutlined, LogoutOutlined } from '@ant-design/icons'
import { Outlet } from 'react-router-dom'
import { useAuth } from 'react-oidc-context'
import type { PersonInfo } from '@ark-iam/shared'
import { getUserinfo, logoutAllAPI } from '../api/auth'
import { setUserProvider } from '../utils/request'

const { Header, Content } = Layout

const MainLayout = () => {
  const [personInfo, setPersonInfo] = useState<PersonInfo | null>(null)
  const initializedRef = useRef(false)
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
    auth.signoutRedirect()
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
