import request from '../utils/request'

export interface Tenant {
  tenantID: number
  name: string
  tag: string
  dbUser: string
  isSuspended: number
  type: string
  createdAt: string
}

export interface TenantPageListReq {
  page: number
  pageSize: number
  keyword?: string
}

export interface TenantPageListResp {
  list: Tenant[]
  total: number
}

export interface TenantCreateReq {
  name: string
  tag?: string
  type?: string
}

export interface TenantUpdateReq {
  tenantID: number
  name?: string
  tag?: string
  type?: string
}

export const getTenantPageList = (data: TenantPageListReq) => {
  return request.post<TenantPageListReq, TenantPageListResp>('/tenant/pageList', data)
}

export const getTenantDetail = (tenantID: number) => {
  return request.get<any, Tenant>('/tenant/detail', { params: { tenantID } })
}

export const createTenant = (data: TenantCreateReq) => {
  return request.post<TenantCreateReq, { tenantID: number }>('/tenant/create', data)
}

export const updateTenant = (data: TenantUpdateReq) => {
  return request.post<TenantUpdateReq, string>('/tenant/update', data)
}

export const deleteTenant = (tenantID: number) => {
  return request.post<any, string>('/tenant/delete', { tenantID })
}
