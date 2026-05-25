import request from '../utils/request'
import type { ApiResponse } from '../utils/response'
import type {
  PersonInfo,
  PersonToken,
  TenantMembership,
  TenantToken,
  UserInfo,
} from '../types/auth'

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

export interface SwitchTenantResp {
  tenantToken: TenantToken
}

export interface SwitchTenantReq {
  tenantID: number
}

export interface RefreshTokenResp {
  accessToken: string
  refreshToken: string
  expiresIn: number
  tokenType: string
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
  return request.post<LoginReq, ApiResponse<LoginResp>>('/auth/login', data)
}

export const register = (data: RegisterReq) => {
  return request.post<RegisterReq, ApiResponse<RegisterResp>>('/auth/register', data)
}

export const logout = (refreshToken: string) => {
  return request.post<any, ApiResponse<null>>('/auth/logout', { refreshToken })
}

export const refreshTokenApi = (refreshToken: string) => {
  return request.post<any, ApiResponse<RefreshTokenResp>>('/auth/refreshToken', { refreshToken })
}

export const getUserinfo = () => {
  return request.get<any, ApiResponse<UserinfoResp>>('/auth/userinfo')
}

export const selectTenant = (data: SelectTenantReq) => {
  return request.post<SelectTenantReq, ApiResponse<SelectTenantResp>>('/auth/selectTenant', data)
}

export const switchTenant = (data: SwitchTenantReq) => {
  return request.post<SwitchTenantReq, ApiResponse<SwitchTenantResp>>('/auth/switchTenant', data)
}

export const getMyTenants = (params: MyTenantsReq = {}) => {
  return request.get<MyTenantsReq, ApiResponse<MyTenantsResp>>('/auth/myTenants', {
    params,
  })
}

export const getConnectorAuthorizationUrl = (data: ConnectorAuthorizationReq) => {
  return request.post<ConnectorAuthorizationReq, ApiResponse<ConnectorAuthorizationResp>>(`/connector/${data.connectorId}/authorize`, data)
}

export const completeConnectorCallback = (data: ConnectorCallbackReq) => {
  return request.get<ConnectorCallbackReq, ApiResponse<ConnectorCallbackResp>>('/connector/callback', {
    params: data,
  })
}
