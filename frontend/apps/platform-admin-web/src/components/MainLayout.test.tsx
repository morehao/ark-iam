import { describe, it, expect, vi } from 'vitest'
import { render, screen } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import MainLayout from './MainLayout'

vi.mock('react-oidc-context', () => ({
  useAuth: () => ({
    isAuthenticated: true,
    isLoading: false,
    user: { access_token: 'mock-access', refresh_token: 'mock-refresh' },
    signoutRedirect: vi.fn().mockResolvedValue(undefined),
  }),
}))

vi.mock('../api/auth', () => ({
  getUserinfo: vi.fn().mockResolvedValue({ personInfo: { personID: 1, name: 'Test User', avatar: '' } }),
  logoutAllAPI: vi.fn().mockResolvedValue(undefined),
}))

vi.mock('../utils/request', () => ({
  setUserProvider: vi.fn(),
  default: { get: vi.fn(), post: vi.fn() },
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
