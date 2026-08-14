import { useEffect } from 'react'
import type { AuthContextProps } from 'react-oidc-context'

/**
 * SSO 会话探活：监听 silent renew 失败，SSO 会话失效（其他端点全局登出）时
 * 清除本地 user，实现"一处登出、处处登出"（在 token 自动续期时生效）。
 *
 * 注意：不要通过主动调用 auth.signinSilent() 做轮询探活。实测在当前后端
 * （SP 拥有 refresh_token）下，每次 signinSilent 都会刷新 user 并触发
 * isAuthenticated 翻转 → 组件重载 → 再次探活，形成每数百毫秒一次的自激
 * renew 循环，导致页面周期性白屏；ref 无法阻断（翻转伴随 remount 重置）。
 * 因此这里仅以 silent renew 失败事件兜底登出，token 自动续期交给
 * `automaticSilentRenew`。
 */
export function useSSOSessionProbe(auth: AuthContextProps) {
  useEffect(() => {
    if (!auth.events?.addSilentRenewError) return
    const unsub = auth.events.addSilentRenewError(() => {
      void auth.removeUser()
    })
    return unsub
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [auth.events])
}
