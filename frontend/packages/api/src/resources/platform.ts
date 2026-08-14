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
  request.post<any, PageListResp<UserItem>>('/user/pageList', data)
export const getUserDetail = (userID: number) => request.get<any, UserItem>('/user/detail', { params: { userID } })
export const createUser = (data: UserCreateReq) => request.post<any, { userID: number }>('/user/create', data)
export const updateUser = (data: UserUpdateReq) => request.post<any, string>('/user/update', data)
export const deleteUser = (userID: number) => request.post<any, string>('/user/delete', { userID })
export const updateUserStatus = (data: UserStatusUpdateReq) => request.post<any, string>('/user/updateStatus', data)
export const updateUserPassword = (data: UserPasswordUpdateReq) => request.post<any, string>('/user/updatePassword', data)

export const getUserIdentityByUser = (userID: number) =>
  request.get<any, PageListResp<UserIdentityItem>>('/user/getUserIdentityByUser', { params: { userID } })
export const createUserIdentity = (data: UserIdentityCreateReq) =>
  request.post<any, { userIdentityID: number }>('/user/createUserIdentity', data)
export const deleteUserIdentity = (userIdentityID: number) =>
  request.post<any, string>('/user/deleteUserIdentity', { userIdentityID })

export const getUserLoginLogByUser = (userID: number) =>
  request.get<any, PageListResp<UserLoginLogItem>>('/user/getUserLoginLogByUser', { params: { userID } })

export const getUserDepartmentByUser = (userID: number) =>
  request.get<any, PageListResp<UserDepartmentItem>>('/user/getUserDepartmentByUser', { params: { userID } })
export const assignUserDepartments = (userID: number, departmentIDs: number[]) =>
  request.post<any, string>('/user/assignDepartments', { userID, departmentIDs })

// ==================== 角色 ====================
export const getRolePageList = (data: { page: number; pageSize: number; name?: string }) =>
  request.post<any, PageListResp<RoleItem>>('/role/pageList', data)
export const getRoleDetail = (roleID: number) => request.get<any, RoleItem>('/role/detail', { params: { roleID } })
export const createRole = (data: RoleCreateReq) => request.post<any, { roleID: number }>('/role/create', data)
export const updateRole = (data: RoleUpdateReq) => request.post<any, string>('/role/update', data)
export const deleteRole = (roleID: number) => request.post<any, string>('/role/delete', { roleID })
export const getRoleUsers = (roleId: number) => request.get<any, { total: number; users: RoleUserItem[] }>('/role/users', { params: { roleId } })
export const assignRoleUsers = (roleId: number, userIds: number[]) => request.post<any, string>('/role/assignUsers', { roleId, userIds })
export const removeRoleUser = (roleId: number, userId: number) => request.delete<any, string>(`/role/users/${roleId}/${userId}`)

// ==================== 部门 ====================
export const getDepartmentTree = () => request.get<any, DepartmentTreeResp>('/department/tree')
export const getDepartmentPageList = (data: { page: number; pageSize: number; name?: string }) =>
  request.post<any, PageListResp<DepartmentItem>>('/department/pageList', data)
export const getDepartmentDetail = (departmentID: number) =>
  request.get<any, DepartmentItem>('/department/detail', { params: { departmentID } })
export const createDepartment = (data: Partial<DepartmentItem>) => request.post<any, { departmentID: number }>('/department/create', data)
export const updateDepartment = (data: Partial<DepartmentItem>) => request.post<any, string>('/department/update', data)
export const deleteDepartment = (departmentID: number) => request.post<any, string>('/department/delete', { departmentID })

// ==================== 应用 ====================
export const getApplicationPageList = (data: { page: number; pageSize: number; name?: string }) =>
  request.post<any, PageListResp<ApplicationItem>>('/application/pageList', data)
export const getApplicationDetail = (appId: number) => request.get<any, ApplicationItem>('/application/detail', { params: { appId } })
export const createApplication = (data: ApplicationCreateReq) => request.post<any, { appId: number; code: string }>('/application/create', data)
export const updateApplication = (data: ApplicationUpdateReq) => request.post<any, string>('/application/update', data)
export const deleteApplication = (appId: number) => request.post<any, string>('/application/delete', { appId })

// ==================== OAuth 客户端 ====================
export const getOAuthClientPageList = (data: { page: number; pageSize: number; name?: string }) =>
  request.post<any, PageListResp<OAuthClientItem>>('/applicationClient/pageList', data)
export const getOAuthClientDetail = (applicationClientId: number) =>
  request.get<any, OAuthClientDetail>('/applicationClient/detail', { params: { applicationClientId } })
export const createOAuthClient = (data: OAuthClientCreateReq) =>
  request.post<any, { applicationClientId: number; clientID: string }>('/applicationClient/create', data)
export const updateOAuthClient = (data: OAuthClientUpdateReq) => request.post<any, string>('/applicationClient/update', data)
export const deleteOAuthClient = (applicationClientId: number) =>
  request.post<any, string>('/applicationClient/delete', { applicationClientId })
export const listOAuthSecrets = (applicationClientId: number) =>
  request.get<any, { total: number; secrets: OAuthSecretItem[] }>('/applicationClient/secrets', { params: { applicationClientId } })
export const createOAuthSecret = (data: { applicationClientId: number; name: string; expiresAt?: string }) =>
  request.post<any, OAuthSecretCreateResp>('/applicationClient/secrets', data)
export const deleteOAuthSecret = (secretId: number) => request.delete<any, string>(`/applicationClient/secrets/${secretId}`)

// ==================== 租户 ====================
export const getTenantPageList = (data: { page: number; pageSize: number; name?: string }) =>
  request.post<any, PageListResp<TenantItem>>('/tenant/pageList', data)
export const getTenantDetail = (tenantID: number) => request.get<any, TenantItem>('/tenant/detail', { params: { tenantID } })
export const createTenant = (data: TenantCreateReq) => request.post<any, { tenantID: number }>('/tenant/create', data)
export const updateTenant = (data: TenantUpdateReq) => request.post<any, string>('/tenant/update', data)
export const deleteTenant = (tenantID: number) => request.post<any, string>('/tenant/delete', { tenantID })

// ==================== 租户应用 ====================
export const getTenantApplicationPageList = (data: { page: number; pageSize: number; status?: string }) =>
  request.post<any, PageListResp<TenantApplicationItem>>('/tenantApplication/pageList', data)
export const getTenantApplicationDetail = (tenantAppId: number) =>
  request.get<any, TenantApplicationItem>('/tenantApplication/detail', { params: { tenantAppId } })
export const createTenantApplication = (data: TenantApplicationCreateReq) =>
  request.post<any, { tenantAppId: number }>('/tenantApplication/create', data)
export const updateTenantApplication = (data: TenantApplicationUpdateReq) =>
  request.post<any, string>('/tenantApplication/update', data)
export const deleteTenantApplication = (tenantAppId: number) =>
  request.post<any, string>('/tenantApplication/delete', { tenantAppId })

// ==================== API Key ====================
export const getApiKeyPageList = (data: { page: number; pageSize: number; name?: string }) =>
  request.post<any, PageListResp<ApiKeyItem>>('/apiKey/pageList', data)
export const createApiKey = (data: { name: string; scope?: string; expiresAt?: string }) =>
  request.post<any, ApiKeyCreateResp>('/apiKey/create', data)
export const revokeApiKey = (id: number) => request.post<any, string>('/apiKey/revoke', { id })
export const deleteApiKey = (id: number) => request.post<any, string>('/apiKey/delete', { id })

// ==================== 菜单 ====================
export const getMenuTree = () => request.get<any, MenuTreeResp>('/menu/tree')
export const getMenuPageList = (data: { page: number; pageSize: number; name?: string }) =>
  request.post<any, PageListResp<MenuItem>>('/menu/pageList', data)
export const getMenuDetail = (menuID: number) => request.get<any, MenuItem>('/menu/detail', { params: { menuID } })
export const createMenu = (data: Partial<MenuItem>) => request.post<any, { menuID: number }>('/menu/create', data)
export const updateMenu = (data: Partial<MenuItem>) => request.post<any, string>('/menu/update', data)
export const deleteMenu = (menuID: number) => request.post<any, string>('/menu/delete', { menuID })

// ==================== 权限域 ====================
export const getScopePageList = (data: { page: number; pageSize: number; name?: string }) =>
  request.post<any, PageListResp<ScopeItem>>('/scope/pageList', data)
export const getScopeDetail = (scopeID: number) => request.get<any, ScopeItem>('/scope/detail', { params: { scopeID } })
export const createScope = (data: Partial<ScopeItem>) => request.post<any, { scopeID: number }>('/scope/create', data)
export const updateScope = (data: Partial<ScopeItem>) => request.post<any, string>('/scope/update', data)
export const deleteScope = (scopeID: number) => request.post<any, string>('/scope/delete', { scopeID })

// ==================== 资源 ====================
export const getResourcePageList = (data: { page: number; pageSize: number; name?: string }) =>
  request.post<any, PageListResp<ResourceItem>>('/resource/pageList', data)
export const getResourceDetail = (resourceID: number) => request.get<any, ResourceItem>('/resource/detail', { params: { resourceID } })
export const createResource = (data: Partial<ResourceItem>) => request.post<any, { resourceID: number }>('/resource/create', data)
export const updateResource = (data: Partial<ResourceItem>) => request.post<any, string>('/resource/update', data)
export const deleteResource = (resourceID: number) => request.post<any, string>('/resource/delete', { resourceID })

// ==================== 域名 ====================
export const getDomainPageList = (data: { page: number; pageSize: number; domain?: string }) =>
  request.post<any, PageListResp<DomainItem>>('/domain/pageList', data)
export const getDomainDetail = (id: number) => request.get<any, DomainItem>('/domain/detail', { params: { id } })
export const createDomain = (domain: string) => request.post<any, { id: number }>('/domain/create', { domain })
export const updateDomain = (data: { id: number; domain?: string; isVerified?: number }) =>
  request.post<any, string>('/domain/update', data)
export const deleteDomain = (id: number) => request.post<any, string>('/domain/delete', { id })

// ==================== 系统配置 ====================
export const getSystemPageList = (data: { page: number; pageSize: number; key?: string }) =>
  request.post<any, PageListResp<SystemConfigItem>>('/system/pageList', data)
export const getSystemDetail = (systemID: number) => request.get<any, SystemConfigItem>('/system/detail', { params: { systemID } })
export const createSystemConfig = (data: { key: string; value?: unknown }) => request.post<any, { systemID: number }>('/system/create', data)
export const updateSystemConfig = (data: { systemID: number; key?: string; value?: unknown }) =>
  request.post<any, string>('/system/update', data)
export const deleteSystemConfig = (systemID: number) => request.post<any, string>('/system/delete', { systemID })

// ==================== 审计日志 ====================
export const getAuditLogPageList = (data: { page: number; pageSize: number; key?: string }) =>
  request.post<any, PageListResp<AuditLogItem>>('/log/pageList', data)
export const getAuditLogDetail = (logID: number) => request.get<any, AuditLogItem>('/log/detail', { params: { logID } })

// ==================== Connector（auth） ====================
export const getConnectorPageList = (data: { page: number; pageSize: number; name?: string }) =>
  request.post<any, PageListResp<ConnectorItem>>('/connector/pageList', data)
export const getConnectorDetail = (connectorId: number) => request.get<any, ConnectorItem>('/connector/detail', { params: { connectorId } })
export const createConnector = (data: Partial<ConnectorItem>) => request.post<any, { connectorId: number }>('/connector/create', data)
export const updateConnector = (data: Partial<ConnectorItem>) => request.post<any, string>('/connector/update', data)
export const deleteConnector = (connectorId: number) => request.post<any, string>('/connector/delete', { connectorId })
export const getConnectorFactoryList = (data?: { protocol?: string; provider?: string }) =>
  request.post<any, { list: ConnectorFactoryItem[] }>('/connector/getFactoryList', data)
