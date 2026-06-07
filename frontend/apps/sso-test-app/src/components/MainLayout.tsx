import { useEffect, useRef } from 'react'
import { Layout, Avatar, Dropdown } from 'antd'
import { UserOutlined, LogoutOutlined } from '@ant-design/icons'
import { Outlet } from 'react-router-dom'
import { getUserinfo, logoutAPI } from '../api/auth'
import { useAuthStore } from '../stores/authStore'
import {
  getEndSessionURL,
  generatePKCEParams,
  generateCodeChallenge,
  buildAuthorizeURL,
  storePKCEParams,
} from '../utils/oidc'

const { Header, Content } = Layout

const MainLayout = () => {
  const initializedRef = useRef(false)
  const checkGenRef = useRef(0)
  const delayTimerRef = useRef<ReturnType<typeof setTimeout>>()
  const iframeRef = useRef<HTMLIFrameElement>(null)
  const authStage = useAuthStore((state) => state.authStage)
  const clearSession = useAuthStore((state) => state.clearSession)
  const setPersonInfo = useAuthStore((state) => state.setPersonInfo)

  const handleLogout = async () => {
    const store = useAuthStore.getState()
    const currentIdToken = store.idToken

    try {
      await logoutAPI(store.refreshToken ?? '')
    } catch {
      // 即使接口调用失败也继续退出流程
    }

    clearSession()

    if (currentIdToken) {
      const el = document.createElement('script')
      el.src = getEndSessionURL(currentIdToken)
      el.onload = () => el.remove()
      el.onerror = () => el.remove()
      document.head.appendChild(el)
    }
  }

  const triggerSilentCheck = async () => {
    const gen = ++checkGenRef.current
    try {
      const params = generatePKCEParams()
      params.codeChallenge = await generateCodeChallenge(params.codeVerifier)
      if (gen !== checkGenRef.current) return
      sessionStorage.setItem('oidc_silent', '1')
      storePKCEParams(params, 'silent')
      const url = buildAuthorizeURL(params, 'silent')
      if (iframeRef.current) {
        iframeRef.current.src = url
      }
    } catch (err) {
      console.error('[MainLayout] triggerSilentCheck failed:', err)
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

    const handleMessage = (event: MessageEvent) => {
      if (event.origin !== window.location.origin) return
      if (event.data?.type !== 'oidc-silent') return
      if (event.data?.status === 'expired') {
        clearSession()
        window.location.href = '/login'
        return
      }
      if (event.data?.status === 'success' && event.data?.tokens) {
        useAuthStore.getState().updateTokens(event.data.tokens)
        if (active) loadUserContext().catch(console.error)
      }
    }
    window.addEventListener('message', handleMessage)

    const handleVisibilityChange = () => {
      if (document.visibilityState !== 'visible' || !active) return
      clearTimeout(delayTimerRef.current)
      delayTimerRef.current = setTimeout(() => {
        if (active) triggerSilentCheck().catch(console.error)
      }, 2000)
    }
    document.addEventListener('visibilitychange', handleVisibilityChange)

    const intervalId = setInterval(() => {
      if (active) triggerSilentCheck().catch(console.error)
    }, 60000)

    loadUserContext().catch(console.error)

    return () => {
      active = false
      clearTimeout(delayTimerRef.current)
      window.removeEventListener('message', handleMessage)
      document.removeEventListener('visibilitychange', handleVisibilityChange)
      clearInterval(intervalId)
    }
  }, [authStage, setPersonInfo, clearSession])

  const userMenuItems = [
    { key: 'logout', icon: <LogoutOutlined />, label: '退出登录', onClick: handleLogout },
  ]

  return (
    <Layout style={{ minHeight: '100vh' }}>
      <iframe
        ref={iframeRef}
        style={{ display: 'none' }}
        title="oidc-silent-check"
        sandbox="allow-scripts allow-same-origin allow-forms"
        onError={() => console.error('[MainLayout] silent check iframe load failed')}
      />
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
