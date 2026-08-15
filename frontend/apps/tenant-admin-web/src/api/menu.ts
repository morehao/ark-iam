import { request } from '@ark-iam/api'
import type { MenuTreeResp, TenantAppItem } from '@ark-iam/types'

/** 获取当前租户的菜单树（租户自服务控制台侧边栏动态菜单） */
export const getMyMenuTree = () => request.get<any, MenuTreeResp>('/tenant/menus/tree')

/** 获取当前租户订阅的应用列表（角色归属 / 菜单授权的应用选项） */
export const getTenantApps = () => request.get<any, { list: TenantAppItem[] }>('/tenant/apps')
