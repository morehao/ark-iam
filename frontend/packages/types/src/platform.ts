// platformadmin 领域类型（与 backend/apps/platformadmin/internal/dto 对齐）

// ---------- 用户 ----------
export interface UserItem {
  userID: string
  tenantID: string
  username: string
  primaryEmail: string
  primaryPhone: string
  name: string
  avatar: string
  isSuspended: number
  createdAt?: number
}

export interface UserCreateReq {
  personID?: string
  password?: string
  tenantID?: string
  username?: string
  primaryEmail?: string
  primaryPhone?: string
  name?: string
  avatar?: string
  isSuspended?: number
}

export interface UserUpdateReq {
  userID: string
  tenantID?: string
  name?: string
  avatar?: string
  isSuspended?: number
}

export interface UserStatusUpdateReq {
  userID: string
  isSuspended: number
}

export interface UserPasswordUpdateReq {
  userID: string
  password: string
}

export interface UserIdentityItem {
  userIdentityID: string
  tenantID: string
  userID: string
  issuer: string
  identityID: string
  detail?: unknown
  createdAt?: number
}

export interface UserIdentityCreateReq {
  tenantID: string
  userID: string
  issuer: string
  identityID: string
  detail?: unknown
}

export interface UserLoginLogItem {
  userLoginLogID: string
  tenantID: string
  userID: string
  loginIP: string
  userAgent: string
  loginTime: number
  createdAt?: number
}

// ---------- 角色 ----------
export interface RoleItem {
  roleID: string
  tenantID: string
  name: string
  code: string
  description: string
  type: string
  isDefault: number
  createdAt?: number
}

export interface RoleCreateReq {
  tenantID?: string
  name: string
  code: string
  description?: string
  type?: string
  isDefault?: number
}

export interface RoleUpdateReq {
  roleID: string
  name?: string
  code?: string
  description?: string
  type?: string
  isDefault?: number
}

export interface RoleUserItem {
  userID: string
  username: string
  name: string
  email: string
  roleID: string
  createdAt: string
}

// ---------- 部门 ----------
// ---------- 应用 ----------
export interface ApplicationItem {
  appID: string
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
  appID: string
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
  applicationClientID: string
  appID: string
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
  tenantID: string
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
  appID: string
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
  applicationClientID: string
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
  id: string
  applicationClientID: string
  name: string
  valuePrefix: string
  expiresAt: string | null
  createdAt: string
}

export interface OAuthSecretCreateResp {
  id: string
  name: string
  valuePrefix: string
  secret: string
}

// ---------- 租户 ----------
export interface TenantItem {
  tenantID: string
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
  tenantID: string
  name?: string
  code?: string
  dbUser?: string
  isSuspended?: number
  tag?: string
  type?: string
}

// ---------- 租户应用 ----------
export interface TenantApplicationItem {
  tenantAppID: string
  tenantID: string
  appID: string
  status: string
  config?: string
  grantedScope?: string
  createdAt?: string
}

export interface TenantApplicationCreateReq {
  appID: string
  status?: string
  config?: string
  grantedScope?: string
}

export interface TenantApplicationUpdateReq {
  tenantAppID: string
  status?: string
  config?: string
  grantedScope?: string
}

// ---------- API Key ----------
export interface ApiKeyItem {
  id: string
  name: string
  keyPrefix: string
  scope: string
  expiresAt: string
  lastUsedAt: string
  revokedAt: string
  createdAt: string
}

export interface ApiKeyCreateResp {
  id: string
  name: string
  key: string
  keyPrefix: string
  expiresAt: string
}

// ---------- 菜单 ----------
export interface MenuItem {
  menuID: string
  appID: string
  parentID: string
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
  scopeID: string
  tenantID: string
  resourceID: string
  name: string
  description: string
  createdAt?: number
}

export interface ResourceItem {
  resourceID: string
  tenantID: string
  name: string
  indicator: string
  isDefault: number
  accessTokenTtl: number
  createdAt?: number
}

// ---------- 域名 ----------
export interface DomainItem {
  id: string
  domain: string
  isVerified: number
  verifiedAt: string
  createdAt: string
}

// ---------- 系统配置 ----------
export interface SystemConfigItem {
  systemID: string
  tenantID: string
  key: string
  value?: unknown
  createdAt?: number
}

// ---------- 审计日志 ----------
export interface AuditLogItem {
  logID: string
  tenantID: string
  key: string
  payload?: unknown
  createdAt?: number
}

// ---------- Connector（auth） ----------
export interface ConnectorItem {
  connectorID: string
  tenantID: string
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
  factoryID: string
  protocol: string
  provider: string
  displayName: string
  isStandard: boolean
  defaultScopes: string[]
  capabilities: string[]
  configSchema?: unknown
}
