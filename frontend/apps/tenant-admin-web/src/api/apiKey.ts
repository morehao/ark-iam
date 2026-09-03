import { request } from '@ark-iam/api'
import type { PageListResp, TenantApiKeyCreateReq, TenantApiKeyCreateResp, TenantApiKeyItem } from '@ark-iam/types'

/**
 * API 密钥分页列表（租户端，需系统管理能力 super）。
 * machineUserID 可选：传=该服务账号密钥；不传/空=租户全部密钥（「个人密钥/我的密钥」已下线，
 * 不传 machineUserID 不再代表当前用户本人密钥）。
 */
export const getApiKeyPageList = (params?: {
  page?: number
  pageSize?: number
  name?: string
  machineUserID?: string
}) => request.get<any, PageListResp<TenantApiKeyItem>>('/tenant/api-keys', { params })

/** 创建 API 密钥（machineUserID 必填=归属服务账号；expiredAt=unix秒，缺省/0=永不过期；需系统管理能力 super） */
export const createApiKey = (data: TenantApiKeyCreateReq) =>
  request.post<any, TenantApiKeyCreateResp>('/tenant/api-keys', data)

/** 吊销 API 密钥（需系统管理能力 super） */
export const revokeApiKey = (apiKeyID: string) => request.post<any, string>(`/tenant/api-keys/${apiKeyID}/revoke`)

/** 删除 API 密钥（需系统管理能力 super） */
export const deleteApiKey = (apiKeyID: string) => request.delete<any, string>(`/tenant/api-keys/${apiKeyID}`)
