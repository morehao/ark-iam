import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, waitFor } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import App from './App'

let mockAuthStage = 'anonymous'
let beginCheckingFn = vi.fn()
let markAnonymousFn = vi.fn()

vi.mock('./stores/authStore', () => ({
  useAuthStore: (selector: any) => {
    const state = {
      authStage: mockAuthStage,
      beginChecking: beginCheckingFn,
      markAnonymous: markAnonymousFn,
    }
    return selector ? selector(state) : state
  },
}))

vi.mock('./utils/oidc', () => ({
  buildAuthorizeURL: vi.fn().mockReturnValue('/v1/iam/oidc/authorize?client_id=test'),
  generateCodeChallenge: vi.fn().mockResolvedValue('mock-challenge'),
  generatePKCEParams: vi.fn().mockReturnValue({ codeVerifier: 'v', codeChallenge: 'c', state: 's' }),
  storePKCEParams: vi.fn(),
}))

vi.mock('./pages/auth/Login', () => ({
  default: () => <div>Login Page</div>,
}))

vi.mock('./components/MainLayout', () => ({
  default: () => <div>Main Layout</div>,
}))

describe('App', () => {
  beforeEach(() => {
    mockAuthStage = 'anonymous'
    beginCheckingFn = vi.fn()
    markAnonymousFn = vi.fn()
  })

  it('shows loading spinner while checking session', () => {
    mockAuthStage = 'checking'
    const { container } = render(
      <MemoryRouter initialEntries={['/']}>
        <App />
      </MemoryRouter>
    )
    expect(container.querySelector('.ant-spin')).toBeInTheDocument()
  })

  it('renders login page for anonymous users on /login without triggering silent auth', () => {
    const { getByText } = render(
      <MemoryRouter initialEntries={['/login']}>
        <App />
      </MemoryRouter>
    )
    expect(getByText('Login Page')).toBeInTheDocument()
    expect(beginCheckingFn).not.toHaveBeenCalled()
  })

  it('renders main layout for authenticated users', () => {
    mockAuthStage = 'authenticated'
    const { getByText } = render(
      <MemoryRouter initialEntries={['/']}>
        <App />
      </MemoryRouter>
    )
    expect(getByText('Main Layout')).toBeInTheDocument()
  })

  it('initiates silent OIDC flow once when anonymous on /', async () => {
    const replaceSpy = vi.fn()
    const originalReplace = window.location.replace
    Object.defineProperty(window, 'location', {
      configurable: true,
      value: { ...window.location, replace: replaceSpy },
    })

    try {
      render(
        <MemoryRouter initialEntries={['/']}>
          <App />
        </MemoryRouter>
      )

      await waitFor(() => {
        expect(beginCheckingFn).toHaveBeenCalledTimes(1)
      })
      await waitFor(() => {
        expect(replaceSpy).toHaveBeenCalledTimes(1)
      })
      expect(replaceSpy.mock.calls[0][0]).toContain('/v1/iam/oidc/authorize')
    } finally {
      Object.defineProperty(window, 'location', {
        configurable: true,
        value: { replace: originalReplace },
      })
    }
  })

  it('falls back to anonymous when silent flow throws', async () => {
    const oidcModule = await import('./utils/oidc')
    vi.spyOn(oidcModule, 'generateCodeChallenge').mockRejectedValueOnce(new Error('crypto failed'))

    render(
      <MemoryRouter initialEntries={['/']}>
        <App />
      </MemoryRouter>
    )

    await waitFor(() => {
      expect(markAnonymousFn).toHaveBeenCalledTimes(1)
    })
  })
})
