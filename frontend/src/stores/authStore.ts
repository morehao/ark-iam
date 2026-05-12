import { create } from 'zustand'
import { persist } from 'zustand/middleware'
import type { AuthStage, PersonInfo, TenantMembership, UserInfo } from '../types/auth'

interface AuthState {
  authStage: AuthStage
  personToken: string | null
  tenantToken: string | null
  refreshToken: string | null
  tenants: TenantMembership[]
  currentTenant: TenantMembership | null
  personInfo: PersonInfo | null
  userInfo: UserInfo | null
  accessToken: string | null

  setPersonSession: (payload: {
    personToken: string
    refreshToken: string
    tenants: TenantMembership[]
    personInfo?: PersonInfo | null
  }) => void
  setTokens: (personToken: string, refreshToken: string) => void
  setTenantSession: (payload: {
    tenantToken: string
    refreshToken: string
    currentTenant: TenantMembership
    userInfo?: UserInfo | null
  }) => void
  clearTenantSession: () => void
  setPersonInfo: (personInfo: PersonInfo | null) => void
  setUserInfo: (userInfo: UserInfo | null) => void
  setTenants: (tenants: AuthState['tenants']) => void
  setCurrentTenant: (tenant: AuthState['currentTenant']) => void
  logout: () => void
}

const persistSession = (key: string, value: string | null) => {
  if (value === null) {
    localStorage.removeItem(key)
    return
  }
  localStorage.setItem(key, value)
}

const personRefreshTokenStorageKey = 'personRefreshToken'

export const useAuthStore = create<AuthState>()(
  persist(
    (set) => ({
      authStage: 'anonymous',
      personToken: null,
      tenantToken: null,
      refreshToken: null,
      tenants: [],
      currentTenant: null,
      personInfo: null,
      userInfo: null,
      accessToken: null,

      setPersonSession: ({ personToken, refreshToken, tenants, personInfo = null }) => {
        persistSession('personToken', personToken)
        persistSession('tenantToken', null)
        persistSession(personRefreshTokenStorageKey, refreshToken)
        persistSession('refreshToken', refreshToken)
        set({
          authStage: 'person',
          personToken,
          tenantToken: null,
          refreshToken,
          tenants,
          currentTenant: null,
          personInfo,
          userInfo: null,
          accessToken: personToken,
        })
      },

      setTokens: (personToken, refreshToken) => {
        persistSession('personToken', personToken)
        persistSession('tenantToken', null)
        persistSession(personRefreshTokenStorageKey, refreshToken)
        persistSession('refreshToken', refreshToken)
        set({
          authStage: 'person',
          personToken,
          tenantToken: null,
          refreshToken,
          currentTenant: null,
          userInfo: null,
          accessToken: personToken,
        })
      },

      setTenantSession: ({ tenantToken, refreshToken, currentTenant, userInfo = null }) => {
        persistSession('tenantToken', tenantToken)
        persistSession('refreshToken', refreshToken)
        set((state) => ({
          authStage: 'tenant',
          tenantToken,
          refreshToken,
          currentTenant,
          userInfo,
          accessToken: tenantToken,
          personToken: state.personToken,
        }))
      },

      clearTenantSession: () => {
        persistSession('tenantToken', null)
        const personRefreshToken = localStorage.getItem(personRefreshTokenStorageKey)
        persistSession('refreshToken', personRefreshToken)
        set((state) => ({
          authStage: state.personToken ? 'person' : 'anonymous',
          tenantToken: null,
          refreshToken: state.personToken ? personRefreshToken : null,
          currentTenant: null,
          userInfo: null,
          accessToken: state.personToken,
        }))
      },

      setPersonInfo: (personInfo) => set({ personInfo }),

      setUserInfo: (userInfo) => set({ userInfo }),

      setTenants: (tenants) => set({ tenants }),

      setCurrentTenant: (tenant) => set({ currentTenant: tenant }),

      logout: () => {
        persistSession('personToken', null)
        persistSession('tenantToken', null)
        persistSession(personRefreshTokenStorageKey, null)
        persistSession('refreshToken', null)
        set({
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
      },
    }),
    {
      name: 'auth-storage',
      partialize: (state) => ({
        authStage: state.authStage,
        personToken: state.personToken,
        tenantToken: state.tenantToken,
        refreshToken: state.refreshToken,
        tenants: state.tenants,
        currentTenant: state.currentTenant,
        personInfo: state.personInfo,
        userInfo: state.userInfo,
        accessToken: state.accessToken,
      }),
    }
  )
)
