export interface ApiResponse<T> {
  code: number
  msg: string
  data: T
}

export enum BizCode {
  Success = 0,
  Unauthorized = 110000,
  Forbidden = 110001,
  TokenInvalid = 110002,
  TokenExpired = 110003,
  PermissionDenied = 110004,
  ParamInvalid = 100104,
}
