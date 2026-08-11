import { request } from '@ark-iam/api'
import type {
  OrganizationItem,
  OrganizationRoleItem,
  OrganizationUserItem,
  OrganizationRoleUserItem,
  PageListResp,
} from '@ark-iam/types'

export const getOrganizationPage = (params?: { page?: number; pageSize?: number; name?: string }) =>
  request.post<any, PageListResp<OrganizationItem>>('/organization/pageList', params)
export const createOrganization = (data: Partial<OrganizationItem>) => request.post<any, void>('/organization/create', data)
export const updateOrganization = (data: Partial<OrganizationItem>) => request.post<any, void>('/organization/update', data)
export const deleteOrganization = (id: number) => request.post<any, void>('/organization/delete', { organizationID: id })

export const getOrganizationRolePage = (params?: { page?: number; pageSize?: number; name?: string }) =>
  request.post<any, PageListResp<OrganizationRoleItem>>('/organizationRole/pageList', params)
export const createOrganizationRole = (data: Partial<OrganizationRoleItem>) => request.post<any, void>('/organizationRole/create', data)
export const updateOrganizationRole = (data: Partial<OrganizationRoleItem>) => request.post<any, void>('/organizationRole/update', data)
export const deleteOrganizationRole = (id: number) => request.post<any, void>('/organizationRole/delete', { organizationRoleID: id })

export const getOrganizationUserPage = (params?: { page?: number; pageSize?: number }) =>
  request.post<any, PageListResp<OrganizationUserItem>>('/organizationUser/pageList', params)
export const createOrganizationUser = (data: { organizationID: number; userID: number }) => request.post<any, void>('/organizationUser/create', data)
export const deleteOrganizationUser = (data: { organizationID: number; userID: number }) => request.post<any, void>('/organizationUser/delete', data)

export const getOrganizationRoleUserPage = (params?: { page?: number; pageSize?: number }) =>
  request.post<any, PageListResp<OrganizationRoleUserItem>>('/organizationRoleUser/pageList', params)
export const createOrganizationRoleUser = (data: { organizationID: number; organizationRoleID: number; userID: number }) => request.post<any, void>('/organizationRoleUser/create', data)
export const deleteOrganizationRoleUser = (data: { organizationID: number; organizationRoleID: number; userID: number }) => request.post<any, void>('/organizationRoleUser/delete', data)
