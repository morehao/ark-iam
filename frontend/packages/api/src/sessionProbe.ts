import { useEffect, useRef } from 'react'
import type { AuthContextProps } from 'react-oidc-context'

/**
 * SSO 会话探活：页面加载且已有本地 user 时，通过 silent renew（prompt=none）
 * 向后端校验 SSO 会话是否仍有效，实现"一处登出、处处登出"。
 */
export function useSSOSessionProbe(auth: AuthContextProps) {
  const probingRef = useRef(false)

  useEffect(() => {
    if (!auth.isAuthenticated || auth.activeNavigator || probingRef.current) return
    probingRef.current = true
    void auth
      .signinSilent({ prompt: 'none', forceIframeAuth: true })
      .then((user) => {
        if (!user) return auth.removeUser()
        return undefined
      })
      .catch(() => {
        return auth.removeUser()
      })
      .finally(() => {
        probingRef.current = false
      })
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [auth.isAuthenticated])

  useEffect(() => {
    if (!auth.events?.addSilentRenewError) return
    const unsub = auth.events.addSilentRenewError(() => {
      void auth.removeUser()
    })
    return unsub
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [auth.events])
}
