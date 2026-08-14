import { request } from '@ark-iam/api'

export interface TenantApplication {
  tenantAppId: number
  tenantId: number
  appId: number
  status: string
  config: string
  createdAt: string
}

export interface TenantApplicationPageListReq {
  page: number
  pageSize: number
  status?: string
}

export interface TenantApplicationPageListResp {
  list: TenantApplication[]
  total: number
}

export interface TenantApplicationCreateReq {
  appId: number
  status?: string
  config?: string
}

export interface TenantApplicationUpdateReq {
  tenantAppId: number
  status?: string
  config?: string
}

export const getTenantApplicationPageList = (data: TenantApplicationPageListReq) => {
  return request.get<TenantApplicationPageListReq, TenantApplicationPageListResp>('/tenantApplication/pageList', { params: data })
}

export const getTenantApplicationDetail = (tenantAppId: number) => {
  return request.get<any, TenantApplication>('/tenantApplication/detail', { params: { tenantAppId } })
}

export const createTenantApplication = (data: TenantApplicationCreateReq) => {
  return request.post<TenantApplicationCreateReq, { tenantAppId: number }>('/tenantApplication/create', data)
}

export const updateTenantApplication = (data: TenantApplicationUpdateReq) => {
  return request.post<TenantApplicationUpdateReq, string>('/tenantApplication/update', data)
}

export const deleteTenantApplication = (tenantAppId: number) => {
  return request.post<any, string>('/tenantApplication/delete', { tenantAppId })
}
