import { useCallback, useEffect, useState } from 'react'
import { useAuth } from 'react-oidc-context'
import { message } from 'antd'
import { getMyTenants, logoutAllAPI, setUserProvider, useSSOSessionProbe } from '@ark-iam/api'
import { getCurrentTenantId, setCurrentTenantId } from './tenant'

export interface TenantSwitchInfo {
  tenants: { tenantID: number; name: string }[]
  loadTenants: () => Promise<void>
  handleSwitchTenant: (tenantID: number) => Promise<void>
}

export function useTenantSwitching(): TenantSwitchInfo {
  const auth = useAuth()
  const [tenants, setTenants] = useState<{ tenantID: number; name: string }[]>([])

  const loadTenants = useCallback(async () => {
    try {
      const resp = await getMyTenants()
      setTenants(resp.list || [])
    } catch {
      message.warning('获取可用租户失败')
    }
  }, [])

  const handleSwitchTenant = useCallback(
    async (tenantID: number) => {
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

  useEffect(() => {
    const claim = (auth.user?.profile as Record<string, unknown> | undefined)?.tenant_id
    if (claim != null && getCurrentTenantId() !== String(claim)) {
      setCurrentTenantId(claim as string | number)
    }
  }, [auth.user])

  return auth
}
