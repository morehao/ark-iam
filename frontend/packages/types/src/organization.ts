export interface OrganizationItem {
  organizationID: string
  parentID: string
  orgPath: string
  orgDepth: number
  name: string
  code: string
  sort: number
  status: string
  children?: OrganizationItem[]
}

export interface OrganizationUserItem {
  organizationID: string
  userID: string
  userName: string
  username: string
  primaryEmail: string
  primaryPhone: string
  avatar: string
  isSuspended: boolean
  relationType: string
  isPrimary: boolean
  joinedAt?: number
}

export interface UserOrganizationItem {
  organizationID: string
  organizationName: string
  relationType: string
  isPrimary: boolean
}

export interface OrganizationTreeResp {
  list: OrganizationItem[]
}

export interface PageListResp<T> {
  list: T[]
  total: number
}
