import { describe, it, expect, vi } from 'vitest'
import { render, screen } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import MainLayout from './MainLayout'

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

describe('MainLayout', () => {
  it('renders the sider title', () => {
    render(
      <MemoryRouter>
        <MainLayout />
      </MemoryRouter>
    )
    expect(screen.getByText('IAM 管理平台')).toBeInTheDocument()
  })
})
