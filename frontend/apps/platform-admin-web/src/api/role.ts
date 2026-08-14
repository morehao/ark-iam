import { request } from '@ark-iam/api'

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

export interface RolePageListResp {
  list: Role[]
  total: number
}

export const getRolePageList = (data: RolePageListReq) => {
  return request.post<RolePageListReq, RolePageListResp>('/role/pageList', data)
}

export const getRoleDetail = (id: number) => {
  return request.get<any, Role>('/role/detail', { params: { roleID: id } })
}