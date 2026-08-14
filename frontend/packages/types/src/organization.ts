export interface OrganizationItem {
  organizationID: number
  tenantID: number
  name: string
  description: string
  isMFARequired: number
}

export interface OrganizationRoleItem {
  organizationRoleID: number
  tenantID: number
  organizationID: number
  name: string
  description: string
  type: string
}

export interface OrganizationUserItem {
  organizationID: number
  userID: number
  tenantID: number
}

export interface OrganizationRoleUserItem {
  organizationID: number
  organizationRoleID: number
  userID: number
  tenantID: number
}

export interface PageListResp<T> {
  list: T[]
  total: number
}
