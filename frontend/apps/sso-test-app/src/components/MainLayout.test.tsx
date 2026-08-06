import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import MainLayout from './MainLayout'

const { mockRemoveUser, mockSignoutRedirect, mockLogoutAllAPI } = vi.hoisted(() => ({
  mockRemoveUser: vi.fn().mockResolvedValue(undefined),
  mockSignoutRedirect: vi.fn().mockResolvedValue(undefined),
  mockLogoutAllAPI: vi.fn().mockResolvedValue(undefined),
}))

// 简化 antd Dropdown：将菜单项内联渲染，便于在 jsdom 中确定性地触发菜单项 onClick
vi.mock('antd', async (importOriginal) => {
  const actual = await importOriginal<typeof import('antd')>()
  const MockDropdown = ({ menu, children }: { menu?: { items?: Array<{ key?: string; onClick?: () => void; label?: string }> }; children?: React.ReactNode }) => {
    const items = menu?.items ?? []
    return (
      <div>
        {children}
        <ul>
          {items.map((item) => (
            <li key={item.key}>
              <button onClick={() => item.onClick?.()}>{item.label}</button>
            </li>
          ))}
        </ul>
      </div>
    )
  }
  return { ...actual, Dropdown: MockDropdown }
})

vi.mock('react-oidc-context', () => ({
  useAuth: () => ({
    isAuthenticated: true,
    isLoading: false,
    user: { access_token: 'mock-access', refresh_token: 'mock-refresh', id_token: 'mock-id-token' },
    signoutRedirect: mockSignoutRedirect,
    removeUser: mockRemoveUser,
  }),
}))

vi.mock('../api/auth', () => ({
  getUserinfo: vi.fn().mockResolvedValue({ personInfo: { personID: 1, name: 'Test User', avatar: '' } }),
  logoutAllAPI: mockLogoutAllAPI,
}))

vi.mock('../utils/request', () => ({
  setUserProvider: vi.fn(),
  default: { get: vi.fn(), post: vi.fn() },
}))

describe('MainLayout', () => {
  beforeEach(() => {
    mockRemoveUser.mockClear()
    mockSignoutRedirect.mockClear()
    mockLogoutAllAPI.mockClear()
  })

  it('renders the header title', () => {
    render(
      <MemoryRouter>
        <MainLayout />
      </MemoryRouter>
    )
    expect(screen.getByText('SSO 测试应用')).toBeInTheDocument()
  })

  it('triggers logoutAll, removeUser and signoutRedirect on logout', async () => {
    render(
      <MemoryRouter>
        <MainLayout />
      </MemoryRouter>
    )
    fireEvent.click(screen.getByText('退出登录'))
    await waitFor(() => {
      expect(mockLogoutAllAPI).toHaveBeenCalledWith('mock-refresh')
      expect(mockRemoveUser).toHaveBeenCalled()
      expect(mockSignoutRedirect).toHaveBeenCalled()
    })
  })
})
