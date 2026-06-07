import { useEffect, useRef } from 'react'
import { Layout, Avatar, Dropdown } from 'antd'
import { UserOutlined, LogoutOutlined } from '@ant-design/icons'
import { Outlet } from 'react-router-dom'
import { getUserinfo, logoutAPI } from '../api/auth'
import { useAuthStore } from '../stores/authStore'
import { getEndSessionURL } from '../utils/oidc'

const { Header, Content } = Layout

const MainLayout = () => {
  const initializedRef = useRef(false)
  const authStage = useAuthStore((state) => state.authStage)
  const clearSession = useAuthStore((state) => state.clearSession)
  const setPersonInfo = useAuthStore((state) => state.setPersonInfo)

  const handleLogout = async () => {
    const store = useAuthStore.getState()
    const currentIdToken = store.idToken
    try { await logoutAPI(store.refreshToken ?? '') } catch {}
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
      } catch {}
    }
    void loadUserContext()
    return () => { active = false }
  }, [authStage, setPersonInfo])

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
