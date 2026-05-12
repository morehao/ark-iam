import request from '../utils/request'

export interface Role {
  id: number
  name: string
  code: string
  description: string
  status: string
  createdAt: string
}

export interface RolePageListReq {
  page: number
  pageSize: number
  keyword?: string
}

export const getRolePageList = (data: RolePageListReq) => {
  return request.post<any, any>('/role/pageList', data)
}

export const getRoleDetail = (id: number) => {
  return request.get<any, any>('/role/detail', { params: { roleID: id } })
}