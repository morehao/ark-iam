// platformadmin 领域类型（与 backend/apps/platformadmin/internal/dto 对齐）

// 系统管理类型：admin=管理员角色（具备系统管理能力），normal=普通角色（不具备）。
export type SysAdminType = 'admin' | 'normal'

// 用户与角色的平台端类型已下线：两者按租户归属，类型见 tenant.ts
// （TenantUserItem / TenantUserIdentityItem / TenantUserLoginLogItem / TenantRoleItem 等）。

// ---------- 部门 ----------
// ---------- 应用 ----------
/**
 * 应用/客户端来源（后端 model.AppSource / model.ApplicationClientSource）：
 * builtin-内置（平台随产品交付的控制台应用：平台管理后台 / 租户管理后台，受删除保护）、
 * first_party-第一方受控应用（平台自建但非内置，当前由运维产生）、third_party-第三方接入。
 * 控制台创建的资源恒为 third_party；builtin / first_party 只能由种子与运维产生。
 */
export type AppSource = 'builtin' | 'first_party' | 'third_party'

/**
 * 启停状态（后端具名类型 model.AppStatus / ApplicationClientStatus / TenantApplicationStatus）。
 * 取值与后端常量一一对应，禁止在页面里裸写字面量。
 */
export type AppStatus = 'enable' | 'disable'
export type ApplicationClientStatus = 'enable' | 'disable'
export type TenantApplicationStatus = 'enable' | 'disable'
/** 连接器状态（后端具名类型 model.ConnectorStatus）：全局启停语义 enable/disable */
export type ConnectorStatus = 'enable' | 'disable'

export interface ApplicationItem {
  appID: string
  code: string
  name: string
  description: string
  logoUrl: string
  homepageUrl: string
  source: AppSource
  status: AppStatus
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
  sort?: number
  allowPersonCreateTenant?: boolean
  allowJoinByInvite?: boolean
}

export interface ApplicationUpdateReq {
  appID: string
  /** 应用编码（自建应用可改，留空不修改；内置应用 source=builtin 恒只读）：小写字母开头，仅含小写字母、数字与下划线（后端 model.AppCodePattern）。 */
  code?: string
  name?: string
  description?: string
  logoUrl?: string
  homepageUrl?: string
  status?: AppStatus
  sort?: number
  allowPersonCreateTenant?: boolean
  allowJoinByInvite?: boolean
}

// ---------- OAuth 客户端 ----------
/** 授权类型（后端具名类型 model.GrantType，RFC 6749 标准字面量）。 */
export type GrantType = 'authorization_code' | 'client_credentials' | 'refresh_token'
/** 令牌端点认证方式（后端具名类型 model.TokenEndpointAuthMethod，RFC 8414 标准字面量）。 */
export type TokenEndpointAuthMethod = 'client_secret_basic' | 'client_secret_post' | 'none'

export interface OAuthClientItem {
  applicationClientID: string
  appID: string
  /** 所属应用名称（后端列表/详情回填，避免前端只拿到 appID） */
  appName: string
  /** 客户端编码 = OIDC client_id（后端 model.ApplicationClientEntity.Code，全局唯一；允许连字符） */
  code: string
  name: string
  source: AppSource
  status: ApplicationClientStatus
  grantTypes: GrantType[]
  tokenEndpointAuthMethod: TokenEndpointAuthMethod
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
  /** 是否强制 PKCE：后端 DTO 是 Go bool，JSON 为 true/false（不是 0/1） */
  requirePKCE: boolean
  /** 是否要求 id_token 带 auth_time：后端 DTO 是 Go bool，JSON 为 true/false */
  requireAuthTime: boolean
  defaultScopes: string[]
  accessTokenTTL: number
  refreshTokenTTL: number
}

export interface OAuthClientCreateReq {
  appID: string
  /** 客户端编码（= OIDC client_id）：小写字母开头，仅含小写字母与下划线（后端 model.ClientCodePattern）。 */
  code: string
  name: string
  redirectURIs?: string[]
  postLogoutRedirectURIs?: string[]
  backChannelLogoutURI?: string
  grantTypes?: GrantType[]
  responseTypes?: string[]
  tokenEndpointAuthMethod?: TokenEndpointAuthMethod
  allowedOrigins?: string[]
  /** 后端 DTO 是 Go bool（true/false）：传 1/0 会被 binding 拒绝 */
  requirePKCE?: boolean
  requireAuthTime?: boolean
  defaultScopes?: string[]
  accessTokenTTL?: number
  refreshTokenTTL?: number
}

export interface OAuthClientUpdateReq {
  applicationClientID: string
  /** 客户端编码（= OIDC client_id，可改，留空不修改）：仅小写字母与下划线；内置客户端拒改。 */
  code?: string
  name?: string
  status?: ApplicationClientStatus
  redirectURIs?: string[]
  postLogoutRedirectURIs?: string[]
  backChannelLogoutURI?: string
  grantTypes?: GrantType[]
  responseTypes?: string[]
  tokenEndpointAuthMethod?: TokenEndpointAuthMethod
  allowedOrigins?: string[]
  /** 后端 DTO 是 Go bool（true/false）：传 1/0 会被 binding 拒绝 */
  requirePKCE?: boolean
  requireAuthTime?: boolean
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
/**
 * 租户类型（后端具名类型 model.TenantType）：
 * customer-客户租户 / platform-平台租户（分类标识，不参与隔离判定）。
 */
export type TenantType = 'customer' | 'platform'

export interface TenantItem {
  tenantID: string
  code: string
  dbUser: string
  name: string
  status: TenantStatus
  tag: string
  type: TenantType
  createdAt?: number
  updatedAt?: number
}

/**
 * 建租户时的内置管理员（必填）：建租户即产出可登录的租户管理员。
 * 不含密码——初始临时密码由服务端生成并仅在创建响应返回一次（见
 * docs/design/system-design.md §5.8）。
 */
export interface TenantAdminCreateReq {
  name: string
  username?: string
  primaryEmail?: string
  primaryPhone?: string
}

export interface TenantCreateReq {
  name: string
  dbUser?: string
  status?: TenantStatus
  tag?: string
  type?: TenantType
  admin: TenantAdminCreateReq
}

/**
 * 建租户响应。adminInitialPassword 为空表示该管理员的邮箱/手机命中了已存在的自然人
 * （其密码未被改动），此时不会回显任何凭据。
 */
export interface TenantCreateResp {
  tenantID: string
  adminUserID: string
  adminInitialPassword: string
}

/** 重置租户内置管理员密码响应：新临时密码仅此一次返回，该管理员下次登录必须改密。 */
export interface TenantAdminResetPasswordResp {
  userID: string
  initialPassword: string
}

export interface TenantUpdateReq {
  tenantID: string
  name?: string
  dbUser?: string
  status?: TenantStatus
  tag?: string
  type?: TenantType
}

// ---------- 租户应用 ----------
export interface TenantApplicationItem {
  tenantAppID: string
  tenantID: string
  tenantName?: string
  appID: string
  appName?: string
  /**
   * 所属应用来源（后端 model.AppSource）。builtin=订阅的是内置应用（平台管理后台/租户管理后台），
   * 该订阅由系统开通（种子 / ProvisionTenantAdmin），不可删除——前端据此隐藏「删除」。
   */
  appSource?: AppSource
  status: TenantApplicationStatus
  config?: string
  grantedScope?: string
  createdAt?: number
  updatedAt?: number
}

export interface TenantApplicationCreateReq {
  tenantID: string
  appID: string
  status?: TenantApplicationStatus
  config?: string
  grantedScope?: string
}

export interface TenantApplicationUpdateReq {
  tenantAppID: string
  status?: TenantApplicationStatus
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
/** 连接器协议（后端具名类型 model.ConnectorProtocol）。 */
export type ConnectorProtocol = 'oidc' | 'oauth2'
/** 连接器身份提供商（后端具名类型 model.ConnectorProvider）。 */
export type ConnectorProvider = 'google' | 'github' | 'microsoft' | 'wechat'
/** 连接器能力（后端具名类型 model.ConnectorCapability）。 */
export type ConnectorCapability = 'authorize' | 'callback' | 'claim_mapping' | 'domain_policy' | 'profile_sync'

export interface ConnectorItem {
  connectorID: string
  tenantID: string
  name: string
  displayName: string
  protocol: ConnectorProtocol
  provider: ConnectorProvider
  status: ConnectorStatus
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
  protocol: ConnectorProtocol
  provider: ConnectorProvider
  displayName: string
  isStandard: boolean
  defaultScopes: string[]
  capabilities: ConnectorCapability[]
  configSchema?: unknown
}
