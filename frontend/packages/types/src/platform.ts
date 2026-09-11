// platformadmin 领域类型（与 backend/apps/platformadmin/internal/dto 对齐）

// 系统管理等级：member=普通租户成员（无系统管理能力），super=超级管理员。
export type AdminLevel = 'member' | 'super'

// 用户与角色的平台端类型已下线：两者按租户归属，类型见 tenant.ts
// （TenantUserItem / TenantUserIdentityItem / TenantUserLoginLogItem / TenantRoleItem 等）。

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
  allowPersonCreateTenant?: boolean
  allowJoinByInvite?: boolean
  createdAt?: number
  updatedAt?: number
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
  allowPersonCreateTenant?: boolean
  allowJoinByInvite?: boolean
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
  allowPersonCreateTenant?: boolean
  allowJoinByInvite?: boolean
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
  createdAt?: number
  updatedAt?: number
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
  expiresAt: number | null
  createdAt: number
}

export interface OAuthSecretCreateResp {
  id: string
  name: string
  valuePrefix: string
  secret: string
}

// ---------- 租户 ----------
// 租户编码由服务端自动生成（规则 t_<12 位随机 hex>，例 t_3f7a9c1d2e4b），创建/编辑入参均不传 code。
// 时间字段为秒级时间戳（见 AGENTS.md）。
// 租户状态（后端 model.TenantStatus）：active-正常 / suspended-已挂起；
// 非 active 的租户其成员无法登录、令牌不签发，且不能挂起操作者自己所在租户。
export type TenantStatus = 'active' | 'suspended'

export interface TenantItem {
  tenantID: string
  code: string
  dbUser: string
  name: string
  status: TenantStatus
  tag: string
  type: string
  createdAt?: number
  updatedAt?: number
}

export interface TenantCreateReq {
  name: string
  dbUser?: string
  status?: TenantStatus
  tag?: string
  type?: string
}

export interface TenantUpdateReq {
  tenantID: string
  name?: string
  dbUser?: string
  status?: TenantStatus
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
  createdAt?: number
  updatedAt?: number
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

// ---------- 菜单 ----------
export type MenuType = 'directory' | 'menu' | 'button'
export type MenuStatus = 'enable' | 'disable'
export type MenuVisibility = 'public' | 'member' | 'admin'

export interface MenuItem {
  menuID: string
  appID: string
  parentID: string
  name: string
  code: string
  path: string
  icon: string
  sort: number
  type: MenuType
  visibility?: MenuVisibility
  component: string
  redirect: string
  hidden: number
  externalLink: number
  keepAlive: number
  status: MenuStatus
  createdAt?: number
  updatedAt?: number
  children?: MenuItem[]
}

export interface MenuTreeResp {
  list: MenuItem[]
}

// 当前用户可见菜单树（平台侧边栏动态菜单）
export interface MenuMyTreeResp {
  list: MenuItem[]
}

// ---------- 域名 ----------
export interface DomainItem {
  id: string
  domain: string
  isVerified: number
  verifiedAt: number | null
  createdAt: number
  updatedAt: number
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
