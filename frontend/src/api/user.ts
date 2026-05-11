import request from '../utils/request'

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

export const getUserPageList = (data: UserPageListReq) => {
  return request.post<any, any>('/user/pageList', data)
}

export const getUserDetail = (id: number) => {
  return request.get<any, any>('/user/detail', { params: { userID: id } })
}