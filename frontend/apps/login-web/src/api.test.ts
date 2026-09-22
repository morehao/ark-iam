import { describe, expect, it } from 'vitest'
import type { AxiosResponse, InternalAxiosRequestConfig } from 'axios'
import {
  ApiError,
  INSTALL_PENDING_REDIRECT,
  getInstallStatus,
  initializeInstall,
  installApi,
  installRedirectForError,
  isInstallPendingError,
  oidcApi,
  oidcLogin,
} from './api'
import type { InstallInitializeReq, InstallStatus } from './install/types'

/** 用自定义 adapter 直接产出响应，避免真实网络请求。 */
function respondWith(body: unknown, status = 200) {
  return async (config: InternalAxiosRequestConfig) =>
    ({ data: body, status, statusText: 'OK', headers: {}, config }) as AxiosResponse
}

/** 模拟 axios 对非 2xx 的 reject（install 端点返回真实 HTTP 状态码）。 */
function rejectWith(status: number, data: unknown) {
  return async (config: InternalAxiosRequestConfig): Promise<AxiosResponse> => {
    throw Object.assign(new Error(`Request failed with status code ${status}`), {
      isAxiosError: true,
      config,
      response: { data, status, statusText: '', headers: {}, config },
    })
  }
}

function readHeader(config: InternalAxiosRequestConfig, name: string): string | undefined {
  const headers = config.headers as unknown as { get?: (n: string) => unknown; [k: string]: unknown }
  if (typeof headers.get === 'function') {
    const value = headers.get(name)
    if (value !== undefined && value !== null) return String(value)
  }
  const direct = headers[name] ?? headers[name.toLowerCase()]
  return direct === undefined || direct === null ? undefined : String(direct)
}

const STATUS_BODY: InstallStatus = {
  initialized: false,
  tokenRequired: true,
  schemaReady: true,
  consoles: { platformAdminWeb: 'http://localhost:4001', tenantAdminWeb: 'http://localhost:4002' },
}

const INIT_REQUEST: InstallInitializeReq = {
  tenantName: '平台运营中心',
  adminUsername: 'admin',
  adminPassword: 'Admin123',
  adminEmail: 'admin@example.com',
}

describe('getInstallStatus', () => {
  it('兼容裸对象响应（契约示例形态）', async () => {
    installApiAdapter(respondWith(STATUS_BODY))
    await expect(getInstallStatus()).resolves.toEqual(STATUS_BODY)
  })

  it('兼容 {code:0,data} 统一信封', async () => {
    installApiAdapter(respondWith({ code: 0, msg: 'success', data: STATUS_BODY }))
    await expect(getInstallStatus()).resolves.toEqual(STATUS_BODY)
  })
})

describe('initializeInstall', () => {
  it('令牌只走 X-Bootstrap-Token 头，响应取 loginURL/adminUsername', async () => {
    let seen: InternalAxiosRequestConfig | undefined
    installApiAdapter(async (config: InternalAxiosRequestConfig) => {
      seen = config
      return {
        data: {
          report: { tenantId: 't_platform', changes: [{ entity: 'tenant', key: 't_platform', action: 'created' }] },
          adminUsername: 'admin',
          loginURL: 'http://localhost:4001/login',
        },
        status: 200,
        statusText: 'OK',
        headers: {},
        config,
      } as AxiosResponse
    })

    const resp = await initializeInstall(INIT_REQUEST, 'bootstrap-token-value')
    expect(resp.adminUsername).toBe('admin')
    expect(resp.loginURL).toBe('http://localhost:4001/login')
    expect(seen?.method?.toLowerCase()).toBe('post')
    expect(readHeader(seen!, 'X-Bootstrap-Token')).toBe('bootstrap-token-value')
    expect(JSON.stringify(seen?.data)).not.toContain('bootstrap-token-value')
  })

  it('令牌首尾空白与后端对称归一（粘贴带尾随换行也能通过）', async () => {
    let seen: InternalAxiosRequestConfig | undefined
    installApiAdapter(async (config: InternalAxiosRequestConfig) => {
      seen = config
      return {
        data: { report: { tenantId: 't', changes: [] }, adminUsername: 'admin', loginURL: '/login' },
        status: 200,
        statusText: 'OK',
        headers: {},
        config,
      } as AxiosResponse
    })

    await initializeInstall(INIT_REQUEST, '  bootstrap-token-value\n')
    expect(readHeader(seen!, 'X-Bootstrap-Token')).toBe('bootstrap-token-value')
  })

  it('409 + 107000 -> ApiError(107000)，页面据此切"已初始化"', async () => {
    installApiAdapter(rejectWith(409, { code: 107000, msg: '系统已完成初始化' }))
    const err = await initializeInstall(INIT_REQUEST, 'bad').catch((e: unknown) => e)
    expect(err).toBeInstanceOf(ApiError)
    expect((err as ApiError).code).toBe(107000)
  })

  it('401 + 107001 -> ApiError(107001)，保留表单并标红令牌', async () => {
    installApiAdapter(rejectWith(401, { code: 107001, msg: '初始化令牌缺失或不正确' }))
    const err = await initializeInstall(INIT_REQUEST, 'bad').catch((e: unknown) => e)
    expect((err as ApiError).code).toBe(107001)
  })

  it('503 无响应体时按状态码兜底为 107002', async () => {
    installApiAdapter(rejectWith(503, ''))
    const err = await initializeInstall(INIT_REQUEST, 'bad').catch((e: unknown) => e)
    expect((err as ApiError).code).toBe(107002)
  })

  it('500 -> 107003（系统错误）', async () => {
    installApiAdapter(rejectWith(500, { code: 107003 }))
    const err = await initializeInstall(INIT_REQUEST, 'bad').catch((e: unknown) => e)
    expect((err as ApiError).code).toBe(107003)
  })

  it('400 + 107005/107006 保持各自业务码', async () => {
    installApiAdapter(rejectWith(400, { code: 107005, msg: '初始化参数不合法' }))
    const bad = await initializeInstall(INIT_REQUEST, 'bad').catch((e: unknown) => e)
    expect((bad as ApiError).code).toBe(107005)

    installApiAdapter(rejectWith(400, { code: 107006, msg: '口令强度不足' }))
    const weak = await initializeInstall(INIT_REQUEST, 'bad').catch((e: unknown) => e)
    expect((weak as ApiError).code).toBe(107006)
  })
})

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

function installApiAdapter(adapter: unknown) {
  installApi.defaults.adapter = adapter as never
}

function oidcApiAdapter(adapter: unknown) {
  oidcApi.defaults.adapter = adapter as never
}
