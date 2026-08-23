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
  personID?: string
  allowPersonCreateTenant?: boolean
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
