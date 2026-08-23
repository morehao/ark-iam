// tenantadmin 领域类型（与 backend/apps/tenantadmin/internal/dto/dtotenant 对齐）

import type { UserOrganizationItem } from './organization'
import type { AdminLevel, MenuItem } from './platform'

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

// MemberItem 成员总表条目（以人为维度，含部门关系数组:主/非主/负责）。
export interface MemberItem {
  userID: string
  tenantID: string
  username: string
  primaryEmail: string
  primaryPhone: string
  name: string
  avatar: string
  isSuspended: boolean
  roleCount: number
  createdAt?: number
  primaryOrgID: string
  organizations: UserOrganizationItem[]
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
  secondaryOrgIDs?: string[]
  leaderOrgIDs?: string[]
}

// TenantUserOrgUpdate 编辑成员时的信息更新（PATCH 局部更新；字段省略=不改变）。
export interface TenantUserOrgUpdate {
  username?: string
  primaryEmail?: string
  primaryPhone?: string
  primaryOrgID?: string
  secondaryOrgIDs?: string[]
  leaderOrgIDs?: string[]
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
  adminLevel?: AdminLevel
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
