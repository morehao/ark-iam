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
  request.post<any, PageListResp<UserItem>>('/platform/user/pageList', data)
export const getUserDetail = (userID: number) => request.get<any, UserItem>('/platform/user/detail', { params: { userID } })
export const createUser = (data: UserCreateReq) => request.post<any, { userID: number }>('/platform/user/create', data)
export const updateUser = (data: UserUpdateReq) => request.post<any, string>('/platform/user/update', data)
export const deleteUser = (userID: number) => request.post<any, string>('/platform/user/delete', { userID })
export const updateUserStatus = (data: UserStatusUpdateReq) => request.post<any, string>('/platform/user/updateStatus', data)
export const updateUserPassword = (data: UserPasswordUpdateReq) => request.post<any, string>('/platform/user/updatePassword', data)

export const getUserIdentityByUser = (userID: number) =>
  request.get<any, PageListResp<UserIdentityItem>>('/platform/user/getUserIdentityByUser', { params: { userID } })
export const createUserIdentity = (data: UserIdentityCreateReq) =>
  request.post<any, { userIdentityID: number }>('/platform/user/createUserIdentity', data)
export const deleteUserIdentity = (userIdentityID: number) =>
  request.post<any, string>('/platform/user/deleteUserIdentity', { userIdentityID })

export const getUserLoginLogByUser = (userID: number) =>
  request.get<any, PageListResp<UserLoginLogItem>>('/platform/user/getUserLoginLogByUser', { params: { userID } })

export const getUserDepartmentByUser = (userID: number) =>
  request.get<any, PageListResp<UserDepartmentItem>>('/platform/user/getUserDepartmentByUser', { params: { userID } })
export const assignUserDepartments = (userID: number, departmentIDs: number[]) =>
  request.post<any, string>('/platform/user/assignDepartments', { userID, departmentIDs })

// ==================== 角色 ====================
export const getRolePageList = (data: { page: number; pageSize: number; name?: string }) =>
  request.post<any, PageListResp<RoleItem>>('/platform/role/pageList', data)
export const getRoleDetail = (roleID: number) => request.get<any, RoleItem>('/platform/role/detail', { params: { roleID } })
export const createRole = (data: RoleCreateReq) => request.post<any, { roleID: number }>('/platform/role/create', data)
export const updateRole = (data: RoleUpdateReq) => request.post<any, string>('/platform/role/update', data)
export const deleteRole = (roleID: number) => request.post<any, string>('/platform/role/delete', { roleID })
export const getRoleUsers = (roleID: number) => request.get<any, { total: number; users: RoleUserItem[] }>('/platform/role/users', { params: { roleID } })
export const assignRoleUsers = (roleID: number, userIDs: number[]) => request.post<any, string>('/platform/role/assignUsers', { roleID, userIDs })
export const removeRoleUser = (roleID: number, userID: number) => request.delete<any, string>(`/platform/role/users/${roleID}/${userID}`)

// ==================== 部门 ====================
export const getDepartmentTree = () => request.get<any, DepartmentTreeResp>('/platform/department/tree')
export const getDepartmentPageList = (data: { page: number; pageSize: number; name?: string }) =>
  request.post<any, PageListResp<DepartmentItem>>('/platform/department/pageList', data)
export const getDepartmentDetail = (departmentID: number) =>
  request.get<any, DepartmentItem>('/platform/department/detail', { params: { departmentID } })
export const createDepartment = (data: Partial<DepartmentItem>) => request.post<any, { departmentID: number }>('/platform/department/create', data)
export const updateDepartment = (data: Partial<DepartmentItem>) => request.post<any, string>('/platform/department/update', data)
export const deleteDepartment = (departmentID: number) => request.post<any, string>('/platform/department/delete', { departmentID })

// ==================== 应用 ====================
export const getApplicationPageList = (data: { page: number; pageSize: number; name?: string }) =>
  request.post<any, PageListResp<ApplicationItem>>('/platform/application/pageList', data)
export const getApplicationDetail = (appID: number) => request.get<any, ApplicationItem>('/platform/application/detail', { params: { appID } })
export const createApplication = (data: ApplicationCreateReq) => request.post<any, { appID: number; code: string }>('/platform/application/create', data)
export const updateApplication = (data: ApplicationUpdateReq) => request.post<any, string>('/platform/application/update', data)
export const deleteApplication = (appID: number) => request.post<any, string>('/platform/application/delete', { appID })

// ==================== OAuth 客户端 ====================
export const getOAuthClientPageList = (data: { page: number; pageSize: number; name?: string }) =>
  request.post<any, PageListResp<OAuthClientItem>>('/platform/applicationClient/pageList', data)
export const getOAuthClientDetail = (applicationClientID: number) =>
  request.get<any, OAuthClientDetail>('/platform/applicationClient/detail', { params: { applicationClientID } })
export const createOAuthClient = (data: OAuthClientCreateReq) =>
  request.post<any, { applicationClientID: number; clientID: string }>('/platform/applicationClient/create', data)
export const updateOAuthClient = (data: OAuthClientUpdateReq) => request.post<any, string>('/platform/applicationClient/update', data)
export const deleteOAuthClient = (applicationClientID: number) =>
  request.post<any, string>('/platform/applicationClient/delete', { applicationClientID })
export const listOAuthSecrets = (applicationClientID: number) =>
  request.get<any, { total: number; secrets: OAuthSecretItem[] }>('/platform/applicationClient/secrets', { params: { applicationClientID } })
export const createOAuthSecret = (data: { applicationClientID: number; name: string; expiresAt?: string }) =>
  request.post<any, OAuthSecretCreateResp>('/platform/applicationClient/secrets', data)
export const deleteOAuthSecret = (secretID: number) => request.delete<any, string>(`/platform/applicationClient/secrets/${secretID}`)

// ==================== 租户 ====================
export const getTenantPageList = (data: { page: number; pageSize: number; name?: string }) =>
  request.post<any, PageListResp<TenantItem>>('/platform/tenant/pageList', data)
export const getTenantDetail = (tenantID: number) => request.get<any, TenantItem>('/platform/tenant/detail', { params: { tenantID } })
export const createTenant = (data: TenantCreateReq) => request.post<any, { tenantID: number }>('/platform/tenant/create', data)
export const updateTenant = (data: TenantUpdateReq) => request.post<any, string>('/platform/tenant/update', data)
export const deleteTenant = (tenantID: number) => request.post<any, string>('/platform/tenant/delete', { tenantID })

// ==================== 租户应用 ====================
export const getTenantApplicationPageList = (data: { page: number; pageSize: number; status?: string }) =>
  request.post<any, PageListResp<TenantApplicationItem>>('/platform/tenantApplication/pageList', data)
export const getTenantApplicationDetail = (tenantAppID: number) =>
  request.get<any, TenantApplicationItem>('/platform/tenantApplication/detail', { params: { tenantAppID } })
export const createTenantApplication = (data: TenantApplicationCreateReq) =>
  request.post<any, { tenantAppID: number }>('/platform/tenantApplication/create', data)
export const updateTenantApplication = (data: TenantApplicationUpdateReq) =>
  request.post<any, string>('/platform/tenantApplication/update', data)
export const deleteTenantApplication = (tenantAppID: number) =>
  request.post<any, string>('/platform/tenantApplication/delete', { tenantAppID })

// ==================== API Key ====================
export const getApiKeyPageList = (data: { page: number; pageSize: number; name?: string }) =>
  request.post<any, PageListResp<ApiKeyItem>>('/platform/apiKey/pageList', data)
export const createApiKey = (data: { name: string; scope?: string; expiresAt?: string }) =>
  request.post<any, ApiKeyCreateResp>('/platform/apiKey/create', data)
export const revokeApiKey = (id: number) => request.post<any, string>('/platform/apiKey/revoke', { id })
export const deleteApiKey = (id: number) => request.post<any, string>('/platform/apiKey/delete', { id })

// ==================== 菜单 ====================
export const getMenuTree = (appID: number) => request.get<any, MenuTreeResp>('/platform/menu/tree', { params: { appID } })
export const getMenuPageList = (data: { page: number; pageSize: number; appID?: number; name?: string }) =>
  request.post<any, PageListResp<MenuItem>>('/platform/menu/pageList', data)
export const getMenuDetail = (menuID: number) => request.get<any, MenuItem>('/platform/menu/detail', { params: { menuID } })
export const createMenu = (data: Partial<MenuItem>) => request.post<any, { menuID: number }>('/platform/menu/create', data)
export const updateMenu = (data: Partial<MenuItem>) => request.post<any, string>('/platform/menu/update', data)
export const deleteMenu = (menuID: number) => request.post<any, string>('/platform/menu/delete', { menuID })

// ==================== 权限域 ====================
export const getScopePageList = (data: { page: number; pageSize: number; name?: string }) =>
  request.post<any, PageListResp<ScopeItem>>('/platform/scope/pageList', data)
export const getScopeDetail = (scopeID: number) => request.get<any, ScopeItem>('/platform/scope/detail', { params: { scopeID } })
export const createScope = (data: Partial<ScopeItem>) => request.post<any, { scopeID: number }>('/platform/scope/create', data)
export const updateScope = (data: Partial<ScopeItem>) => request.post<any, string>('/platform/scope/update', data)
export const deleteScope = (scopeID: number) => request.post<any, string>('/platform/scope/delete', { scopeID })

// ==================== 资源 ====================
export const getResourcePageList = (data: { page: number; pageSize: number; name?: string }) =>
  request.post<any, PageListResp<ResourceItem>>('/platform/resource/pageList', data)
export const getResourceDetail = (resourceID: number) => request.get<any, ResourceItem>('/platform/resource/detail', { params: { resourceID } })
export const createResource = (data: Partial<ResourceItem>) => request.post<any, { resourceID: number }>('/platform/resource/create', data)
export const updateResource = (data: Partial<ResourceItem>) => request.post<any, string>('/platform/resource/update', data)
export const deleteResource = (resourceID: number) => request.post<any, string>('/platform/resource/delete', { resourceID })

// ==================== 域名 ====================
export const getDomainPageList = (data: { page: number; pageSize: number; domain?: string }) =>
  request.post<any, PageListResp<DomainItem>>('/platform/domain/pageList', data)
export const getDomainDetail = (id: number) => request.get<any, DomainItem>('/platform/domain/detail', { params: { id } })
export const createDomain = (domain: string) => request.post<any, { id: number }>('/platform/domain/create', { domain })
export const updateDomain = (data: { id: number; domain?: string; isVerified?: number }) =>
  request.post<any, string>('/platform/domain/update', data)
export const deleteDomain = (id: number) => request.post<any, string>('/platform/domain/delete', { id })

// ==================== 系统配置 ====================
export const getSystemPageList = (data: { page: number; pageSize: number; key?: string }) =>
  request.post<any, PageListResp<SystemConfigItem>>('/platform/system/pageList', data)
export const getSystemDetail = (systemID: number) => request.get<any, SystemConfigItem>('/platform/system/detail', { params: { systemID } })
export const createSystemConfig = (data: { key: string; value?: unknown }) => request.post<any, { systemID: number }>('/platform/system/create', data)
export const updateSystemConfig = (data: { systemID: number; key?: string; value?: unknown }) =>
  request.post<any, string>('/platform/system/update', data)
export const deleteSystemConfig = (systemID: number) => request.post<any, string>('/platform/system/delete', { systemID })

// ==================== 审计日志 ====================
export const getAuditLogPageList = (data: { page: number; pageSize: number; key?: string }) =>
  request.post<any, PageListResp<AuditLogItem>>('/platform/log/pageList', data)
export const getAuditLogDetail = (logID: number) => request.get<any, AuditLogItem>('/platform/log/detail', { params: { logID } })

// ==================== Connector（auth） ====================
export const getConnectorPageList = (data: { page: number; pageSize: number; name?: string }) =>
  request.post<any, PageListResp<ConnectorItem>>('/auth/connector/pageList', data)
export const getConnectorDetail = (connectorID: number) => request.get<any, ConnectorItem>('/auth/connector/detail', { params: { connectorID } })
export const createConnector = (data: Partial<ConnectorItem>) => request.post<any, { connectorID: number }>('/auth/connector/create', data)
export const updateConnector = (data: Partial<ConnectorItem>) => request.post<any, string>('/auth/connector/update', data)
export const deleteConnector = (connectorID: number) => request.post<any, string>('/auth/connector/delete', { connectorID })
export const getConnectorFactoryList = (data?: { protocol?: string; provider?: string }) =>
  request.post<any, { list: ConnectorFactoryItem[] }>('/auth/connector/getFactoryList', data)
