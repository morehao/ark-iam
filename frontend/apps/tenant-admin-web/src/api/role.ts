import { request } from '@ark-iam/api'
import type { PageListResp, TenantRoleCreateReq, TenantRoleItem, TenantRoleMenuResp } from '@ark-iam/types'

/** 租户角色分页列表 */
export const getTenantRolePageList = (params?: { page?: number; pageSize?: number; appID?: string; keyword?: string }) =>
  request.get<any, PageListResp<TenantRoleItem>>('/tenant/roles', { params })

/** 创建租户角色 */
export const createTenantRole = (data: TenantRoleCreateReq) => request.post<any, { roleID: string }>('/tenant/roles', data)

/** 更新租户角色 */
export const updateTenantRole = (data: { roleID: string; name?: string; code?: string; description?: string; type?: string }) => {
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
