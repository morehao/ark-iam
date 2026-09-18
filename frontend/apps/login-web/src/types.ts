/**
 * 租户拥有者类型（后端具名类型 model.OwnerType）：owner-拥有者 / normal-普通成员。
 * login-web 不依赖 @ark-iam/types，故在此按后端同名同形定义一份。
 */
export type OwnerType = 'owner' | 'normal'

/** 租户选择项（登录/注册响应回传，ownerType 标识该租户内的归属） */
export interface TenantSelectionItem {
  tenantID: string
  name: string
  tag?: string
  userID?: string
  ownerType?: OwnerType
}

export interface OIDCLoginReq {
  authRequestID: string
  identifier: string
  password: string
}

export interface ApiResponse<T> {
  code: number
  msg: string
  data: T
}

export interface OIDCLoginResp {
  continueURL: string
  requiresTenantSelection?: boolean
  /** 该账号持临时密码，必须先用 changePassword 设置新密码后才能继续登录 */
  requiresPasswordChange?: boolean
  tenants?: TenantSelectionItem[]
  personID?: string
  /** 计算结果（应用策略 ∧ 零租户），后端 dtooidc 为 bool，非字典列 */
  allowPersonCreateTenant?: boolean
}

export interface OIDCChangePasswordReq {
  authRequestID: string
  currentPassword: string
  newPassword: string
}

export interface OIDCSelectTenantReq {
  authRequestID: string
  tenantID: string
}

export interface RegisterPersonReq {
  authRequestID: string
  username?: string
  primaryEmail?: string
  primaryPhone?: string
  password: string
  name?: string
}

export interface RegisterPersonResp {
  personID: string
  requiresPasswordLogin?: boolean
  requiresTenantSelection: boolean
  tenants?: TenantSelectionItem[]
  /** 计算结果（应用策略 ∧ 零租户），后端 dtooidc 为 bool，非字典列 */
  allowPersonCreateTenant: boolean
}

export interface CreateTenantReq {
  authRequestID: string
  tenantName: string
  tenantCode?: string
}

export interface CreateTenantResp {
  tenantID: string
  personID: string
}

export interface OIDCLoginConfigReq {
  authRequestID: string
}

export interface OIDCLoginConfigResp {
  /** 计算结果（应用策略），后端 dtooidc 为 bool，非字典列 */
  allowPersonCreateTenant: boolean
}
