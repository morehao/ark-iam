import { describe, expect, it } from 'vitest'
import type { AxiosResponse, InternalAxiosRequestConfig } from 'axios'
import {
  ApiError,
  INSTALL_PENDING_REDIRECT,
  installRedirectForError,
  isInstallPendingError,
  oidcApi,
  oidcLogin,
} from '../api'

/** 用自定义 adapter 直接产出响应，避免真实网络请求。 */
function respondWith(body: unknown, status = 200) {
  return async (config: InternalAxiosRequestConfig) =>
    ({ data: body, status, statusText: 'OK', headers: {}, config }) as AxiosResponse
}

/** 模拟 axios 对非 2xx 的 reject（守卫/业务端点返回真实 HTTP 状态码）。 */
function rejectWith(status: number, data: unknown) {
  return async (config: InternalAxiosRequestConfig): Promise<AxiosResponse> => {
    throw Object.assign(new Error(`Request failed with status code ${status}`), {
      isAxiosError: true,
      config,
      response: { data, status, statusText: '', headers: {}, config },
    })
  }
}

describe('登录侧 107004 守卫：任何登录/认证调用失败都要跳 /install', () => {
  it('登录返回 107004 时抛出携带业务码的 ApiError，并给出 /install 跳转目标', async () => {
    oidcApiAdapter(respondWith({ code: 107004, msg: '系统尚未初始化，请先完成初始化', data: null }))
    const err = await oidcLogin({ authRequestID: 'req-1', identifier: 'admin', password: 'x' }).catch((e: unknown) => e)

    expect(err).toBeInstanceOf(ApiError)
    expect((err as ApiError).code).toBe(107004)
    expect(isInstallPendingError(err)).toBe(true)
    expect(installRedirectForError(err)).toBe('/install')
    expect(INSTALL_PENDING_REDIRECT).toBe('/install')
  })

  // 回归：守卫返回的是**真实 HTTP 409**（不是 200 + 信封），axios 会直接 reject。
  // 这条用例曾在生产路径上失效——错误原先只覆盖了 respondWith(200) 的形态，
  // 于是代码里那句 `body.code !== 0` 永远走不到，登录页也就永远不会跳 /install。
  it('登录返回 HTTP 409 + code=107004 时同样归一为 ApiError 并跳 /install', async () => {
    oidcApiAdapter(rejectWith(409, { code: 107004, msg: '系统尚未初始化，请先完成初始化', data: {} }))
    const err = await oidcLogin({ authRequestID: 'req-1', identifier: 'admin', password: 'x' }).catch((e: unknown) => e)

    expect(err).toBeInstanceOf(ApiError)
    expect((err as ApiError).code).toBe(107004)
    expect(installRedirectForError(err)).toBe('/install')
    // 展示文案取后端中文 msg，而不是 axios 的英文 "Request failed with status code 409"
    expect((err as Error).message).toBe('系统尚未初始化，请先完成初始化')
  })

  it('非 2xx 但响应体没有业务码时不归一（保持原错误，仍给出可读文案）', async () => {
    oidcApiAdapter(rejectWith(502, 'Bad Gateway'))
    const err = await oidcLogin({ authRequestID: 'req-1', identifier: 'admin', password: 'x' }).catch((e: unknown) => e)

    expect(installRedirectForError(err)).toBeNull()
    expect(String((err as Error).message)).not.toBe('')
  })

  it('其它业务码不触发跳转（仍按 message 展示）', async () => {
    oidcApiAdapter(respondWith({ code: 100104, msg: '参数不合法', data: null }))
    const err = await oidcLogin({ authRequestID: 'req-1', identifier: 'admin', password: 'x' }).catch((e: unknown) => e)

    expect((err as ApiError).code).toBe(100104)
    expect(isInstallPendingError(err)).toBe(false)
    expect(installRedirectForError(err)).toBeNull()
    expect((err as Error).message).toBe('参数不合法')
  })

  it('网络错误（非 ApiError）不触发跳转', () => {
    expect(installRedirectForError(new Error('Network Error'))).toBeNull()
    expect(isInstallPendingError(new Error('Network Error'))).toBe(false)
  })
})

function oidcApiAdapter(adapter: unknown) {
  oidcApi.defaults.adapter = adapter as never
}
