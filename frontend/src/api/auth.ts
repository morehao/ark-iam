import request from '../utils/request'

export interface LoginReq {
  identifier: string
  password: string
}

export interface LoginResp {
  personToken: {
    accessToken: string
    refreshToken: string
    expiresIn: number
    tokenType: string
  }
  tenants: Array<{
    tenantID: number
    name: string
    tag: string
    userID: number
    isOwner: number
  }>
}

export interface RegisterReq {
  username: string
  password: string
  name: string
  primaryEmail?: string
  primaryPhone?: string
  tenantID: number
}

export interface RegisterResp {
  userID: number
}

export interface UserinfoResp {
  personInfo: {
    personID: number
    name: string
    avatar: string
  }
  userInfo: {
    userID: number
    name: string
    tenantID: number
    isOwner: number
  }
}

export const login = (data: LoginReq) => {
  return request.post<any, any>('/auth/login', data)
}

export const register = (data: RegisterReq) => {
  return request.post<any, any>('/auth/register', data)
}

export const logout = (refreshToken: string) => {
  return request.post<any, any>('/auth/logout', { refreshToken })
}

export const refreshToken = (refreshToken: string) => {
  return request.post<any, any>('/auth/refreshToken', { refreshToken })
}

export const getUserinfo = () => {
  return request.get<any, any>('/auth/userinfo')
}