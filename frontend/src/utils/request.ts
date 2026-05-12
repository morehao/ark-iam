import axios, { AxiosError, InternalAxiosRequestConfig } from 'axios'
import { message } from 'antd'
import { useAuthStore } from '../stores/authStore'

const request = axios.create({
  baseURL: '/v1/iam',
  timeout: 30000,
})

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
  (response) => response.data,
  async (error: AxiosError) => {
    const status = error.response?.status
    const data = error.response?.data as any

    if (status === 401) {
      useAuthStore.getState().clearTenantSession()
      return Promise.reject(error)
    }

    if (data?.msg) {
      message.error(data.msg)
    } else {
      message.error('请求失败')
    }

    return Promise.reject(error)
  }
)

export default request
