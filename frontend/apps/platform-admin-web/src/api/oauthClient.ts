import request from '../utils/request'

export interface OAuthClient {
  oauthClientId: number
  appId: number
  clientID: string
  name: string
  type: string
  status: string
  isThirdParty: number
  grantTypes: string[]
  tokenEndpointAuthMethod: string
  createdAt: string
}

export interface OAuthClientDetail extends OAuthClient {
  tenantId: number
  redirectURIs: string[]
  postLogoutRedirectURIs: string[]
  responseTypes: string[]
  allowedOrigins: string[]
  requirePKCE: number
  requireAuthTime: number
  defaultScopes: string[]
  accessTokenTTL: number
  refreshTokenTTL: number
}

export interface OAuthClientPageListReq {
  page: number
  pageSize: number
  name?: string
  type?: string
  status?: string
}

export interface OAuthClientPageListResp {
  list: OAuthClient[]
  total: number
}

export interface OAuthClientCreateReq {
  appId: number
  name: string
  type?: string
  isThirdParty?: number
  redirectURIs?: string[]
  postLogoutRedirectURIs?: string[]
  grantTypes?: string[]
  responseTypes?: string[]
  tokenEndpointAuthMethod?: string
  allowedOrigins?: string[]
  requirePKCE?: number
  requireAuthTime?: number
  defaultScopes?: string[]
  accessTokenTTL?: number
  refreshTokenTTL?: number
}

export interface OAuthClientUpdateReq {
  oauthClientId: number
  name?: string
  type?: string
  status?: string
  isThirdParty?: number
  redirectURIs?: string[]
  postLogoutRedirectURIs?: string[]
  grantTypes?: string[]
  responseTypes?: string[]
  tokenEndpointAuthMethod?: string
  allowedOrigins?: string[]
  requirePKCE?: number
  requireAuthTime?: number
  defaultScopes?: string[]
  accessTokenTTL?: number
  refreshTokenTTL?: number
}

export interface SecretResp {
  id: number
  oauthClientId: number
  name: string
  valuePrefix: string
  expiresAt: string | null
  createdAt: string
}

export interface CreateSecretResp {
  id: number
  name: string
  valuePrefix: string
  secret: string
}

export const getOAuthClientPageList = (data: OAuthClientPageListReq) => {
  return request.post<OAuthClientPageListReq, OAuthClientPageListResp>('/oauthClient/pageList', data)
}

export const getOAuthClientDetail = (oauthClientId: number) => {
  return request.get<any, OAuthClientDetail>('/oauthClient/detail', { params: { oauthClientId } })
}

export const createOAuthClient = (data: OAuthClientCreateReq) => {
  return request.post<OAuthClientCreateReq, { oauthClientId: number; clientID: string }>('/oauthClient/create', data)
}

export const updateOAuthClient = (data: OAuthClientUpdateReq) => {
  return request.post<OAuthClientUpdateReq, string>('/oauthClient/update', data)
}

export const deleteOAuthClient = (oauthClientId: number) => {
  return request.post<any, string>('/oauthClient/delete', { oauthClientId })
}

export const listSecrets = (oauthClientId: number) => {
  return request.get<any, { total: number; secrets: SecretResp[] }>('/oauthClient/secrets', { params: { oauthClientId } })
}

export const createSecret = (data: { oauthClientId: number; name: string; expiresAt?: string }) => {
  return request.post<any, CreateSecretResp>('/oauthClient/secrets', data)
}

export const deleteSecret = (secretId: number) => {
  return request.delete<any, string>(`/oauthClient/secrets/${secretId}`)
}
