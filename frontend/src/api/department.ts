import request from '../utils/request'
import type { ApiResponse } from '../utils/response'

export interface Department {
  id: number
  name: string
  parentID: number
  orderNum: number
  status: string
  createdAt: string
}

export interface DepartmentListResp {
  list: Department[]
}

export const getDepartmentList = () => {
  return request.get<any, ApiResponse<Department[]>>('/department/list')
}

export const createDepartment = (data: Partial<Department>) => {
  return request.post<Partial<Department>, ApiResponse<Department>>('/department/create', data)
}

export const updateDepartment = (data: Partial<Department>) => {
  return request.post<Partial<Department>, ApiResponse<Department>>('/department/update', data)
}

export const deleteDepartment = (id: number) => {
  return request.post<any, ApiResponse<null>>('/department/delete', { id })
}