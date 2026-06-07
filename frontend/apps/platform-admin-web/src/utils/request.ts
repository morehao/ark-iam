import axios, { AxiosError, InternalAxiosRequestConfig } from 'axios'
import { message } from 'antd'
import { BizCode } from '@ark-iam/shared'
import { useAuthStore } from '../stores/authStore'
import { refreshTokens } from './oidc'

const request = axios.create({ baseURL: '/v1/iam', timeout: 30000 })

let isRefreshing = false
let pendingRequests: Array<(token: string) => void> = []

function handleAnonymousRedirect() {
  useAuthStore.getState().clearSession()
  window.location.href = '/login'
}

async function handleTokenExpired(originalConfig: InternalAxiosRequestConfig): Promise<any> {
  const store = useAuthStore.getState()
  if (!store.refreshToken) {
    handleAnonymousRedirect()
    return Promise.reject(new Error('no refresh token'))
  }
  if (!isRefreshing) {
    isRefreshing = true
    try {
      const resp = await refreshTokens(store.refreshToken)
      store.updateTokens({
        accessToken: resp.access_token,
        idToken: resp.id_token,
        refreshToken: resp.refresh_token,
        expiresIn: resp.expires_in,
      })
      pendingRequests.forEach((cb) => cb(resp.access_token))
      pendingRequests = []
      originalConfig.headers!.Authorization = `Bearer ${resp.access_token}`
      return request(originalConfig)
    } catch {
      pendingRequests = []
      handleAnonymousRedirect()
      return Promise.reject(new Error('token refresh failed'))
    } finally {
      isRefreshing = false
    }
  }
  return new Promise((resolve) => {
    pendingRequests.push((newToken: string) => {
      originalConfig.headers!.Authorization = `Bearer ${newToken}`
      resolve(request(originalConfig))
    })
  })
}

request.interceptors.request.use((config: InternalAxiosRequestConfig) => {
  const token = useAuthStore.getState().accessToken
  if (token && config.headers) {
    config.headers.Authorization = `Bearer ${token}`
  }
  return config
})

request.interceptors.response.use(
  (response) => {
    const { code, msg } = response.data as { code: number; msg: string }
    if (code === BizCode.Success) return response.data.data
    const isLogoutRequest = response.config.url?.includes('/auth/logout')
    if (code === BizCode.TokenExpired) {
      if (isLogoutRequest) {
        return Promise.reject(new Error('session expired'))
      }
      return handleTokenExpired(response.config)
    }
    if (code === BizCode.TokenInvalid || code === BizCode.Unauthorized) {
      if (isLogoutRequest) {
        return Promise.reject(new Error(msg || '未认证'))
      }
      handleAnonymousRedirect()
      return Promise.reject(new Error(msg || '未认证'))
    }
    if (code === BizCode.Forbidden || code === BizCode.PermissionDenied) {
      message.warning('暂无权限访问')
      return Promise.reject(new Error(msg || '暂无权限'))
    }
    message.error(msg || '请求失败')
    return Promise.reject(new Error(msg || '请求失败'))
  },
  async (error: AxiosError) => {
    if (error.response?.status === 401) {
      handleAnonymousRedirect()
      return Promise.reject(error)
    }
    const data = error.response?.data as any
    message.error(data?.msg || '请求失败')
    return Promise.reject(error)
  }
)

export default request
