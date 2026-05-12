import request from '../utils/request'
import type {
  PersonInfo,
  PersonToken,
  TenantMembership,
  TenantToken,
  UserInfo,
} from '../types/auth'

interface ApiResponse<T> {
  code: number
  msg: string
  data: T
}

export interface LoginReq {
  identifier: string
  password: string
}

export interface LoginResp {
  personToken: PersonToken
  tenants: TenantMembership[]
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
  personInfo: PersonInfo
  userInfo: UserInfo
}

export interface SelectTenantReq {
  tenantID: number
  personToken?: string
}

export interface SelectTenantResp {
  tenantToken: TenantToken
}

export interface SwitchTenantReq {
  tenantID: number
}

export interface SwitchTenantResp {
  tenantToken: TenantToken
}

export interface MyTenantsResp {
  list: TenantMembership[]
}

export interface MyTenantsReq {
  personToken?: string
}

export interface ConnectorAuthorizationReq {
  connectorId: number
  redirectUri: string
  state?: string
  loginHint?: string
  responseMode?: string
}

export interface ConnectorAuthorizationResp {
  authorizationUrl: string
}

export interface ConnectorCallbackReq {
  connectorId?: number
  code: string
  state: string
}

export interface ConnectorCallbackResp {
  personToken: PersonToken
  tenants: TenantMembership[]
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

export const selectTenant = (data: SelectTenantReq) => {
  return request.post<any, any>('/auth/selectTenant', data)
}

export const switchTenant = (data: SwitchTenantReq) => {
  return request.post<any, any>('/auth/switchTenant', data)
}

export const getMyTenants = (params: MyTenantsReq = {}) => {
  return request.get<any, ApiResponse<MyTenantsResp>>('/auth/myTenants', {
    params,
  })
}

export const getConnectorAuthorizationUrl = (data: ConnectorAuthorizationReq) => {
  return request.post<any, ApiResponse<ConnectorAuthorizationResp>>(`/connector/${data.connectorId}/authorize`, data)
}

export const completeConnectorCallback = (data: ConnectorCallbackReq) => {
  return request.get<any, ApiResponse<ConnectorCallbackResp>>('/connector/callback', {
    params: data,
  })
}
