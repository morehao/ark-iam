// platformadmin 领域类型（与 apps/platformadmin/internal/dto 对齐）

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

/**
 * 应用策略开关（后端具名类型，取值 enable/disable；语义等价旧 bool：true=enable / false=disable）：
 * - AppPersonCreateTenantPolicy：application.allow_person_create_tenant（个人自助创建租户）
 * - AppJoinByInvitePolicy：application.allow_join_by_invite（凭邀请码加入租户）
 */
export type AppPersonCreateTenantPolicy = 'enable' | 'disable'
export type AppJoinByInvitePolicy = 'enable' | 'disable'

/** 客户端协议开关（后端具名类型）：requirePKCE / requireAuthTime，取值 enable/disable。 */
export type ClientPKCEPolicy = 'enable' | 'disable'
export type ClientAuthTimeClaimPolicy = 'enable' | 'disable'

/**
 * 应用角色模板项：本应用对外提供的一个跨系统授权契约值。
 * 下游按「claim_prefix + code」认策略名（策略在下游是全局命名实体、全租户共用一条），
 * 故 code 由应用方在这里定义一次，租户只能"授权"、不能造值（后端 model.RoleTemplateItem）。
 */
export interface ApplicationRoleTemplateItem {
  /** 契约值（= OIDC ID token 的 groups 取值）：小写字母开头，仅含小写字母、数字与下划线，模板内唯一 */
  code: string
  /** 展示名：物化为租户角色的名称，非空且不超过 128 字符 */
  name: string
}

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
  allowPersonCreateTenant?: AppPersonCreateTenantPolicy
  allowJoinByInvite?: AppJoinByInvitePolicy
  /** 应用角色模板：本应用对外的契约值清单，开通应用时物化到各租户（后端 model.ApplicationEntity.RoleTemplate）。 */
  roleTemplate?: ApplicationRoleTemplateItem[]
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
  allowPersonCreateTenant?: AppPersonCreateTenantPolicy
  allowJoinByInvite?: AppJoinByInvitePolicy
  /** 应用角色模板：见 ApplicationItem.roleTemplate，留空表示无。 */
  roleTemplate?: ApplicationRoleTemplateItem[]
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
  allowPersonCreateTenant?: AppPersonCreateTenantPolicy
  allowJoinByInvite?: AppJoinByInvitePolicy
  /** 应用角色模板：不传表示不修改，传 [] 表示清空，传值即全量替换（移除项会在各租户撤下该角色及其授权）。 */
  roleTemplate?: ApplicationRoleTemplateItem[]
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
  /** 是否强制 PKCE（后端 model.ClientPKCEPolicy）：enable/disable。 */
  requirePKCE: ClientPKCEPolicy
  /** 是否要求 id_token 带 auth_time（后端 model.ClientAuthTimeClaimPolicy）：enable/disable。 */
  requireAuthTime: ClientAuthTimeClaimPolicy
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
  /** 规则开关（后端具名类型 ClientPKCEPolicy）：enable/disable */
  requirePKCE?: ClientPKCEPolicy
  requireAuthTime?: ClientAuthTimeClaimPolicy
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
  /** 规则开关（后端具名类型 ClientPKCEPolicy）：enable/disable */
  requirePKCE?: ClientPKCEPolicy
  requireAuthTime?: ClientAuthTimeClaimPolicy
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
// 租户状态（后端 model.TenantStatus）：provisioning-建租中 / active-正常 / suspended-已挂起；
// 仅 active 的租户其成员可登录、令牌可签发。D 租户的 provisioning 在 primary 域名 activate 后转 active，
// suspended 由 offboard 动作置位；D 编辑（PUT）不提交 status，状态只经 activate/offboard 动作流转。
export type TenantStatus = 'provisioning' | 'active' | 'suspended'
/**
 * 租户隔离模式（后端具名类型 model.TenantIsolationMode）：
 * dedicated-专属（自定义域 + 独立 issuer，D）/ shared-共享（中央 issuer，S）。
 * create-only：P0 控制台仅新建时可选，PUT 不可改。
 */
export type TenantIsolationMode = 'dedicated' | 'shared'
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
  /** 隔离模式（后端 model.TenantIsolationMode）：dedicated-专属 / shared-共享。 */
  isolationMode?: TenantIsolationMode
  /** D 租户派生缓存 issuer（active 时 = active primary host 派生）；S/平台为空。只读回显。 */
  issuer?: string
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
  /** 状态：普通（shared）建租户由服务端置 active；D（dedicated）强制 provisioning（传值被忽略/拒绝）。 */
  status?: TenantStatus
  tag?: string
  type?: TenantType
  /** 隔离模式（后端 model.TenantIsolationMode）：缺省/空=shared，dedicated=新建 D 专属租户。 */
  isolationMode?: TenantIsolationMode
  /** 仅 dedicated 新建时使用：同事务创建该 primary（draft 域名）。普通建租户忽略。 */
  primaryDomain?: string
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
  /** 临时密码是否已通过邮件/短信投递（true 时 adminInitialPassword 为空）。 */
  delivered: boolean
}

/** 重置租户内置管理员密码响应：新临时密码仅此一次返回，该管理员下次登录必须改密。 */
export interface TenantAdminResetPasswordResp {
  userID: string
  initialPassword: string
  delivered: boolean
}

export interface TenantUpdateReq {
  tenantID: string
  name?: string
  dbUser?: string
  /**
   * 状态仅对普通（shared）租户可经 PUT 修改；D（dedicated）租户编辑**不提交** status
   * （状态只经 activate/offboard 动作流转，后端对 D 的 PUT 改 status 会拒写）。
   */
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
  /**
   * 该租户下已准入（user_application.status=enable）本应用的成员数。
   * 停用/删除订阅会让这些成员失去本应用访问，故用于确认文案回显「影响人数」。
   */
  grantedUserCount?: number
  createdAt?: number
  updatedAt?: number
}

export interface TenantApplicationCreateReq {
  tenantID: string
  appID: string
  status?: TenantApplicationStatus
}

export interface TenantApplicationUpdateReq {
  tenantAppID: string
  status?: TenantApplicationStatus
}

// ---------- 菜单 ----------
export type MenuType = 'directory' | 'menu' | 'button'
export type MenuStatus = 'enable' | 'disable'
export type MenuVisibility = 'public' | 'member' | 'admin'
/**
 * 菜单展示开关（后端具名类型，取值 enable/disable；语义等价旧 0/1：1=enable / 0=disable）：
 * MenuHiddenFlag / MenuExternalLinkFlag / MenuKeepAliveFlag。
 */
export type MenuHiddenFlag = 'enable' | 'disable'
export type MenuExternalLinkFlag = 'enable' | 'disable'
export type MenuKeepAliveFlag = 'enable' | 'disable'

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
  /** 是否隐藏（后端 model.MenuHiddenFlag）：enable/disable */
  hidden: MenuHiddenFlag
  /** 是否外链（后端 model.MenuExternalLinkFlag）：enable/disable */
  externalLink: MenuExternalLinkFlag
  /** 是否缓存（后端 model.MenuKeepAliveFlag）：enable/disable */
  keepAlive: MenuKeepAliveFlag
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
/**
 * 域名生命周期状态（后端具名类型 model.DomainStatus，7 态单一状态机，替代原布尔「是否已验证」）：
 * - draft：已占 host、未验证；不建 TLS，不解析/不签发；
 * - verified：域控证明通过（verifiedAt）；不解析/不签发；
 * - tls_ready：证书就绪（tlsReadyAt）；不解析/不签发；
 * - active：唯一可用态；primary 可签发，login/both alias 可 302；
 * - disabled：停止解析/签发，可回 active（仅 alias）；
 * - retired：墓碑；host 仍占用；协议请求 400/404；
 * - released：隔离期满/审批后软删；host 可重新登记（终态）。
 */
export type DomainStatus = 'draft' | 'verified' | 'tls_ready' | 'active' | 'disabled' | 'retired' | 'released'

/**
 * 域名角色（后端具名类型 model.DomainRole）：
 * - primary：主域（每 active D 租户恰一个 active primary；作为 issuer 来源）；
 * - alias：别名（302 到 primary，不签 token）；
 * - branded：仅 P1/P2 预留，P0 不消费。
 */
export type DomainRole = 'primary' | 'alias' | 'branded'

/**
 * 域名用途（后端具名类型 model.DomainUsage）：
 * - login：登录宿主（可作为 issuer / 302 target）；
 * - email：邮箱域发现（P0 只登记，不路由/不签发）；
 * - both：登录 + 邮箱。
 */
export type DomainUsage = 'login' | 'email' | 'both'

/** 域控证明方式（后端具名类型 model.DomainVerificationMethod）：manual-人工留证 / dns_txt-DNS TXT 挑战。 */
export type DomainVerificationMethod = 'manual' | 'dns_txt'

/** 域名列表项（后端 DomainPageListItem）。 */
export interface DomainItem {
  id: string
  tenantID: string
  domain: string
  domainRole: DomainRole
  domainUsage: DomainUsage
  status: DomainStatus
  createdAt: number
  updatedAt: number
}

/** 域名详情（后端 DomainDetailResp）：比列表多验证方式、target、各里程碑时间、退役/释放审计字段。 */
export interface DomainDetail extends DomainItem {
  targetDomainID: string
  verificationMethod: DomainVerificationMethod
  /** unix 秒，空为 null */
  verifiedAt: number | null
  tlsReadyAt: number | null
  retiredAt: number | null
  releasedAt: number | null
  retireReason: string
  releaseApproval: string
}

/** 创建域名入参（后端 DomainCreateReq）。P0 只创建 draft，状态流转走动作端点。 */
export interface DomainCreateReq {
  tenantID: string
  domain: string
  domainRole: DomainRole
  /** primary 只允许 login/both；alias/email 需 targetDomain。 */
  domainUsage: DomainUsage
  /** alias/email 目标 primary 的 host（服务端解析为同租户 target_domain_id）。 */
  targetDomain?: string
  /** manual/dns_txt（默认 manual）。 */
  verificationMethod?: DomainVerificationMethod
}

/** 域名分页筛选参数（后端 DomainPageListReq）。 */
export interface DomainPageListParams {
  page: number
  pageSize: number
  tenantID?: string
  domain?: string
  status?: DomainStatus
  domainRole?: DomainRole
  domainUsage?: DomainUsage
}

/** 域名动作端点响应（后端 DomainActionResp）：回传动作后的最新状态。 */
export interface DomainActionResp {
  domainId: string
  status: DomainStatus
}

/** verify 动作入参（后端 DomainVerifyReq 的 body 部分；domainID 走 path）。 */
export interface DomainVerifyReq {
  method?: DomainVerificationMethod
  token?: string
}

// ---------- Connector（platform 全局对象） ----------
/** 连接器协议（后端具名类型 model.ConnectorProtocol）。 */
export type ConnectorProtocol = 'oidc' | 'oauth2'
/** 连接器身份提供商（后端具名类型 model.ConnectorProvider）。 */
export type ConnectorProvider = 'google' | 'github' | 'microsoft' | 'wechat'
/** 连接器能力（后端具名类型 model.ConnectorCapability）。 */
export type ConnectorCapability = 'authorize' | 'callback' | 'claim_mapping' | 'domain_policy' | 'profile_sync'

/**
 * 连接器开关（后端具名类型，取值 enable/disable；语义等价旧 bool：true=enable / false=disable）：
 * ConnectorAutoCreateUserFlag / ConnectorSyncProfileFlag / ConnectorTokenStorageFlag。
 */
export type ConnectorAutoCreateUserFlag = 'enable' | 'disable'
export type ConnectorSyncProfileFlag = 'enable' | 'disable'
export type ConnectorTokenStorageFlag = 'enable' | 'disable'

/**
 * 连接器协议配置（后端 model.ConnectorConfig，落 connector.config，JSON 列）。
 * OIDC 用 issuer（+ microsoft 额外 tenant）；OAuth2 用 authUrl/tokenUrl/userInfoUrl。
 * protocol/provider 由所选工厂带入，一般不由用户手填。
 * 注意：这里的 `tenant` 是**协议级**配置（部分 IdP 如 Microsoft 的租户标识），
 * 与连接器的 `defaultTenantID`（自动创建用户默认落租户）语义完全不同，勿混淆。
 */
export interface ConnectorConfig {
  protocol?: ConnectorProtocol
  provider?: ConnectorProvider
  issuer?: string
  authUrl?: string
  tokenUrl?: string
  userInfoUrl?: string
  clientID?: string
  clientSecret?: string
  redirectUri?: string
  scopes?: string[]
  tenant?: string
}

/** 声明映射（后端 model.ConnectorClaimMapping，落 connector.claim_mapping，JSON 列）。 */
export interface ConnectorClaimMapping {
  subject?: string
  email?: string
  username?: string
  displayName?: string
  avatarUrl?: string
}

/** 域策略（后端 model.ConnectorDomainPolicy，落 connector.domain_policy，JSON 列）。 */
export interface ConnectorDomainPolicy {
  allowedDomains?: string[]
  blockedDomains?: string[]
}

/** 连接器工厂 metadata 的 configSchema（当前后端只下发 required 列表）。 */
export interface ConnectorConfigSchema {
  type?: string
  required?: string[]
}

export interface ConnectorItem {
  connectorID: string
  /** 自动创建用户默认落租户（后端 connector.default_tenant_id）。 */
  defaultTenantID: string
  /** 默认落租户名称（列表/详情展示用，后端回填）。 */
  defaultTenantName?: string
  name: string
  displayName: string
  protocol: ConnectorProtocol
  provider: ConnectorProvider
  status: ConnectorStatus
  allowAutoCreateUser: ConnectorAutoCreateUserFlag
  syncProfile: ConnectorSyncProfileFlag
  enableTokenStorage: ConnectorTokenStorageFlag
  config?: ConnectorConfig
  claimMapping?: ConnectorClaimMapping
  domainPolicy?: ConnectorDomainPolicy
  createdAt?: number
  updatedAt?: number
}

/**
 * 创建/更新连接器请求体（后端 dtoauth.ConnectorCreateReq / ConnectorUpdateReq）。
 * `defaultTenantID`（自动创建用户默认落租户）仅在 allowAutoCreateUser=enable 时必填。
 */
export interface ConnectorCreateReq {
  name: string
  displayName: string
  protocol: ConnectorProtocol
  provider: ConnectorProvider
  status?: ConnectorStatus
  defaultTenantID?: string
  allowAutoCreateUser?: ConnectorAutoCreateUserFlag
  syncProfile?: ConnectorSyncProfileFlag
  enableTokenStorage?: ConnectorTokenStorageFlag
  config?: ConnectorConfig
  claimMapping?: ConnectorClaimMapping
  domainPolicy?: ConnectorDomainPolicy
}

/** 更新请求与创建同构；status 传空串表示「不修改状态」。 */
export type ConnectorUpdateReq = ConnectorCreateReq

export interface ConnectorFactoryItem {
  factoryID: string
  protocol: ConnectorProtocol
  provider: ConnectorProvider
  displayName: string
  isStandard: boolean
  defaultScopes: string[]
  capabilities: ConnectorCapability[]
  configSchema?: ConnectorConfigSchema
}

// ==================== 系统设置（运行时配置中心） ====================

/** 配置项值类型 */
export type SettingValueType = 'json' | 'toml' | 'yaml' | 'string' | 'int' | 'bool' | 'float'

/** 配置项状态 */
export type SettingStatus = 'enabled' | 'disabled'

/** 配置项加密模式 */
export type SettingEncryptionMode = 'plain' | 'encrypted'

/** 配置项（敏感项 value 为掩码 "******"） */
export interface SettingItem {
  id: string
  group: string
  key: string
  valueType: SettingValueType
  value: string
  encryptionMode: SettingEncryptionMode
  description?: string
  status: SettingStatus
  createdAt: number
  updatedAt: number
}

/** 创建配置项请求 */
export interface SettingCreateReq {
  group: string
  key: string
  valueType?: SettingValueType
  value: string
  encrypted?: boolean
  status?: SettingStatus
  description?: string
}

/** 更新配置项请求（value 传掩码表示不改） */
export interface SettingUpdateReq {
  value?: string
  encrypted?: boolean
  status?: SettingStatus
  description?: string
}

/** 配置分组元数据（供渲染分组下拉与说明；单一事实源在后端 svcsetting） */
export interface SettingGroupMeta {
  group: string
  description?: string
}
