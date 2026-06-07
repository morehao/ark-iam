import { describe, it, expect, vi } from 'vitest'
import { render } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import App from './App'

vi.mock('./stores/authStore', () => ({
  useAuthStore: (selector: any) => {
    const state = {
      authStage: 'checking',
      beginChecking: vi.fn(),
    }
    return selector ? selector(state) : state
  },
}))

describe('App', () => {
  it('shows loading spinner while checking session', () => {
    const { container } = render(
      <MemoryRouter initialEntries={['/']}>
        <App />
      </MemoryRouter>
    )
    expect(container.querySelector('.ant-spin')).toBeInTheDocument()
  })
})
