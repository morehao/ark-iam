// 部门节点状态（后端具名类型 model.DeptNodeStatus）：全局启停语义 enable/disable。
export type DeptNodeStatus = 'enable' | 'disable'

export interface DepartmentItem {
  departmentID: string
  parentID: string
  deptPath: string
  deptDepth: number
  name: string
  sort: number
  status: DeptNodeStatus
  createdAt?: number
  children?: DepartmentItem[]
}

export interface DepartmentChildItem {
  departmentID: string
  parentID: string
  deptDepth: number
  name: string
  sort: number
  status: DeptNodeStatus
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
