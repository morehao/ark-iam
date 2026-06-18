import axios from 'axios'
import type { ApiResponse, OIDCLoginReq, OIDCLoginResp } from './types'

const api = axios.create({
  baseURL: '/v1/auth',
  timeout: 10000,
})

export async function oidcLogin(data: OIDCLoginReq): Promise<OIDCLoginResp> {
  const resp = await api.post<ApiResponse<OIDCLoginResp>>('/oidc/login', data)
  return resp.data.data
}
