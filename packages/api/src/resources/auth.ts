import request from '../request'
import type {
  MyTenantsResp,
  PersonDetailResp,
  PersonUpdatePasswordReq,
  SessionListResp,
  UserinfoResp,
} from '@ark-iam/types'

export const getUserinfo = () => request.get<any, UserinfoResp>('/auth/userinfo')
export const getMyTenants = () => request.get<any, MyTenantsResp>('/auth/me/tenants')
export const logoutAPI = (refreshToken: string) => request.post<any, void>('/auth/logout', { refreshToken })
export const logoutAllAPI = (refreshToken: string) => request.post<any, void>('/auth/logoutAll', { refreshToken })
export const getPersonDetail = () => request.get<any, PersonDetailResp>('/auth/me')
export const updatePassword = (req: PersonUpdatePasswordReq) => request.post<any, void>('/auth/me/changePassword', req)
export const getSessionList = (params?: { page?: number; pageSize?: number }) =>
  request.get<any, SessionListResp>('/auth/me/sessions', { params })
export const revokeSession = (sessionID: string) => request.delete<any, void>(`/auth/me/sessions/${sessionID}`)
export const revokeAllSessions = () => request.delete<any, void>('/auth/me/sessions')
