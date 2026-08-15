import request from '../request'
import type {
  ApiKeyCreateResp,
  ApiKeyItem,
  ApplicationCreateReq,
  ApplicationItem,
  ApplicationUpdateReq,
  AuditLogItem,
  ConnectorFactoryItem,
  ConnectorItem,
  DepartmentItem,
  DepartmentTreeResp,
  DomainItem,
  MenuItem,
  MenuTreeResp,
  OAuthClientCreateReq,
  OAuthClientDetail,
  OAuthClientItem,
  OAuthClientUpdateReq,
  OAuthSecretCreateResp,
  OAuthSecretItem,
  PageListResp,
  ResourceItem,
  RoleCreateReq,
  RoleItem,
  RoleUpdateReq,
  RoleUserItem,
  ScopeItem,
  SystemConfigItem,
  TenantApplicationCreateReq,
  TenantApplicationItem,
  TenantApplicationUpdateReq,
  TenantCreateReq,
  TenantItem,
  TenantUpdateReq,
  UserCreateReq,
  UserDepartmentItem,
  UserIdentityCreateReq,
  UserIdentityItem,
  UserItem,
  UserLoginLogItem,
  UserPasswordUpdateReq,
  UserStatusUpdateReq,
  UserUpdateReq,
} from '@ark-iam/types'

// ==================== 用户 ====================
export const getUserPageList = (data: { page: number; pageSize: number; name?: string; isSuspended?: number }) =>
  request.get<any, PageListResp<UserItem>>('/platform/users', { params: data })
export const getUserDetail = (userID: number) => request.get<any, UserItem>(`/platform/users/${userID}`)
export const createUser = (data: UserCreateReq) => request.post<any, { userID: number }>('/platform/users', data)
export const updateUser = (data: UserUpdateReq) => {
  const { userID, ...body } = data
  return request.put<any, string>(`/platform/users/${userID}`, body)
}
export const deleteUser = (userID: number) => request.delete<any, string>(`/platform/users/${userID}`)
export const updateUserStatus = (data: UserStatusUpdateReq) => {
  const { userID, ...body } = data
  return request.patch<any, string>(`/platform/users/${userID}`, body)
}
export const updateUserPassword = (data: UserPasswordUpdateReq) => {
  const { userID, ...body } = data
  return request.post<any, string>(`/platform/users/${userID}/changePassword`, body)
}

export const getUserIdentityByUser = (userID: number) =>
  request.get<any, PageListResp<UserIdentityItem>>(`/platform/users/${userID}/identities`)
export const createUserIdentity = (data: UserIdentityCreateReq) => {
  const { userID, ...body } = data
  return request.post<any, { userIdentityID: number }>(`/platform/users/${userID}/identities`, body)
}
export const deleteUserIdentity = (userID: number, userIdentityID: number) =>
  request.delete<any, string>(`/platform/users/${userID}/identities/${userIdentityID}`)

export const getUserLoginLogByUser = (userID: number) =>
  request.get<any, PageListResp<UserLoginLogItem>>(`/platform/users/${userID}/login-logs`)

export const getUserDepartmentByUser = (userID: number) =>
  request.get<any, PageListResp<UserDepartmentItem>>(`/platform/users/${userID}/departments`)
export const assignUserDepartments = (userID: number, departmentIDs: number[]) =>
  request.put<any, string>(`/platform/users/${userID}/departments`, { departmentIDs })

// ==================== 角色 ====================
export const getRolePageList = (data: { page: number; pageSize: number; name?: string }) =>
  request.get<any, PageListResp<RoleItem>>('/platform/roles', { params: data })
export const getRoleDetail = (roleID: number) => request.get<any, RoleItem>(`/platform/roles/${roleID}`)
export const createRole = (data: RoleCreateReq) => request.post<any, { roleID: number }>('/platform/roles', data)
export const updateRole = (data: RoleUpdateReq) => {
  const { roleID, ...body } = data
  return request.put<any, string>(`/platform/roles/${roleID}`, body)
}
export const deleteRole = (roleID: number) => request.delete<any, string>(`/platform/roles/${roleID}`)
export const getRoleUsers = (roleID: number) => request.get<any, { total: number; users: RoleUserItem[] }>(`/platform/roles/${roleID}/users`)
export const assignRoleUsers = (roleID: number, userIDs: number[]) => request.put<any, string>(`/platform/roles/${roleID}/users`, { userIDs })
export const removeRoleUser = (roleID: number, userID: number) => request.delete<any, string>(`/platform/roles/${roleID}/users/${userID}`)

// ==================== 部门 ====================
export const getDepartmentTree = () => request.get<any, DepartmentTreeResp>('/platform/departments/tree')
export const getDepartmentPageList = (data: { page: number; pageSize: number; name?: string }) =>
  request.get<any, PageListResp<DepartmentItem>>('/platform/departments', { params: data })
export const getDepartmentDetail = (departmentID: number) =>
  request.get<any, DepartmentItem>(`/platform/departments/${departmentID}`)
export const createDepartment = (data: Partial<DepartmentItem>) => request.post<any, { departmentID: number }>('/platform/departments', data)
export const updateDepartment = (data: Partial<DepartmentItem>) => {
  const { departmentID, ...body } = data
  return request.put<any, string>(`/platform/departments/${departmentID}`, body)
}
export const deleteDepartment = (departmentID: number) => request.delete<any, string>(`/platform/departments/${departmentID}`)

// ==================== 应用 ====================
export const getApplicationPageList = (data: { page: number; pageSize: number; name?: string }) =>
  request.get<any, PageListResp<ApplicationItem>>('/platform/applications', { params: data })
export const getApplicationDetail = (appID: number) => request.get<any, ApplicationItem>(`/platform/applications/${appID}`)
export const createApplication = (data: ApplicationCreateReq) => request.post<any, { appID: number; code: string }>('/platform/applications', data)
export const updateApplication = (data: ApplicationUpdateReq) => {
  const { appID, ...body } = data
  return request.put<any, string>(`/platform/applications/${appID}`, body)
}
export const deleteApplication = (appID: number) => request.delete<any, string>(`/platform/applications/${appID}`)

// ==================== OAuth 客户端 ====================
export const getOAuthClientPageList = (data: { page: number; pageSize: number; name?: string }) =>
  request.get<any, PageListResp<OAuthClientItem>>('/platform/application-clients', { params: data })
export const getOAuthClientDetail = (applicationClientID: number) =>
  request.get<any, OAuthClientDetail>(`/platform/application-clients/${applicationClientID}`)
export const createOAuthClient = (data: OAuthClientCreateReq) =>
  request.post<any, { applicationClientID: number; clientID: string }>('/platform/application-clients', data)
export const updateOAuthClient = (data: OAuthClientUpdateReq) => {
  const { applicationClientID, ...body } = data
  return request.put<any, string>(`/platform/application-clients/${applicationClientID}`, body)
}
export const deleteOAuthClient = (applicationClientID: number) =>
  request.delete<any, string>(`/platform/application-clients/${applicationClientID}`)
export const listOAuthSecrets = (applicationClientID: number) =>
  request.get<any, { total: number; secrets: OAuthSecretItem[] }>(`/platform/application-clients/${applicationClientID}/secrets`)
export const createOAuthSecret = (data: { applicationClientID: number; name: string; expiresAt?: string }) => {
  const { applicationClientID, ...body } = data
  return request.post<any, OAuthSecretCreateResp>(`/platform/application-clients/${applicationClientID}/secrets`, body)
}
export const deleteOAuthSecret = (applicationClientID: number, secretID: number) =>
  request.delete<any, string>(`/platform/application-clients/${applicationClientID}/secrets/${secretID}`)

// ==================== 租户 ====================
export const getTenantPageList = (data: { page: number; pageSize: number; name?: string }) =>
  request.get<any, PageListResp<TenantItem>>('/platform/tenants', { params: data })
export const getTenantDetail = (tenantID: number) => request.get<any, TenantItem>(`/platform/tenants/${tenantID}`)
export const createTenant = (data: TenantCreateReq) => request.post<any, { tenantID: number }>('/platform/tenants', data)
export const updateTenant = (data: TenantUpdateReq) => {
  const { tenantID, ...body } = data
  return request.put<any, string>(`/platform/tenants/${tenantID}`, body)
}
export const deleteTenant = (tenantID: number) => request.delete<any, string>(`/platform/tenants/${tenantID}`)

// ==================== 租户应用 ====================
export const getTenantApplicationPageList = (data: { page: number; pageSize: number; status?: string }) =>
  request.get<any, PageListResp<TenantApplicationItem>>('/platform/tenant-applications', { params: data })
export const getTenantApplicationDetail = (tenantAppID: number) =>
  request.get<any, TenantApplicationItem>(`/platform/tenant-applications/${tenantAppID}`)
export const createTenantApplication = (data: TenantApplicationCreateReq) =>
  request.post<any, { tenantAppID: number }>('/platform/tenant-applications', data)
export const updateTenantApplication = (data: TenantApplicationUpdateReq) => {
  const { tenantAppID, ...body } = data
  return request.put<any, string>(`/platform/tenant-applications/${tenantAppID}`, body)
}
export const deleteTenantApplication = (tenantAppID: number) =>
  request.delete<any, string>(`/platform/tenant-applications/${tenantAppID}`)

// ==================== API Key ====================
export const getApiKeyPageList = (data: { page: number; pageSize: number; name?: string }) =>
  request.get<any, PageListResp<ApiKeyItem>>('/platform/api-keys', { params: data })
export const createApiKey = (data: { name: string; scope?: string; expiresAt?: string }) =>
  request.post<any, ApiKeyCreateResp>('/platform/api-keys', data)
export const revokeApiKey = (id: number) => request.post<any, string>(`/platform/api-keys/${id}/revoke`)
export const deleteApiKey = (id: number) => request.delete<any, string>(`/platform/api-keys/${id}`)

// ==================== 菜单 ====================
export const getMenuTree = (appID: number) => request.get<any, MenuTreeResp>('/platform/menus/tree', { params: { appID } })
export const getMenuPageList = (data: { page: number; pageSize: number; appID?: number; name?: string }) =>
  request.get<any, PageListResp<MenuItem>>('/platform/menus', { params: data })
export const getMenuDetail = (menuID: number) => request.get<any, MenuItem>(`/platform/menus/${menuID}`)
export const createMenu = (data: Partial<MenuItem>) => request.post<any, { menuID: number }>('/platform/menus', data)
export const updateMenu = (data: Partial<MenuItem>) => {
  const { menuID, ...body } = data
  return request.put<any, string>(`/platform/menus/${menuID}`, body)
}
export const deleteMenu = (menuID: number) => request.delete<any, string>(`/platform/menus/${menuID}`)

// ==================== 权限域 ====================
export const getScopePageList = (data: { page: number; pageSize: number; name?: string }) =>
  request.get<any, PageListResp<ScopeItem>>('/platform/scopes', { params: data })
export const getScopeDetail = (scopeID: number) => request.get<any, ScopeItem>(`/platform/scopes/${scopeID}`)
export const createScope = (data: Partial<ScopeItem>) => request.post<any, { scopeID: number }>('/platform/scopes', data)
export const updateScope = (data: Partial<ScopeItem>) => {
  const { scopeID, ...body } = data
  return request.put<any, string>(`/platform/scopes/${scopeID}`, body)
}
export const deleteScope = (scopeID: number) => request.delete<any, string>(`/platform/scopes/${scopeID}`)

// ==================== 资源 ====================
export const getResourcePageList = (data: { page: number; pageSize: number; name?: string }) =>
  request.get<any, PageListResp<ResourceItem>>('/platform/resources', { params: data })
export const getResourceDetail = (resourceID: number) => request.get<any, ResourceItem>(`/platform/resources/${resourceID}`)
export const createResource = (data: Partial<ResourceItem>) => request.post<any, { resourceID: number }>('/platform/resources', data)
export const updateResource = (data: Partial<ResourceItem>) => {
  const { resourceID, ...body } = data
  return request.put<any, string>(`/platform/resources/${resourceID}`, body)
}
export const deleteResource = (resourceID: number) => request.delete<any, string>(`/platform/resources/${resourceID}`)

// ==================== 域名 ====================
export const getDomainPageList = (data: { page: number; pageSize: number; domain?: string }) =>
  request.get<any, PageListResp<DomainItem>>('/platform/domains', { params: data })
export const getDomainDetail = (id: number) => request.get<any, DomainItem>(`/platform/domains/${id}`)
export const createDomain = (domain: string) => request.post<any, { id: number }>('/platform/domains', { domain })
export const updateDomain = (data: { id: number; domain?: string; isVerified?: number }) => {
  const { id, ...body } = data
  return request.put<any, string>(`/platform/domains/${id}`, body)
}
export const deleteDomain = (id: number) => request.delete<any, string>(`/platform/domains/${id}`)

// ==================== 系统配置 ====================
export const getSystemPageList = (data: { page: number; pageSize: number; key?: string }) =>
  request.get<any, PageListResp<SystemConfigItem>>('/platform/systems', { params: data })
export const getSystemDetail = (systemID: number) => request.get<any, SystemConfigItem>(`/platform/systems/${systemID}`)
export const createSystemConfig = (data: { key: string; value?: unknown }) => request.post<any, { systemID: number }>('/platform/systems', data)
export const updateSystemConfig = (data: { systemID: number; key?: string; value?: unknown }) => {
  const { systemID, ...body } = data
  return request.put<any, string>(`/platform/systems/${systemID}`, body)
}
export const deleteSystemConfig = (systemID: number) => request.delete<any, string>(`/platform/systems/${systemID}`)

// ==================== 审计日志 ====================
export const getAuditLogPageList = (data: { page: number; pageSize: number; key?: string }) =>
  request.get<any, PageListResp<AuditLogItem>>('/platform/logs', { params: data })
export const getAuditLogDetail = (logID: number) => request.get<any, AuditLogItem>(`/platform/logs/${logID}`)

// ==================== Connector（auth） ====================
export const getConnectorPageList = (data: { page: number; pageSize: number; name?: string }) =>
  request.get<any, PageListResp<ConnectorItem>>('/auth/connectors', { params: data })
export const getConnectorDetail = (connectorID: number) => request.get<any, ConnectorItem>(`/auth/connectors/${connectorID}`)
export const createConnector = (data: Partial<ConnectorItem>) => request.post<any, { connectorID: number }>('/auth/connectors', data)
export const updateConnector = (data: Partial<ConnectorItem>) => {
  const { connectorID, ...body } = data
  return request.put<any, string>(`/auth/connectors/${connectorID}`, body)
}
export const deleteConnector = (connectorID: number) => request.delete<any, string>(`/auth/connectors/${connectorID}`)
export const getConnectorFactoryList = (data?: { protocol?: string; provider?: string }) =>
  request.get<any, { list: ConnectorFactoryItem[] }>('/auth/connector-factories', { params: data })
