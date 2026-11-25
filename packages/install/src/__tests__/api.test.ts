import { describe, expect, it } from 'vitest'
import type { AxiosResponse, InternalAxiosRequestConfig } from 'axios'
import { ApiError } from '@ark-iam/login'
import { getInstallStatus, initializeInstall, installApi } from '../api'
import type { InstallInitializeReq, InstallStatus } from '../types'

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

function installApiAdapter(adapter: unknown) {
  installApi.defaults.adapter = adapter as never
}
