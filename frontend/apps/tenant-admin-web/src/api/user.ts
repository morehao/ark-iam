import { request } from '@ark-iam/api'
import type { PageListResp, UserItem } from '@ark-iam/types'

/** 获取当前租户内的用户列表（租户自服务端用户目录，替代平台端 /platform/users） */
export const getTenantUserPageList = (params?: { page?: number; pageSize?: number }) =>
  request.get<any, PageListResp<UserItem>>('/tenant/users', { params })
