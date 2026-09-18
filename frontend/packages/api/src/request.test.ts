import { beforeEach, describe, expect, it, vi } from 'vitest'
import type { AxiosResponse } from 'axios'
import { BizCode } from '@ark-iam/types'

const mockMessageError = vi.fn()
const mockMessageWarning = vi.fn()

vi.mock('antd', () => ({
  message: {
    error: (...args: unknown[]) => mockMessageError(...args),
    warning: (...args: unknown[]) => mockMessageWarning(...args),
  },
}))

const { default: request, setSessionExpiredHandler } = await import('./request')

/** 用自定义 adapter 直接产出统一信封响应，避免真实网络请求。 */
function respondWith(body: { code: number; msg: string }) {
  request.defaults.adapter = async (config) =>
    ({ data: body, status: 200, statusText: 'OK', headers: {}, config }) as AxiosResponse
}

/**
 * silent 回归（2026-09）：菜单页首屏恢复"上次选择的应用"是 best-effort 探测——
 * 应用可能已被删除、或开发库删库重建后主键变化导致旧 ID 失效（服务端 100735 应用不存在）。
 * 这类失败由调用方静默回退，不得再弹全局错误提示；但 silent 只抑制提示，
 * 401 的会话处置（重新登录）属流程控制，必须照常生效。
 */
describe('request 全局错误提示的 silent 开关', () => {
  beforeEach(() => {
    mockMessageError.mockReset()
    mockMessageWarning.mockReset()
  })

  it('业务失败默认弹全局错误提示', async () => {
    respondWith({ code: 100735, msg: '应用不存在' })

    await expect(request.get('/platform/applications/stale-id')).rejects.toThrow('应用不存在')
    expect(mockMessageError).toHaveBeenCalledWith('应用不存在')
  })

  it('silent 请求失败不弹提示，仍以 reject 交调用方兜底', async () => {
    respondWith({ code: 100735, msg: '应用不存在' })

    await expect(request.get('/platform/applications/stale-id', { silent: true })).rejects.toThrow('应用不存在')
    expect(mockMessageError).not.toHaveBeenCalled()
    expect(mockMessageWarning).not.toHaveBeenCalled()
  })

  it('silent 不改变 401 的会话处置：仍触发重新登录', async () => {
    // request.ts 的会话处置会写 window.location，node 环境下补最小桩
    const globals = globalThis as { window?: unknown }
    const originalWindow = globals.window
    globals.window = { location: { href: '' } }
    const sessionExpired = vi.fn()
    setSessionExpiredHandler(sessionExpired)

    try {
      respondWith({ code: BizCode.Unauthorized, msg: '未认证' })

      await expect(request.get('/platform/applications/stale-id', { silent: true })).rejects.toThrow('未认证')
      expect(sessionExpired).toHaveBeenCalledTimes(1)
      expect(mockMessageError).not.toHaveBeenCalled()
    } finally {
      globals.window = originalWindow
    }
  })
})
