import request from '../utils/request'

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

export const getApplicationPageList = (data: ApplicationPageListReq) => {
  return request.post<any, any>('/application/pageList', data)
}

export const getApplicationDetail = (id: number) => {
  return request.get<any, any>('/application/detail', { params: { applicationID: id } })
}

export const createApplication = (data: any) => {
  return request.post<any, any>('/application/create', data)
}

export const updateApplication = (data: any) => {
  return request.post<any, any>('/application/update', data)
}

export const deleteApplication = (id: number) => {
  return request.post<any, any>('/application/delete', { id })
}

export const getApplicationRoles = (applicationId: number) => {
  return request.get<any, any>('/application/roles', { params: { applicationId } })
}