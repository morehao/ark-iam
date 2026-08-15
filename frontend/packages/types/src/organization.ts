export interface OrganizationItem {
  organizationID: string
  tenantID: string
  name: string
  description: string
  isMFARequired: number
}

export interface OrganizationRoleItem {
  organizationRoleID: string
  tenantID: string
  organizationID: string
  name: string
  description: string
  type: string
}

export interface OrganizationUserItem {
  organizationID: string
  userID: string
  tenantID: string
}

export interface OrganizationRoleUserItem {
  organizationID: string
  organizationRoleID: string
  userID: string
  tenantID: string
}

export interface PageListResp<T> {
  list: T[]
  total: number
}
