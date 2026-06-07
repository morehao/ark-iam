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

  it('beginChecking transitions to checking', () => {
    useAuthStore.getState().beginChecking()
    expect(useAuthStore.getState().authStage).toBe('checking')
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

  it('markAnonymous sets stage to anonymous without clearing tokens', () => {
    useAuthStore.getState().setAuthenticatedSession({
      accessToken: 'eyJhbGciOiJSUzI1NiJ9.eyJ0ZW5hbnRfaWQiOjEsImV4cCI6OTk5OTk5OTk5OX0.fake',
      idToken: 'id-token',
      refreshToken: 'refresh-token',
      expiresIn: 3600,
    })
    useAuthStore.getState().markAnonymous()
    expect(useAuthStore.getState().authStage).toBe('anonymous')
    expect(useAuthStore.getState().accessToken).toBeTruthy()
  })

  it('clearSession clears all state', () => {
    useAuthStore.getState().setAuthenticatedSession({
      accessToken: 'eyJhbGciOiJSUzI1NiJ9.eyJ0ZW5hbnRfaWQiOjEsImV4cCI6OTk5OTk5OTk5OX0.fake',
      idToken: 'id-token',
      refreshToken: 'refresh-token',
      expiresIn: 3600,
    })
    useAuthStore.getState().clearSession()
    expect(useAuthStore.getState().authStage).toBe('anonymous')
    expect(useAuthStore.getState().accessToken).toBeNull()
  })
})
