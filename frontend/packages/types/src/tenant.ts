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

// ---------- 服务账号（租户内机器主体，user_type=machine） ----------
export interface TenantMachineUserItem {
  machineUserID: string
  tenantID: string
  name: string
  description: string
  isSuspended: boolean
  createdAt?: number
}

export interface TenantMachineUserDetail extends TenantMachineUserItem {
  roles: TenantUserRoleItem[]
}

export interface TenantMachineUserCreateReq {
  name: string
  description?: string
}

export interface TenantMachineUserUpdateReq {
  machineUserID: string
  name: string
  description?: string
}

// ---------- 租户 API 密钥 ----------
// 密钥归属两类主体：真实用户本人（member）或服务账号（machine，开发者模式）。
export type TenantApiKeyOwnerType = 'member' | 'machine'

export interface TenantApiKeyItem {
  keyID: string
  name: string
  keyPrefix: string
  ownerUserID: string
  ownerType: TenantApiKeyOwnerType
  ownerName: string
  createdBy: string
  creatorName: string
  expiredAt: number | null
  lastUsedAt: number | null
  revokedAt: number | null
  createdAt: number
}

export interface TenantApiKeyCreateResp {
  id: string
  name: string
  key: string
  keyPrefix: string
  expiredAt: number
}
