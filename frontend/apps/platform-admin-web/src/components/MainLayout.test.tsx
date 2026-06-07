import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import MainLayout from './MainLayout'

const replaceSpy = vi.fn()

vi.mock('../stores/authStore', () => ({
  useAuthStore: (selector: any) => {
    const state = {
      authStage: 'authenticated',
      idToken: 'test-id-token',
      accessToken: 'test-access-token',
      refreshToken: 'test-refresh-token',
      personInfo: null,
      clearSession: vi.fn(),
      setPersonInfo: vi.fn(),
    }
    return selector(state)
  },
}))

vi.mock('../api/auth', () => ({
  getUserinfo: vi.fn().mockResolvedValue({ personInfo: null }),
  logoutAPI: vi.fn().mockResolvedValue(undefined),
}))

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
    Object.defineProperty(window, 'location', {
      configurable: true,
      value: { ...window.location, replace: replaceSpy },
    })
  })

  it('renders the sider title', () => {
    render(
      <MemoryRouter>
        <MainLayout />
      </MemoryRouter>
    )
    expect(screen.getByText('IAM 管理平台')).toBeInTheDocument()
  })

  it('does not trigger silent login when tab becomes visible', () => {
    Object.defineProperty(document, 'visibilityState', {
      configurable: true,
      get: () => 'visible',
    })

    render(
      <MemoryRouter>
        <MainLayout />
      </MemoryRouter>
    )

    document.dispatchEvent(new Event('visibilitychange'))
    expect(replaceSpy).not.toHaveBeenCalled()
  })
})
