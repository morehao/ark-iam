import { describe, it, expect, vi } from 'vitest'
import { render, screen } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import App from './App'

vi.mock('./stores/authStore', () => ({
  useAuthStore: (selector?: (state: any) => any) => {
    const state = { authStage: 'anonymous', beginChecking: vi.fn() }
    return selector ? selector(state) : state
  },
}))

vi.mock('./pages/auth/Login', () => ({
  default: () => <div>SSO Login Page</div>,
}))

describe('App', () => {
  it('renders login page for anonymous users', () => {
    render(
      <MemoryRouter initialEntries={['/login']}>
        <App />
      </MemoryRouter>
    )
    expect(screen.getByText('SSO Login Page')).toBeInTheDocument()
  })
})
