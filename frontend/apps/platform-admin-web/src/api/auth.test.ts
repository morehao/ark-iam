import { describe, it, expect, vi } from 'vitest'

vi.mock('../utils/request', () => ({
  default: {
    get: vi.fn((url: string) => {
      if (url === '/auth/userinfo') {
        return Promise.resolve({
          personInfo: { personID: 1, name: 'Test', avatar: '' },
          userInfo: { userID: 1, tenantID: 1, name: 'Test', isOwner: 1 },
        })
      }
      if (url === '/auth/myTenants') {
        return Promise.resolve({
          list: [{ tenantID: 1, name: 'Default' }],
        })
      }
      return Promise.reject(new Error('unknown'))
    }),
    post: vi.fn((url: string) => {
      if (url === '/auth/logout') {
        return Promise.resolve()
      }
      if (url === '/auth/logoutAll') {
        return Promise.resolve()
      }
      return Promise.reject(new Error('unknown'))
    }),
  },
}))

import { getUserinfo, getMyTenants, logoutAPI, logoutAllAPI } from './auth'

describe('auth API', () => {
  it('getUserinfo returns person and user info', async () => {
    const resp = await getUserinfo()
    expect(resp.personInfo.personID).toBe(1)
    expect(resp.userInfo.tenantID).toBe(1)
  })

  it('getMyTenants returns tenant list', async () => {
    const resp = await getMyTenants()
    expect(resp.list).toHaveLength(1)
    expect(resp.list[0].tenantID).toBe(1)
  })

  it('logoutAPI calls logout endpoint', async () => {
    await expect(logoutAPI('test-refresh-token')).resolves.toBeUndefined()
  })

  it('logoutAllAPI calls logoutAll endpoint', async () => {
    await expect(logoutAllAPI('test-refresh-token')).resolves.toBeUndefined()
  })
})
