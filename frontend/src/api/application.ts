import request from '../utils/request'
import type { ApiResponse } from '../utils/response'

export interface Application {
  id: number
  name: string
  code: string
  description: string
  status: string
  createdAt: string
}

export interface ApplicationPageListReq {
  page: number
  pageSize: number
  keyword?: string
}

export interface ApplicationPageListResp {
  list: Application[]
  total: number
}

export const getApplicationPageList = (data: ApplicationPageListReq) => {
  return request.post<ApplicationPageListReq, ApiResponse<ApplicationPageListResp>>('/application/pageList', data)
}

export const getApplicationDetail = (id: number) => {
  return request.get<any, ApiResponse<Application>>('/application/detail', { params: { applicationID: id } })
}

export const createApplication = (data: Partial<Application>) => {
  return request.post<Partial<Application>, ApiResponse<Application>>('/application/create', data)
}

export const updateApplication = (data: Partial<Application>) => {
  return request.post<Partial<Application>, ApiResponse<Application>>('/application/update', data)
}

export const deleteApplication = (id: number) => {
  return request.post<any, ApiResponse<null>>('/application/delete', { id })
}

export const getApplicationRoles = (applicationId: number) => {
  return request.get<any, ApiResponse<Role[]>>('/application/roles', { params: { applicationId } })
}

export interface Role {
  id: number
  name: string
  code: string
}
