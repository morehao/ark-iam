import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, waitFor } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'

let mockIsAuthenticated = false
let mockIsLoading = false
let mockActiveNavigator: string | undefined = undefined
let mockError: { name?: string; message?: string } | undefined = undefined
const mockSigninRedirect = vi.fn()
const mockSigninSilent = vi.fn(() => Promise.resolve(null))

vi.mock('react-oidc-context', () => ({
  useAuth: () => ({
    isAuthenticated: mockIsAuthenticated,
    isLoading: mockIsLoading,
    activeNavigator: mockActiveNavigator,
    error: mockError,
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
    SsoLoginPage: () => <div>Login Page</div>,
  }
})

const consoleModule = await import('./console')
const IamRoutes = consoleModule.IamRoutes

// IamRoutes 需外层已有 Router（与生产一致：宿主提供 MemoryRouter 模拟）。
const App = () => <IamRoutes config={{ variant: 'platform' }} />

describe('IamRoutes (platform)', () => {
  beforeEach(() => {
    mockIsAuthenticated = false
    mockIsLoading = false
    mockActiveNavigator = undefined
    mockError = undefined
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

  it('renders visible error (not an infinite spinner) when callback fails on /auth/callback', async () => {
    mockError = { name: 'signinCallback', message: 'No matching state found in storage' }
    const { getByText, container } = render(
      <MemoryRouter initialEntries={['/auth/callback']}>
        <App />
      </MemoryRouter>
    )
    // 关键回归：失败必须可见且有恢复入口，不能停在 spinner。
    await waitFor(() => expect(getByText('登录失败')).toBeInTheDocument())
    expect(getByText('重新登录')).toBeInTheDocument()
    expect(container.querySelector('.ant-spin')).not.toBeInTheDocument()
    // 失败态不应再自动拉起 signinRedirect（避免与错误页打架/死循环）。
    expect(mockSigninRedirect).not.toHaveBeenCalled()
  })

  it('renders error view when auth error occurs on a non-callback route', async () => {
    mockError = { name: 'renewSilent', message: 'renew failed' }
    const { getByText, container } = render(
      <MemoryRouter initialEntries={['/']}>
        <App />
      </MemoryRouter>
    )
    await waitFor(() => expect(getByText('登录失败')).toBeInTheDocument())
    expect(container.querySelector('.ant-spin')).not.toBeInTheDocument()
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
