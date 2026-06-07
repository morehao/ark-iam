import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import AuthCallback from './AuthCallback'

const mockNavigate = vi.fn()
vi.mock('react-router-dom', async () => {
  const actual = await vi.importActual('react-router-dom')
  return { ...actual, useNavigate: () => mockNavigate }
})

const mockSetAuthenticatedSession = vi.fn()
const mockMarkAnonymous = vi.fn()
vi.mock('../../stores/authStore', () => ({
  useAuthStore: (selector: any) => {
    const state = {
      setAuthenticatedSession: mockSetAuthenticatedSession,
      markAnonymous: mockMarkAnonymous,
    }
    return selector(state)
  },
}))

vi.mock('../../utils/oidc', () => ({
  loadPKCEParams: vi.fn(() => ({
    codeVerifier: 'test-verifier',
    codeChallenge: 'test-challenge',
    state: 'test-state',
  })),
  clearPKCEParams: vi.fn(),
  exchangeCodeForTokens: vi.fn().mockResolvedValue({
    access_token: 'access-token',
    id_token: 'id-token',
    refresh_token: 'refresh-token',
    expires_in: 3600,
  }),
  getOIDCFlowMode: vi.fn(() => 'interactive'),
}))

describe('AuthCallback', () => {
  beforeEach(() => {
    mockNavigate.mockClear()
    mockSetAuthenticatedSession.mockClear()
    mockMarkAnonymous.mockClear()
  })

  it('shows loading spinner during callback processing', () => {
    render(
      <MemoryRouter initialEntries={['/auth/callback?code=test-code&state=test-state']}>
        <AuthCallback />
      </MemoryRouter>
    )
    expect(screen.getByText('正在完成登录')).toBeInTheDocument()
  })

  it('marks anonymous and redirects to login for silent login_required callback', async () => {
    const oidcModule = await import('../../utils/oidc')
    vi.spyOn(oidcModule, 'getOIDCFlowMode').mockReturnValueOnce('silent')

    render(
      <MemoryRouter initialEntries={['/auth/callback?error=login_required&state=test-state']}>
        <AuthCallback />
      </MemoryRouter>
    )

    await waitFor(() => {
      expect(mockMarkAnonymous).toHaveBeenCalledTimes(1)
    })
    expect(mockNavigate).toHaveBeenCalledWith('/login', { replace: true })
    expect(mockSetAuthenticatedSession).not.toHaveBeenCalled()
  })
})
