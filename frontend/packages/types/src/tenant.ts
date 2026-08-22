// tenantadmin 领域类型（与 backend/apps/tenantadmin/internal/dto/dtotenant 对齐）

import type { UserOrganizationItem } from './organization'
import type { MenuItem } from './platform'

// ---------- 租户用户 ----------
export interface TenantUserItem {
  userID: string
  tenantID: string
  username: string
  primaryEmail: string
  primaryPhone: string
  name: string
  avatar: string
  isSuspended: boolean
  primaryOrgName: string
  roleCount: number
  createdAt?: number
}

export interface TenantUserCreateReq {
  personID?: string
  username?: string
  primaryEmail?: string
  primaryPhone?: string
  name: string
  avatar?: string
  password?: string
  isSuspended?: boolean
  organizationIDs: string[]
}

export interface TenantUserDetail extends TenantUserItem {
  organizations: UserOrganizationItem[]
  roles: TenantUserRoleItem[]
}

export interface TenantUserRoleItem {
  roleID: string
  appID: string
  appName: string
  name: string
  code: string
  description: string
}

// ---------- 租户订阅应用 ----------
export interface TenantAppItem {
  appID: string
  code: string
  name: string
}

// ---------- 租户角色 ----------
export interface TenantRoleItem {
  roleID: string
  appID: string
  appName: string
  name: string
  code: string
  description: string
  source?: 'builtin' | 'custom' | string
  adminLevel?: 'none' | 'basic' | 'super' | string
  memberCount: number
  menuCount: number
  createdAt?: number
}

export interface TenantRoleCreateReq {
  appID: string
  name: string
  code: string
  description?: string
}

export interface TenantRoleMenuResp {
  list: MenuItem[]
  menuIDs: string[]
}
