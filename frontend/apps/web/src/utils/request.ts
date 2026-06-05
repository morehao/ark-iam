import axios, { AxiosError, InternalAxiosRequestConfig } from 'axios'
import { message } from 'antd'
import { useAuthStore } from '../stores/authStore'
import { BizCode } from '@ark-iam/shared'

const request = axios.create({
  baseURL: '/v1/iam',
  timeout: 30000,
})

let isRefreshing = false
let pendingRequests: Array<(token: string) => void> = []

async function handleTokenExpired(originalConfig: InternalAxiosRequestConfig): Promise<any> {
  const store = useAuthStore.getState()

  if (!store.refreshToken || !store.currentTenant) {
    store.clearTenantSession()
    return Promise.reject(new Error('refresh token or tenant not found'))
  }

  if (!isRefreshing) {
    isRefreshing = true
    try {
      const refreshResp = await axios.post('/v1/iam/auth/refreshToken', {
        refreshToken: store.refreshToken,
      })
      const newTenantToken = refreshResp.data.data.accessToken
      const newRefreshToken = refreshResp.data.data.refreshToken
      store.setTenantSession({
        tenantToken: newTenantToken,
        refreshToken: newRefreshToken,
        currentTenant: store.currentTenant,
        userInfo: store.userInfo,
      })

      pendingRequests.forEach((cb) => cb(newTenantToken))
      pendingRequests = []

      originalConfig.headers!.Authorization = `Bearer ${newTenantToken}`
      return request(originalConfig)
    } catch {
      store.clearTenantSession()
      pendingRequests = []
      return Promise.reject(new Error('refresh token failed'))
    } finally {
      isRefreshing = false
    }
  } else {
    return new Promise((resolve) => {
      pendingRequests.push((newToken: string) => {
        originalConfig.headers!.Authorization = `Bearer ${newToken}`
        resolve(request(originalConfig))
      })
    })
  }
}

request.interceptors.request.use(
  (config: InternalAxiosRequestConfig) => {
    const token = useAuthStore.getState().tenantToken ?? localStorage.getItem('tenantToken')
    if (token && config.headers) {
      config.headers.Authorization = `Bearer ${token}`
    }
    return config
  },
  (error: AxiosError) => {
    return Promise.reject(error)
  }
)

request.interceptors.response.use(
  (response) => {
    const { code, msg } = response.data as { code: number; msg: string }

    if (code === BizCode.Success) {
      return response.data.data
    }

    if (code === BizCode.TokenExpired) {
      return handleTokenExpired(response.config)
    }

    if (code === BizCode.TokenInvalid || code === BizCode.Unauthorized) {
      useAuthStore.getState().clearTenantSession()
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
    const status = error.response?.status
    const data = error.response?.data as any

    if (status === 401) {
      useAuthStore.getState().clearTenantSession()
      return Promise.reject(error)
    }

    message.error(data?.msg || '请求失败')
    return Promise.reject(error)
  }
)

export default request
