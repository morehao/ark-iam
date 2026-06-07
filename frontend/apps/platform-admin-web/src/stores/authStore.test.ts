import { describe, it, expect, beforeEach } from 'vitest'
import { useAuthStore } from './authStore'

describe('authStore', () => {
  beforeEach(() => {
    useAuthStore.setState({
      authStage: 'anonymous',
      accessToken: null,
      idToken: null,
      refreshToken: null,
      expiresAt: null,
      personInfo: null,
      tenantInfo: null,
    })
  })

  it('starts anonymous', () => {
    expect(useAuthStore.getState().authStage).toBe('anonymous')
    expect(useAuthStore.getState().accessToken).toBeNull()
  })

  it('setAuthenticatedSession transitions to authenticated', () => {
    useAuthStore.getState().setAuthenticatedSession({
      accessToken: 'eyJhbGciOiJSUzI1NiJ9.eyJzdWIiOiJwZXJzb246NDIiLCJ0ZW5hbnRfaWQiOjEsImV4cCI6OTk5OTk5OTk5OX0.fake',
      idToken: 'test-id-token',
      refreshToken: 'test-refresh-token',
      expiresIn: 3600,
    })
    expect(useAuthStore.getState().authStage).toBe('authenticated')
    expect(useAuthStore.getState().accessToken).toBeTruthy()
    expect(useAuthStore.getState().refreshToken).toBe('test-refresh-token')
    expect(useAuthStore.getState().tenantInfo?.tenantID).toBe(1)
  })

  it('setAuthenticatedSession stores tokens and switches to authenticated', () => {
    useAuthStore.getState().setAuthenticatedSession({
      accessToken: 'eyJhbGciOiJSUzI1NiJ9.eyJ0ZW5hbnRfaWQiOjEsImV4cCI6OTk5OTk5OTk5OX0.fake',
      idToken: 'id-token',
      refreshToken: 'refresh-token',
      expiresIn: 3600,
    })

    expect(useAuthStore.getState().authStage).toBe('authenticated')
    expect(useAuthStore.getState().refreshToken).toBe('refresh-token')
    expect(useAuthStore.getState().tenantInfo?.tenantID).toBe(1)
  })

  it('updateTokens preserves idToken if not provided', () => {
    useAuthStore.getState().setAuthenticatedSession({
      accessToken: 'old-access',
      idToken: 'old-id-token',
      refreshToken: 'old-refresh',
      expiresIn: 3600,
    })
    useAuthStore.getState().updateTokens({
      accessToken: 'new-access',
      refreshToken: 'new-refresh',
      expiresIn: 3600,
    })
    expect(useAuthStore.getState().idToken).toBe('old-id-token')
    expect(useAuthStore.getState().accessToken).toBe('new-access')
    expect(useAuthStore.getState().refreshToken).toBe('new-refresh')
  })

  it('clearSession clears all state', () => {
    useAuthStore.getState().setAuthenticatedSession({
      accessToken: 'token',
      idToken: 'id',
      refreshToken: 'refresh',
      expiresIn: 3600,
    })
    useAuthStore.getState().clearSession()
    expect(useAuthStore.getState().authStage).toBe('anonymous')
    expect(useAuthStore.getState().accessToken).toBeNull()
    expect(useAuthStore.getState().idToken).toBeNull()
    expect(useAuthStore.getState().refreshToken).toBeNull()
    expect(useAuthStore.getState().tenantInfo).toBeNull()
  })

  it('beginChecking transitions to checking', () => {
    useAuthStore.getState().beginChecking()
    expect(useAuthStore.getState().authStage).toBe('checking')
  })

  it('markAnonymous clears tokens and exits checking mode', () => {
    useAuthStore.getState().beginChecking()
    useAuthStore.getState().markAnonymous()
    expect(useAuthStore.getState().authStage).toBe('anonymous')
    expect(useAuthStore.getState().accessToken).toBeNull()
  })
})
