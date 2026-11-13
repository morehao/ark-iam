import { describe, expect, it, vi, beforeEach, afterEach } from 'vitest'
import { render, screen, waitFor, fireEvent, cleanup } from '@testing-library/react'
import { LoginPage } from '../LoginPage'
import type { OIDCLoginResp, OIDCLoginConfigResp } from '../types'

// 被 mock 的 API 层：LoginPage 只通过 ../api 与 ../config 触达外部世界。
const apiMock = vi.hoisted(() => ({
  oidcLogin: vi.fn(),
  oidcSelectTenant: vi.fn(),
  oidcChangePassword: vi.fn(),
  registerPerson: vi.fn(),
  createTenant: vi.fn(),
  getLoginConfig: vi.fn(),
  installRedirectForError: vi.fn(() => null as string | null),
}))

vi.mock('../api', () => apiMock)
vi.mock('../config', () => ({
  ensureRuntimeConfig: vi.fn(async () => ({})),
  setAppBase: vi.fn(),
  appPath: (p: string) => p,
}))

/** 设置当前 URL 的查询串（authRequestID 读取来源）。 */
function setSearch(search: string) {
  window.history.replaceState({}, '', `/${search ? '?' + search : ''}`)
}

beforeEach(() => {
  Object.values(apiMock).forEach((fn) => {
    if (typeof fn === 'function' && 'mockReset' in fn) (fn as ReturnType<typeof vi.fn>).mockReset()
  })
  apiMock.installRedirectForError.mockReturnValue(null)
  apiMock.getLoginConfig.mockResolvedValue({ allowPersonCreateTenant: false } as OIDCLoginConfigResp)
})

afterEach(() => {
  cleanup()
  window.history.replaceState({}, '', '/')
})

describe('LoginPage', () => {
  it('① 无 authRequestID 时提示「缺少认证请求 ID，请从应用重新发起登录」', async () => {
    setSearch('')
    render(<LoginPage />)
    await waitFor(() =>
      expect(screen.getAllByText(/缺少认证请求 ID，请从应用重新发起登录/).length).toBeGreaterThan(0),
    )
  })

  it('② 登录成功 → 走 onLoggedIn(continueURL)', async () => {
    setSearch('authRequestID=req-1')
    apiMock.oidcLogin.mockResolvedValue({ continueURL: 'https://app/callback?code=abc' } as OIDCLoginResp)
    const onLoggedIn = vi.fn()
    render(<LoginPage onLoggedIn={onLoggedIn} />)

    fireEvent.change(screen.getByPlaceholderText('请输入用户名'), { target: { value: 'admin' } })
    fireEvent.change(screen.getByPlaceholderText('请输入密码'), { target: { value: 'secret' } })
    fireEvent.click(screen.getByRole('button', { name: /登\s*录/ }))

    await waitFor(() => expect(onLoggedIn).toHaveBeenCalledWith('https://app/callback?code=abc'))
    expect(apiMock.oidcLogin).toHaveBeenCalledWith({ authRequestID: 'req-1', identifier: 'admin', password: 'secret' })
  })

  it('③ 登录失败时展示后端错误文案', async () => {
    setSearch('authRequestID=req-1')
    apiMock.oidcLogin.mockRejectedValue(new Error('账号或密码错误'))
    render(<LoginPage />)

    fireEvent.change(screen.getByPlaceholderText('请输入用户名'), { target: { value: 'admin' } })
    fireEvent.change(screen.getByPlaceholderText('请输入密码'), { target: { value: 'bad' } })
    fireEvent.click(screen.getByRole('button', { name: /登\s*录/ }))

    await waitFor(() => expect(screen.getByText('账号或密码错误')).toBeTruthy())
  })

  it('④a 临时密码 → 进入改密分支（显示「设置新密码」）', async () => {
    setSearch('authRequestID=req-1')
    apiMock.oidcLogin.mockResolvedValue({ requiresPasswordChange: true } as OIDCLoginResp)
    render(<LoginPage />)

    fireEvent.change(screen.getByPlaceholderText('请输入用户名'), { target: { value: 'admin' } })
    fireEvent.change(screen.getByPlaceholderText('请输入密码'), { target: { value: 'temp' } })
    fireEvent.click(screen.getByRole('button', { name: /登\s*录/ }))

    await waitFor(() => expect(screen.getByRole('button', { name: /设置新密码/ })).toBeTruthy())
    expect(screen.getByText(/首次登录需修改初始密码/)).toBeTruthy()
  })

  it('④b requiresTenantSelection → 进入选租户分支', async () => {
    setSearch('authRequestID=req-1')
    apiMock.oidcLogin.mockResolvedValue({
      requiresTenantSelection: true,
      tenants: [
        { tenantID: 't1', name: 'Acme 公司' },
        { tenantID: 't2', name: 'Globex 集团' },
      ],
    } as OIDCLoginResp)
    render(<LoginPage />)

    fireEvent.change(screen.getByPlaceholderText('请输入用户名'), { target: { value: 'admin' } })
    fireEvent.change(screen.getByPlaceholderText('请输入密码'), { target: { value: 'secret' } })
    fireEvent.click(screen.getByRole('button', { name: /登\s*录/ }))

    await waitFor(() => expect(screen.getByText('请选择要登录的组织/租户')).toBeTruthy())
    expect(screen.getByText('Acme 公司')).toBeTruthy()
    expect(screen.getByText('Globex 集团')).toBeTruthy()
  })

  it('④c personID 零租户且允许建租户 → 进入建租户分支', async () => {
    setSearch('authRequestID=req-1')
    apiMock.oidcLogin.mockResolvedValue({
      continueURL: '',
      personID: 'p1',
      allowPersonCreateTenant: true,
      tenants: [],
    } as OIDCLoginResp)
    render(<LoginPage />)

    fireEvent.change(screen.getByPlaceholderText('请输入用户名'), { target: { value: 'admin' } })
    fireEvent.change(screen.getByPlaceholderText('请输入密码'), { target: { value: 'secret' } })
    fireEvent.click(screen.getByRole('button', { name: /登\s*录/ }))

    await waitFor(() => expect(screen.getByPlaceholderText('例如：Acme 公司')).toBeTruthy())
    expect(screen.getByRole('button', { name: /创建租户/ })).toBeTruthy()
  })

  it('辅助：选中租户后点击「进入」→ 走 onLoggedIn', async () => {
    setSearch('authRequestID=req-1')
    apiMock.oidcLogin.mockResolvedValue({
      requiresTenantSelection: true,
      tenants: [{ tenantID: 't1', name: 'Acme 公司' }],
    } as OIDCLoginResp)
    apiMock.oidcSelectTenant.mockResolvedValue({ continueURL: 'https://app/cb?code=xyz' } as OIDCLoginResp)
    const onLoggedIn = vi.fn()
    render(<LoginPage onLoggedIn={onLoggedIn} />)

    fireEvent.change(screen.getByPlaceholderText('请输入用户名'), { target: { value: 'admin' } })
    fireEvent.change(screen.getByPlaceholderText('请输入密码'), { target: { value: 'secret' } })
    fireEvent.click(screen.getByRole('button', { name: /登\s*录/ }))

    await waitFor(() => expect(screen.getByText('Acme 公司')).toBeTruthy())
    fireEvent.click(screen.getByRole('radio'))
    fireEvent.click(screen.getByRole('button', { name: /进\s*入/ }))

    await waitFor(() => expect(onLoggedIn).toHaveBeenCalledWith('https://app/cb?code=xyz'))
    expect(apiMock.oidcSelectTenant).toHaveBeenCalledWith({ authRequestID: 'req-1', tenantID: 't1' })
  })
})
