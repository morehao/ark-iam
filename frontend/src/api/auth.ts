import request from '../utils/request'
import type {
  PersonInfo,
  PersonToken,
  TenantMembership,
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
  accessToken: string
  refreshToken: string
  expiresIn: number
  tokenType: string
}

export interface SwitchTenantResp {
  accessToken: string
  refreshToken: string
  expiresIn: number
  tokenType: string
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
  return request.post<LoginReq, LoginResp>('/auth/login', data)
}

export const register = (data: RegisterReq) => {
  return request.post<RegisterReq, RegisterResp>('/auth/register', data)
}

export const logout = (refreshToken: string) => {
  return request.post<any, null>('/auth/logout', { refreshToken })
}

export const refreshTokenApi = (refreshToken: string) => {
  return request.post<any, RefreshTokenResp>('/auth/refreshToken', { refreshToken })
}

export const getUserinfo = () => {
  return request.get<any, UserinfoResp>('/auth/userinfo')
}

export const selectTenant = (data: SelectTenantReq) => {
  return request.post<SelectTenantReq, SelectTenantResp>('/auth/selectTenant', data)
}

export const switchTenant = (data: SwitchTenantReq) => {
  return request.post<SwitchTenantReq, SwitchTenantResp>('/auth/switchTenant', data)
}

export const getMyTenants = (params: MyTenantsReq = {}) => {
  return request.get<MyTenantsReq, MyTenantsResp>('/auth/myTenants', {
    params,
  })
}

export const getConnectorAuthorizationUrl = (data: ConnectorAuthorizationReq) => {
  return request.post<ConnectorAuthorizationReq, ConnectorAuthorizationResp>(`/connector/${data.connectorId}/authorize`, data)
}

export const completeConnectorCallback = (data: ConnectorCallbackReq) => {
  return request.get<ConnectorCallbackReq, ConnectorCallbackResp>('/connector/callback', {
    params: data,
  })
}

export interface OIDCLoginReq {
  authRequestID: string
  identifier: string
  password: string
}

export interface OIDCLoginResp {
  continueURL: string
}

export const oidcLogin = (data: OIDCLoginReq) => {
  return request.post<OIDCLoginReq, OIDCLoginResp>('/oidc/login', data)
}
