// platformadmin 领域类型（与 backend/apps/platformadmin/internal/dto 对齐）

// ---------- 用户 ----------
export interface UserItem {
  userID: number
  tenantID: number
  username: string
  primaryEmail: string
  primaryPhone: string
  name: string
  avatar: string
  isSuspended: number
  createdAt?: number
}

export interface UserCreateReq {
  personID?: number
  password?: string
  tenantID?: number
  username?: string
  primaryEmail?: string
  primaryPhone?: string
  name?: string
  avatar?: string
  isSuspended?: number
}

export interface UserUpdateReq {
  userID: number
  tenantID?: number
  name?: string
  avatar?: string
  isSuspended?: number
}

export interface UserStatusUpdateReq {
  userID: number
  isSuspended: number
}

export interface UserPasswordUpdateReq {
  userID: number
  password: string
}

export interface UserIdentityItem {
  userIdentityID: number
  tenantID: number
  userID: number
  issuer: string
  identityID: string
  detail?: unknown
  createdAt?: number
}

export interface UserIdentityCreateReq {
  tenantID: number
  userID: number
  issuer: string
  identityID: string
  detail?: unknown
}

export interface UserLoginLogItem {
  userLoginLogID: number
  tenantID: number
  userID: number
  loginIP: string
  userAgent: string
  loginTime: number
  createdAt?: number
}

export interface UserDepartmentItem {
  userDepartmentID: number
  tenantID: number
  userID: number
  departmentID: number
  isPrimary: number
}

// ---------- 角色 ----------
export interface RoleItem {
  roleID: number
  tenantID: number
  name: string
  code: string
  description: string
  type: string
  isDefault: number
  createdAt?: number
}

export interface RoleCreateReq {
  tenantID?: number
  name: string
  code: string
  description?: string
  type?: string
  isDefault?: number
}

export interface RoleUpdateReq {
  roleID: number
  name?: string
  code?: string
  description?: string
  type?: string
  isDefault?: number
}

export interface RoleUserItem {
  userId: number
  username: string
  name: string
  email: string
  roleId: number
  createdAt: string
}

// ---------- 部门 ----------
export interface DepartmentItem {
  departmentID: number
  tenantID: number
  parentID: number
  name: string
  code: string
  sort: number
  leaderUserID: number
  createdAt?: number
  children?: DepartmentItem[]
}

export interface DepartmentTreeResp {
  list: DepartmentItem[]
}

// ---------- 应用 ----------
export interface ApplicationItem {
  appId: number
  code: string
  name: string
  description: string
  logoUrl: string
  homepageUrl: string
  type: string
  status: string
  visibility: string
  sort: number
  tenantPolicy?: unknown
  createdAt?: string
}

export interface ApplicationCreateReq {
  code: string
  name: string
  description?: string
  logoUrl?: string
  homepageUrl?: string
  type?: string
  visibility: string
  sort?: number
}

export interface ApplicationUpdateReq {
  appId: number
  name?: string
  description?: string
  logoUrl?: string
  homepageUrl?: string
  type?: string
  visibility?: string
  status?: string
  sort?: number
}

// ---------- OAuth 客户端 ----------
export interface OAuthClientItem {
  applicationClientId: number
  appId: number
  clientID: string
  name: string
  type: string
  status: string
  isThirdParty: number
  grantTypes: string[]
  tokenEndpointAuthMethod: string
  createdAt?: string
}

export interface OAuthClientDetail extends OAuthClientItem {
  tenantId: number
  redirectURIs: string[]
  postLogoutRedirectURIs: string[]
  backChannelLogoutURI: string
  responseTypes: string[]
  allowedOrigins: string[]
  requirePKCE: number
  requireAuthTime: number
  defaultScopes: string[]
  accessTokenTTL: number
  refreshTokenTTL: number
}

export interface OAuthClientCreateReq {
  appId: number
  name: string
  type?: string
  isThirdParty?: number
  redirectURIs?: string[]
  postLogoutRedirectURIs?: string[]
  grantTypes?: string[]
  responseTypes?: string[]
  tokenEndpointAuthMethod?: string
  allowedOrigins?: string[]
  requirePKCE?: number
  requireAuthTime?: number
  defaultScopes?: string[]
  accessTokenTTL?: number
  refreshTokenTTL?: number
}

export interface OAuthClientUpdateReq {
  applicationClientId: number
  name?: string
  type?: string
  status?: string
  isThirdParty?: number
  redirectURIs?: string[]
  postLogoutRedirectURIs?: string[]
  grantTypes?: string[]
  responseTypes?: string[]
  tokenEndpointAuthMethod?: string
  allowedOrigins?: string[]
  requirePKCE?: number
  requireAuthTime?: number
  defaultScopes?: string[]
  accessTokenTTL?: number
  refreshTokenTTL?: number
}

export interface OAuthSecretItem {
  id: number
  applicationClientId: number
  name: string
  valuePrefix: string
  expiresAt: string | null
  createdAt: string
}

export interface OAuthSecretCreateResp {
  id: number
  name: string
  valuePrefix: string
  secret: string
}

// ---------- 租户 ----------
export interface TenantItem {
  tenantID: number
  code: string
  dbUser: string
  isSuspended: number
  name: string
  tag: string
  type: string
  createdAt?: number
}

export interface TenantCreateReq {
  name: string
  code?: string
  dbUser?: string
  isSuspended?: number
  tag?: string
  type?: string
}

export interface TenantUpdateReq {
  tenantID: number
  name?: string
  code?: string
  dbUser?: string
  isSuspended?: number
  tag?: string
  type?: string
}

// ---------- 租户应用 ----------
export interface TenantApplicationItem {
  tenantAppId: number
  tenantId: number
  appId: number
  status: string
  config?: string
  grantedScope?: string
  createdAt?: string
}

export interface TenantApplicationCreateReq {
  appId: number
  status?: string
  config?: string
  grantedScope?: string
}

export interface TenantApplicationUpdateReq {
  tenantAppId: number
  status?: string
  config?: string
  grantedScope?: string
}

// ---------- API Key ----------
export interface ApiKeyItem {
  id: number
  name: string
  keyPrefix: string
  scope: string
  expiresAt: string
  lastUsedAt: string
  revokedAt: string
  createdAt: string
}

export interface ApiKeyCreateResp {
  id: number
  name: string
  key: string
  keyPrefix: string
  expiresAt: string
}

// ---------- 菜单 ----------
export interface MenuItem {
  menuID: number
  appId: number
  parentID: number
  name: string
  code: string
  path: string
  icon: string
  sort: number
  type: string
  component: string
  redirect: string
  hidden: number
  externalLink: number
  keepAlive: number
  permission: string
  status: string
  createdAt?: number
  children?: MenuItem[]
}

export interface MenuTreeResp {
  list: MenuItem[]
}

// ---------- 权限域 / 资源 ----------
export interface ScopeItem {
  scopeID: number
  tenantID: number
  resourceID: number
  name: string
  description: string
  createdAt?: number
}

export interface ResourceItem {
  resourceID: number
  tenantID: number
  name: string
  indicator: string
  isDefault: number
  accessTokenTtl: number
  createdAt?: number
}

// ---------- 域名 ----------
export interface DomainItem {
  id: number
  domain: string
  isVerified: number
  verifiedAt: string
  createdAt: string
}

// ---------- 系统配置 ----------
export interface SystemConfigItem {
  systemID: number
  tenantID: number
  key: string
  value?: unknown
  createdAt?: number
}

// ---------- 审计日志 ----------
export interface AuditLogItem {
  logID: number
  tenantID: number
  key: string
  payload?: unknown
  createdAt?: number
}

// ---------- Connector（auth） ----------
export interface ConnectorItem {
  connectorId: number
  tenantId: number
  name: string
  displayName: string
  protocol: string
  provider: string
  status: string
  allowAutoCreateUser: number
  allowAccountLink: number
  syncProfile: number
  enableTokenStorage: number
  config?: unknown
  claimMapping?: unknown
  domainPolicy?: unknown
  createdAt?: number
}

export interface ConnectorFactoryItem {
  factoryId: string
  protocol: string
  provider: string
  displayName: string
  isStandard: boolean
  defaultScopes: string[]
  capabilities: string[]
  configSchema?: unknown
}
