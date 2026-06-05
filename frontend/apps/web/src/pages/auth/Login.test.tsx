import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { MemoryRouter, Route, Routes } from 'react-router-dom'
import { beforeEach, describe, expect, test, vi } from 'vitest'
import Login from './Login'
import { useAuthStore } from '../../stores/authStore'

const loginMock = vi.fn()
const selectTenantMock = vi.fn()
const messageSuccessMock = vi.fn()
const messageErrorMock = vi.fn()

vi.mock('../../api/auth', () => ({
  login: (...args: unknown[]) => loginMock(...args),
  selectTenant: (...args: unknown[]) => selectTenantMock(...args),
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

describe('Login', () => {
  beforeEach(() => {
    loginMock.mockReset()
    selectTenantMock.mockReset()
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

  test('单租户登录后自动选租户并进入首页', async () => {
    loginMock.mockResolvedValue({
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
      ],
    })
    selectTenantMock.mockResolvedValue({
      accessToken: 'tenant-token',
      refreshToken: 'tenant-refresh-token',
      expiresIn: 3600,
      tokenType: 'Bearer',
    })

    render(
      <MemoryRouter initialEntries={['/login']}>
        <Routes>
          <Route path="/login" element={<Login />} />
          <Route path="/" element={<div>首页</div>} />
          <Route path="/select-tenant" element={<div>选择租户页</div>} />
        </Routes>
      </MemoryRouter>
    )

    fireEvent.change(screen.getByPlaceholderText('用户名/邮箱/手机号'), {
      target: { value: 'alice' },
    })
    fireEvent.change(screen.getByPlaceholderText('密码'), {
      target: { value: 'password' },
    })
    fireEvent.click(screen.getByRole('button', { name: /登\s*录/ }))

    await waitFor(() => {
      expect(selectTenantMock).toHaveBeenCalledWith({
        personToken: 'person-token',
        tenantID: 7,
      })
    })

    await waitFor(() => {
      expect(screen.getByText('首页')).toBeInTheDocument()
    })

    expect(useAuthStore.getState()).toMatchObject({
      authStage: 'tenant',
      personToken: 'person-token',
      tenantToken: 'tenant-token',
      refreshToken: 'tenant-refresh-token',
      currentTenant: {
        tenantID: 7,
      },
      accessToken: 'tenant-token',
    })
  })

  test('多租户登录后跳转到选择租户页', async () => {
    loginMock.mockResolvedValue({
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
    })

    render(
      <MemoryRouter initialEntries={['/login']}>
        <Routes>
          <Route path="/login" element={<Login />} />
          <Route path="/" element={<div>首页</div>} />
          <Route path="/select-tenant" element={<div>选择租户页</div>} />
        </Routes>
      </MemoryRouter>
    )

    fireEvent.change(screen.getByPlaceholderText('用户名/邮箱/手机号'), {
      target: { value: 'alice' },
    })
    fireEvent.change(screen.getByPlaceholderText('密码'), {
      target: { value: 'password' },
    })
    fireEvent.click(screen.getByRole('button', { name: /登\s*录/ }))

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

})
