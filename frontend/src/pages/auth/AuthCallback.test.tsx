import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { MemoryRouter, Route, Routes } from 'react-router-dom'
import { beforeEach, describe, expect, test, vi } from 'vitest'
import AuthCallback from './AuthCallback'
import Login from './Login'
import { useAuthStore } from '../../stores/authStore'

const completeConnectorCallbackMock = vi.fn()
const selectTenantMock = vi.fn()
const getConnectorAuthorizationUrlMock = vi.fn()
const messageSuccessMock = vi.fn()
const messageErrorMock = vi.fn()

vi.mock('../../api/auth', () => ({
  completeConnectorCallback: (...args: unknown[]) => completeConnectorCallbackMock(...args),
  selectTenant: (...args: unknown[]) => selectTenantMock(...args),
  getConnectorAuthorizationUrl: (...args: unknown[]) => getConnectorAuthorizationUrlMock(...args),
  login: vi.fn(),
}))

vi.mock('antd', async () => {
  const actual = await vi.importActual<typeof import('antd')>('antd')

  return {
    ...actual,
    message: {
      ...actual.message,
      success: (...args: unknown[]) => messageSuccessMock(...args),
      error: (...args: unknown[]) => messageErrorMock(...args),
    },
  }
})

describe('AuthCallback', () => {
  beforeEach(() => {
    completeConnectorCallbackMock.mockReset()
    selectTenantMock.mockReset()
    getConnectorAuthorizationUrlMock.mockReset()
    messageSuccessMock.mockReset()
    messageErrorMock.mockReset()
    localStorage.clear()
    useAuthStore.setState({
      authStage: 'anonymous',
      personToken: null,
      tenantToken: null,
      refreshToken: null,
      tenants: [],
      currentTenant: null,
      personInfo: null,
      userInfo: null,
      accessToken: null,
    })
  })

  test('多租户 SSO callback 后跳转到选择租户页', async () => {
    completeConnectorCallbackMock.mockResolvedValue({
      data: {
        personToken: {
          accessToken: 'person-token',
          refreshToken: 'person-refresh-token',
          expiresIn: 3600,
          tokenType: 'Bearer',
        },
        tenants: [
          {
            tenantID: 7,
            name: 'Tenant A',
            tag: 'tenant-a',
            userID: 101,
            isOwner: 1,
          },
          {
            tenantID: 8,
            name: 'Tenant B',
            tag: 'tenant-b',
            userID: 101,
            isOwner: 0,
          },
        ],
      },
    })

    render(
      <MemoryRouter initialEntries={['/auth/callback?code=auth-code&state=callback-state']}>
        <Routes>
          <Route path="/auth/callback" element={<AuthCallback />} />
          <Route path="/login" element={<div>登录页</div>} />
          <Route path="/select-tenant" element={<div>选择租户页</div>} />
          <Route path="/" element={<div>首页</div>} />
        </Routes>
      </MemoryRouter>
    )

    await waitFor(() => {
      expect(completeConnectorCallbackMock).toHaveBeenCalledWith({
        code: 'auth-code',
        state: 'callback-state',
      })
    })

    await waitFor(() => {
      expect(screen.getByText('选择租户页')).toBeInTheDocument()
    })

    expect(selectTenantMock).not.toHaveBeenCalled()
    expect(useAuthStore.getState()).toMatchObject({
      authStage: 'person',
      personToken: 'person-token',
      tenantToken: null,
      tenants: [
        { tenantID: 7 },
        { tenantID: 8 },
      ],
      currentTenant: null,
      accessToken: 'person-token',
    })
  })

  test('配置 SSO connector 时点击企业 SSO 会跳转授权地址', async () => {
    vi.stubEnv('VITE_SSO_CONNECTOR_ID', '12')
    const locationAssignMock = vi.fn()
    getConnectorAuthorizationUrlMock.mockResolvedValue({
      data: {
        authorizationUrl: 'https://sso.example.com/authorize',
      },
    })

    const originalLocation = window.location
    Object.defineProperty(window, 'location', {
      configurable: true,
      value: {
        ...originalLocation,
        origin: 'http://localhost:3000',
        assign: locationAssignMock,
      },
    })

    render(
      <MemoryRouter initialEntries={['/login']}>
        <Routes>
          <Route path="/login" element={<Login />} />
        </Routes>
      </MemoryRouter>
    )

    fireEvent.click(screen.getByRole('button', { name: '企业 SSO 登录' }))

    await waitFor(() => {
      expect(getConnectorAuthorizationUrlMock).toHaveBeenCalledWith({
        connectorId: 12,
        redirectUri: 'http://localhost:3000/auth/callback',
      })
    })

    expect(locationAssignMock).toHaveBeenCalledWith('https://sso.example.com/authorize')

    Object.defineProperty(window, 'location', {
      configurable: true,
      value: originalLocation,
    })
    vi.unstubAllEnvs()
  })

  test('非法 SSO connector id 时不展示企业 SSO 登录入口', () => {
    vi.stubEnv('VITE_SSO_CONNECTOR_ID', '12abc')

    render(
      <MemoryRouter initialEntries={['/login']}>
        <Routes>
          <Route path="/login" element={<Login />} />
        </Routes>
      </MemoryRouter>
    )

    expect(screen.queryByRole('button', { name: '企业 SSO 登录' })).not.toBeInTheDocument()

    vi.unstubAllEnvs()
  })
})
