import request from '../request'
import type {
  ApplicationCreateReq,
  ApplicationItem,
  ApplicationUpdateReq,
  AuditLogItem,
  DomainItem,
  MenuItem,
  MenuMyTreeResp,
  MenuTreeResp,
  OAuthClientCreateReq,
  OAuthClientDetail,
  OAuthClientItem,
  OAuthClientUpdateReq,
  OAuthSecretCreateResp,
  OAuthSecretItem,
  PageListResp,
  TenantApplicationCreateReq,
  TenantApplicationItem,
  TenantApplicationUpdateReq,
  TenantAdminResetPasswordResp,
  TenantCreateReq,
  TenantCreateResp,
  TenantItem,
  TenantStatus,
  TenantUpdateReq,
} from '@ark-iam/types'

// 用户与角色无平台端接口：两者按租户归属，读写与成员管理统一收敛到 /tenant/* （见 tenant-admin-web/src/api）。
// 平台侧仅保留跨租户的「监督/干预」类只读或状态接口：租户、租户应用。

// ==================== 应用 ====================
export const getApplicationPageList = (data: { page: number; pageSize: number; name?: string }) =>
  request.get<any, PageListResp<ApplicationItem>>('/platform/applications', { params: data })
export const getApplicationDetail = (appID: string) => request.get<any, ApplicationItem>(`/platform/applications/${appID}`)
export const createApplication = (data: ApplicationCreateReq) => request.post<any, { appID: string; code: string }>('/platform/applications', data)
export const updateApplication = (data: ApplicationUpdateReq) => {
  const { appID, ...body } = data
  return request.put<any, string>(`/platform/applications/${appID}`, body)
}
export const deleteApplication = (appID: string) => request.delete<any, string>(`/platform/applications/${appID}`)

// ==================== OAuth 客户端 ====================
export const getOAuthClientPageList = (data: { page: number; pageSize: number; name?: string }) =>
  request.get<any, PageListResp<OAuthClientItem>>('/platform/application-clients', { params: data })
export const getOAuthClientDetail = (applicationClientID: string) =>
  request.get<any, OAuthClientDetail>(`/platform/application-clients/${applicationClientID}`)
export const createOAuthClient = (data: OAuthClientCreateReq) =>
  request.post<any, { applicationClientID: string; clientID: string }>('/platform/application-clients', data)
export const updateOAuthClient = (data: OAuthClientUpdateReq) => {
  const { applicationClientID, ...body } = data
  return request.put<any, string>(`/platform/application-clients/${applicationClientID}`, body)
}
export const deleteOAuthClient = (applicationClientID: string) =>
  request.delete<any, string>(`/platform/application-clients/${applicationClientID}`)
export const listOAuthSecrets = (applicationClientID: string) =>
  request.get<any, { total: number; secrets: OAuthSecretItem[] }>(`/platform/application-clients/${applicationClientID}/secrets`)
export const createOAuthSecret = (data: { applicationClientID: string; name: string; expiresAt?: number }) => {
  const { applicationClientID, ...body } = data
  return request.post<any, OAuthSecretCreateResp>(`/platform/application-clients/${applicationClientID}/secrets`, body)
}
export const deleteOAuthSecret = (applicationClientID: string, secretID: string) =>
  request.delete<any, string>(`/platform/application-clients/${applicationClientID}/secrets/${secretID}`)

// ==================== 租户 ====================
export const getTenantPageList = (data: { page: number; pageSize: number; name?: string; status?: TenantStatus }) =>
  request.get<any, PageListResp<TenantItem>>('/platform/tenants', { params: data })
export const createTenant = (data: TenantCreateReq) => request.post<any, TenantCreateResp>('/platform/tenants', data)
/** 重置租户内置管理员（source=builtin）密码：新临时密码仅此一次返回；不作用于租户手工创建的成员 */
export const resetTenantAdminPassword = (tenantID: string) =>
  request.post<any, TenantAdminResetPasswordResp>(`/platform/tenants/${tenantID}/builtin-admin/reset-password`)
export const updateTenant = (data: TenantUpdateReq) => {
  const { tenantID, ...body } = data
  return request.put<any, string>(`/platform/tenants/${tenantID}`, body)
}
export const deleteTenant = (tenantID: string) => request.delete<any, string>(`/platform/tenants/${tenantID}`)

// ==================== 租户应用 ====================
export const getTenantApplicationPageList = (data: { page: number; pageSize: number; status?: string }) =>
  request.get<any, PageListResp<TenantApplicationItem>>('/platform/tenant-applications', { params: data })
export const createTenantApplication = (data: TenantApplicationCreateReq) =>
  request.post<any, { tenantAppID: string }>('/platform/tenant-applications', data)
export const updateTenantApplication = (data: TenantApplicationUpdateReq) => {
  const { tenantAppID, ...body } = data
  return request.put<any, string>(`/platform/tenant-applications/${tenantAppID}`, body)
}
export const deleteTenantApplication = (tenantAppID: string) =>
  request.delete<any, string>(`/platform/tenant-applications/${tenantAppID}`)

// ==================== 菜单 ====================
export const getMenuTree = (appID: string) => request.get<any, MenuTreeResp>('/platform/menus/tree', { params: { appID } })
export const getMyMenuTree = () => request.get<any, MenuMyTreeResp>('/platform/menus/my')
export const createMenu = (data: Partial<MenuItem>) => request.post<any, { menuID: string }>('/platform/menus', data)
export const updateMenu = (data: Partial<MenuItem>) => {
  const { menuID, ...body } = data
  return request.put<any, string>(`/platform/menus/${menuID}`, body)
}
export const deleteMenu = (menuID: string) => request.delete<any, string>(`/platform/menus/${menuID}`)

// ==================== 域名 ====================
export const getDomainPageList = (data: { page: number; pageSize: number; domain?: string }) =>
  request.get<any, PageListResp<DomainItem>>('/platform/domains', { params: data })
export const getDomainDetail = (id: string) => request.get<any, DomainItem>(`/platform/domains/${id}`)
export const createDomain = (domain: string) => request.post<any, { id: string }>('/platform/domains', { domain })
export const updateDomain = (data: { id: string; domain?: string; isVerified?: number }) => {
  const { id, ...body } = data
  return request.put<any, string>(`/platform/domains/${id}`, body)
}
export const deleteDomain = (id: string) => request.delete<any, string>(`/platform/domains/${id}`)

// ==================== 审计日志 ====================
export const getAuditLogPageList = (data: { page: number; pageSize: number; key?: string }) =>
  request.get<any, PageListResp<AuditLogItem>>('/platform/logs', { params: data })
export const getAuditLogDetail = (logID: string) => request.get<any, AuditLogItem>(`/platform/logs/${logID}`)
