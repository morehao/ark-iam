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
// 服务账号与真实用户一致：必须从属主部门（primary，唯一）；可有多条参与部门（secondary）。
export interface TenantMachineUserItem {
  machineUserID: string
  tenantID: string
  name: string
  description: string
  primaryOrgID: string // 主部门ID（服务账号必有主部门）
  primaryOrgName: string // 主部门名称
  isSuspended: boolean
  createdAt?: number
}

// 服务账号组织归属条目（详情用）：primary=主部门（唯一），secondary=参与部门（可多条）。
export interface TenantMachineUserOrganization {
  organizationID: string
  organizationName: string
  relationType: 'primary' | 'secondary'
}

export interface TenantMachineUserDetail extends TenantMachineUserItem {
  organizations: TenantMachineUserOrganization[]
  roles: TenantUserRoleItem[]
}

export interface TenantMachineUserCreateReq {
  name: string
  description?: string
  organizationIDs: string[] // 主部门ID数组（primary，仅允许1个，必填）
  secondaryOrgIDs?: string[] // 参与部门ID数组（secondary，可多条，可选）
}

// 全量更新服务账号；可空字段语义：
// primaryOrgID：不传/null=不变；传值=替换主部门（禁止传空串清空）
// secondaryOrgIDs：不传/null=不变；传[]=清空参与部门；传值=全量替换参与部门
export interface TenantMachineUserUpdateReq {
  machineUserID: string
  name: string
  description?: string
  primaryOrgID?: string | null
  secondaryOrgIDs?: string[] | null
}

// ---------- 租户 API 密钥 ----------
// API 密钥一律归属服务账号（machine，供服务端集成）；「个人密钥（代表真人本人）」已下线。
// OwnerType 保留 member 仅为兼容历史 member 归属数据，UI 不再出现个人密钥。
export type TenantApiKeyOwnerType = 'member' | 'machine'

export interface TenantApiKeyItem {
  keyID: string
  name: string
  keyPrefix: string
  ownerUserID: string // 归属服务账号ID
  ownerType: TenantApiKeyOwnerType
  ownerName: string // 归属服务账号名称
  createdBy: string
  creatorName: string // 创建人名称
  expiredAt: number | null
  lastUsedAt: number | null
  revokedAt: number | null
  createdAt: number
}

export interface TenantApiKeyCreateReq {
  name: string
  machineUserID: string // 归属服务账号ID（必填）
  expiredAt?: number // 过期时间(unix秒；缺省/0=永不过期)
}

export interface TenantApiKeyCreateResp {
  id: string
  name: string
  key: string
  keyPrefix: string
  expiredAt: number
}
