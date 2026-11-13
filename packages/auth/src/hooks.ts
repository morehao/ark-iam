import { useCallback, useEffect, useState } from 'react'
import { useAuth } from 'react-oidc-context'
import { message } from 'antd'
import {
  getMyTenants,
  logoutAllAPI,
  setUserProvider,
  setSessionExpiredHandler,
  useSSOSessionProbe,
} from '@ark-iam/api'
import { getCurrentTenantId, setCurrentTenantId } from './tenant'

export interface TenantSwitchInfo {
  tenants: { tenantID: string; name: string }[]
  loadTenants: () => Promise<void>
  handleSwitchTenant: (tenantID: string) => Promise<void>
}

export function useTenantSwitching(): TenantSwitchInfo {
  const auth = useAuth()
  const [tenants, setTenants] = useState<{ tenantID: string; name: string }[]>([])

  const loadTenants = useCallback(async () => {
    try {
      const resp = await getMyTenants()
      setTenants(resp.list || [])
    } catch {
      message.warning('获取可用租户失败')
    }
  }, [])

  const handleSwitchTenant = useCallback(
    async (tenantID: string) => {
      if (String(tenantID) === getCurrentTenantId()) return
      setCurrentTenantId(tenantID)
      try {
        await auth.removeUser()
      } catch {
        // 本地清理失败不阻断重授权
      }
      await auth.signinRedirect({ extraQueryParams: { tenant: String(tenantID) }, redirectMethod: 'replace' })
    },
    [auth],
  )

  return { tenants, loadTenants, handleSwitchTenant }
}

export function useLogout() {
  const auth = useAuth()
  return useCallback(async () => {
    try {
      await logoutAllAPI(auth.user?.refresh_token ?? '')
    } catch {
      // ignore：撤销自有 refresh token 为尽力而为，失败不阻断登出
    }
    // 先 signoutRedirect 再 removeUser，保证 id_token_hint 仍存在以撤销后端 SSO session
    try {
      await auth.signoutRedirect()
    } finally {
      await auth.removeUser()
    }
  }, [auth])
}

export function useAuthGuard() {
  const auth = useAuth()

  useSSOSessionProbe(auth)

  useEffect(() => {
    setUserProvider(() => auth.user)
  }, [auth.user])

  // 401（SSO 会话已被其它端点全局登出撤销）时，先移除本地 user 再跳转登录，
  // 避免 "401 → 刷新 → sessionStorage 残留 user → 再 401" 的死循环。
  useEffect(() => {
    setSessionExpiredHandler(() => {
      void auth.removeUser()
    })
  }, [auth])

  useEffect(() => {
    const claim = (auth.user?.profile as Record<string, unknown> | undefined)?.tenant_id
    if (claim != null && getCurrentTenantId() !== String(claim)) {
      setCurrentTenantId(claim as string | number)
    }
  }, [auth.user])

  return auth
}
