export interface PersonInfo {
  personID: number
  name: string
  avatar: string
}

export interface UserinfoResp {
  personInfo: PersonInfo
  userInfo: { userID: number; tenantID: number; name: string; isOwner: number }
}

export interface MyTenantsResp {
  list: { tenantID: number; name: string }[]
}

export interface PersonDetailResp {
  personID: number
  username: string
  primaryEmail: string
  primaryPhone: string
  name: string
  avatar: string
  isSuspended: number
}

export interface PersonUpdatePasswordReq {
  oldPassword: string
  newPassword: string
}

export interface SessionResp {
  id: number
  sessionID: string
  appID: number
  tenantID: number
  clientType: string
  clientIP: string
  userAgent: string
  expiresAt?: string | null
  createdAt: string
  isActive: boolean
}

export interface SessionListResp {
  list: SessionResp[]
  total: number
}

export interface SessionListReq {
  page?: number
  pageSize?: number
}
