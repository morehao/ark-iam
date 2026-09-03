import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, waitFor } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'

let mockIsAuthenticated = false
let mockIsLoading = false
let mockActiveNavigator: string | undefined = undefined
const mockSigninRedirect = vi.fn()
const mockSigninSilent = vi.fn(() => Promise.resolve(null))

vi.mock('react-oidc-context', () => ({
  useAuth: () => ({
    isAuthenticated: mockIsAuthenticated,
    isLoading: mockIsLoading,
    activeNavigator: mockActiveNavigator,
    signinRedirect: mockSigninRedirect,
    signinSilent: mockSigninSilent,
    removeUser: vi.fn(),
    events: { addSilentRenewError: vi.fn().mockReturnValue(() => {}) },
    user: null,
  }),
  hasAuthParams: vi.fn(() => false),
}))

vi.mock('@ark-iam/ui', async () => {
  const actual = await vi.importActual<typeof import('@ark-iam/ui')>('@ark-iam/ui')
  return {
    ...actual,
    MainLayout: () => <div>Main Layout</div>,
    LoginPage: () => <div>Login Page</div>,
  }
})

const appModule = await import('./App')
const App = appModule.default

describe('App', () => {
  beforeEach(() => {
    mockIsAuthenticated = false
    mockIsLoading = false
    mockActiveNavigator = undefined
    mockSigninRedirect.mockClear()
    mockSigninSilent.mockClear()
  })

  it('shows loading spinner while still loading', () => {
    mockIsLoading = true
    const { container } = render(
      <MemoryRouter initialEntries={['/']}>
        <App />
      </MemoryRouter>
    )
    expect(container.querySelector('.ant-spin')).toBeInTheDocument()
  })

  it('shows loading spinner during navigation', () => {
    mockActiveNavigator = 'signinRedirect'
    const { container } = render(
      <MemoryRouter initialEntries={['/']}>
        <App />
      </MemoryRouter>
    )
    expect(container.querySelector('.ant-spin')).toBeInTheDocument()
  })

  it('triggers signinRedirect when not authenticated on /', async () => {
    render(
      <MemoryRouter initialEntries={['/']}>
        <App />
      </MemoryRouter>
    )

    await waitFor(() => {
      expect(mockSigninRedirect).toHaveBeenCalled()
    })
  })

  it('triggers signinRedirect on /login when not authenticated', async () => {
    render(
      <MemoryRouter initialEntries={['/login']}>
        <App />
      </MemoryRouter>
    )

    await waitFor(() => {
      expect(mockSigninRedirect).toHaveBeenCalled()
    })
  })

  it('does NOT trigger signinRedirect on /auth/callback', async () => {
    render(
      <MemoryRouter initialEntries={['/auth/callback']}>
        <App />
      </MemoryRouter>
    )

    await waitFor(() => {
      expect(document.querySelector('.ant-spin')).toBeInTheDocument()
    })
    expect(mockSigninRedirect).not.toHaveBeenCalled()
  })

  it('renders main layout when authenticated', async () => {
    mockIsAuthenticated = true
    const { getByText } = render(
      <MemoryRouter initialEntries={['/']}>
        <App />
      </MemoryRouter>
    )
    // 动态菜单接口在测试环境请求失败，catch 后置空菜单并完成加载，最终渲染 MainLayout。
    await waitFor(() => expect(getByText('Main Layout')).toBeInTheDocument())
  })

  it('renders login page when authenticated and on /login', () => {
    mockIsAuthenticated = true
    const { getByText } = render(
      <MemoryRouter initialEntries={['/login']}>
        <App />
      </MemoryRouter>
    )
    expect(getByText('Login Page')).toBeInTheDocument()
  })
})
