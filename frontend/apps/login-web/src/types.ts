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
  tenants?: { tenantID: number; name: string; tag?: string; userID?: number; isOwner?: number }[]
}

export interface OIDCSelectTenantReq {
  authRequestID: string
  tenantID: number
}
