import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, waitFor, fireEvent } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import MainLayout from './MainLayout'

vi.mock('antd', async () => {
  const actual = await vi.importActual<typeof import('antd')>('antd')
  return {
    ...actual,
    Dropdown: ({ menu, children }: any) => (
      <>
        {children}
        <button type="button" onClick={() => menu?.items?.[0]?.onClick?.()}>退出登录</button>
      </>
    ),
  }
})

const testMocks = vi.hoisted(() => {
  const clearSessionSpy = vi.fn()
  const setPersonInfoSpy = vi.fn()
  const logoutAllAPISpy = vi.fn().mockResolvedValue(undefined)
  const getUserinfoSpy = vi.fn().mockResolvedValue({ personInfo: null })
  const fetchSpy = vi.fn().mockResolvedValue({ ok: true, json: async () => ({ authenticated: true }) })
  const state = {
    authStage: 'authenticated',
    idToken: 'tid',
    accessToken: 'ta',
    refreshToken: 'tr',
    personInfo: null,
    clearSession: clearSessionSpy,
    setPersonInfo: setPersonInfoSpy,
  }
  const useAuthStore = ((selector: any) => selector(state)) as any
  useAuthStore.getState = () => state
  return { clearSessionSpy, setPersonInfoSpy, logoutAllAPISpy, getUserinfoSpy, fetchSpy, useAuthStore }
})

const replaceSpy = vi.fn()
const assignSpy = vi.fn()

vi.mock('../stores/authStore', () => ({
  useAuthStore: testMocks.useAuthStore,
}))
vi.mock('../api/auth', () => ({ getUserinfo: testMocks.getUserinfoSpy, logoutAllAPI: testMocks.logoutAllAPISpy }))
vi.mock('../utils/oidc', () => ({
  buildAuthorizeURL: vi.fn().mockReturnValue('/v1/iam/oidc/authorize?client_id=test'),
  generateCodeChallenge: vi.fn().mockResolvedValue('mock-challenge'),
  generatePKCEParams: vi.fn().mockReturnValue({ codeVerifier: 'v', codeChallenge: 'c', state: 's' }),
  getEndSessionURL: vi.fn().mockReturnValue('/v1/iam/oidc/end_session'),
  storePKCEParams: vi.fn(),
}))

describe('MainLayout', () => {
  beforeEach(() => {
    replaceSpy.mockClear()
    assignSpy.mockClear()
    testMocks.clearSessionSpy.mockClear()
    testMocks.setPersonInfoSpy.mockClear()
    testMocks.logoutAllAPISpy.mockClear()
    testMocks.fetchSpy.mockClear()
    Object.defineProperty(window, 'location', {
      configurable: true,
      value: { ...window.location, replace: replaceSpy, assign: assignSpy },
    })
    vi.stubGlobal('fetch', testMocks.fetchSpy)
  })

  it('renders header title', () => {
    render(<MemoryRouter><MainLayout /></MemoryRouter>)
    expect(screen.getByText('SSO 测试应用')).toBeInTheDocument()
  })

  it('does not trigger top-level redirect when tab becomes visible', async () => {
    Object.defineProperty(document, 'visibilityState', {
      configurable: true,
      get: () => 'visible',
    })

    render(<MemoryRouter><MainLayout /></MemoryRouter>)

    document.dispatchEvent(new Event('visibilitychange'))
    await waitFor(() => {
      expect(testMocks.fetchSpy).toHaveBeenCalled()
      expect(replaceSpy).not.toHaveBeenCalled()
    })
  })

  it('does not start a second silent session check while one is in flight', async () => {
    Object.defineProperty(document, 'visibilityState', {
      configurable: true,
      get: () => 'visible',
    })

    render(<MemoryRouter><MainLayout /></MemoryRouter>)

    document.dispatchEvent(new Event('visibilitychange'))
    document.dispatchEvent(new Event('visibilitychange'))
    await waitFor(() => {
      expect(testMocks.fetchSpy).toHaveBeenCalledTimes(1)
      expect(replaceSpy).not.toHaveBeenCalled()
    })
  })

  it('clears session and redirects to login when session status is anonymous', async () => {
    testMocks.fetchSpy.mockResolvedValueOnce({ ok: true, json: async () => ({ authenticated: false }) })

    Object.defineProperty(document, 'visibilityState', {
      configurable: true,
      get: () => 'visible',
    })

    render(<MemoryRouter><MainLayout /></MemoryRouter>)

    document.dispatchEvent(new Event('visibilitychange'))
    await waitFor(() => {
      expect(testMocks.clearSessionSpy).toHaveBeenCalled()
      expect(assignSpy).toHaveBeenCalledWith('/login')
    })
  })

  it('uses logoutAll and redirects to end session on logout', async () => {
    render(<MemoryRouter><MainLayout /></MemoryRouter>)

    fireEvent.click(screen.getByText('退出登录'))

    await waitFor(() => {
      expect(testMocks.logoutAllAPISpy).toHaveBeenCalledWith('tr')
      expect(testMocks.clearSessionSpy).toHaveBeenCalled()
      expect(assignSpy).toHaveBeenCalledWith('/v1/iam/oidc/end_session')
    })
  })
})
