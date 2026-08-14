import axios from 'axios'
import type { ApiResponse, OIDCLoginReq, OIDCLoginResp, OIDCSelectTenantReq } from './types'

const api = axios.create({
  baseURL: '/oidc',
  timeout: 10000,
})

// 后端统一响应信封：业务失败也返回 HTTP 200，需按 code 判断
export async function oidcLogin(data: OIDCLoginReq): Promise<OIDCLoginResp> {
  const resp = await api.post<ApiResponse<OIDCLoginResp>>('/login', data)
  const body = resp.data
  if (body.code !== 0) {
    throw new Error(body.msg || '登录失败，请重试')
  }
  return body.data
}

export async function oidcSelectTenant(data: OIDCSelectTenantReq): Promise<OIDCLoginResp> {
  const resp = await api.post<ApiResponse<OIDCLoginResp>>('/login/selectTenant', data)
  const body = resp.data
  if (body.code !== 0) {
    throw new Error(body.msg || '选择租户失败，请重试')
  }
  return body.data
}
