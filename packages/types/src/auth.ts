import type { OwnerType } from './tenant'

/**
 * 自然人可用性（后端具名类型 model.PersonStatus）：active-正常 / suspended-已挂起
 * （suspended 禁止登录与签发令牌，语义对齐 tenant.status 的挂起）。
 */
export type PersonStatus = 'active' | 'suspended'

/** 自然人密码状态（后端具名类型 model.PasswordStatus）：normal-正常 / must_change-必须改密。 */
export type PasswordStatus = 'normal' | 'must_change'

export interface PersonInfo {
  personID: string
  name: string
  avatar: string
}

export interface UserinfoResp {
  personInfo: PersonInfo
  userInfo: { userID: string; tenantID: string; name: string; ownerType: OwnerType }
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
  status: PersonStatus
  passwordStatus: PasswordStatus
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
  expiresAt?: number | null
  createdAt: number
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
