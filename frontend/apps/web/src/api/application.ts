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

export interface ApplicationPageListResp {
  list: Application[]
  total: number
}

export const getApplicationPageList = (data: ApplicationPageListReq) => {
  return request.post<ApplicationPageListReq, ApplicationPageListResp>('/application/pageList', data)
}

export const getApplicationDetail = (id: number) => {
  return request.get<any, Application>('/application/detail', { params: { appId: id } })
}

export const createApplication = (data: Partial<Application>) => {
  return request.post<Partial<Application>, Application>('/application/create', data)
}

export const updateApplication = (data: Partial<Application>) => {
  return request.post<Partial<Application>, Application>('/application/update', data)
}

export const deleteApplication = (id: number) => {
  return request.post<any, null>('/application/delete', { id })
}

export const getApplicationRoles = (appId: number) => {
  return request.get<any, Role[]>('/application/roles', { params: { appId } })
}

export interface Role {
  id: number
  name: string
  code: string
}
