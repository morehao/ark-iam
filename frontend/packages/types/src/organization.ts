export interface OrganizationItem {
  organizationID: string
  parentID: string
  orgPath: string
  orgDepth: number
  name: string
  code: string
  sort: number
  status: string
  createdAt?: number
  children?: OrganizationItem[]
}

export interface OrganizationChildItem {
  organizationID: string
  parentID: string
  orgDepth: number
  name: string
  code: string
  sort: number
  status: string
  createdAt?: number
  hasChildren?: boolean
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
  joinedAt?: number
}

export interface UserOrganizationItem {
  organizationID: string
  organizationName: string
  relationType: string
}

export interface OrganizationTreeResp {
  list: OrganizationItem[]
}

export interface OrganizationChildrenResp {
  list: OrganizationChildItem[]
  total: number
}

export interface PageListResp<T> {
  list: T[]
  total: number
}
