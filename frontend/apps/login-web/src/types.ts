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
  tenants?: { tenantID: string; name: string; tag?: string; userID?: string; isOwner?: number }[]
  personID?: string
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
  tenants?: { tenantID: string; name: string; tag?: string; userID?: string; isOwner?: number }[]
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
  allowPersonCreateTenant: boolean
}
