import { request } from '@ark-iam/api'
import type {
  PageListResp,
  TenantMachineUserCreateReq,
  TenantMachineUserDetail,
  TenantMachineUserItem,
  TenantMachineUserUpdateReq,
  TenantUserRoleItem,
} from '@ark-iam/types'

/** 服务账号分页列表（租户内机器主体，user_type=machine；name=名称模糊，isSuspended=挂起过滤） */
export const getMachineUserPageList = (params?: {
  page?: number
  pageSize?: number
  name?: string
  isSuspended?: boolean
}) => request.get<any, PageListResp<TenantMachineUserItem>>('/tenant/machine-users', { params })

/** 创建服务账号（需系统管理能力 super） */
export const createMachineUser = (data: TenantMachineUserCreateReq) =>
  request.post<any, { machineUserID: string }>('/tenant/machine-users', data)

/** 服务账号详情（基础信息 + 已分配角色） */
export const getMachineUserDetail = (machineUserID: string) =>
  request.get<any, TenantMachineUserDetail>(`/tenant/machine-users/${machineUserID}`)

/** 全量更新服务账号（name/description） */
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

/** 全量替换服务账号角色（禁止授予系统管理角色） */
export const updateMachineUserRoles = (machineUserID: string, roleIDs: string[]) =>
  request.put<any, string>(`/tenant/machine-users/${machineUserID}/roles`, { roleIDs })
