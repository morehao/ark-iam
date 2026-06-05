import axios from 'axios'
import type { OIDCLoginReq, OIDCLoginResp } from './types'

const api = axios.create({
  baseURL: '/v1/iam',
  timeout: 10000,
})

export async function oidcLogin(data: OIDCLoginReq): Promise<OIDCLoginResp> {
  const resp = await api.post<OIDCLoginResp>('/oidc/login', data)
  return resp.data
}
