import { request } from '@ark-iam/api'
import type { MenuTreeResp } from '@ark-iam/types'

/** 获取当前租户的菜单树（租户自服务控制台侧边栏动态菜单） */
export const getMyMenuTree = () => request.get<any, MenuTreeResp>('/myMenu/tree')
