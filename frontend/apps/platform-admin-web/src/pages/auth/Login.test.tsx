import { describe, it, expect, vi } from 'vitest'
import { render, screen } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import Login from './Login'

vi.mock('../../utils/oidc', () => ({
  generatePKCEParams: vi.fn(() => ({
    codeVerifier: 'test-verifier',
    codeChallenge: 'test-challenge',
    state: 'test-state',
  })),
  generateCodeChallenge: vi.fn().mockResolvedValue('test-challenge'),
  buildAuthorizeURL: vi.fn(() => 'http://localhost/authorize?test=1'),
  storePKCEParams: vi.fn(),
}))

describe('Login', () => {
  it('renders IAM login button', () => {
    render(
      <MemoryRouter>
        <Login />
      </MemoryRouter>
    )
    expect(screen.getByText('IAM 账号登录')).toBeInTheDocument()
  })
})
