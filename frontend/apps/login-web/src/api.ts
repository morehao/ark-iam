import axios from 'axios'
import type { ApiResponse, OIDCLoginReq, OIDCLoginResp, OIDCSelectTenantReq } from './types'

const api = axios.create({
  baseURL: '/oidc',
  timeout: 10000,
})

export async function oidcLogin(data: OIDCLoginReq): Promise<OIDCLoginResp> {
  const resp = await api.post<ApiResponse<OIDCLoginResp>>('/login', data)
  return resp.data.data
}

export async function oidcSelectTenant(data: OIDCSelectTenantReq): Promise<OIDCLoginResp> {
  const resp = await api.post<ApiResponse<OIDCLoginResp>>('/login/selectTenant', data)
  return resp.data.data
}
