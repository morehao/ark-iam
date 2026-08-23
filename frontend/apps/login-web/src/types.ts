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
  tenants?: { tenantID: string; name: string; tag?: string; userID?: string; isOwner?: number }[]
}

export interface OIDCSelectTenantReq {
  authRequestID: string
  tenantID: string
}

// 通道 A：自助开通租户（POST /v1/auth/register）
export interface RegisterOrgReq {
  tenantName: string
  tenantCode?: string
  username?: string
  primaryEmail?: string
  primaryPhone?: string
  password: string
  name?: string
}

export interface RegisterOrgResp {
  userID: string
  tenantID: string
  sessionID?: string
}

// 通道 B：凭邀请加入租户（POST /v1/auth/joinTenant）
export interface JoinTenantReq {
  inviteCode: string
}

export interface JoinTenantResp {
  userID: string
}
