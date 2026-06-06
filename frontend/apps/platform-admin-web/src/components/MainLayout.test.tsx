import { describe, it, expect, vi } from 'vitest'
import { render, screen } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import MainLayout from './MainLayout'

vi.mock('../../utils/oidc', () => ({
  getEndSessionURL: vi.fn(() => 'http://localhost/end_session'),
}))

vi.mock('../../stores/authStore', () => ({
  useAuthStore: (selector?: (s: any) => any) => {
    const state = {
      authStage: 'authenticated',
      idToken: 'test-id-token',
      accessToken: 'test-access-token',
      tenantInfo: { tenantID: 1, tenantName: 'Test Tenant' },
      logout: vi.fn(),
      setPersonInfo: vi.fn(),
    }
    return selector ? selector(state) : state
  },
}))

vi.mock('../../api/auth', () => ({
  getUserinfo: vi.fn().mockResolvedValue({
    personInfo: { personID: 1, name: 'Test User', avatar: '' },
    userInfo: { userID: 1, tenantID: 1, name: 'Test User', isOwner: 1 },
  }),
}))

describe('MainLayout', () => {
  it('renders sidebar menu items', () => {
    render(
      <MemoryRouter initialEntries={['/']}>
        <MainLayout />
      </MemoryRouter>
    )
    expect(screen.getByText('仪表盘')).toBeInTheDocument()
    expect(screen.getByText('用户管理')).toBeInTheDocument()
  })
})
