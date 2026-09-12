export interface DepartmentItem {
  departmentID: string
  parentID: string
  deptPath: string
  deptDepth: number
  name: string
  code: string
  sort: number
  status: string
  createdAt?: number
  children?: DepartmentItem[]
}

export interface DepartmentChildItem {
  departmentID: string
  parentID: string
  deptDepth: number
  name: string
  code: string
  sort: number
  status: string
  createdAt?: number
  updatedAt?: number
  hasChildren?: boolean
}

export interface DepartmentUserItem {
  departmentID: string
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

export interface UserDepartmentItem {
  departmentID: string
  departmentName: string
  relationType: string
}

export interface DepartmentTreeResp {
  list: DepartmentItem[]
}

export interface DepartmentChildrenResp {
  list: DepartmentChildItem[]
  total: number
}

export interface PageListResp<T> {
  list: T[]
  total: number
}
