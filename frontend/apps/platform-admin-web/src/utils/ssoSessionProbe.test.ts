import { beforeEach, describe, expect, test, vi } from 'vitest'
import { renderHook } from '@testing-library/react'
import { useSSOSessionProbe } from './ssoSessionProbe'

const mockRemoveUser = vi.fn().mockResolvedValue(undefined)
const mockSigninSilent = vi.fn()
const mockUnsubscribe = vi.fn()
const mockAddSilentRenewError = vi.fn().mockReturnValue(mockUnsubscribe)

vi.mock('react-oidc-context', () => ({
  useAuth: () => ({
    isAuthenticated: true,
    isLoading: false,
    activeNavigator: undefined,
    signinSilent: mockSigninSilent,
    removeUser: mockRemoveUser,
  }),
}))

describe('useSSOSessionProbe', () => {
  beforeEach(() => {
    mockRemoveUser.mockClear()
    mockSigninSilent.mockClear()
    mockUnsubscribe.mockClear()
    mockAddSilentRenewError.mockClear()
  })

  test('registers a silent renew error listener that calls removeUser', () => {
    mockAddSilentRenewError.mockReturnValue(mockUnsubscribe)
    mockSigninSilent.mockResolvedValue(undefined)

    const auth = {
      isAuthenticated: true,
      activeNavigator: undefined,
      signinSilent: mockSigninSilent,
      removeUser: mockRemoveUser,
      events: { addSilentRenewError: mockAddSilentRenewError },
    } as any

    const { unmount } = renderHook(() => useSSOSessionProbe(auth))

    expect(mockAddSilentRenewError).toHaveBeenCalledWith(expect.any(Function))

    const cb = mockAddSilentRenewError.mock.calls[0][0] as () => void
    cb()
    expect(mockRemoveUser).toHaveBeenCalled()

    unmount()
    expect(mockUnsubscribe).toHaveBeenCalled()
  })

  test('clears user when initial probe fails', async () => {
    mockSigninSilent.mockRejectedValue(new Error('login_required'))

    const auth = {
      isAuthenticated: true,
      activeNavigator: undefined,
      signinSilent: mockSigninSilent,
      removeUser: mockRemoveUser,
      events: { addSilentRenewError: () => () => {} },
    } as any

    renderHook(() => useSSOSessionProbe(auth))

    await new Promise((r) => setTimeout(r, 0))
    expect(mockSigninSilent).toHaveBeenCalledWith({ prompt: 'none', forceIframeAuth: true })
    expect(mockRemoveUser).toHaveBeenCalled()
  })

  test('clears user when silent renew resolves null (SSO session revoked)', async () => {
    mockSigninSilent.mockResolvedValue(null)

    const auth = {
      isAuthenticated: true,
      activeNavigator: undefined,
      signinSilent: mockSigninSilent,
      removeUser: mockRemoveUser,
      events: { addSilentRenewError: () => () => {} },
    } as any

    renderHook(() => useSSOSessionProbe(auth))

    await new Promise((r) => setTimeout(r, 0))
    expect(mockSigninSilent).toHaveBeenCalledWith({ prompt: 'none', forceIframeAuth: true })
    expect(mockRemoveUser).toHaveBeenCalled()
  })

  test('keeps user when silent renew returns a valid user', async () => {
    const mockUser = { id_token: 'x', access_token: 'y' }
    mockSigninSilent.mockResolvedValue(mockUser)

    const auth = {
      isAuthenticated: true,
      activeNavigator: undefined,
      signinSilent: mockSigninSilent,
      removeUser: mockRemoveUser,
      events: { addSilentRenewError: () => () => {} },
    } as any

    renderHook(() => useSSOSessionProbe(auth))

    await new Promise((r) => setTimeout(r, 0))
    expect(mockSigninSilent).toHaveBeenCalledWith({ prompt: 'none', forceIframeAuth: true })
    expect(mockRemoveUser).not.toHaveBeenCalled()
  })
})
