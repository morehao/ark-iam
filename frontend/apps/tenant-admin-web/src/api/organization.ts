import { request } from '@ark-iam/api'
import type {
  OrganizationItem,
  OrganizationRoleItem,
  OrganizationUserItem,
  OrganizationRoleUserItem,
  PageListResp,
} from '@ark-iam/types'

export const getOrganizationPage = (params?: { page?: number; pageSize?: number; name?: string }) =>
  request.post<any, PageListResp<OrganizationItem>>('/tenant/organization/pageList', params)
export const createOrganization = (data: Partial<OrganizationItem>) => request.post<any, void>('/tenant/organization/create', data)
export const updateOrganization = (data: Partial<OrganizationItem>) => request.post<any, void>('/tenant/organization/update', data)
export const deleteOrganization = (id: number) => request.post<any, void>('/tenant/organization/delete', { organizationID: id })

export const getOrganizationRolePage = (params?: { page?: number; pageSize?: number; name?: string }) =>
  request.post<any, PageListResp<OrganizationRoleItem>>('/tenant/organizationRole/pageList', params)
export const createOrganizationRole = (data: Partial<OrganizationRoleItem>) => request.post<any, void>('/tenant/organizationRole/create', data)
export const updateOrganizationRole = (data: Partial<OrganizationRoleItem>) => request.post<any, void>('/tenant/organizationRole/update', data)
export const deleteOrganizationRole = (id: number) => request.post<any, void>('/tenant/organizationRole/delete', { organizationRoleID: id })

export const getOrganizationUserPage = (params?: { page?: number; pageSize?: number }) =>
  request.post<any, PageListResp<OrganizationUserItem>>('/tenant/organizationUser/pageList', params)
export const createOrganizationUser = (data: { organizationID: number; userID: number }) => request.post<any, void>('/tenant/organizationUser/create', data)
export const deleteOrganizationUser = (data: { organizationID: number; userID: number }) => request.post<any, void>('/tenant/organizationUser/delete', data)

export const getOrganizationRoleUserPage = (params?: { page?: number; pageSize?: number }) =>
  request.post<any, PageListResp<OrganizationRoleUserItem>>('/tenant/organizationRoleUser/pageList', params)
export const createOrganizationRoleUser = (data: { organizationID: number; organizationRoleID: number; userID: number }) => request.post<any, void>('/tenant/organizationRoleUser/create', data)
export const deleteOrganizationRoleUser = (data: { organizationID: number; organizationRoleID: number; userID: number }) => request.post<any, void>('/tenant/organizationRoleUser/delete', data)
