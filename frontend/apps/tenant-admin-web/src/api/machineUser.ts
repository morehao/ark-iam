import { request } from '@ark-iam/api'
import type {
  PageListResp,
  TenantMachineUserCreateReq,
  TenantMachineUserDetail,
  TenantMachineUserItem,
  TenantMachineUserUpdateReq,
  TenantUserRoleItem,
} from '@ark-iam/types'

/** 服务账号分页列表（租户内机器主体，user_type=machine；name=名称模糊，isSuspended=挂起过滤；条目含主部门 primaryOrgID/primaryOrgName） */
export const getMachineUserPageList = (params?: {
  page?: number
  pageSize?: number
  name?: string
  isSuspended?: boolean
}) => request.get<any, PageListResp<TenantMachineUserItem>>('/tenant/machine-users', { params })

/** 创建服务账号（需系统管理能力 super）：organizationIDs=主部门ID数组(primary,仅1个,必填)，secondaryOrgIDs=参与部门(secondary,可多条,可选) */
export const createMachineUser = (data: TenantMachineUserCreateReq) =>
  request.post<any, { machineUserID: string }>('/tenant/machine-users', data)

/** 服务账号详情（基础信息含主部门 + 组织归属 organizations + 已分配角色） */
export const getMachineUserDetail = (machineUserID: string) =>
  request.get<any, TenantMachineUserDetail>(`/tenant/machine-users/${machineUserID}`)

/**
 * 全量更新服务账号（需系统管理能力 super，name 必填）。
 * primaryOrgID/secondaryOrgIDs 可空字段语义：不传=不变；传 primaryOrgID=替换主部门；
 * 传 secondaryOrgIDs=全量替换参与部门（[] 表示清空）。
 */
export const updateMachineUser = (data: TenantMachineUserUpdateReq) => {
  const { machineUserID, ...body } = data
  return request.put<any, string>(`/tenant/machine-users/${machineUserID}`, body)
}

/** 局部更新服务账号状态（isSuspended：挂起/启用） */
export const updateMachineUserStatus = (machineUserID: string, isSuspended: boolean) =>
  request.patch<any, string>(`/tenant/machine-users/${machineUserID}`, { isSuspended })

/** 删除服务账号（须先删除其全部 API 密钥） */
export const deleteMachineUser = (machineUserID: string) =>
  request.delete<any, string>(`/tenant/machine-users/${machineUserID}`)

/** 服务账号已分配角色 */
export const getMachineUserRoles = (machineUserID: string) =>
  request.get<any, { list: TenantUserRoleItem[] }>(`/tenant/machine-users/${machineUserID}/roles`)

/** 按应用全量替换服务账号角色（appID 空串=系统/未归属应用组；仅替换该应用下授权，不影响其它应用；禁授系统管理角色） */
export const updateMachineUserRoles = (machineUserID: string, appID: string, roleIDs: string[]) =>
  request.put<any, string>(`/tenant/machine-users/${machineUserID}/roles`, { appID, roleIDs })
