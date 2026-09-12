import { request } from '@ark-iam/api'
import type {
  DepartmentChildrenResp,
  DepartmentItem,
  DepartmentTreeResp,
  DepartmentUserItem,
  PageListResp,
} from '@ark-iam/types'

// ---------- 部门树 ----------
export const getDepartmentTree = (params?: { name?: string; status?: string }) =>
  request.get<any, DepartmentTreeResp>('/tenant/departments/tree', { params })
export const getDepartmentChildren = (
  departmentID: string,
  params?: { page?: number; pageSize?: number; name?: string; status?: string },
) => request.get<any, DepartmentChildrenResp>(`/tenant/departments/${departmentID}/children`, { params })
export const createDepartment = (data: { parentID?: string; name: string; code?: string; sort?: number; status?: string }) =>
  request.post<any, { departmentID: string }>('/tenant/departments', data)
export const updateDepartment = (data: { departmentID: string; parentID?: string; name?: string; code?: string; sort?: number; status?: string }) => {
  const { departmentID, ...body } = data
  return request.put<any, string>(`/tenant/departments/${departmentID}`, body)
}
export const updateDepartmentStatus = (departmentID: string, status: string) =>
  request.patch<any, string>(`/tenant/departments/${departmentID}`, { status })
export const deleteDepartment = (id: string, cascade = false) =>
  request.delete<any, string>(`/tenant/departments/${id}`, { params: { cascade } })

// ---------- 部门关系 ----------
export const getDepartmentUserPage = (departmentID: string, params?: { page?: number; pageSize?: number; relationType?: string; keyword?: string }) =>
  request.get<any, PageListResp<DepartmentUserItem>>(`/tenant/departments/${departmentID}/users`, { params })
export const createDepartmentUser = (departmentID: string, data: { userID: string; relationType?: string }) =>
  request.post<any, void>(`/tenant/departments/${departmentID}/users`, data)
export const updateDepartmentUser = (departmentID: string, userID: string, data: { relationType?: string }) =>
  request.put<any, string>(`/tenant/departments/${departmentID}/users/${userID}`, data)
export const deleteDepartmentUser = (departmentID: string, userID: string) =>
  request.delete<any, string>(`/tenant/departments/${departmentID}/users/${userID}`)

export type { DepartmentItem }
