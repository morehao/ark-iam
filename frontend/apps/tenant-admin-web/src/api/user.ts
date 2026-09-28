import { request } from '@ark-iam/api'
import type {
  PageListResp,
  TenantUserCreateReq,
  TenantUserDetail,
  TenantUserIdentityCreateReq,
  TenantUserIdentityItem,
  TenantUserItem,
  TenantUserLoginLogItem,
  TenantUserDepartmentUpdate,
  TenantUserResetPasswordResp,
  TenantUserRoleItem,
  TenantUserCreateResp,
  TenantUserApplicationsListResp,
  TenantApplicationUsersListResp,
  UserStatus,
} from '@ark-iam/types'

/** 获取当前租户内的用户目录（分页，关键词=姓名/用户名/邮箱/手机；status=状态过滤，空串=不过滤；departmentID=仅筛选恰在该部门的用户） */
export const getTenantUserPageList = (params?: {
  page?: number
  pageSize?: number
  keyword?: string
  status?: UserStatus
  departmentID?: string
}) => request.get<any, PageListResp<TenantUserItem>>('/tenant/users', { params })

/** 创建租户用户（person 不存在则先创建；含行政主部门[primary,单] + 参与部门[secondary] + 负责部门[leader]） */
export const createTenantUser = (data: TenantUserCreateReq) => request.post<any, TenantUserCreateResp>('/tenant/users', data)

/** 用户详情（基础信息 + 部门归属 + 角色） */
export const getTenantUserDetail = (userID: string) => request.get<any, TenantUserDetail>(`/tenant/users/${userID}`)

/** 局部更新用户（姓名/头像/状态 + 主/参与/负责部门；status 省略=不变） */
export const updateTenantUser = (data: { userID: string } & TenantUserDepartmentUpdate & { name?: string; avatar?: string; status?: UserStatus }) => {
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

/**
 * 用户的应用准入清单：返回「本租户订阅应用 ∪ 该用户准入行」的全集，
 * 每行带 subscribed/granted/roleCount，供抽屉呈现选中态与撤销确认文案。
 */
export const getTenantUserApplications = (userID: string) =>
  request.get<any, TenantUserApplicationsListResp>(`/tenant/users/${userID}/applications`)

/**
 * 全量替换用户的应用准入（`appIDs` 即"最终应被准入的应用集合"）。
 * 必须显式传 `appIDs`：缺字段会被服务端拒绝（防"漏传即清空"），传 `[]` 表示清空全部准入。
 * 撤销某应用准入会在同一事务内级联撤销该应用下的角色绑定，故前端需按 roleCount 二次确认。
 */
export const updateTenantUserApplications = (userID: string, appIDs: string[]) =>
  request.put<any, string>(`/tenant/users/${userID}/applications`, { appIDs })

/** 某应用下被准入的成员列表（应用视角；撤销前的影响面核对） */
export const getTenantApplicationUsers = (appID: string) =>
  request.get<any, TenantApplicationUsersListResp>(`/tenant/applications/${appID}/users`)

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
