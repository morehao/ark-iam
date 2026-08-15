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
export const getPersonDetail = () => request.get<any, PersonDetailResp>('/auth/person/detail')
export const updatePassword = (req: PersonUpdatePasswordReq) => request.post<any, void>('/auth/person/updatePassword', req)
export const getSessionList = (params?: { page?: number; pageSize?: number }) =>
  request.get<any, SessionListResp>('/auth/user/sessions', { params })
export const revokeSession = (sessionID: number) => request.delete<any, void>(`/auth/user/sessions/${sessionID}`)
export const revokeAllSessions = () => request.delete<any, void>('/auth/user/sessions')
