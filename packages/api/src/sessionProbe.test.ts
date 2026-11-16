import { describe, expect, it, vi } from 'vitest'
import { createSilentRenewErrorHandler } from './sessionProbe'

/**
 * 回归：silent renew 失败时的「自愈」策略。
 *
 * 背景 bug：SSO 会话仍有效、但 sessionStorage 里的 refresh token 因单次轮换/多标签页/后台节流
 * 等原因失效时，旧实现无条件 removeUser() → 跳登录页，表现为「过一会儿就跳登录页」。
 * 现策略：先尝试静默重登（prompt=none，用仍有效的 SSO 会话）救回；救不回才登出。
 */
describe('silent renew 失败自愈', () => {
  it('静默重登成功时不登出、也不清 user', async () => {
    const signinSilent = vi.fn().mockResolvedValue(undefined)
    const removeUser = vi.fn()
    const onRestored = vi.fn()

    const handler = createSilentRenewErrorHandler({ signinSilent, removeUser, onRestored })
    handler()
    // 等待内部 async 落定
    await vi.waitFor(() => expect(signinSilent).toHaveBeenCalledTimes(1))
    await Promise.resolve()

    expect(removeUser).not.toHaveBeenCalled()
    expect(onRestored).toHaveBeenCalledTimes(1)
  })

  it('静默重登失败（SSO 会话确实失效）时才登出', async () => {
    const signinSilent = vi.fn().mockRejectedValue(new Error('login_required'))
    const removeUser = vi.fn().mockResolvedValue(undefined)

    const handler = createSilentRenewErrorHandler({ signinSilent, removeUser })
    handler()
    await vi.waitFor(() => expect(removeUser).toHaveBeenCalledTimes(1))
    expect(signinSilent).toHaveBeenCalledTimes(1)
  })

  it('同一轮错误风暴里并发触发只尝试一次静默重登（防重入，避免 refresh token 互踩）', async () => {
    let resolveSilent: () => void = () => {}
    const signinSilent = vi.fn(
      () => new Promise<void>((res) => { resolveSilent = res }),
    )
    const removeUser = vi.fn()

    const handler = createSilentRenewErrorHandler({ signinSilent, removeUser })
    // 模拟 silent renew 失败事件在极短时间内连续触发多次
    handler()
    handler()
    handler()

    expect(signinSilent).toHaveBeenCalledTimes(1)
    resolveSilent()
    await vi.waitFor(() => expect(removeUser).not.toHaveBeenCalled())
    expect(signinSilent).toHaveBeenCalledTimes(1)
  })

  it('上一轮恢复完成后，后续失败可再次尝试（锁被正确释放）', async () => {
    const signinSilent = vi.fn().mockRejectedValue(new Error('boom'))
    const removeUser = vi.fn().mockResolvedValue(undefined)

    const handler = createSilentRenewErrorHandler({ signinSilent, removeUser })
    handler()
    await vi.waitFor(() => expect(removeUser).toHaveBeenCalledTimes(1))

    // 模拟下一轮（例如用户重新登录后又失效）：仍应能再次尝试
    handler()
    await vi.waitFor(() => expect(signinSilent).toHaveBeenCalledTimes(2))
  })
})
