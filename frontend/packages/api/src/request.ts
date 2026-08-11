import axios, { AxiosError, InternalAxiosRequestConfig } from 'axios'
import { message } from 'antd'
import { BizCode } from '@ark-iam/types'

let userProvider: (() => { access_token: string } | null | undefined) | null = null

export function setUserProvider(provider: () => { access_token: string } | null | undefined) {
  userProvider = provider
}

const request = axios.create({ baseURL: '/v1/iam', timeout: 30000 })

request.interceptors.request.use((config: InternalAxiosRequestConfig) => {
  const user = userProvider?.()
  if (user?.access_token && config.headers) {
    config.headers.Authorization = `Bearer ${user.access_token}`
  }
  return config
})

request.interceptors.response.use(
  (response) => {
    const { code, msg } = response.data as { code: number; msg: string }
    if (code === BizCode.Success) return response.data.data

    const isLogoutRequest = response.config.url?.includes('/auth/logout')

    if (code === BizCode.TokenExpired || code === BizCode.TokenInvalid || code === BizCode.Unauthorized) {
      if (isLogoutRequest) return Promise.reject(new Error(msg || 'session expired'))
      window.location.href = '/'
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
      window.location.href = '/'
      return Promise.reject(error)
    }
    const data = error.response?.data as any
    message.error(data?.msg || '请求失败')
    return Promise.reject(error)
  },
)

export default request
