import { beforeEach, describe, expect, test, vi } from 'vitest'

const requestMock = vi.hoisted(() => ({
  get: vi.fn(),
  post: vi.fn(),
}))

vi.mock('../utils/request', () => ({
  default: requestMock,
}))

import { getMyTenants, selectTenant } from './auth'

describe('auth api', () => {
  beforeEach(() => {
    requestMock.get.mockReset()
    requestMock.post.mockReset()
  })

  test('selectTenant posts personToken in request body', () => {
    selectTenant({
      tenantID: 7,
      personToken: 'person-token',
    })

    expect(requestMock.post).toHaveBeenCalledWith(
      '/auth/selectTenant',
      {
        tenantID: 7,
        personToken: 'person-token',
      }
    )
  })

  test('getMyTenants sends personToken as query param', () => {
    getMyTenants({ personToken: 'person-token' })

    expect(requestMock.get).toHaveBeenCalledWith('/auth/myTenants', {
      params: {
        personToken: 'person-token',
      },
    })
  })
})
