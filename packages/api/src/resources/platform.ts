import type { AxiosRequestConfig } from 'axios'

import request from '../request'
import type {
  ApplicationCreateReq,
  ApplicationItem,
  ApplicationUpdateReq,
  ConnectorCreateReq,
  ConnectorFactoryItem,
  ConnectorItem,
  ConnectorUpdateReq,
  DomainActionResp,
  DomainCreateReq,
  DomainDetail,
  DomainItem,
  DomainPageListParams,
  DomainVerifyReq,
  MenuMyTreeResp,
  MenuTreeResp,
  OAuthClientCreateReq,
  OAuthClientDetail,
  OAuthClientItem,
  OAuthClientUpdateReq,
  OAuthSecretCreateResp,
  OAuthSecretItem,
  PageListResp,
  SettingCreateReq,
  SettingGroupMeta,
  SettingItem,
  SettingStatus,
  SettingUpdateReq,
  SettingValueType,
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
// config 用于传递 silent 等请求级选项：菜单页首屏恢复"上次选择的应用"属 best-effort 探测，
// 应用已被删除/换库（旧 ID 失效）时静默回退，不弹全局提示（见 request.ts 的 silent）。
export const getApplicationDetail = (appID: string, config?: AxiosRequestConfig) =>
  request.get<any, ApplicationItem>(`/platform/applications/${appID}`, config)
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
  request.post<any, { applicationClientID: string; code: string }>('/platform/application-clients', data)
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
/**
 * offboard 租户（P0 D 唯一停用入口）：tenant→suspended、primary disabled、清 issuer、撤 session/refresh。
 * D 租户 DELETE 一律 403，只能走此入口。
 */
export const offboardTenant = (tenantID: string) =>
  request.post<any, string>(`/platform/tenants/${tenantID}/offboard`)

// ==================== 租户应用 ====================
export const getTenantApplicationPageList = (data: { page: number; pageSize: number; tenantID?: string; status?: string }) =>
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
// 菜单写接口已下放到租户侧（/tenant/applications/{appID}/menus，见 resources/tenant.ts）：
// createMenu/updateMenu/deleteMenu 已迁出并改为 createAppMenu/updateAppMenu/deleteAppMenu。
// 平台侧仅保留只读：侧栏菜单树与平台审计菜单树。
export const getMenuTree = (appID: string) => request.get<any, MenuTreeResp>('/platform/menus/tree', { params: { appID } })
export const getMyMenuTree = () => request.get<any, MenuMyTreeResp>('/platform/menus/my')

// ==================== 域名 ====================
// P0 契约：无通用 PUT 改状态。创建产出 draft，状态流转一律走动作端点（verify/tls-ready/.../release）。
export const getDomainPageList = (data: DomainPageListParams) =>
  request.get<any, PageListResp<DomainItem>>('/platform/domains', { params: data })
export const getDomainDetail = (id: string) => request.get<any, DomainDetail>(`/platform/domains/${id}`)
export const createDomain = (data: DomainCreateReq) =>
  request.post<any, { id: string }>('/platform/domains', data)
export const deleteDomain = (id: string) => request.delete<any, string>(`/platform/domains/${id}`)

// ---- 域名状态机动作端点（POST /platform/domains/{id}/{action}） ----
export const verifyDomain = (id: string, data?: DomainVerifyReq) =>
  request.post<any, DomainActionResp>(`/platform/domains/${id}/verify`, data ?? {})
/** 证书就绪：verified → tls_ready（P0 certRef 仅记录）。 */
export const tlsReadyDomain = (id: string, certRef?: string) =>
  request.post<any, DomainActionResp>(`/platform/domains/${id}/tls-ready`, { certRef })
export const activateDomain = (id: string) =>
  request.post<any, DomainActionResp>(`/platform/domains/${id}/activate`)
/** 停用：仅 alias active → disabled（primary 只能走 offboard）。 */
export const disableDomain = (id: string) =>
  request.post<any, DomainActionResp>(`/platform/domains/${id}/disable`)
/** 恢复：仅 alias disabled → active。 */
export const enableDomain = (id: string) =>
  request.post<any, DomainActionResp>(`/platform/domains/${id}/enable`)
/** 退役：primary disabled→retired（offboard 后）；alias active/disabled→retired。 */
export const retireDomain = (id: string, reason?: string) =>
  request.post<any, DomainActionResp>(`/platform/domains/${id}/retire`, { reason })
/** 释放（正式）：仅 retired→released；需满隔离期且有审批单。 */
export const releaseDomain = (id: string, approvalTicket: string) =>
  request.post<any, DomainActionResp>(`/platform/domains/${id}/release`, { approvalTicket })

// ==================== 系统设置（运行时配置中心） ====================
export const getSettingGroups = () => request.get<any, { list: SettingGroupMeta[] }>('/platform/settings/groups')
export const getSettingPageList = (data: {
  group?: string
  key?: string
  valueType?: SettingValueType
  status?: SettingStatus
  page?: number
  pageSize?: number
}) => request.get<any, PageListResp<SettingItem>>('/platform/settings', { params: data })
export const getSettingDetail = (id: string) => request.get<any, SettingItem>(`/platform/settings/${id}`)
export const createSetting = (data: SettingCreateReq) => request.post<any, string>('/platform/settings', data)
export const updateSetting = (id: string, data: SettingUpdateReq) => request.put<any, string>(`/platform/settings/${id}`, data)
export const updateSettingStatus = (id: string, status: SettingStatus) =>
  request.patch<any, string>(`/platform/settings/${id}/status`, { status })
export const deleteSetting = (id: string) => request.delete<any, string>(`/platform/settings/${id}`)

// ==================== 身份连接器（平台级全局对象） ====================
/** 连接器分页列表筛选参数（defaultTenantID=按默认落租户过滤） */
export interface ConnectorPageListParams {
  page?: number
  pageSize?: number
  defaultTenantID?: string
  protocol?: string
  provider?: string
  status?: string
  name?: string
  displayName?: string
}

export const createConnector = (req: ConnectorCreateReq) =>
  request.post<any, { connectorID: string }>('/platform/connectors', req)
export const getConnectorPageList = (params?: ConnectorPageListParams) =>
  request.get<any, { list: ConnectorItem[]; total: number }>('/platform/connectors', { params })
export const getConnectorDetail = (connectorID: string) =>
  request.get<any, ConnectorItem>(`/platform/connectors/${connectorID}`)
export const updateConnector = (connectorID: string, req: ConnectorUpdateReq) =>
  request.put<any, void>(`/platform/connectors/${connectorID}`, req)
export const deleteConnector = (connectorID: string) =>
  request.delete<any, void>(`/platform/connectors/${connectorID}`)
export const getConnectorFactories = (params?: { protocol?: string; provider?: string }) =>
  request.get<any, { list: ConnectorFactoryItem[] }>('/platform/connector-factories', { params })
