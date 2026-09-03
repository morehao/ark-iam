import { request } from '@ark-iam/api'
import type { PageListResp, TenantApiKeyCreateResp, TenantApiKeyItem } from '@ark-iam/types'

/**
 * API 密钥分页列表（租户端）。
 * 语义：不给 machineUserID、不给 all → 当前真实用户本人密钥；
 * 给 machineUserID → 该服务账号密钥（需系统管理能力 super）；all=true → 租户全部（需 super）。
 */
export const getApiKeyPageList = (params?: {
  page?: number
  pageSize?: number
  name?: string
  machineUserID?: string
  all?: boolean
}) => request.get<any, PageListResp<TenantApiKeyItem>>('/tenant/api-keys', { params })

/** 创建 API 密钥：machineUserID 空=代表当前真实用户本人；expiredAt(unix秒,0=永不过期) */
export const createApiKey = (data: { name: string; machineUserID?: string; expiredAt?: number }) =>
  request.post<any, TenantApiKeyCreateResp>('/tenant/api-keys', data)

/** 吊销 API 密钥 */
export const revokeApiKey = (apiKeyID: string) => request.post<any, string>(`/tenant/api-keys/${apiKeyID}/revoke`)

/** 删除 API 密钥 */
export const deleteApiKey = (apiKeyID: string) => request.delete<any, string>(`/tenant/api-keys/${apiKeyID}`)
