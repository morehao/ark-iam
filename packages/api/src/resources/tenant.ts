import request from '../request'
import type {
  DepartmentChildrenResp,
  DepartmentItem,
  DepartmentTreeResp,
  DepartmentUserItem,
  DeptNodeStatus,
  MenuTreeResp,
  PageListResp,
  TenantApiKeyCreateReq,
  TenantApiKeyCreateResp,
  TenantApiKeyItem,
  TenantApplicationUsersListResp,
  TenantMachineUserCreateReq,
  TenantMachineUserDetail,
  TenantMachineUserItem,
  TenantMachineUserUpdateReq,
  TenantRoleCreateReq,
  TenantRoleItem,
  TenantRoleMenuResp,
  TenantRoleUpdateReq,
  TenantSubscribedApplicationItem,
  TenantUserApplicationsListResp,
  TenantUserCreateReq,
  TenantUserCreateResp,
  TenantUserDepartmentUpdate,
  TenantUserDetail,
  TenantUserIdentityCreateReq,
  TenantUserIdentityItem,
  TenantUserItem,
  TenantUserLoginLogItem,
  TenantUserResetPasswordResp,
  TenantUserRoleItem,
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

/** 租户角色分页列表（`unassigned=true` 只查未归属应用的系统角色） */
export const getTenantRolePageList = (params?: {
  page?: number
  pageSize?: number
  appID?: string
  keyword?: string
  unassigned?: boolean
}) => request.get<any, PageListResp<TenantRoleItem>>('/tenant/roles', { params })

/** 创建租户角色 */
export const createTenantRole = (data: TenantRoleCreateReq) => request.post<any, { roleID: string }>('/tenant/roles', data)

/** 更新租户角色（全量更新，code 必填：它是对下游系统的授权契约值） */
export const updateTenantRole = (data: TenantRoleUpdateReq) => {
  const { roleID, ...body } = data
  return request.put<any, string>(`/tenant/roles/${roleID}`, body)
}

/** 删除租户角色（级联清理成员/菜单关联） */
export const deleteTenantRole = (roleID: string) => request.delete<any, string>(`/tenant/roles/${roleID}`)

/** 角色菜单授权回显（菜单树 + 已授权ID） */
export const getTenantRoleMenus = (roleID: string) => request.get<any, TenantRoleMenuResp>(`/tenant/roles/${roleID}/menus`)

/** 全量替换角色菜单授权 */
export const updateTenantRoleMenus = (roleID: string, menuIDs: string[]) =>
  request.put<any, string>(`/tenant/roles/${roleID}/menus`, { menuIDs })

// ---------- 部门树 ----------
export const getDepartmentTree = (params?: { name?: string; status?: DeptNodeStatus }) =>
  request.get<any, DepartmentTreeResp>('/tenant/departments/tree', { params })
export const getDepartmentChildren = (
  departmentID: string,
  params?: { page?: number; pageSize?: number; name?: string; status?: DeptNodeStatus },
) => request.get<any, DepartmentChildrenResp>(`/tenant/departments/${departmentID}/children`, { params })
export const createDepartment = (data: { parentID?: string; name: string; sort?: number; status?: DeptNodeStatus }) =>
  request.post<any, { departmentID: string }>('/tenant/departments', data)
export const updateDepartment = (data: { departmentID: string; parentID?: string; name?: string; sort?: number; status?: DeptNodeStatus }) => {
  const { departmentID, ...body } = data
  return request.put<any, string>(`/tenant/departments/${departmentID}`, body)
}
export const updateDepartmentStatus = (departmentID: string, status: DeptNodeStatus) =>
  request.patch<any, string>(`/tenant/departments/${departmentID}`, { status })
export const deleteDepartment = (id: string, cascade = false) =>
  request.delete<any, string>(`/tenant/departments/${id}`, { params: { cascade } })

// ---------- 部门关系 ----------
export const getDepartmentUserPage = (departmentID: string, params?: { page?: number; pageSize?: number; relationType?: string; keyword?: string }) =>
  request.get<any, PageListResp<DepartmentUserItem>>(`/tenant/departments/${departmentID}/users`, { params })
export const createDepartmentUser = (departmentID: string, data: { userID: string; relationType?: string }) =>
  request.post<any, void>(`/tenant/departments/${departmentID}/users`, data)
export const updateDepartmentUser = (departmentID: string, userID: string, data: { relationType?: string }) =>
  request.put<any, string>(`/tenant/departments/${departmentID}/users/${userID}`, data)
export const deleteDepartmentUser = (departmentID: string, userID: string) =>
  request.delete<any, string>(`/tenant/departments/${departmentID}/users/${userID}`)

export type { DepartmentItem }

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

/** 服务账号分页列表（租户内机器主体，user_type=machine；name=名称模糊，status=状态过滤（空串=不过滤）；条目含主部门 primaryDepartmentID/primaryDepartmentName） */
export const getMachineUserPageList = (params?: {
  page?: number
  pageSize?: number
  name?: string
  status?: UserStatus
}) => request.get<any, PageListResp<TenantMachineUserItem>>('/tenant/machine-users', { params })

/** 创建服务账号（需系统管理能力 super）：primaryDepartmentID=主部门ID(primary,单值,必填)，secondaryDepartmentIDs=参与部门(secondary,可多条,可选) */
export const createMachineUser = (data: TenantMachineUserCreateReq) =>
  request.post<any, { machineUserID: string }>('/tenant/machine-users', data)

/** 服务账号详情（基础信息含主部门 + 部门归属 departments + 已分配角色） */
export const getMachineUserDetail = (machineUserID: string) =>
  request.get<any, TenantMachineUserDetail>(`/tenant/machine-users/${machineUserID}`)

/**
 * 全量更新服务账号（需系统管理能力 super，name 必填）。
 * primaryDepartmentID/secondaryDepartmentIDs 可空字段语义：不传=不变；传 primaryDepartmentID=替换主部门；
 * 传 secondaryDepartmentIDs=全量替换参与部门（[] 表示清空）。
 */
export const updateMachineUser = (data: TenantMachineUserUpdateReq) => {
  const { machineUserID, ...body } = data
  return request.put<any, string>(`/tenant/machine-users/${machineUserID}`, body)
}

/** 局部更新服务账号状态（status：active=启用 / suspended=挂起） */
export const updateMachineUserStatus = (machineUserID: string, status: UserStatus) =>
  request.patch<any, string>(`/tenant/machine-users/${machineUserID}`, { status })

/** 删除服务账号（须先删除其全部 API 密钥） */
export const deleteMachineUser = (machineUserID: string) =>
  request.delete<any, string>(`/tenant/machine-users/${machineUserID}`)

/** 服务账号已分配角色 */
export const getMachineUserRoles = (machineUserID: string) =>
  request.get<any, { list: TenantUserRoleItem[] }>(`/tenant/machine-users/${machineUserID}/roles`)

/** 按应用全量替换服务账号角色（appID 空串=系统/未归属应用组；仅替换该应用下授权，不影响其它应用；禁授管理员角色） */
export const updateMachineUserRoles = (machineUserID: string, appID: string, roleIDs: string[]) =>
  request.put<any, string>(`/tenant/machine-users/${machineUserID}/roles`, { appID, roleIDs })

/** 获取当前租户的菜单树（租户自服务控制台侧边栏动态菜单） */
export const getTenantMenuTree = () => request.get<any, MenuTreeResp>('/tenant/menus/tree')

/** 获取当前租户订阅的应用列表（角色归属 / 菜单授权 / 应用准入的应用选项） */
export const getTenantApps = () => request.get<any, { list: TenantSubscribedApplicationItem[] }>('/tenant/applications')