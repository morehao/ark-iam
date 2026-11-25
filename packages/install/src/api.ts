/**
 * /install/* 协议调用（从 @ark-iam/login 拆出，与账密页解耦）。
 *
 * 依赖方向：install → login（复用 login 的运行时 base 解析 getRuntimeConfig 与 ApiError），
 * login 不反向依赖本包。错误映射见 ./errors，契约类型见 ./types。
 */
import axios, { type InternalAxiosRequestConfig } from 'axios'
import { ApiError, getRuntimeConfig } from '@ark-iam/login'
import { describeInstallError, resolveInstallError, unwrapEnvelope } from './errors'
import type { InstallInitializeReq, InstallInitializeResp, InstallStatus } from './types'

/**
 * 在请求前注入运行时 baseURL（内嵌形态前缀自适应）。
 *
 * 构建期无法确定挂载前缀，故每次请求从 `getRuntimeConfig()` 取后端下发的 installBase；
 * 若运行时配置尚未落定，回退到实例自身的 baseURL（旧根行为）。
 */
function applyDynamicBase(getBase: () => string) {
  return (config: InternalAxiosRequestConfig): InternalAxiosRequestConfig => {
    const base = getBase()
    if (base) config.baseURL = base
    return config
  }
}

/** /install 调用（baseURL 由 config.json 的 installBase 运行时下发；公开只读 status + 唯一写端点 initialize）。 */
export const installApi = axios.create({ timeout: 15000 })
installApi.interceptors.request.use(
  applyDynamicBase(() => getRuntimeConfig()?.installBase || '/install'),
)

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
