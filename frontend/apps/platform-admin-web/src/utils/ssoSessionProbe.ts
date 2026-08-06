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

  // 页面加载时做一次探活：校验 SSO 会话是否仍有效
  useEffect(() => {
    if (!auth.isAuthenticated || auth.activeNavigator || probingRef.current) return
    probingRef.current = true
    void auth
      .signinSilent({ prompt: 'none', forceIframeAuth: true })
      .then((user) => {
        // signinSilent 解析为 null 说明后端返回 login_required：
        // 即该用户的 SSO 会话已被撤销（例如在兄弟应用全局登出），
        // 清除本地登录态，交由 App 的 isAuthenticated 逻辑回到登录页。
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

  // 打开中的标签页借助 automaticSilentRenew 定时出发的 silent renew：
  // 任一 silent renew 失败（含登出导致的 login_required）即清除本地登录态，
  // 由 App 的 isAuthenticated 逻辑触发重新认证，实现"一处登出、处处登出"
  useEffect(() => {
    if (!auth.events?.addSilentRenewError) return
    const unsub = auth.events.addSilentRenewError(() => {
      void auth.removeUser()
    })
    return unsub
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [auth.events])
}
