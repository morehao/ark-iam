import { request } from '@ark-iam/api'
import type {
  PageListResp,
  TenantUserCreateReq,
  TenantUserDetail,
  TenantUserItem,
  TenantUserRoleItem,
} from '@ark-iam/types'

/** 获取当前租户内的用户目录（分页，关键词=姓名/用户名/邮箱/手机） */
export const getTenantUserPageList = (params?: { page?: number; pageSize?: number; keyword?: string; isSuspended?: boolean }) =>
  request.get<any, PageListResp<TenantUserItem>>('/tenant/users', { params })

/** 创建租户用户（person 不存在则先创建） */
export const createTenantUser = (data: TenantUserCreateReq) => request.post<any, { userID: string }>('/tenant/users', data)

/** 用户详情（基础信息 + 组织归属 + 角色） */
export const getTenantUserDetail = (userID: string) => request.get<any, TenantUserDetail>(`/tenant/users/${userID}`)

/** 局部更新用户（姓名/头像/状态） */
export const updateTenantUser = (data: { userID: string; name?: string; avatar?: string; isSuspended?: boolean }) => {
  const { userID, ...body } = data
  return request.patch<any, string>(`/tenant/users/${userID}`, body)
}

/** 重置密码 */
export const resetTenantUserPassword = (userID: string, password: string) =>
  request.post<any, string>(`/tenant/users/${userID}/reset-password`, { password })

/** 用户已分配角色 */
export const getTenantUserRoles = (userID: string) => request.get<any, { list: TenantUserRoleItem[] }>(`/tenant/users/${userID}/roles`)

/** 全量替换用户角色 */
export const updateTenantUserRoles = (userID: string, roleIDs: string[]) =>
  request.put<any, string>(`/tenant/users/${userID}/roles`, { roleIDs })
