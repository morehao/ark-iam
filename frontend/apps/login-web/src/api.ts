import axios from 'axios'
import type {
  ApiResponse,
  OIDCLoginReq,
  OIDCLoginResp,
  OIDCSelectTenantReq,
  RegisterOrgReq,
  RegisterOrgResp,
  JoinTenantReq,
  JoinTenantResp,
} from './types'

const api = axios.create({
  baseURL: '/oidc',
  timeout: 10000,
})

// /v1/* 业务端点（经 gateway:8100 代理），后接 auth 的 /v1/auth/*
const authApi = axios.create({
  baseURL: '/v1/auth',
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

// 通道 A：自助开通租户（注册人自任租户 owner）
export async function registerOrg(data: RegisterOrgReq): Promise<RegisterOrgResp> {
  const resp = await authApi.post<ApiResponse<RegisterOrgResp>>('/register', data)
  const body = resp.data
  if (body.code !== 0) {
    throw new Error(body.msg || '注册失败，请重试')
  }
  return body.data
}

// 通道 B：凭邀请加入已有租户（普通成员，非 owner）
export async function joinWithInvite(data: JoinTenantReq): Promise<JoinTenantResp> {
  const resp = await authApi.post<ApiResponse<JoinTenantResp>>('/joinTenant', data)
  const body = resp.data
  if (body.code !== 0) {
    throw new Error(body.msg || '加入租户失败，请重试')
  }
  return body.data
}
