import { beforeEach, describe, expect, test } from 'vitest'
import { useAuthStore } from './authStore'

describe('useAuthStore', () => {
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

  test('sets person session before tenant selection', () => {
    const store = useAuthStore.getState()

    store.setPersonSession({
      personToken: 'person-token',
      refreshToken: 'refresh-token',
      tenants: [
        {
          tenantID: 1,
          name: 'Tenant A',
          tag: 'tenant-a',
          userID: 100,
          isOwner: 1,
        },
      ],
      personInfo: {
        personID: 10,
        name: 'Alice',
        avatar: 'avatar.png',
      },
    })

    expect(useAuthStore.getState()).toMatchObject({
      authStage: 'person',
      personToken: 'person-token',
      tenantToken: null,
      refreshToken: 'refresh-token',
      tenants: [
        {
          tenantID: 1,
          name: 'Tenant A',
        },
      ],
      personInfo: {
        personID: 10,
      },
      currentTenant: null,
      userInfo: null,
    })
    expect(localStorage.getItem('personToken')).toBe('person-token')
    expect(localStorage.getItem('tenantToken')).toBeNull()
  })

  test('promotes person session to tenant session and clears only tenant scope', () => {
    const store = useAuthStore.getState()

    store.setPersonSession({
      personToken: 'person-token',
      refreshToken: 'refresh-token',
      tenants: [
        {
          tenantID: 1,
          name: 'Tenant A',
          tag: 'tenant-a',
          userID: 100,
          isOwner: 1,
        },
      ],
    })
    store.setTenantSession({
      tenantToken: 'tenant-token',
      refreshToken: 'tenant-refresh-token',
      currentTenant: {
        tenantID: 1,
        name: 'Tenant A',
        tag: 'tenant-a',
        userID: 100,
        isOwner: 1,
      },
      userInfo: {
        userID: 100,
        name: 'Alice Admin',
        tenantID: 1,
        isOwner: 1,
      },
    })

    expect(useAuthStore.getState()).toMatchObject({
      authStage: 'tenant',
      personToken: 'person-token',
      tenantToken: 'tenant-token',
      refreshToken: 'tenant-refresh-token',
      currentTenant: {
        tenantID: 1,
      },
      userInfo: {
        userID: 100,
      },
    })
    expect(localStorage.getItem('tenantToken')).toBe('tenant-token')

    useAuthStore.getState().clearTenantSession()

    expect(useAuthStore.getState()).toMatchObject({
      authStage: 'person',
      personToken: 'person-token',
      tenantToken: null,
      refreshToken: 'refresh-token',
      currentTenant: null,
      userInfo: null,
    })
    expect(localStorage.getItem('personToken')).toBe('person-token')
    expect(localStorage.getItem('tenantToken')).toBeNull()
    expect(localStorage.getItem('refreshToken')).toBe('refresh-token')
  })

  test('supports legacy setTokens by seeding person stage', () => {
    const store = useAuthStore.getState()

    store.setTokens('legacy-person-token', 'legacy-refresh-token')

    expect(useAuthStore.getState()).toMatchObject({
      authStage: 'person',
      personToken: 'legacy-person-token',
      tenantToken: null,
      refreshToken: 'legacy-refresh-token',
      accessToken: 'legacy-person-token',
    })
    expect(localStorage.getItem('personToken')).toBe('legacy-person-token')
    expect(localStorage.getItem('refreshToken')).toBe('legacy-refresh-token')
  })

  test('keeps legacy accessToken available after person session seeding', () => {
    const store = useAuthStore.getState()

    store.setPersonSession({
      personToken: 'person-token',
      refreshToken: 'refresh-token',
      tenants: [],
    })

    expect(useAuthStore.getState()).toMatchObject({
      authStage: 'person',
      personToken: 'person-token',
      accessToken: 'person-token',
    })
  })
})
