import { request } from '@ark-iam/api'

export interface User {
  id: number
  username: string
  name: string
  email: string
  phone: string
  status: string
  createdAt: string
}

export interface UserPageListReq {
  page: number
  pageSize: number
  keyword?: string
}

export interface UserPageListResp {
  list: User[]
  total: number
}

export const getUserPageList = (data: UserPageListReq) => {
  return request.post<UserPageListReq, UserPageListResp>('/user/pageList', data)
}

export const getUserDetail = (id: number) => {
  return request.get<any, User>('/user/detail', { params: { userID: id } })
}