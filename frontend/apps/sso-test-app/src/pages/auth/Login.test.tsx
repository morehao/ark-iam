import { describe, it, expect, vi } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import Login from './Login'

vi.mock('../../utils/oidc', () => ({
  generatePKCEParams: vi.fn(() => ({ codeVerifier: 'verifier', codeChallenge: 'challenge', state: 'state' })),
  generateCodeChallenge: vi.fn().mockResolvedValue('challenge'),
  buildAuthorizeURL: vi.fn(() => 'http://localhost/authorize?client_id=test-rp-client'),
  storePKCEParams: vi.fn(),
}))

describe('Login', () => {
  it('renders login button', () => {
    render(<MemoryRouter><Login /></MemoryRouter>)
    expect(screen.getByText('IAM 账号登录')).toBeInTheDocument()
  })

  it('starts login without prompt=login', async () => {
    const mockAssign = vi.fn()
    const originalLocation = window.location
    delete (window as any).location
    ;(window as any).location = { ...originalLocation, assign: mockAssign }
    render(<MemoryRouter><Login /></MemoryRouter>)
    fireEvent.click(screen.getByText('IAM 账号登录'))
    await Promise.resolve()
    expect(mockAssign).toHaveBeenCalledWith('http://localhost/authorize?client_id=test-rp-client')
    ;(window as any).location = originalLocation
  })
})
