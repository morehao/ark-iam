import { create } from 'zustand'
import { persist } from 'zustand/middleware'

interface AuthState {
  accessToken: string | null
  refreshToken: string | null
  userInfo: {
    userID: number
    name: string
    tenantID: number
    isOwner: number
  } | null
  personInfo: {
    personID: number
    name: string
    avatar: string
  } | null
  currentTenant: {
    tenantID: number
    name: string
    tag: string
    userID: number
    isOwner: number
  } | null
  tenants: Array<{
    tenantID: number
    name: string
    tag: string
    userID: number
    isOwner: number
  }>

  setTokens: (accessToken: string, refreshToken: string) => void
  setUserinfo: (userInfo: AuthState['userInfo'], personInfo: AuthState['personInfo']) => void
  setTenants: (tenants: AuthState['tenants']) => void
  setCurrentTenant: (tenant: AuthState['currentTenant']) => void
  logout: () => void
}

export const useAuthStore = create<AuthState>()(
  persist(
    (set) => ({
      accessToken: null,
      refreshToken: null,
      userInfo: null,
      personInfo: null,
      currentTenant: null,
      tenants: [],

      setTokens: (accessToken, refreshToken) => {
        localStorage.setItem('accessToken', accessToken)
        localStorage.setItem('refreshToken', refreshToken)
        set({ accessToken, refreshToken })
      },

      setUserinfo: (userInfo, personInfo) => set({ userInfo, personInfo }),

      setTenants: (tenants) => set({ tenants }),

      setCurrentTenant: (tenant) => set({ currentTenant: tenant }),

      logout: () => {
        localStorage.removeItem('accessToken')
        localStorage.removeItem('refreshToken')
        set({
          accessToken: null,
          refreshToken: null,
          userInfo: null,
          personInfo: null,
          currentTenant: null,
          tenants: [],
        })
      },
    }),
    {
      name: 'auth-storage',
      partialize: (state) => ({
        accessToken: state.accessToken,
        refreshToken: state.refreshToken,
        userInfo: state.userInfo,
        personInfo: state.personInfo,
        currentTenant: state.currentTenant,
        tenants: state.tenants,
      }),
    }
  )
)