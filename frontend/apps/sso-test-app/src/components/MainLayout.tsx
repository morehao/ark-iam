import { useEffect, useRef } from 'react'
import { Layout, Avatar, Dropdown } from 'antd'
import { UserOutlined, LogoutOutlined } from '@ant-design/icons'
import { Outlet } from 'react-router-dom'
import { getUserinfo, logoutAllAPI } from '../api/auth'
import { useAuthStore } from '../stores/authStore'
import { getEndSessionURL } from '../utils/oidc'

const { Header, Content } = Layout

const MainLayout = () => {
  const initializedRef = useRef(false)
  const silentCheckRef = useRef(false)
  const authStage = useAuthStore((state) => state.authStage)
  const clearSession = useAuthStore((state) => state.clearSession)
  const setPersonInfo = useAuthStore((state) => state.setPersonInfo)

  const handleLogout = async () => {
    const store = useAuthStore.getState()
    const currentIdToken = store.idToken
    try { await logoutAllAPI(store.refreshToken ?? '') } catch {}
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

  useEffect(() => {
    if (authStage !== 'authenticated') {
      silentCheckRef.current = false
      return
    }

    const handleVisibilityChange = () => {
      if (document.visibilityState !== 'visible' || silentCheckRef.current) return

      silentCheckRef.current = true
      void (async () => {
        try {
          const resp = await fetch('/v1/iam/oidc/session/status', { credentials: 'include' })
          const data = await resp.json() as { authenticated?: boolean }
          if (data.authenticated === false) {
            clearSession()
            window.location.assign('/login')
          }
        } catch {
        } finally {
          silentCheckRef.current = false
        }
      })()
    }

    document.addEventListener('visibilitychange', handleVisibilityChange)
    return () => {
      document.removeEventListener('visibilitychange', handleVisibilityChange)
    }
  }, [authStage, clearSession])

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
