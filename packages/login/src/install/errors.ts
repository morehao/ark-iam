import { INSTALL_MESSAGES } from './messages'

/**
 * /install 领域错误码与"页面该做什么"的映射。
 *
 * 事实源：backend/pkg/code/install.go（107xxx 段）。要点：与其它领域不同，
 * 这些端点返回**真实 HTTP 状态码**（409/401/503/500/400），因此解码必须同时
 * 覆盖两种形态——
 *   1. axios 因非 2xx 抛错，业务码在 error.response.data（正常路径）；
 *   2. 万一后端用 gincontext.Fail（恒 200）返回，业务码在响应体信封里。
 * 另外后端成功响应可能是 {code,msg,data} 信封、也可能是裸对象（契约示例即裸对象），
 * unwrapEnvelope 对两种形态都兼容。
 */

/** install 领域错误码（后端 pkg/code/install.go）。 */
export const INSTALL_ERROR_CODE = {
  /** 409 已初始化（终态） */
  alreadyInitialized: 107000,
  /** 401 令牌缺失或不匹配 */
  tokenInvalid: 107001,
  /** 503 服务端未配置 BOOTSTRAP_TOKEN */
  tokenNotConfigured: 107002,
  /** 500 初始化执行失败（系统错误） */
  initializeFailed: 107003,
  /** 409 守卫：系统尚未初始化（业务接口返回，前端跳 /install） */
  pending: 107004,
  /** 400 入参非法 */
  badRequest: 107005,
  /** 400 口令强度不足 */
  passwordWeak: 107006,
} as const

/** 错误码对应的展示位置：form=表单级横幅；其余为字段级。 */
export type InstallErrorField = 'form' | 'bootstrapToken' | 'adminPassword'

export interface InstallErrorBehavior {
  /** 归一后的业务码；0 表示无法识别（按未知系统错误兜底） */
  code: number
  message: string
  field: InstallErrorField
  /** 已初始化终态：切到"已初始化"屏并展示控制台入口，不再允许提交 */
  alreadyInitialized: boolean
  /** 系统尚未初始化：路由跳回 /install */
  redirectToInstall: boolean
  /** 服务端未配置 BOOTSTRAP_TOKEN：提交按钮置灰（重试无意义） */
  disableSubmit: boolean
  /** 是否命中已知 /install 错误码 */
  known: boolean
}

/** 错误码 -> 默认文案。终态/环境类错误用我方可执行文案；后端 msg 更具体的码见 SERVER_MESSAGE_CODES。 */
const DEFAULT_MESSAGE_BY_CODE: Record<number, string> = {
  [INSTALL_ERROR_CODE.alreadyInitialized]: INSTALL_MESSAGES.alreadyInitialized,
  [INSTALL_ERROR_CODE.tokenInvalid]: INSTALL_MESSAGES.tokenInvalid,
  [INSTALL_ERROR_CODE.tokenNotConfigured]: INSTALL_MESSAGES.tokenNotConfigured,
  [INSTALL_ERROR_CODE.initializeFailed]: INSTALL_MESSAGES.initializeFailed,
  [INSTALL_ERROR_CODE.pending]: INSTALL_MESSAGES.pending,
  [INSTALL_ERROR_CODE.badRequest]: INSTALL_MESSAGES.badRequest,
  [INSTALL_ERROR_CODE.passwordWeak]: INSTALL_MESSAGES.passwordWeak,
}

/**
 * 这些码的后端 msg 比前端兜底文案更具体（如"用户名不合法"），优先回显；
 * 其余码（终态/环境类）必须用前端可执行文案，否则会退化成"只描述现象、不给出下一步"。
 */
const SERVER_MESSAGE_CODES = new Set<number>([
  INSTALL_ERROR_CODE.tokenInvalid,
  INSTALL_ERROR_CODE.badRequest,
  INSTALL_ERROR_CODE.passwordWeak,
])

/** 业务码 -> 页面处置。 */
export function describeInstallError(code: number, serverMsg?: string): InstallErrorBehavior {
  const known = Object.prototype.hasOwnProperty.call(DEFAULT_MESSAGE_BY_CODE, code)
  const server = (serverMsg ?? '').trim()
  const fallback = DEFAULT_MESSAGE_BY_CODE[code]
  let message: string
  if (SERVER_MESSAGE_CODES.has(code) && server) {
    message = server
  } else if (fallback) {
    message = fallback
  } else {
    message = server || INSTALL_MESSAGES.unknown
  }
  return {
    code,
    message,
    field:
      code === INSTALL_ERROR_CODE.tokenInvalid
        ? 'bootstrapToken'
        : code === INSTALL_ERROR_CODE.passwordWeak
          ? 'adminPassword'
          : 'form',
    alreadyInitialized: code === INSTALL_ERROR_CODE.alreadyInitialized,
    redirectToInstall: code === INSTALL_ERROR_CODE.pending,
    disableSubmit: code === INSTALL_ERROR_CODE.tokenNotConfigured,
    known,
  }
}

/** 无响应体业务码时，按 HTTP 状态码兜底。 */
export function codeFromHTTPStatus(status?: number): number | undefined {
  switch (status) {
    case 401:
      return INSTALL_ERROR_CODE.tokenInvalid
    case 503:
      return INSTALL_ERROR_CODE.tokenNotConfigured
    case 500:
      return INSTALL_ERROR_CODE.initializeFailed
    case 400:
      return INSTALL_ERROR_CODE.badRequest
    // 409 有两义（107000 已初始化 / 107004 未初始化）。缺少响应体时按"已初始化"保守处置：
    // 停在终态屏并给出控制台入口，比把一个可能已初始化的系统推回表单更安全。
    case 409:
      return INSTALL_ERROR_CODE.alreadyInitialized
    default:
      return undefined
  }
}

function asRecord(value: unknown): Record<string, unknown> | undefined {
  return typeof value === 'object' && value !== null ? (value as Record<string, unknown>) : undefined
}

function toNumber(value: unknown): number | undefined {
  if (typeof value === 'number' && Number.isFinite(value)) return value
  if (typeof value === 'string' && value.trim() !== '') {
    const parsed = Number(value)
    if (Number.isFinite(parsed)) return parsed
  }
  return undefined
}

function pickMessage(...candidates: unknown[]): string {
  for (const candidate of candidates) {
    if (typeof candidate === 'string' && candidate.trim() !== '') return candidate.trim()
  }
  return ''
}

/**
 * 从统一信封解码业务码/文案，兼容三种形态：
 *   顶层 {code,msg}（golib gincontext 的标准形态）、
 *   嵌套 {data:{code,msg}} / {error:{code,msg}}。
 */
export function decodeErrorEnvelope(payload: unknown): { code: number; message: string } | undefined {
  const record = asRecord(payload)
  if (!record) return undefined
  const candidates = [record, asRecord(record.data), asRecord(record.error)]
  for (const candidate of candidates) {
    if (!candidate) continue
    const code = toNumber(candidate.code)
    if (code === undefined) continue
    return { code, message: pickMessage(candidate.msg, candidate.message, record.msg, record.message) }
  }
  return undefined
}

/** 把任意异常（axios HTTP 错误 / ApiError / 网络错误）归一为页面处置。 */
export function resolveInstallError(error: unknown): InstallErrorBehavior {  const record = asRecord(error)
  const response = asRecord(record?.response)
  const decoded =
    decodeErrorEnvelope(response?.data) ?? decodeErrorEnvelope(record?.data) ?? decodeErrorEnvelope(record)
  const code = decoded?.code ?? codeFromHTTPStatus(toNumber(response?.status))
  if (code === undefined || code === 0) {
    return {
      code: 0,
      message: decoded?.message || INSTALL_MESSAGES.unknown,
      field: 'form',
      alreadyInitialized: false,
      redirectToInstall: false,
      disableSubmit: false,
      known: false,
    }
  }
  return describeInstallError(code, decoded?.message)
}

/**
 * GET /install/status 失败时的用户可见文案。
 *
 * 与 initialize 不同：状态查询的 503 是"服务暂时给不出状态"（后端为系统错误，code=-1），
 * 不是 107002「未配置 BOOTSTRAP_TOKEN」。因此这里**不按状态码映射业务码**，只取信封里的
 * 后端文案；拿不到（网络错误 / axios 英文原文）时给本页可执行提示，绝不回显英文报错。
 */
export function statusUnavailableMessage(error: unknown): string {
  const record = asRecord(error)
  const response = asRecord(record?.response)
  const decoded = decodeErrorEnvelope(response?.data) ?? decodeErrorEnvelope(record?.data)
  return decoded?.message || INSTALL_MESSAGES.statusUnavailable
}

export type UnwrapResult<T> = { ok: true; data: T } | { ok: false; code: number; message: string }/**
 * 解包成功响应：{code:0,data} 信封 -> data；裸对象 -> 原样返回。
 * code!==0（后端用恒 200 的 Fail 返回时）-> 失败，交调用方按错误码处置。
 */
export function unwrapEnvelope<T>(body: unknown): UnwrapResult<T> {
  const record = asRecord(body)
  if (record && 'code' in record) {
    const code = toNumber(record.code)
    if (code === undefined || code !== 0) {
      return { ok: false, code: code ?? 0, message: pickMessage(record.msg, record.message) }
    }
    return { ok: true, data: record.data as T }
  }
  return { ok: true, data: body as T }
}
