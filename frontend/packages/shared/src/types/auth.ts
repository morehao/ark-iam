export type AuthStage = 'anonymous' | 'authenticated'

export interface TokenSet {
  accessToken: string
  idToken: string
  refreshToken: string
  expiresAt: number
}

export interface PersonInfo {
  personID: number
  name: string
  avatar: string
}

export interface TenantInfo {
  tenantID: number
  tenantName: string
}
