// tenantadmin 领域类型（与 backend/apps/tenantadmin/internal/dto/dtotenant 对齐）

import type { UserDepartmentItem } from './department'
import type { MenuItem, SysAdminType } from './platform'

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
  primaryDepartmentName: string
  roleCount: number
  createdAt?: number
  updatedAt?: number
}

/**
 * 建成员入参。不含密码：初始临时密码由服务端生成、仅在创建响应返回一次，
 * 且该成员首次登录必须改密（见 docs/design/tenant-admin-provisioning-design-20260912.md D3/D7）。
 */
export interface TenantUserCreateReq {
  personID?: string
  username?: string
  primaryEmail?: string
  primaryPhone?: string
  name: string
  avatar?: string
  isSuspended?: boolean
  primaryDepartmentID: string // 行政主部门ID（primary，单值，必填：用户必须从属部门）
  secondaryDepartmentIDs?: string[]
  leaderDepartmentIDs?: string[]
}

/**
 * 建成员响应。initialPassword 为空表示邮箱/手机命中了已存在的自然人（其密码未被改动），
 * 此时不会回显任何凭据。
 */
export interface TenantUserCreateResp {
  userID: string
  initialPassword: string
}

/** 重置成员密码响应：新临时密码仅此一次返回，成员下次登录必须改密。 */
export interface TenantUserResetPasswordResp {
  userID: string
  initialPassword: string
}

// TenantUserDepartmentUpdate 编辑成员时的信息更新（PATCH 局部更新；字段省略=不改变）。
export interface TenantUserDepartmentUpdate {
  username?: string
  primaryEmail?: string
  primaryPhone?: string
  primaryDepartmentID?: string
  secondaryDepartmentIDs?: string[]
  leaderDepartmentIDs?: string[]
}

export interface TenantUserDetail extends TenantUserItem {
  departments: UserDepartmentItem[]
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

// ---------- 租户用户第三方身份（用户子资源） ----------
// 身份按自然人归属，租户维度由后端从登录上下文取得，入参不传 tenantID。
export interface TenantUserIdentityItem {
  userIdentityID: string
  issuer: string
  identityID: string
  detail?: unknown
  createdAt: number
  updatedAt: number
}

export interface TenantUserIdentityCreateReq {
  userID: string
  issuer: string
  identityID: string
  detail?: unknown
}

// ---------- 租户用户登录日志（用户子资源） ----------
export interface TenantUserLoginLogItem {
  userLoginLogID: string
  loginIP: string
  userAgent: string
  loginTime: number
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
  adminType: SysAdminType
  memberCount: number
  menuCount: number
  createdAt?: number
  updatedAt?: number
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
  primaryDepartmentID: string // 主部门ID（服务账号必有主部门）
  primaryDepartmentName: string // 主部门名称
  isSuspended: boolean
  createdAt?: number
  updatedAt?: number
}

// 服务账号部门归属条目（详情用）：primary=主部门（唯一），secondary=参与部门（可多条）。
export interface TenantMachineUserDepartment {
  departmentID: string
  departmentName: string
  relationType: 'primary' | 'secondary'
}

export interface TenantMachineUserDetail extends TenantMachineUserItem {
  departments: TenantMachineUserDepartment[]
  roles: TenantUserRoleItem[]
}

export interface TenantMachineUserCreateReq {
  name: string
  description?: string
  primaryDepartmentID: string // 主部门ID（primary，单值，必填：服务账号必须从属部门）
  secondaryDepartmentIDs?: string[] // 参与部门ID数组（secondary，可多条，可选）
}

// 全量更新服务账号；可空字段语义：
// primaryDepartmentID：不传/null=不变；传值=替换主部门（禁止传空串清空）
// secondaryDepartmentIDs：不传/null=不变；传[]=清空参与部门；传值=全量替换参与部门
export interface TenantMachineUserUpdateReq {
  machineUserID: string
  name: string
  description?: string
  primaryDepartmentID?: string | null
  secondaryDepartmentIDs?: string[] | null
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
  updatedAt: number
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
