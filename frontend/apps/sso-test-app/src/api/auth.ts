import request from '../utils/request'
import type { PersonInfo } from '@ark-iam/shared'

export interface UserinfoResp {
  personInfo: PersonInfo
  userInfo: { userID: number; tenantID: number; name: string; isOwner: number }
}

export const getUserinfo = () => {
  return request.get<any, UserinfoResp>('/auth/userinfo')
}

export const logoutAPI = (refreshToken: string) => {
  return request.post<any, void>('/auth/logout', { refreshToken })
}
