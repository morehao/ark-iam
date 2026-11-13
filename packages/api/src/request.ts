import axios, { AxiosError, AxiosInstance, InternalAxiosRequestConfig } from 'axios'
import { message } from 'antd'
import { BizCode } from '@ark-iam/types'

/**
 * silent：本次请求属于 best-effort 探测，失败由调用方自行兜底
 * （典型场景：菜单页首屏恢复"上次选择的应用"，该应用可能已被删除或换库，旧 ID 失效），
 * 此时不弹全局提示，避免每次进页面都弹一个调用方已经处理掉的错误。
 *
 * 语义边界：silent 只抑制「提示」（message.error / message.warning）；
 * 401/令牌失效的会话处置（handleSessionExpired → 跳登录页）是流程控制，不受 silent 影响。
 */
declare module 'axios' {
  export interface AxiosRequestConfig {
    silent?: boolean
  }
}

let userProvider: (() => { access_token: string } | null | undefined) | null = null

// sessionExpiredHandler 由 auth 层注册（见 @ark-iam/auth hooks）：
// 当 API 返回 401（SSO 会话已被其它端点全局登出撤销）时，先清除本地 user，
// 再跳转登录页，避免"401 → 刷新 → sessionStorage 残留 user → 再 401"的循环。
let sessionExpiredHandler: (() => void) | null = null

export function setUserProvider(provider: () => { access_token: string } | null | undefined) {
  userProvider = provider
}

export function setSessionExpiredHandler(handler: () => void) {
  sessionExpiredHandler = handler
}

export interface CreateApiClientOptions {
  /** 业务 API 基址（含版本段），如 '/v1' 或 'https://iam.example.com/v1'。默认 '/v1'。 */
  baseURL?: string
  /** 请求超时（毫秒），默认 30000。 */
  timeout?: number
  /** 追加到每个请求的静态请求头。 */
  headers?: Record<string, string>
}

function attachInterceptors(instance: AxiosInstance) {
  instance.interceptors.request.use((config: InternalAxiosRequestConfig) => {
    const user = userProvider?.()
    if (user?.access_token && config.headers) {
      config.headers.Authorization = `Bearer ${user.access_token}`
    }
    return config
  })

  instance.interceptors.response.use(
    (response) => {
      const { code, msg } = response.data as { code: number; msg: string }
      if (code === BizCode.Success) return response.data.data

      const isLogoutRequest = response.config.url?.includes('/auth/logout')

      if (code === BizCode.TokenExpired || code === BizCode.TokenInvalid || code === BizCode.Unauthorized) {
        if (isLogoutRequest) return Promise.reject(new Error(msg || 'session expired'))
        handleSessionExpired()
        return Promise.reject(new Error(msg || '未认证'))
      }
      if (code === BizCode.Forbidden || code === BizCode.PermissionDenied) {
        if (!response.config.silent) message.warning('暂无权限访问')
        return Promise.reject(new Error(msg || '暂无权限'))
      }
      if (!response.config.silent) message.error(msg || '请求失败')
      return Promise.reject(new Error(msg || '请求失败'))
    },
    async (error: AxiosError) => {
      if (error.response?.status === 401) {
        handleSessionExpired()
        return Promise.reject(error)
      }
      const data = error.response?.data as any
      if (!error.config?.silent) message.error(data?.msg || '请求失败')
      return Promise.reject(error)
    },
  )
  return instance
}

function handleSessionExpired() {
  // 先清本地 user（避免刷新后 sessionStorage 残留导致 401 循环），再跳转登录。
  sessionExpiredHandler?.()
  window.location.href = '/'
}

/**
 * createApiClient 创建一个独立配置的业务 API 客户端（axios 实例）。
 * 外部接入方用它指定自己的 baseURL（外部应用通常不带 /v1 前缀或指向别的域）。
 * 401/令牌失效、权限、业务错误的拦截逻辑与仓内默认实例完全一致。
 */
export function createApiClient(opts: CreateApiClientOptions = {}): AxiosInstance {
  const instance = axios.create({
    baseURL: opts.baseURL ?? '/v1',
    timeout: opts.timeout ?? 30000,
    headers: opts.headers,
  })
  return attachInterceptors(instance)
}

// 默认实例：供仓内（与 IAM 同源、API 挂在 /v1 下）的应用直接使用。
const request = createApiClient()

export default request
