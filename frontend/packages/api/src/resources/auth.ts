import request from '../request'
import type {
  MyTenantsResp,
  PersonDetailResp,
  PersonUpdatePasswordReq,
  SessionListResp,
  UserinfoResp,
} from '@ark-iam/types'

export const getUserinfo = () => request.get<any, UserinfoResp>('/auth/userinfo')
export const getMyTenants = () => request.get<any, MyTenantsResp>('/auth/myTenants')
export const logoutAPI = (refreshToken: string) => request.post<any, void>('/auth/logout', { refreshToken })
export const logoutAllAPI = (refreshToken: string) => request.post<any, void>('/auth/logoutAll', { refreshToken })
export const getPersonDetail = () => request.get<any, PersonDetailResp>('/person/detail')
export const updatePassword = (req: PersonUpdatePasswordReq) => request.post<any, void>('/person/updatePassword', req)
export const getSessionList = (params?: { page?: number; pageSize?: number }) =>
  request.get<any, SessionListResp>('/user/sessions', { params })
export const revokeSession = (sessionId: number) => request.delete<any, void>(`/user/sessions/${sessionId}`)
export const revokeAllSessions = () => request.delete<any, void>('/user/sessions')
