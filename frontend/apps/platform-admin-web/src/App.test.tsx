import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import App from './App'

vi.mock('./stores/authStore', () => ({
  useAuthStore: () => ({ authStage: 'anonymous' }),
}))

vi.mock('./pages/auth/Login', () => ({
  default: () => <div>Login Page</div>,
}))

vi.mock('./pages/auth/AuthCallback', () => ({
  default: () => <div>Auth Callback</div>,
}))

describe('App', () => {
  beforeEach(() => {
    sessionStorage.setItem('oidc_silent_failed', '1')
  })

  it('renders login page for anonymous users at root', () => {
    render(
      <MemoryRouter initialEntries={['/login']}>
        <App />
      </MemoryRouter>
    )
    expect(screen.getByText('Login Page')).toBeInTheDocument()
  })
})
