import { useEffect, useRef } from 'react'
import type { AuthContextProps } from 'react-oidc-context'

/**
 * silent renew 失败后的「自愈」策略。
 *
 * 背景（为什么不能直接登出）：
 * - 前端在 sessionStorage 存 OIDC user（含 refresh_token），而 refresh token 是**单次轮换**
 *   （用一次即撤销，服务端只有极短的并发宽限）。多标签页各自轮换、标签页被浏览器冻结后恢复、
 *   后台节流漏掉一次 renew 后的重放等场景，都会让下一次 `signinSilent()` 用**已被撤销的**
 *   refresh token 而抛 `invalid_grant`。
 * - 但此时用户仍持有有效的 **SSO 会话**（`iam_sso_session` cookie，24h 滑动续期）。SSO 会话
 *   还在时本可以用 `prompt=none` 静默重新登录，无声换回一整套新 token——无需打扰用户。
 * - 旧实现收到 silent renew error 就无条件 `removeUser()` → 触发登录跳转，等于「SSO 明明还活着
 *   却把用户踢出」，表现为「过一会儿（access token 15 分钟到期触发首次 renew）就跳登录页」。
 *
 * 现策略：silent renew 失败时先尝试一次静默重登；只有静默重登也失败（SSO 会话确实没了，
 * 例如在别处全局登出）才清除本地 user 跳登录页，保留「一处登出、处处登出」的即时性。
 *
 * 注意：这里**不是轮询**——只在 silent renew 失败事件上兜底触发一次，不会自激；并通过
 * 单次防重入锁保证同一轮错误风暴里只跑一次静默重登（并发多次 signinSilent 会用同一把
 * refresh token 互相作废）。
 */

export interface SilentRenewRecoveryDeps {
  /**
   * 静默重新登录：必须**强制走 iframe `prompt=none`**（forceIframeAuth=true），用仍有效的
   * SSO 会话换回全新 token；失败时抛错。
   *
   * 关键：普通 `signinSilent()` 在本地存在 refresh_token 时会走 refresh token 轮换，
   * 而失效的 refresh token（轮换失配/多标签页/后台节流）只会再次 `invalid_grant`、且
   * **不会自动回退到 iframe**。因此这里必须显式 forceIframeAuth，绕过本地 refresh token。
   */
  signinSilent: () => Promise<unknown>
  /** 彻底清除本地 user（SSO 会话已失效时的最终处置）。 */
  removeUser: () => Promise<void> | void
  /** 静默重登成功后回调（默认为空；供测试/诊断观察）。 */
  onRestored?: () => void
}

/**
 * 构造 silent renew 失败处理器。抽成纯函数便于在不依赖 React/浏览器环境的单元测试里直接验证。
 * 返回的处理器可被多次调用（如 `auth.events.addSilentRenewError`），内部做防重入。
 */
export function createSilentRenewErrorHandler(deps: SilentRenewRecoveryDeps): () => void {
  let recovering = false
  return () => {
    // 防重入：一轮错误风暴里只尝试一次静默重登，避免并发 signinSilent 用同一把 refresh token 互踩。
    if (recovering) return
    recovering = true
    void (async () => {
      try {
        await deps.signinSilent()
        // 静默重登成功：新的 token 已写回 store，isAuthenticated 保持 true，用户无感。
        deps.onRestored?.()
      } catch {
        // SSO 会话确实失效（或静默登录亦失败）：此时才真正清除本地 user 并跳登录页。
        await deps.removeUser()
      } finally {
        recovering = false
      }
    })()
  }
}

/**
 * SSO 会话探活（自愈式）：监听 silent renew 失败，**先尝试静默重登救回**，救不回（SSO 会话
 * 失效，例如其他端点全局登出）才清除本地 user，实现"一处登出、处处登出"。
 *
 * 注意：不要通过主动调用 auth.signinSilent() 做轮询探活（那会形成 isAuthenticated 反复翻转
 * → 组件重载 → 再次探活的自激循环，导致页面周期性白屏）。这里仅在 silent renew **失败事件**
 * 上兜底触发一次，token 的正常自动续期仍交给 `automaticSilentRenew`。
 */
export function useSSOSessionProbe(auth: AuthContextProps) {
  // 用 ref 承接最新 auth，避免 signinSilent/removeUser 引用变化导致反复解绑重绑事件。
  const authRef = useRef(auth)
  authRef.current = auth

  // 处理器只创建一次（防重入锁贯穿组件生命周期）。
  const handlerRef = useRef<(() => void) | null>(null)
  if (handlerRef.current === null) {
    handlerRef.current = createSilentRenewErrorHandler({
      // forceIframeAuth：强制走 iframe 的 prompt=none，用 SSO cookie 重新授权，
      // 绕过本地可能已失效的 refresh token（否则会再次 invalid_grant 而救不回）。
      signinSilent: () => authRef.current.signinSilent({ forceIframeAuth: true }),
      removeUser: () => authRef.current.removeUser(),
    })
  }

  useEffect(() => {
    if (!auth.events?.addSilentRenewError) return
    const unsub = auth.events.addSilentRenewError(() => handlerRef.current?.())
    return unsub
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [auth.events])
}
