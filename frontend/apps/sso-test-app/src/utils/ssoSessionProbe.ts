import { useEffect, useRef } from 'react'
import type { AuthContextProps } from 'react-oidc-context'

/**
 * SSO 会话探活：页面加载且已有本地 user 时，通过 silent renew（prompt=none）
 * 向后端校验 SSO 会话是否仍有效。
 *
 * 若用户已在其他应用执行全局登出，后端 SSO session 被撤销，
 * silent renew 会返回 login_required，此时清除本地登录态，
 * 由 App 的 isAuthenticated 逻辑触发重新认证，实现"一处登出、处处登出"。
 */
export function useSSOSessionProbe(auth: AuthContextProps) {
  const probingRef = useRef(false)

  useEffect(() => {
    if (!auth.isAuthenticated || auth.activeNavigator || probingRef.current) return
    probingRef.current = true
    void auth
      .signinSilent({ prompt: 'none' })
      .catch(() => {
        // silent renew 失败（含 login_required）说明 SSO 会话已失效
        return auth.removeUser()
      })
      .finally(() => {
        probingRef.current = false
      })
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [auth.isAuthenticated])
}
