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
  relationType: string
  isPrimary: boolean
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
