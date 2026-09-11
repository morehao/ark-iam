import { request } from '@ark-iam/api'
import type {
  PageListResp,
  TenantUserCreateReq,
  TenantUserDetail,
  TenantUserIdentityCreateReq,
  TenantUserIdentityItem,
  TenantUserItem,
  TenantUserLoginLogItem,
  TenantUserOrgUpdate,
  TenantUserResetPasswordResp,
  TenantUserRoleItem,
  TenantUserCreateResp,
} from '@ark-iam/types'

/** 获取当前租户内的用户目录（分页，关键词=姓名/用户名/邮箱/手机；organizationID=仅筛选恰在该组织的用户） */
export const getTenantUserPageList = (params?: {
  page?: number
  pageSize?: number
  keyword?: string
  isSuspended?: boolean
  organizationID?: string
}) => request.get<any, PageListResp<TenantUserItem>>('/tenant/users', { params })

/** 创建租户用户（person 不存在则先创建；含行政主组织[primary,单] + 参与组织[secondary] + 负责组织[leader]） */
export const createTenantUser = (data: TenantUserCreateReq) => request.post<any, TenantUserCreateResp>('/tenant/users', data)

/** 用户详情（基础信息 + 组织归属 + 角色） */
export const getTenantUserDetail = (userID: string) => request.get<any, TenantUserDetail>(`/tenant/users/${userID}`)

/** 局部更新用户（姓名/头像/状态 + 主/参与/负责组织） */
export const updateTenantUser = (data: { userID: string } & TenantUserOrgUpdate & { name?: string; avatar?: string; isSuspended?: boolean }) => {
  const { userID, ...body } = data
  return request.patch<any, string>(`/tenant/users/${userID}`, body)
}

/** 重置成员密码：新临时密码由服务端生成并仅此一次返回，成员下次登录必须改密 */
export const resetTenantUserPassword = (userID: string) =>
  request.post<any, TenantUserResetPasswordResp>(`/tenant/users/${userID}/reset-password`)

/** 用户已分配角色 */
export const getTenantUserRoles = (userID: string) => request.get<any, { list: TenantUserRoleItem[] }>(`/tenant/users/${userID}/roles`)

/** 按应用全量替换用户角色（appID 空串=系统/未归属应用组；仅替换该应用下授权，不影响其它应用） */
export const updateTenantUserRoles = (userID: string, appID: string, roleIDs: string[]) =>
  request.put<any, string>(`/tenant/users/${userID}/roles`, { appID, roleIDs })

/** 用户已绑定的第三方身份 */
export const getTenantUserIdentities = (userID: string) =>
  request.get<any, PageListResp<TenantUserIdentityItem>>(`/tenant/users/${userID}/identities`)

/** 为用户绑定第三方身份（租户取自登录上下文，不传 tenantID） */
export const createTenantUserIdentity = (data: TenantUserIdentityCreateReq) => {
  const { userID, ...body } = data
  return request.post<any, { userIdentityID: string }>(`/tenant/users/${userID}/identities`, body)
}

/** 解绑用户的第三方身份 */
export const deleteTenantUserIdentity = (userID: string, userIdentityID: string) =>
  request.delete<any, string>(`/tenant/users/${userID}/identities/${userIdentityID}`)

/** 用户登录日志 */
export const getTenantUserLoginLogs = (userID: string) =>
  request.get<any, PageListResp<TenantUserLoginLogItem>>(`/tenant/users/${userID}/login-logs`)
