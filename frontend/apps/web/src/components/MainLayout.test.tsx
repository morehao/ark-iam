import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { MemoryRouter, Route, Routes } from 'react-router-dom'
import { beforeEach, describe, expect, test, vi } from 'vitest'
import MainLayout from './MainLayout'
import { useAuthStore } from '../stores/authStore'

const getUserinfoMock = vi.fn()
const getMyTenantsMock = vi.fn()
const switchTenantMock = vi.fn()

vi.mock('../api/auth', () => ({
  getUserinfo: (...args: unknown[]) => getUserinfoMock(...args),
  getMyTenants: (...args: unknown[]) => getMyTenantsMock(...args),
  switchTenant: (...args: unknown[]) => switchTenantMock(...args),
}))

describe('MainLayout', () => {
  beforeEach(() => {
    getUserinfoMock.mockReset()
    getMyTenantsMock.mockReset()
    switchTenantMock.mockReset()
    localStorage.clear()
    useAuthStore.setState({
      authStage: 'tenant',
      personToken: 'person-token',
      tenantToken: 'tenant-token-a',
      refreshToken: 'tenant-refresh-token-a',
      tenants: [
        {
          tenantID: 1,
          name: 'Tenant A',
          tag: 'tenant-a',
          userID: 101,
          isOwner: 1,
        },
        {
          tenantID: 2,
          name: 'Tenant B',
          tag: 'tenant-b',
          userID: 102,
          isOwner: 0,
        },
      ],
      currentTenant: {
        tenantID: 1,
        name: 'Tenant A',
        tag: 'tenant-a',
        userID: 101,
        isOwner: 1,
      },
      personInfo: null,
      userInfo: null,
      accessToken: 'tenant-token-a',
    })
  })

  test('从头部切换到另一个租户后调用 switchTenant 且更新 currentTenant', async () => {
    getUserinfoMock
      .mockResolvedValueOnce({
        personInfo: { personID: 9, name: 'Alice', avatar: '' },
        userInfo: { userID: 101, name: 'Alice', tenantID: 1, isOwner: 1 },
      })
      .mockResolvedValueOnce({
        personInfo: { personID: 9, name: 'Alice', avatar: '' },
        userInfo: { userID: 102, name: 'Alice', tenantID: 2, isOwner: 0 },
      })
    switchTenantMock.mockResolvedValue({
      accessToken: 'tenant-token-b',
      refreshToken: 'tenant-refresh-token-b',
      expiresIn: 3600,
      tokenType: 'Bearer',
    })

    render(
      <MemoryRouter initialEntries={['/']}>
        <Routes>
          <Route path="/" element={<MainLayout />}>
            <Route index element={<div>首页</div>} />
          </Route>
          <Route path="/login" element={<div>登录页</div>} />
        </Routes>
      </MemoryRouter>
    )

    await waitFor(() => {
      expect(screen.getByRole('button', { name: 'Tenant A' })).toBeInTheDocument()
    })

    fireEvent.click(screen.getByRole('button', { name: 'Tenant A' }))
    fireEvent.click(await screen.findByRole('menuitem', { name: 'Tenant B' }))

    await waitFor(() => {
      expect(switchTenantMock).toHaveBeenCalledWith({ tenantID: 2 })
    })

    await waitFor(() => {
      expect(useAuthStore.getState()).toMatchObject({
        tenantToken: 'tenant-token-b',
        refreshToken: 'tenant-refresh-token-b',
        currentTenant: {
          tenantID: 2,
          name: 'Tenant B',
        },
        userInfo: {
          tenantID: 2,
        },
      })
    })
  })

  test('切换租户后即使 userinfo 拉取失败也应保留新的 tenant session', async () => {
    getUserinfoMock
      .mockResolvedValueOnce({
        personInfo: { personID: 9, name: 'Alice', avatar: '' },
        userInfo: { userID: 101, name: 'Alice', tenantID: 1, isOwner: 1 },
      })
      .mockRejectedValueOnce(new Error('userinfo failed'))
    switchTenantMock.mockResolvedValue({
      accessToken: 'tenant-token-b',
      refreshToken: 'tenant-refresh-token-b',
      expiresIn: 3600,
      tokenType: 'Bearer',
    })

    useAuthStore.setState({
      userInfo: { userID: 101, name: 'Alice', tenantID: 1, isOwner: 1 },
      personInfo: { personID: 9, name: 'Alice', avatar: '' },
    })

    render(
      <MemoryRouter initialEntries={['/']}>
        <Routes>
          <Route path="/" element={<MainLayout />}>
            <Route index element={<div>首页</div>} />
          </Route>
          <Route path="/login" element={<div>登录页</div>} />
        </Routes>
      </MemoryRouter>
    )

    await waitFor(() => {
      expect(screen.getByRole('button', { name: 'Tenant A' })).toBeInTheDocument()
    })

    fireEvent.click(screen.getByRole('button', { name: 'Tenant A' }))
    fireEvent.click(await screen.findByRole('menuitem', { name: 'Tenant B' }))

    await waitFor(() => {
      expect(useAuthStore.getState()).toMatchObject({
        tenantToken: 'tenant-token-b',
        refreshToken: 'tenant-refresh-token-b',
        currentTenant: {
          tenantID: 2,
          name: 'Tenant B',
        },
        userInfo: {
          tenantID: 1,
        },
      })
    })
  })
})
