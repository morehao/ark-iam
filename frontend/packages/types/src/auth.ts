export interface PersonInfo {
  personID: string
  name: string
  avatar: string
}

export interface UserinfoResp {
  personInfo: PersonInfo
  userInfo: { userID: string; tenantID: string; name: string; isOwner: number }
}

export interface MyTenantsResp {
  list: { tenantID: string; name: string }[]
}

export interface PersonDetailResp {
  personID: string
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
  id: string
  sessionID: string
  appID: string
  tenantID: string
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
