import { request } from '@ark-iam/api'
import type {
  OrganizationItem,
  OrganizationRoleItem,
  OrganizationUserItem,
  OrganizationRoleUserItem,
  PageListResp,
} from '@ark-iam/types'

export const getOrganizationPage = (params?: { page?: number; pageSize?: number; name?: string }) =>
  request.get<any, PageListResp<OrganizationItem>>('/tenant/organizations', { params })
export const createOrganization = (data: Partial<OrganizationItem>) => request.post<any, void>('/tenant/organizations', data)
export const updateOrganization = (data: Partial<OrganizationItem>) => {
  const { organizationID, ...body } = data
  return request.put<any, void>(`/tenant/organizations/${organizationID}`, body)
}
export const deleteOrganization = (id: string) => request.delete<any, void>(`/tenant/organizations/${id}`)

export const getOrganizationRolePage = (params?: { page?: number; pageSize?: number; name?: string }) =>
  request.get<any, PageListResp<OrganizationRoleItem>>('/tenant/organization-roles', { params })
export const createOrganizationRole = (data: Partial<OrganizationRoleItem>) => request.post<any, void>('/tenant/organization-roles', data)
export const updateOrganizationRole = (data: Partial<OrganizationRoleItem>) => {
  const { organizationRoleID, ...body } = data
  return request.put<any, void>(`/tenant/organization-roles/${organizationRoleID}`, body)
}
export const deleteOrganizationRole = (id: string) => request.delete<any, void>(`/tenant/organization-roles/${id}`)

export const getOrganizationUserPage = (params?: { page?: number; pageSize?: number; organizationID?: string; userID?: string }) =>
  request.get<any, PageListResp<OrganizationUserItem>>('/tenant/organization-users', { params })
export const createOrganizationUser = (data: { organizationID: string; userID: string }) =>
  request.post<any, void>('/tenant/organization-users', data)
export const deleteOrganizationUser = (data: { organizationID: string; userID: string }) =>
  request.delete<any, void>(`/tenant/organization-users/${data.organizationID}/${data.userID}`)

export const getOrganizationRoleUserPage = (params?: { page?: number; pageSize?: number; organizationRoleID?: string; userID?: string }) =>
  request.get<any, PageListResp<OrganizationRoleUserItem>>('/tenant/organization-role-users', { params })
export const createOrganizationRoleUser = (data: { organizationID: string; organizationRoleID: string; userID: string }) =>
  request.post<any, void>('/tenant/organization-role-users', data)
export const deleteOrganizationRoleUser = (data: { organizationID: string; organizationRoleID: string; userID: string }) =>
  request.delete<any, void>(`/tenant/organization-role-users/${data.organizationRoleID}/${data.userID}`)
