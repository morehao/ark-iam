import { request } from '@ark-iam/api'

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
  return request.get<any, Department[]>('/department/list')
}

export const createDepartment = (data: Partial<Department>) => {
  return request.post<Partial<Department>, Department>('/department/create', data)
}

export const updateDepartment = (data: Partial<Department>) => {
  return request.post<Partial<Department>, Department>('/department/update', data)
}

export const deleteDepartment = (id: number) => {
  return request.post<any, null>('/department/delete', { id })
}