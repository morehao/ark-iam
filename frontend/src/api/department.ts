import request from '../utils/request'

export interface Department {
  id: number
  name: string
  parentID: number
  orderNum: number
  status: string
  createdAt: string
}

export const getDepartmentList = () => {
  return request.get<any, any>('/department/list')
}

export const createDepartment = (data: any) => {
  return request.post<any, any>('/department/create', data)
}

export const updateDepartment = (data: any) => {
  return request.post<any, any>('/department/update', data)
}

export const deleteDepartment = (id: number) => {
  return request.post<any, any>('/department/delete', { id })
}