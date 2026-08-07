import axios from 'axios'
import type { ApiResponse, OIDCLoginReq, OIDCLoginResp } from './types'

const api = axios.create({
  baseURL: '/oidc',
  timeout: 10000,
})

export async function oidcLogin(data: OIDCLoginReq): Promise<OIDCLoginResp> {
  const resp = await api.post<ApiResponse<OIDCLoginResp>>('/login', data)
  return resp.data.data
}
