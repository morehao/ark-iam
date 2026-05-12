export type AuthStage = 'anonymous' | 'person' | 'tenant'

export interface PersonToken {
  accessToken: string
  refreshToken: string
  expiresIn: number
  tokenType: string
}

export interface TenantToken {
  accessToken: string
  refreshToken: string
  expiresIn: number
  tokenType: string
}

export interface TenantMembership {
  tenantID: number
  name: string
  tag: string
  userID: number
  isOwner: number
}

export interface PersonInfo {
  personID: number
  name: string
  avatar: string
}

export interface UserInfo {
  userID: number
  name: string
  tenantID: number
  isOwner: number
}

export interface ConnectorAuthorizationResp {
  authorizationURL: string
  state: string
}

export interface ConnectorCallbackResp {
  personToken: PersonToken
  tenants: TenantMembership[]
}
