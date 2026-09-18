import axios, { AxiosError, InternalAxiosRequestConfig } from 'axios'
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

const request = axios.create({ baseURL: '/v1', timeout: 30000 })

request.interceptors.request.use((config: InternalAxiosRequestConfig) => {
  const user = userProvider?.()
  if (user?.access_token && config.headers) {
    config.headers.Authorization = `Bearer ${user.access_token}`
  }
  return config
})

function handleSessionExpired() {
  // 先清本地 user（避免刷新后 sessionStorage 残留导致 401 循环），再跳转登录。
  sessionExpiredHandler?.()
  window.location.href = '/'
}

request.interceptors.response.use(
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

export default request
