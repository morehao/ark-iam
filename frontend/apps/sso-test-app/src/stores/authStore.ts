import { create } from 'zustand'
import { persist } from 'zustand/middleware'
import type { AuthStage, PersonInfo, TenantInfo } from '@ark-iam/shared'
import { parseJWT } from '../utils/oidc'

interface AuthState {
  authStage: AuthStage
  accessToken: string | null
  idToken: string | null
  refreshToken: string | null
  expiresAt: number | null
  personInfo: PersonInfo | null
  tenantInfo: TenantInfo | null
  beginChecking: () => void
  setAuthenticatedSession: (tokens: {
    accessToken: string
    idToken: string
    refreshToken: string
    expiresIn: number
  }) => void
  updateTokens: (tokens: {
    accessToken: string
    idToken?: string
    refreshToken: string
    expiresIn: number
  }) => void
  setPersonInfo: (info: PersonInfo | null) => void
  setTenantInfo: (info: TenantInfo | null) => void
  markAnonymous: () => void
  clearSession: () => void
}

function extractTenantFromToken(accessToken: string): TenantInfo | null {
  const claims = parseJWT(accessToken)
  if (!claims) return null
  const tenantID = claims['tenant_id'] as number
  if (!tenantID) return null
  return { tenantID, tenantName: '' }
}

export const useAuthStore = create<AuthState>()(
  persist(
    (set) => ({
      authStage: 'anonymous',
      accessToken: null,
      idToken: null,
      refreshToken: null,
      expiresAt: null,
      personInfo: null,
      tenantInfo: null,
      beginChecking: () => set((state) => ({ authStage: state.accessToken ? 'authenticated' : 'checking' })),
      setAuthenticatedSession: ({ accessToken, idToken, refreshToken, expiresIn }) => {
        const expiresAt = Date.now() + expiresIn * 1000
        const tenantInfo = extractTenantFromToken(accessToken)
        set({ authStage: 'authenticated', accessToken, idToken, refreshToken, expiresAt, tenantInfo, personInfo: null })
      },
      updateTokens: ({ accessToken, idToken, refreshToken, expiresIn }) => {
        const expiresAt = Date.now() + expiresIn * 1000
        const tenantInfo = extractTenantFromToken(accessToken)
        set((state) => ({ authStage: 'authenticated', accessToken, idToken: idToken ?? state.idToken, refreshToken, expiresAt, tenantInfo: tenantInfo ?? state.tenantInfo }))
      },
      setPersonInfo: (personInfo) => set({ personInfo }),
      setTenantInfo: (tenantInfo) => set({ tenantInfo }),
      markAnonymous: () => set({ authStage: 'anonymous', accessToken: null, idToken: null, refreshToken: null, expiresAt: null, personInfo: null, tenantInfo: null }),
      clearSession: () => set({ authStage: 'anonymous', accessToken: null, idToken: null, refreshToken: null, expiresAt: null, personInfo: null, tenantInfo: null }),
    }),
    {
      name: 'auth-storage',
      partialize: (state) => ({
        authStage: state.authStage === 'checking' ? 'anonymous' : state.authStage,
        accessToken: state.accessToken,
        idToken: state.idToken,
        refreshToken: state.refreshToken,
        expiresAt: state.expiresAt,
        personInfo: state.personInfo,
        tenantInfo: state.tenantInfo,
      }),
    }
  )
)
