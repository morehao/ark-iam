import { render, screen } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import { beforeEach, describe, expect, test } from 'vitest'
import App from './App'
import { useAuthStore } from './stores/authStore'

describe('App routing', () => {
  beforeEach(() => {
    localStorage.clear()
    useAuthStore.setState({
      authStage: 'anonymous',
      personToken: null,
      tenantToken: null,
      refreshToken: null,
      tenants: [],
      currentTenant: null,
      personInfo: null,
      userInfo: null,
      accessToken: null,
    })
  })

  test('person 阶段访问 /login 时跳转到 /select-tenant', () => {
    useAuthStore.setState({
      authStage: 'person',
      personToken: 'person-token',
      tenantToken: null,
      refreshToken: 'person-refresh-token',
      tenants: [
        {
          tenantID: 7,
          name: 'Tenant A',
          tag: 'tenant-a',
          userID: 101,
          isOwner: 1,
        },
      ],
      currentTenant: null,
      personInfo: null,
      userInfo: null,
      accessToken: 'person-token',
    })

    render(
      <MemoryRouter initialEntries={['/login']}>
        <App />
      </MemoryRouter>
    )

    expect(screen.getByText('选择租户')).toBeInTheDocument()
    expect(screen.queryByText('IAM 管理平台')).not.toBeInTheDocument()
  })
})
