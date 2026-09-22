import axios from 'axios'
import type {
  ApiResponse,
  OIDCLoginReq,
  OIDCLoginResp,
  OIDCSelectTenantReq,
  RegisterPersonReq,
  RegisterPersonResp,
  CreateTenantReq,
  CreateTenantResp,
  OIDCLoginConfigReq,
  OIDCLoginConfigResp,
  OIDCChangePasswordReq,
} from './types'
import { INSTALL_ERROR_CODE, describeInstallError, resolveInstallError, unwrapEnvelope } from './install/errors'
import type { InstallInitializeReq, InstallInitializeResp, InstallStatus } from './install/types'

/**
 * 携带后端业务码的 API 错误。
 *
 * 为什么需要它：业务失败后端恒返回 HTTP 200 + 信封 code，原先把错误压成裸 Error 会丢掉 code，
 * 登录侧就无法识别 107004（系统尚未初始化守卫）并跳 /install。展示文案仍走 message，
 * 既有 `err?.message` 调用点行为不变。
 */
export class ApiError extends Error {
  readonly code: number

  constructor(code: number, message: string) {
    super(message)
    this.name = 'ApiError'
    this.code = code
  }
}

/** 107004：业务接口在系统未初始化时的守卫错误 -> 必须把用户送回 /install。 */
export const INSTALL_PENDING_REDIRECT = '/install'

/** 是否命中"系统尚未初始化"守卫错误。 */
export function isInstallPendingError(err: unknown): boolean {
  return err instanceof ApiError && err.code === INSTALL_ERROR_CODE.pending
}

/** 需要跳转 /install 时返回目标路径，否则返回 null（便于调用点与测试复用同一判定）。 */
export function installRedirectForError(err: unknown): string | null {
  return isInstallPendingError(err) ? INSTALL_PENDING_REDIRECT : null
}

/**
 * 把"非 2xx 且响应体带业务码"的 axios 错误归一成 ApiError。
 *
 * 为什么必须有这一步：`/oidc` 与 `/install` 下**并非所有**失败都是「HTTP 200 + 信封 code」。
 * 未初始化守卫（`BootstrapGuard`）对业务端点返回的是**真实 HTTP 409** + 信封
 * `{"code":107004,...}`——axios 会把非 2xx 直接 reject，函数体里那句 `body.code !== 0`
 * 根本没机会执行，于是错误退化成 axios 的英文 "Request failed with status code 409"，
 * `err.code` 也不存在，登录页的 107004 跳转判定永远不成立（这正是本文件曾经的真实缺陷）。
 * 归一后：所有调用点都稳定拿到 `ApiError.code`，`err.message` 也变成后端的中文文案
 * （比 axios 英文原文更适合直接展示），既有 `err?.message` 调用点行为不变。
 */
function normalizeAxiosError(error: unknown): unknown {
  const response = (error as { response?: { data?: unknown } } | undefined)?.response
  const body = response?.data as { code?: unknown; msg?: unknown } | undefined
  if (body && typeof body.code === 'number' && body.code !== 0) {
    const message = typeof body.msg === 'string' && body.msg ? body.msg : '请求失败，请重试'
    return new ApiError(body.code, message)
  }
  return error
}

/** login-web 的 OIDC 凭证调用（baseURL /oidc，走 Vite 代理到 gateway）。 */
export const oidcApi = axios.create({
  baseURL: '/oidc',
  timeout: 10000,
})
// 守卫返回真实 409（见 normalizeAxiosError 注释），必须归一后才能识别 code=107004
oidcApi.interceptors.response.use((resp) => resp, (error) => Promise.reject(normalizeAxiosError(error)))

/** /install 调用（baseURL /install，同一代理机制；公开只读 status + 唯一写端点 initialize）。 */
export const installApi = axios.create({
  baseURL: '/install',
  timeout: 15000,
})

// 后端统一响应信封：业务失败也返回 HTTP 200，需按 code 判断
export async function oidcLogin(data: OIDCLoginReq): Promise<OIDCLoginResp> {
  const resp = await oidcApi.post<ApiResponse<OIDCLoginResp>>('/login', data)
  const body = resp.data
  if (body.code !== 0) {
    throw new ApiError(body.code, body.msg || '登录失败，请重试')
  }
  return body.data
}

export async function oidcSelectTenant(data: OIDCSelectTenantReq): Promise<OIDCLoginResp> {
  const resp = await oidcApi.post<ApiResponse<OIDCLoginResp>>('/login/selectTenant', data)
  const body = resp.data
  if (body.code !== 0) {
    throw new ApiError(body.code, body.msg || '选择租户失败，请重试')
  }
  return body.data
}

/** 首次登录强制改密（临时密码）：改密成功后需重新登录（会话已全局撤销）。 */
export async function oidcChangePassword(data: OIDCChangePasswordReq): Promise<void> {
  const resp = await oidcApi.post<ApiResponse<string>>('/login/changePassword', data)
  const body = resp.data
  if (body.code !== 0) {
    throw new ApiError(body.code, body.msg || '修改密码失败，请重试')
  }
}

export async function registerPerson(data: RegisterPersonReq): Promise<RegisterPersonResp> {
  const resp = await oidcApi.post<ApiResponse<RegisterPersonResp>>('/registerPerson', data)
  const body = resp.data
  if (body.code !== 0) {
    throw new ApiError(body.code, body.msg || '注册失败，请重试')
  }
  return body.data
}

export async function createTenant(data: CreateTenantReq): Promise<CreateTenantResp> {
  const resp = await oidcApi.post<ApiResponse<CreateTenantResp>>('/createTenant', data)
  const body = resp.data
  if (body.code !== 0) {
    throw new ApiError(body.code, body.msg || '创建租户失败，请重试')
  }
  return body.data
}

export async function getLoginConfig(data: OIDCLoginConfigReq): Promise<OIDCLoginConfigResp> {
  const resp = await oidcApi.post<ApiResponse<OIDCLoginConfigResp>>('/login-config', data)
  const body = resp.data
  if (body.code !== 0) {
    throw new ApiError(body.code, body.msg || '获取登录配置失败，请重试')
  }
  return body.data
}

/**
 * GET /install/status（公开只读）。响应可能是 {code,msg,data} 信封，也可能是裸对象，
 * unwrapEnvelope 两种都兼容。
 */
export async function getInstallStatus(): Promise<InstallStatus> {
  const resp = await installApi.get<unknown>('/status')
  const unwrapped = unwrapEnvelope<InstallStatus>(resp.data)
  if (!unwrapped.ok) {
    const behavior = describeInstallError(unwrapped.code, unwrapped.message)
    throw new ApiError(behavior.code, behavior.message)
  }
  return unwrapped.data
}

/**
 * POST /install/initialize —— 唯一写端点，bootstrap token 经 X-Bootstrap-Token 头传递。
 * 错误按真实 HTTP 状态码返回，resolveInstallError 同时覆盖 HTTP 错误与恒 200 信封两种形态。
 */
export async function initializeInstall(
  data: InstallInitializeReq,
  bootstrapToken: string,
): Promise<InstallInitializeResp> {
  let resp
  try {
    resp = await installApi.post<unknown>('/initialize', data, {
      // 与后端 svcinstall.trimToken 对称归一：部署脚本注入/页面粘贴的令牌常带尾随换行或空格，
      // 不归一会被判成 401「令牌不正确」，而肉眼完全看不出差异。
      headers: { 'X-Bootstrap-Token': bootstrapToken.trim() },
    })
  } catch (err) {
    if (err instanceof ApiError) throw err
    const behavior = resolveInstallError(err)
    throw new ApiError(behavior.code, behavior.message)
  }
  const unwrapped = unwrapEnvelope<InstallInitializeResp>(resp.data)
  if (!unwrapped.ok) {
    const behavior = describeInstallError(unwrapped.code, unwrapped.message)
    throw new ApiError(behavior.code, behavior.message)
  }
  return unwrapped.data
}
