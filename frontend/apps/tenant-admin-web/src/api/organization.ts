import { request } from '@ark-iam/api'
import type {
  OrganizationItem,
  OrganizationTreeResp,
  OrganizationUserItem,
  PageListResp,
  UserOrganizationItem,
} from '@ark-iam/types'

// ---------- 组织树 ----------
export const getOrganizationTree = (params?: { name?: string; status?: string }) =>
  request.get<any, OrganizationTreeResp>('/tenant/organizations/tree', { params })
export const createOrganization = (data: { parentID?: string; name: string; code?: string; sort?: number; status?: string }) =>
  request.post<any, { organizationID: string }>('/tenant/organizations', data)
export const updateOrganization = (data: { organizationID: string; parentID?: string; name?: string; code?: string; sort?: number; status?: string }) => {
  const { organizationID, ...body } = data
  return request.put<any, string>(`/tenant/organizations/${organizationID}`, body)
}
export const updateOrganizationStatus = (organizationID: string, status: string) =>
  request.patch<any, string>(`/tenant/organizations/${organizationID}`, { status })
export const deleteOrganization = (id: string, cascade = false) =>
  request.delete<any, string>(`/tenant/organizations/${id}`, { params: { cascade } })
export const getOrganizationDetail = (organizationID: string) =>
  request.get<any, { organizationID: string; parentID: string; orgPath: string; orgDepth: number; name: string; code: string; sort: number; status: string; ancestors: { organizationID: string; name: string }[] }>(
    `/tenant/organizations/${organizationID}`,
  )

// ---------- 组织关系 ----------
export const getOrganizationUserPage = (organizationID: string, params?: { page?: number; pageSize?: number; relationType?: string; keyword?: string }) =>
  request.get<any, PageListResp<OrganizationUserItem>>(`/tenant/organizations/${organizationID}/users`, { params })
export const createOrganizationUser = (organizationID: string, data: { userID: string; relationType?: string; isPrimary?: boolean }) =>
  request.post<any, void>(`/tenant/organizations/${organizationID}/users`, data)
export const updateOrganizationUser = (organizationID: string, userID: string, data: { relationType?: string; isPrimary?: boolean }) =>
  request.put<any, string>(`/tenant/organizations/${organizationID}/users/${userID}`, data)
export const deleteOrganizationUser = (organizationID: string, userID: string) =>
  request.delete<any, string>(`/tenant/organizations/${organizationID}/users/${userID}`)
export const getOrganizationSubtreeUsers = (organizationID: string) =>
  request.get<any, { list: { userID: string; userName: string }[] }>(`/tenant/organizations/${organizationID}/users/descendants`)

// ---------- 用户组织归属 ----------
export const getUserOrganizations = (userID: string) =>
  request.get<any, { list: UserOrganizationItem[] }>(`/tenant/users/${userID}/organizations`)
export const updateUserOrganizations = (userID: string, organizationIDs: string[]) =>
  request.put<any, string>(`/tenant/users/${userID}/organizations`, { organizationIDs })

export type { OrganizationItem }
