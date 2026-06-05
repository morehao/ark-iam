export interface OIDCLoginReq {
  authRequestID: string
  identifier: string
  password: string
}

export interface OIDCLoginResp {
  continueURL: string
}
