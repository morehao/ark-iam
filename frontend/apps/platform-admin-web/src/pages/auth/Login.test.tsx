import { describe, it, expect, vi } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import Login from './Login'

vi.mock('../../utils/oidc', () => ({
  generatePKCEParams: vi.fn(() => ({ codeVerifier: 'verifier', codeChallenge: 'challenge', state: 'state' })),
  generateCodeChallenge: vi.fn().mockResolvedValue('challenge'),
  buildAuthorizeURL: vi.fn(() => 'http://localhost/authorize?client_id=platform-admin-web'),
  storePKCEParams: vi.fn(),
}))

import * as oidc from '../../utils/oidc'

describe('Login', () => {
  it('renders IAM login button', () => {
    render(<MemoryRouter><Login /></MemoryRouter>)
    expect(screen.getByText('IAM 账号登录')).toBeInTheDocument()
  })

  it('calls storePKCEParams and buildAuthorizeURL with interactive mode', async () => {
    render(<MemoryRouter><Login /></MemoryRouter>)
    fireEvent.click(screen.getByText('IAM 账号登录'))
    await Promise.resolve()
    expect(oidc.storePKCEParams).toHaveBeenCalledWith(
      expect.objectContaining({ codeVerifier: 'verifier' }),
      'interactive'
    )
    expect(oidc.buildAuthorizeURL).toHaveBeenCalledWith(
      expect.objectContaining({ codeVerifier: 'verifier' }),
      'interactive'
    )
  })
})
