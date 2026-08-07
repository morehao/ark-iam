import request from '../utils/request'
import type { PersonInfo } from '@ark-iam/shared'

export interface UserinfoResp {
  personInfo: PersonInfo
  userInfo: { userID: number; tenantID: number; name: string; isOwner: number }
}

export interface MyTenantsResp {
  list: { tenantID: number; name: string }[]
}

export interface MyTenantsReq {
  personToken?: string
}

export const getUserinfo = () => {
  return request.get<any, UserinfoResp>('/auth/userinfo')
}

export const getMyTenants = (params: MyTenantsReq = {}) => {
  return request.get<MyTenantsReq, MyTenantsResp>('/auth/myTenants', { params })
}

export const logoutAPI = (refreshToken: string) => {
  return request.post<any, void>('/auth/logout', { refreshToken })
}

export const logoutAllAPI = (refreshToken: string) => {
  return request.post<any, void>('/auth/logoutAll', { refreshToken })
}
